package write

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const LedgerCap = 1 << 20

type Entry struct {
	At        time.Time `json:"at"`
	Context   string    `json:"context"`
	Level     string    `json:"level"`
	Order     string    `json:"order"`
	Object    string    `json:"object"`
	Code      int       `json:"code"`
	Answer    string    `json:"answer"`
	Container string    `json:"container,omitempty"`
	Command   string    `json:"command,omitempty"`
	Ended     string    `json:"ended,omitempty"`
}

type Ledger struct {
	mu   sync.Mutex
	path string
	cap  int64
}

func ContextKey(c Context) string {
	sum := sha256.Sum256([]byte(c.Name + "\x00" + c.Server + "\x00" + c.User))
	return hex.EncodeToString(sum[:])[:12]
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func LedgerFile(dir string, c Context) string {
	if dir == "" {
		return ""
	}
	name := unsafeName.ReplaceAllString(c.Name, "_")
	if len(name) > 48 {
		name = name[:48]
	}
	return filepath.Join(dir, "live-ledger-"+name+"-"+ContextKey(c)+".jsonl")
}

func OpenLedger(dir string, c Context) *Ledger {
	return &Ledger{path: LedgerFile(dir, c), cap: LedgerCap}
}

func (l *Ledger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

func (l *Ledger) Append(e Entry) error {
	if l == nil || l.path == "" {
		return nil
	}
	if SecretWrite(e.Object, e.Order) {
		e.Order, e.Command = SecretCommand(e.Order), SecretCommand(e.Command)
	}
	e.Order, e.Answer, e.Object, e.Command = Mask(e.Order), Mask(e.Answer), Mask(e.Object), Mask(e.Command)
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, werr := f.Write(append(line, '\n'))
	cerr := f.Close()
	if err := errors.Join(werr, cerr); err != nil {
		return err
	}
	info, err := os.Stat(l.path)
	if err != nil || info.Size() <= l.cap {
		return err
	}
	return l.halve()
}

func (l *Ledger) halve() error {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return err
	}
	lines := bytes.SplitAfter(data, []byte("\n"))
	if n := len(lines); n > 0 && len(lines[n-1]) == 0 {
		lines = lines[:n-1]
	}
	keep := bytes.Join(lines[len(lines)/2:], nil)
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, keep, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, l.path)
}

func (l *Ledger) Read() ([]Entry, error) {
	if l == nil || l.path == "" {
		return nil, nil
	}
	f, err := os.Open(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for sc.Scan() {
		var e Entry
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			out = append(out, e)
		}
	}
	return out, sc.Err()
}

const Masked = "****"

var (
	maskedFlags     = regexp.MustCompile(`(--(?:docker-password|password|token|client-key-data|client-certificate-data)[= ])("[^"]*"|'[^']*'|\S+)`)
	literalFlag     = regexp.MustCompile(`(--from-literal[= ](?:"|')?[^=\s"']+=)("[^"]*"|'[^']*'|[^\s"']+)`)
	secretValues    = regexp.MustCompile(`("(?:data|stringData)"\s*:\s*\{)([^{}]*)(\})`)
	jsonValue       = regexp.MustCompile(`("[^"\\]*(?:\\.[^"\\]*)*"\s*:\s*)("[^"\\]*(?:\\.[^"\\]*)*"|[^,}\s]+)`)
	yamlData        = regexp.MustCompile(`((?:^|\n)\s*(?:data|stringData):\s*\{)([^{}]*)(\})`)
	patchOp         = regexp.MustCompile(`\{(?:[^{}]|\{[^{}]*\})*\}`)
	secretPatchPath = regexp.MustCompile(`"path"\s*:\s*"/(?:data|stringData)(?:/|")`)
	patchValue      = regexp.MustCompile(`("value"\s*:\s*)("[^"\\]*(?:\\.[^"\\]*)*"|\{[^{}]*\}|[^,}\s]+)`)
)

var (
	valueFlag  = regexp.MustCompile(`((?:^|\s)(?:-p|--patch|--from-literal|--from-file|--from-env-file)(?:=|\s+))('[^']*'|"(?:[^"\\]|\\.)*"|(?:\\.|[^\s'"])+)`)
	secretWord = regexp.MustCompile(`(?i)(?:^|[\s,/])secrets?(?:$|[\s,/.])`)
)

func SecretWrite(object, command string) bool {
	return strings.HasPrefix(object, "secret ") || strings.HasPrefix(object, "secret/") || secretWord.MatchString(command)
}

func SecretCommand(command string) string {
	return valueFlag.ReplaceAllStringFunc(command, func(m string) string {
		parts := valueFlag.FindStringSubmatch(m)
		flag, value := parts[1], parts[2]
		if strings.Contains(flag, "--from-") {
			plain := strings.Trim(value, `'"`)
			if key, _, ok := strings.Cut(plain, "="); ok && !strings.ContainsAny(key, " \\") {
				return flag + key + "=" + Masked
			}
		}
		return flag + Masked
	})
}

func Mask(s string) string {
	masked := mask(s)
	plain := strings.ReplaceAll(masked, `\"`, `"`)
	if plain == masked {
		return masked
	}
	if unescaped := mask(plain); unescaped != plain {
		return unescaped
	}
	return masked
}

func mask(s string) string {
	s = maskedFlags.ReplaceAllString(s, "${1}"+Masked)
	s = literalFlag.ReplaceAllString(s, "${1}"+Masked)
	values := func(m string, re *regexp.Regexp) string {
		parts := re.FindStringSubmatch(m)
		return parts[1] + jsonValue.ReplaceAllString(parts[2], `${1}"`+Masked+`"`) + parts[3]
	}
	s = patchOp.ReplaceAllStringFunc(s, func(op string) string {
		if !secretPatchPath.MatchString(op) {
			return op
		}
		return patchValue.ReplaceAllString(op, `${1}"`+Masked+`"`)
	})
	s = secretValues.ReplaceAllStringFunc(s, func(m string) string { return values(m, secretValues) })
	s = yamlData.ReplaceAllStringFunc(s, func(m string) string { return values(m, yamlData) })
	return strings.ToValidUTF8(s, "?")
}
