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
	"slices"
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
	key := c.Name + "\x00" + c.Server + "\x00" + c.User
	if c.ClusterID != "" {
		key += "\x00" + c.ClusterID
	}
	sum := sha256.Sum256([]byte(key))
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
	if order, ok := ProgramOnly(e.Order); ok {
		e.Order, e.Command = order, firstWord(e.Command)
	}
	e.Order, e.Answer, e.Object, e.Command = Mask(e.Order), Mask(e.Answer), Mask(e.Object), Mask(e.Command)
	e.Order, e.Answer, e.Object, e.Command, e.Ended = Redact(e.Order), Redact(e.Answer), Redact(e.Object), Redact(e.Command), Redact(e.Ended)
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := PrivateDir(filepath.Dir(l.path)); err != nil {
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
	maskedFlags     = regexp.MustCompile(`(--(?:docker[-_]password|password|token|client[-_]key[-_]data|client[-_]certificate[-_]data)[= ])("[^"]*"|'[^']*'|\S+)`)
	literalFlag     = regexp.MustCompile(`(--from[-_]literal[= ](?:"|')?[^=\s"']+=)("[^"]*"|'[^']*'|[^\s"']+)`)
	secretValues    = regexp.MustCompile(`("(?:data|stringData)"\s*:\s*\{)([^{}]*)(\})`)
	jsonValue       = regexp.MustCompile(`("[^"\\]*(?:\\.[^"\\]*)*"\s*:\s*)("[^"\\]*(?:\\.[^"\\]*)*"|[^,}\s]+)`)
	yamlData        = regexp.MustCompile(`((?:^|\n)\s*(?:data|stringData):\s*\{)([^{}]*)(\})`)
	patchOp         = regexp.MustCompile(`\{(?:[^{}]|\{[^{}]*\})*\}`)
	secretPatchPath = regexp.MustCompile(`"path"\s*:\s*"/(?:data|stringData)(?:/|")`)
	patchValue      = regexp.MustCompile(`("value"\s*:\s*)("[^"\\]*(?:\\.[^"\\]*)*"|\{[^{}]*\}|[^,}\s]+)`)
)

var secretWord = regexp.MustCompile(`(?i)(?:^|[\s,/])secrets?(?:$|[\s,/.])`)

func SecretWrite(object, command string) bool {
	return strings.HasPrefix(object, "secret ") || strings.HasPrefix(object, "secret/") || secretWord.MatchString(command)
}

var secretValueFlags = map[string]bool{
	"--patch": true, "--from-literal": true, "--from-file": true, "--from-env-file": true,
	"--docker-password": true, "--password": true, "--token": true,
}

var keyedValues = map[string]bool{"--from-literal": true, "--from-file": true}

const valueShorthands = "nsvpfkolLce"

func SecretCommand(command string) string {
	words, err := Words(command)
	if err != nil {
		return Masked
	}
	out := make([]string, 0, len(words))
	pending := ""
	positional := false
	for _, w := range words {
		switch {
		case pending != "":
			out = append(out, maskedValue(pending, w))
			pending = ""
		case positional || w == "-" || !strings.HasPrefix(w, "-"):
			out = append(out, w)
		case w == "--":
			positional = true
			out = append(out, w)
		case strings.HasPrefix(w, "--"):
			flag, value, attached := strings.Cut(w, "=")
			name := FlagName(flag)
			switch {
			case !secretValueFlags[name]:
				out = append(out, w)
			case attached:
				out = append(out, flag+"="+maskedValue(name, value))
			default:
				pending = name
				out = append(out, w)
			}
		default:
			word, next := shorthands(w)
			out = append(out, word)
			pending = next
		}
	}
	for i, w := range out {
		out[i] = quoteWord(w)
	}
	return strings.Join(out, " ")
}

func FlagName(flag string) string {
	if !strings.HasPrefix(flag, "--") {
		return flag
	}
	return "--" + strings.ReplaceAll(flag[2:], "_", "-")
}

