package write

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNoAnswerOrLedgerLineShowsTheUserinfoOfAServerAddress(t *testing.T) {
	err := &url.Error{Op: "Post", URL: "https://alice:SERVER-PW@prod.example:6443/api/v1/namespaces/shop/pods", Err: errors.New("dial tcp: connection refused")}
	if text := (Answer{Err: err}).Text(); strings.Contains(text, "alice") || strings.Contains(text, "SERVER-PW") {
		t.Errorf("a gate answer shows %q", text)
	}
	dir := t.TempDir()
	ctx := Context{Name: "prod", Server: "https://prod.example:6443", User: "admin"}
	l := OpenLedger(dir, ctx)
	if err := l.Append(Entry{At: time.Unix(0, 0), Context: "prod", Level: "WRITES", Order: "kubectl get --server=https://alice:SERVER-PW@x", Object: "pod shop/web", Code: 0, Answer: err.Error()}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(l.Path())
	if strings.Contains(string(data), "SERVER-PW") || strings.Contains(string(data), "alice") {
		t.Errorf("the ledger holds a server's userinfo:\n%s", data)
	}
	for in, want := range map[string]string{
		"Get \"https://alice:***@host:6443/api\": EOF":       "Get \"https://host:6443/api\": EOF",
		"at http://u@[::1]:80/x and https://a:b@c@d/e":       "at http://[::1]:80/x and https://d/e",
		"no address here, only email bob@example.com and :/": "no address here, only email bob@example.com and :/",
	} {
		if got := Redact(in); got != want {
			t.Errorf("Redact(%q) = %q, want %q", in, got, want)
		}
	}
}
