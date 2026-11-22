package write

import (
	"encoding/json"
	"math/rand/v2"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const userinfoBytes = "ABCXYZabcxyz0189-._:~!$&'()*+,;=%@"

func TestRedactDropsTheUserinfoOfEveryURLWhateverItHolds(t *testing.T) {
	rng := rand.New(rand.NewPCG(4, 34))
	tried := 0
	for range 20000 {
		n := 1 + rng.IntN(24)
		var b strings.Builder
		for range n {
			b.WriteByte(userinfoBytes[rng.IntN(len(userinfoBytes))])
		}
		info := b.String()
		for _, tail := range []string{"", "/api/v1/pods", "?watch=1", "#top", "\": EOF", "' refused", " and more"} {
			raw := "https://" + info + "@prod.example:6443" + tail
			u, err := url.Parse("https://" + info + "@prod.example:6443")
			if err != nil || u.User == nil {
				continue
			}
			tried++
			got := Redact("Get " + raw)
			if want := "Get https://prod.example:6443" + tail; got != want {
				t.Fatalf("Redact(%q) = %q, want %q", "Get "+raw, got, want)
			}
		}
	}
	if tried < 10000 {
		t.Fatalf("only %d addresses parsed", tried)
	}
	for in, want := range map[string]string{
		"https://alice:pa'ss@prod.example:6443/api":   "https://prod.example:6443/api",
		"alice:pa'ss@prod.example:6443":               "prod.example:6443",
		"alice@prod.example":                          "prod.example",
		"https://prod.example:6443":                   "https://prod.example:6443",
		"prod.example:6443":                           "prod.example:6443",
		"https://u:p@[::1]:6443":                      "https://[::1]:6443",
		"https://u:p@h@prod.example:6443/a@b?c=d@e#f": "https://prod.example:6443/a@b?c=d@e#f",
	} {
		if got := RedactServer(in); got != want {
			t.Errorf("RedactServer(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAChoicesFileWrittenWithAServersUserinfoIsRewrittenWithoutIt(t *testing.T) {
	dir := t.TempDir()
	old := Memory{Contexts: []Remembered{
		{Context: Context{Name: "prod", Server: "https://alice:SERVER'PW@prod.example:6443", User: "admin", Identity: "id"}, Level: Writes, Warned: Writes},
		{Context: Context{Name: "bare", Server: "bob:BARE-PW@bare.example:6443", User: "admin", Identity: "id2"}, Level: Seamless, Warned: Seamless},
		{Context: Context{Name: "kind", Server: "https://127.0.0.1:6443", User: "kind", Identity: "id3"}, Level: Writes, Warned: Writes},
	}}
	data, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(MemoryPath(dir), data, 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := LoadMemory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Contexts) != 3 {
		t.Fatalf("the choices %+v", m.Contexts)
	}
	disk, err := os.ReadFile(MemoryPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"alice", "SERVER", "bob", "BARE-PW"} {
		if strings.Contains(string(disk), secret) {
			t.Errorf("the choices file still holds %q:\n%s", secret, disk)
		}
		for _, r := range m.Contexts {
			if strings.Contains(r.Context.Server, secret) {
				t.Errorf("the remembered %s keeps %q", r.Context.Name, r.Context.Server)
			}
		}
	}
	if r, _, ok := m.Recall(Context{Name: "prod", Server: "https://prod.example:6443", User: "admin", Identity: "id"}); !ok || r.Level != Writes {
		t.Errorf("the rewritten choice is not recalled for the same identity: %+v %v", r, ok)
	}
	ledger := OpenLedger(dir, Context{Name: "prod", Server: "https://prod.example:6443", User: "admin"})
	if err := ledger.Append(Entry{At: time.Unix(0, 0), Context: "prod", Level: "WRITES", Order: "kubectl get pods", Object: "pod shop/web", Answer: `Get "https://alice:SERVER'PW@prod.example:6443/api": EOF`}); err != nil {
		t.Fatal(err)
	}
	if line, _ := os.ReadFile(ledger.Path()); strings.Contains(string(line), "SERVER") {
		t.Errorf("the ledger holds a server's userinfo:\n%s", line)
	}
}