func shorthands(w string) (string, string) {
	for i := 1; i < len(w); i++ {
		c := w[i]
		if !strings.ContainsRune(valueShorthands, rune(c)) {
			continue
		}
		rest := w[i+1:]
		if rest == "" {
			if c == 'p' {
				return w, "--patch"
			}
			return w, ""
		}
		if c != 'p' {
			return w, ""
		}
		value, eq := strings.CutPrefix(rest, "=")
		head := w[:i+1]
		if eq {
			head += "="
		}
		return head + maskedValue("--patch", value), ""
	}
	return w, ""
}

func maskedValue(flag, value string) string {
	if keyedValues[flag] {
		if key, _, ok := strings.Cut(value, "="); ok {
			return key + "=" + Masked
		}
	}
	return Masked
}

func quoteWord(w string) string {
	if w != "" && strings.IndexFunc(w, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./:=,@%+*", r))
	}) < 0 {
		return w
	}
	return "'" + strings.ReplaceAll(w, "'", `'\''`) + "'"
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

var runValueFlags = map[string]bool{
	"-n": true, "--namespace": true, "-l": true, "--labels": true, "--image": true, "--restart": true, "--env": true,
	"--port": true, "--overrides": true, "--override-type": true, "--image-pull-policy": true, "-o": true, "--output": true,
	"--template": true, "--timeout": true, "--pod-running-timeout": true, "--grace-period": true, "--field-manager": true,
	"--request-timeout": true, "--annotations": true, "-f": true, "--filename": true, "-v": true, "--v": true,
}

func runVerb(words []string) int {
	for i := 0; i < len(words); i++ {
		w := words[i]
		switch {
		case i == 0 && w == "kubectl":
		case strings.HasPrefix(w, "-"):
			if name, _, attached := strings.Cut(w, "="); !attached && runValueFlags[FlagName(name)] {
				i++
			}
		case w == "run":
			return i
		default:
			return -1
		}
	}
	return -1
}

func runOrder(words []string, verb int) []string {
	out := slices.Clone(words[:verb+1])
	named := false
	for i := verb + 1; i < len(words); i++ {
		w := words[i]
		name, _, attached := strings.Cut(w, "=")
		name = FlagName(name)
		switch {
		case w == "--":
			return append(out, "--", Masked)
		case strings.HasPrefix(w, "-") && name == "--overrides":
			if attached {
				out = append(out, name+"="+Masked)
				continue
			}
			out = append(out, w)
			if i+1 < len(words) {
				out = append(out, Masked)
				i++
			}
		case strings.HasPrefix(w, "-"):
			out = append(out, w)
			if !attached && runValueFlags[name] && i+1 < len(words) {
				i++
				out = append(out, words[i])
			}
		case !named:
			named = true
			out = append(out, w)
		default:
			out = append(out, Masked)
		}
	}
	return out
}

func ProgramOnly(order string) (string, bool) {
	words, err := Words(order)
	if err != nil {
		if fields := strings.Fields(order); runVerb(fields) >= 0 {
			return strings.Join(fields[:runVerb(fields)+1], " ") + " " + Masked, true
		}
		head, _, found := strings.Cut(order, " -- ")
		if !found {
			return order, false
		}
		return head + " -- " + Masked, true
	}
	if verb := runVerb(words); verb >= 0 {
		kept := runOrder(words, verb)
		if slices.Equal(kept, words) {
			return order, false
		}
		for i, w := range kept {
			if w != Masked && !strings.HasSuffix(w, "="+Masked) {
				kept[i] = quoteWord(w)
			}
		}
		return strings.Join(kept, " "), true
	}
	at := slices.Index(words, "--")
	if at < 0 || at+2 >= len(words) {
		return order, false
	}
	kept := slices.Clone(words[:at+2])
	for i, w := range kept {
		kept[i] = quoteWord(w)
	}
	return strings.Join(kept, " "), true
}

func firstWord(command string) string {
	if words, err := Words(command); err == nil {
		if len(words) == 0 {
			return ""
		}
		return quoteWord(words[0])
	}
	if f := strings.Fields(command); len(f) > 0 {
		return f[0]
	}
	return ""
}
