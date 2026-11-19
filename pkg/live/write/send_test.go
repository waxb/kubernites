package write

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/rest"
)

type fakeServer struct {
	mu      sync.Mutex
	seen    []string
	answer  func(r *http.Request, body string) (int, string)
	agents  []string
	options []string
}

func (f *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	line := r.Method + " " + r.URL.Path
	if r.URL.RawQuery != "" {
		line += "?" + r.URL.RawQuery
	}
	f.seen = append(f.seen, line)
	f.agents = append(f.agents, r.UserAgent())
	f.options = append(f.options, r.Header.Get("Content-Type"))
	answer := f.answer
	f.mu.Unlock()
	code, body := http.StatusOK, string(data)
	if answer != nil {
		code, body = answer(r, string(data))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	io.WriteString(w, body)
}

func (f *fakeServer) lines() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.seen...)
}

func bound(t *testing.T, l Level, f *fakeServer, classes ...Class) *Gate {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	g, err := New(Choice{Context: Context{Name: "kind-kubernites", Server: srv.URL, User: "kind-kubernites"}, Level: l, Classes: classes})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Client(&rest.Config{Host: srv.URL}); err != nil {
		t.Fatal(err)
	}
	return g
}

func status(code int, reason, message string) string {
	data, _ := json.Marshal(map[string]any{"kind": "Status", "apiVersion": "v1", "status": "Failure", "code": code, "reason": reason, "message": message})
	return string(data)
}

func TestAtWritesNothingButReviewsAndDryRunsLeaveUntilConfirm(t *testing.T) {
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		if strings.HasSuffix(r.URL.Path, "selfsubjectaccessreviews") {
			return http.StatusCreated, `{"status":{"allowed":true}}`
		}
		if r.URL.Query().Get("dryRun") == "All" {
			return http.StatusOK, `{"kind":"Scale","spec":{"replicas":5}}`
		}
		return http.StatusOK, `{"kind":"Scale","spec":{"replicas":5}}`
	}}
	g := bound(t, Writes, f)
	o := everyOrder(t)["crew-scale"]
	ctx := context.Background()
	if v := g.Review(ctx, o.Access); !v.Allowed || v.Err != nil {
		t.Fatalf("review %+v", v)
	}
	ans, fields := g.Preview(ctx, o)
	if !ans.OK() || len(fields) != 1 || fields[0].Before != "2" || fields[0].After != "5" {
		t.Fatalf("preview %+v %+v", ans, fields)
	}
	if ans := g.Send(ctx, o); !errors.Is(ans.Err, ErrRefused) {
		t.Fatalf("an unconfirmed scale: %+v", ans)
	}
	want := []string{
		"POST /apis/authorization.k8s.io/v1/selfsubjectaccessreviews",
		"PATCH /apis/apps/v1/namespaces/shop/deployments/web/scale?dryRun=All&fieldManager=kubernites",
	}
	if strings.Join(f.lines(), "\n") != strings.Join(want, "\n") {
		t.Fatalf("the server saw\n%s", strings.Join(f.lines(), "\n"))
	}
	g.Issue(o.Request)
	if ans := g.Send(ctx, o); !ans.OK() {
		t.Fatalf("the confirmed scale: %+v", ans)
	}
	if ans := g.Send(ctx, o); !errors.Is(ans.Err, ErrRefused) {
		t.Fatalf("the ticket passed twice: %+v", ans)
	}
	got := f.lines()
	if len(got) != 3 || got[2] != "PATCH /apis/apps/v1/namespaces/shop/deployments/web/scale?fieldManager=kubernites" {
		t.Fatalf("the server saw %v", got)
	}
	for _, a := range f.agents {
		if a != UserAgent(Writes) {
			t.Errorf("user agent %q", a)
		}
	}
	if f.options[1] != MergePatch {
		t.Errorf("content type %q", f.options[1])
	}
}

func TestAtSeamlessEachOrderSendsExactlyOneRequest(t *testing.T) {
	for name, o := range everyOrder(t) {
		f := &fakeServer{}
		g := bound(t, Seamless, f, AllClasses...)
		if ans := g.Send(context.Background(), o); !ans.OK() {
			t.Fatalf("%s: %+v", name, ans)
		}
		got := f.lines()
		want := o.Request.Method + " " + o.Request.Path
		if q := o.Request.Query.Encode(); q != "" {
			want += "?" + q
		}
		if len(got) != 1 || got[0] != want {
			t.Errorf("%s: the server saw %v, want %s", name, got, want)
		}
	}
}

