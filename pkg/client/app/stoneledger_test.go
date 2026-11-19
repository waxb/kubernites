package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

const scaleAnswer = `{"apiVersion":"autoscaling/v1","kind":"Scale","metadata":{"name":"web-b","namespace":"parity","resourceVersion":"7"},"spec":{"replicas":4},"status":{"replicas":3,"selector":"app=web-b"}}`

const stoneScale = "scale deployment web-b -n parity --replicas=4"

type frontSeen struct {
	mu   sync.Mutex
	list []string
}

func (f *frontSeen) add(s string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.list = append(f.list, s)
}

func (f *frontSeen) count(want string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.list {
		if s == want {
			n++
		}
	}
	return n
}

func (f *frontSeen) all() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.list...)
}

func stoneFront(t *testing.T) (*httptest.Server, *frontSeen) {
	t.Helper()
	rec, err := livetest.ReadRecording("../../live/testdata/stone-kind-replay.json")
	if err != nil {
		t.Fatal(err)
	}
	replay, _ := livetest.ServeRecordingSeen(rec)
	t.Cleanup(replay.Close)
	back, _ := livetest.ServeAs(t, livetest.Managed(), livetest.Options{User: "admin@example.com"})
	recorded := map[string]bool{}
	for _, e := range rec.Exchanges {
		if e.Method == http.MethodGet && e.Query == "" {
			recorded[e.Path] = true
		}
	}
	proxy := func(raw string) *httputil.ReverseProxy {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return httputil.NewSingleHostReverseProxy(u)
	}
	toReplay, toBack := proxy(replay.URL), proxy(back.URL)
	seen := &frontSeen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.add(r.Method + " " + r.URL.RequestURI())
		switch {
		case r.Method == http.MethodGet && recorded[r.URL.Path]:
			toReplay.ServeHTTP(w, r)
		case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/deployments/web-b/scale"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, scaleAnswer)
		default:
			toBack.ServeHTTP(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, seen
}

func connectStone(t *testing.T, level string) (*liveRig, *frontSeen) {
	t.Helper()
	srv, seen := stoneFront(t)
	return connectStoneTo(t, level, srv.URL), seen
}

func connectStoneTo(t *testing.T, level, url string) *liveRig {
	t.Helper()
	dir := t.TempDir()
	g := NewGame(nil, Options{LiveOnly: true, SettingsDir: dir, Live: LiveSource{Files: []string{livetest.Kubeconfig(t, url)}}}, nil, nil)
	rig := &liveRig{g: g, dir: dir}
	rig.title(hud.Intent{Kind: hud.IntentLive})
	rig.title(hud.Intent{Kind: hud.IntentLiveList})
	rig.press(t, "live:row:0")
	rig.press(t, "live:choose")
	if level != hud.LevelReadOnly {
		rig.press(t, hud.LiveBoxID)
		if level == hud.LevelSeamless {
			rig.press(t, "live:seam")
		}
		rig.press(t, "live:allow")
	}
	if got := g.Picker.ChosenLevel(); got != level {
		t.Fatalf("the picker chose %s, want %s", got, level)
	}
	rig.press(t, "live:confirm")
	for deadline := time.Now().Add(30 * time.Second); g.Scene != SceneRealm; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the cluster never opened: %q %s", g.Picker.Step, g.Picker.Err)
		}
		g.StepLive()
	}
	rig.s = g.Sess
	if w := rig.s.Writer; w != nil {
		w.sync = true
		w.now = func() time.Time { return time.Date(2026, 11, 16, 3, 4, 5, 0, time.UTC) }
		rig.gate = w.gate
	}
	rig.s.Hud.SetStoneMode(hud.StoneModeKube)
	return rig
}

func stoneFrame(s *Session, keys ...ebiten.Key) {
	s.Update(hud.Input{MX: -10, MY: -10, Keys: keys}, 0, &Pointer{}, true)
}

func seenLong(s *Session) {
	for range hud.AskSeenFrames {
		stoneFrame(s)
	}
}

func askedOnTheStone(t *testing.T, s *Session, cmd string) hud.StoneAsk {
	t.Helper()
	s.Hud.OpenStone(cmd)
	stoneFrame(s, ebiten.KeyEnter)
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		stoneFrame(s)
		if a, ok := s.Hud.StoneAsking(); ok {
			seenLong(s)
			return a
		}
		if !s.Hud.StoneRunning() {
			t.Fatalf("%s ended without asking", cmd)
		}
	}
	t.Fatalf("%s never asked", cmd)
	return hud.StoneAsk{}
}

func stoneEnds(t *testing.T, s *Session) {
	t.Helper()
	for deadline := time.Now().Add(20 * time.Second); s.Hud.StoneRunning(); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the Stone's kubectl did not finish")
		}
		stoneFrame(s)
	}
}