func TestAConflictIsNeverRetried(t *testing.T) {
	f := &fakeServer{answer: func(*http.Request, string) (int, string) {
		return http.StatusConflict, status(409, "Conflict", `Operation cannot be fulfilled on deployments.apps "web": the object has been modified; please apply your changes to the latest version and try again`)
	}}
	g := bound(t, Seamless, f)
	ans := g.Send(context.Background(), everyOrder(t)["crew-scale"])
	if !ans.Conflict() || !strings.Contains(ans.Text(), "the object has been modified") {
		t.Fatalf("answer %+v", ans)
	}
	if n := len(f.lines()); n != 1 {
		t.Fatalf("the server saw %d requests: %v", n, f.lines())
	}
}

func TestABudgetsRefusalIsReadAndADryRunTheWebhookCannotTakeIsSaid(t *testing.T) {
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/eviction") {
			data, _ := json.Marshal(map[string]any{"kind": "Status", "code": 429, "reason": "TooManyRequests", "message": "Cannot evict pod as it would violate the pod's disruption budget.",
				"details": map[string]any{"causes": []any{map[string]any{"reason": "DisruptionBudget", "message": "The disruption budget web needs 2 healthy pods and has 2 currently"}}}})
			return http.StatusTooManyRequests, string(data)
		}
		return http.StatusBadRequest, status(400, "BadRequest", `admission webhook "audit.example.com" does not support dry run`)
	}}
	g := bound(t, Seamless, f)
	o := everyOrder(t)
	ans := g.Send(context.Background(), o["keep-drain-evict"])
	if !ans.Blocked() || len(ans.Causes) != 1 || !strings.Contains(ans.Causes[0], "The disruption budget web needs 2 healthy pods") {
		t.Fatalf("eviction answer %+v", ans)
	}
	pre, _ := g.Preview(context.Background(), o["crew-set-image"])
	if !pre.NoDryRun {
		t.Fatalf("a webhook's refusal of the dry run was not recognised: %+v", pre)
	}
	if real := g.Send(context.Background(), o["crew-set-image"]); real.NoDryRun {
		t.Fatal("a real request was taken for a dry run")
	}
}

func TestAReviewSaysWhyTheCredentialsRefuse(t *testing.T) {
	f := &fakeServer{answer: func(*http.Request, string) (int, string) {
		return http.StatusCreated, `{"status":{"allowed":false,"reason":"RBAC: no rule allows patch on deployments/scale"}}`
	}}
	g := bound(t, Writes, f)
	v := g.Review(context.Background(), everyOrder(t)["crew-scale"].Access)
	if v.Allowed || v.Reason != "RBAC: no rule allows patch on deployments/scale" {
		t.Fatalf("verdict %+v", v)
	}
}

func TestAnUnboundGateSendsNothing(t *testing.T) {
	g, _ := New(choice(Seamless))
	if ans := g.Send(context.Background(), everyOrder(t)["crew-scale"]); !errors.Is(ans.Err, ErrUnbound) {
		t.Fatalf("answer %+v", ans)
	}
}