func ledgerLines(t *testing.T, rig *liveRig) []string {
	t.Helper()
	data, err := os.ReadFile(rig.s.Hud.LiveCard().Ledger)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func TestAStoneWriteGoesToTheLedgerTheLiveCardAndTheHistory(t *testing.T) {
	rig, seen := connectStone(t, hud.LevelSeamless)
	if out := speakAndWait(t, rig.s, stoneScale); !strings.Contains(out, "deployment.apps/web-b scaled") {
		t.Fatalf("the Stone answered:\n%s", out)
	}
	speakAndWait(t, rig.s, "get deployment web-b -n parity")
	speakAndWait(t, rig.s, stoneScale+" --dry-run=server")
	if n := seen.count("PATCH /apis/apps/v1/namespaces/parity/deployments/web-b/scale"); n != 1 {
		t.Fatalf("the cluster saw %d scales", n)
	}
	lines := ledgerLines(t, rig)
	if len(lines) != 1 || !strings.Contains(lines[0], `"order":"kubectl `+stoneScale+`"`) || !strings.Contains(lines[0], `"object":"deployment parity/web-b (scale)"`) || !strings.Contains(lines[0], `"level":"DANGEROUSLY SEAMLESS"`) || !strings.Contains(lines[0], `"code":200`) {
		t.Fatalf("the ledger file holds %q", lines)
	}
	card := rig.s.Hud.LiveCard()
	if len(card.Writes) != 1 || !strings.Contains(card.Writes[0], "parity/web-b") || !strings.Contains(card.Writes[0], "PATCH") {
		t.Fatalf("the live card lists %q", card.Writes)
	}
	if !slices.ContainsFunc(rig.s.Echoes, func(e hud.Tiding) bool {
		return strings.Contains(e.Text, "WRITE DANGEROUSLY SEAMLESS") && strings.Contains(e.Text, "parity/web-b")
	}) {
		t.Fatal("the write is not in the Stone's history")
	}
}

func TestAtWritesOnlyTheConfirmedStoneWriteIsSentAndListed(t *testing.T) {
	rig, seen := connectStone(t, hud.LevelWrites)
	s := rig.s
	a := askedOnTheStone(t, s, stoneScale)
	if !strings.HasPrefix(a.Title, "CONFIRM: kubectl "+stoneScale) || !a.Confirm {
		t.Fatalf("the Stone asked %+v", a)
	}
	if label, _ := a.GoLabel(); label != "CONFIRM [Y]" {
		t.Fatalf("the ask's button reads %q", label)
	}
	stoneFrame(s, ebiten.KeyEscape)
	stoneEnds(t, s)
	if n := seen.count("PATCH /apis/apps/v1/namespaces/parity/deployments/web-b/scale"); n != 0 || len(ledgerLines(t, rig)) != 0 {
		t.Fatalf("after CANCEL: %d scales, ledger %q", n, ledgerLines(t, rig))
	}
	askedOnTheStone(t, s, stoneScale)
	stoneFrame(s, ebiten.KeyY)
	stoneEnds(t, s)
	if n, dry := seen.count("PATCH /apis/apps/v1/namespaces/parity/deployments/web-b/scale"), seen.count("PATCH /apis/apps/v1/namespaces/parity/deployments/web-b/scale?dryRun=All"); n != 1 || dry != 2 {
		t.Fatalf("the cluster saw %d scales and %d dry runs", n, dry)
	}
	lines := ledgerLines(t, rig)
	if len(lines) != 1 || !strings.Contains(lines[0], `"level":"WRITES"`) {
		t.Fatalf("the ledger file holds %q", lines)
	}
}

func TestAStoneClassWarningIsAskedLikeAnOrdersAndRemembered(t *testing.T) {
	rig, _ := connectStone(t, hud.LevelWrites)
	s := rig.s
	a := askedOnTheStone(t, s, "get secrets -n parity")
	label, _ := a.GoLabel()
	if !strings.Contains(a.Title, "SECRETS") || label != "I UNDERSTAND [Y]" || !strings.Contains(strings.Join(a.Lines, " "), "masked") {
		t.Fatalf("the Stone asked %+v", a)
	}
	stoneFrame(s, ebiten.KeyY)
	stoneEnds(t, s)
	rig.frame()
	mem, err := write.LoadMemory(rig.dir)
	if err != nil {
		t.Fatal(err)
	}
	r, _, ok := mem.Recall(rig.gate.Context())
	if !ok || !slices.Contains(r.Classes, write.Secrets) {
		t.Fatalf("remembered %+v %v", r, ok)
	}
	if !rig.gate.Passed(write.Secrets) {
		t.Fatal("the gate did not pass the warning")
	}
	if len(ledgerLines(t, rig)) != 0 {
		t.Fatal("a read went to the ledger")
	}
}