func TestTheLedgerKeepsOneLinePerWriteAndNoSecretValue(t *testing.T) {
	dir := t.TempDir()
	c := Context{Name: "kind-kubernites", Server: "https://127.0.0.1:6443", User: "kind-kubernites"}
	l := OpenLedger(dir, c)
	if !strings.HasPrefix(filepath.Base(l.Path()), "live-ledger-kind-kubernites-") || filepath.Dir(l.Path()) != dir {
		t.Fatalf("ledger path %s", l.Path())
	}
	at := time.Date(2026, 11, 15, 21, 0, 0, 0, time.UTC)
	writes := []Entry{
		{At: at, Context: c.Name, Level: "WRITES", Order: "kubectl create secret generic db --from-literal=password=hunter2 --from-literal=user=admin", Object: "secret shop/db", Code: 201},
		{At: at, Context: c.Name, Level: "WRITES", Order: `kubectl patch secret db -n shop -p '{"stringData":{"password":"hunter3","token":"abc"}}'`, Object: "secret shop/db", Code: 200},
		{At: at, Context: c.Name, Level: "SEAMLESS", Order: `kubectl patch secret db -p '{"data":{"password":"aHVudGVyMg=="}}'`, Object: "secret shop/db", Code: 200},
		{At: at, Context: c.Name, Level: "SEAMLESS", Order: "kubectl create secret docker-registry reg --docker-password=s3cr3t --docker-username=me", Object: "secret shop/reg", Code: 201},
		{At: at, Context: c.Name, Level: "SEAMLESS", Order: "scale deployment shop/web to 5", Object: "deployment shop/web", Code: 200, Answer: "ok"},
	}
	for _, e := range writes {
		if err := l.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(l.Path())
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, secret := range []string{"hunter2", "hunter3", "admin", "aHVudGVyMg==", "abc", "s3cr3t"} {
		if strings.Contains(text, secret) {
			t.Errorf("the ledger holds %q:\n%s", secret, text)
		}
	}
	if !strings.Contains(text, "--docker-username=me") || !strings.Contains(text, "--from-literal=password="+Masked) {
		t.Errorf("the ledger lost what is not secret:\n%s", text)
	}
	got, err := l.Read()
	if err != nil || len(got) != len(writes) || got[4].Order != "scale deployment shop/web to 5" {
		t.Fatalf("read back %d entries %v: %+v", len(got), err, got)
	}
	if info, _ := os.Stat(l.Path()); info.Mode().Perm() != 0o600 {
		t.Errorf("the ledger is %v", info.Mode().Perm())
	}
}

func TestTheLedgerDropsItsOldestHalfAtItsCap(t *testing.T) {
	dir := t.TempDir()
	l := OpenLedger(dir, Context{Name: "c", Server: "s"})
	l.cap = 4096
	for i := 0; i < 200; i++ {
		if err := l.Append(Entry{Order: strings.Repeat("x", 40) + string(rune('a'+i%26)), Code: i}); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Stat(l.Path())
	if err != nil || info.Size() > l.cap {
		t.Fatalf("size %v %v", info.Size(), err)
	}
	got, _ := l.Read()
	if len(got) == 0 || got[len(got)-1].Code != 199 || got[0].Code == 0 {
		t.Fatalf("kept %d entries, first %d last %d", len(got), got[0].Code, got[len(got)-1].Code)
	}
}

func TestRememberedChoicesAreKeyedByNameServerAndUser(t *testing.T) {
	dir := t.TempDir()
	c := Context{Name: "kind-kind", Server: "https://127.0.0.1:6443", User: "kind-kind"}
	m := Memory{}
	m.Keep(Remembered{Context: c, Level: Seamless, Warned: Seamless, Classes: []Class{Secrets, Destroy}})
	m.Keep(Remembered{Context: Context{Name: "staging", Server: "https://s", User: "ro"}, Level: Writes, Warned: Writes})
	if err := SaveMemory(dir, m); err != nil {
		t.Fatal(err)
	}
	back, err := LoadMemory(dir)
	if err != nil {
		t.Fatal(err)
	}
	r, ch, ok := back.Recall(c)
	if !ok || ch.Any() || r.Level != Seamless || len(r.Classes) != 2 || r.Classes[0] != Destroy {
		t.Fatalf("recall %+v %+v %v", r, ch, ok)
	}
	moved := c
	moved.Server = "https://10.0.0.9:6443"
	if _, ch, ok := back.Recall(moved); ok || !ch.Server || ch.User || ch.Was.Server != c.Server {
		t.Fatalf("a changed server: %+v %v", ch, ok)
	}
	other := c
	other.User = "admin"
	if _, ch, ok := back.Recall(other); ok || !ch.User {
		t.Fatalf("a changed user: %+v %v", ch, ok)
	}
	back.Forget("kind-kind")
	if _, _, ok := back.Recall(c); ok || len(back.Contexts) != 1 {
		t.Fatal("forgotten, still recalled")
	}
	back.ForgetAll()
	if len(back.Contexts) != 0 {
		t.Fatal("forget all kept some")
	}
	data, _ := os.ReadFile(MemoryPath(dir))
	if !strings.Contains(string(data), `"level": "DANGEROUSLY SEAMLESS"`) {
		t.Errorf("the file:\n%s", data)
	}
	if m, err := LoadMemory(t.TempDir()); err != nil || len(m.Contexts) != 0 {
		t.Fatalf("no file: %+v %v", m, err)
	}
	if m, err := LoadMemory(""); err != nil || len(m.Contexts) != 0 {
		t.Fatalf("no directory: %+v %v", m, err)
	}
}

func TestAChoiceRememberedAsSeamlessBeforeTheRenameStillLoads(t *testing.T) {
	dir := t.TempDir()
	c := Context{Name: "kind-kind", Server: "https://127.0.0.1:6443", User: "kind-kind"}
	old := `{"contexts": [{"context": {"Name": "kind-kind", "Server": "https://127.0.0.1:6443", "User": "kind-kind"}, "level": "SEAMLESS", "warned": "SEAMLESS"}]}`
	if err := os.WriteFile(MemoryPath(dir), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := LoadMemory(dir)
	if err != nil {
		t.Fatal(err)
	}
	r, _, ok := m.Recall(c)
	if !ok || r.Level != Seamless || r.Warned != Seamless {
		t.Fatalf("recall %+v %v", r, ok)
	}
	if Seamless.String() != "DANGEROUSLY SEAMLESS" {
		t.Errorf("the level is named %q", Seamless.String())
	}
}
