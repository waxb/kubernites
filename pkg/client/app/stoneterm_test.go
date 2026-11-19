package app

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

const shellPod = `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"shell-1","namespace":"parity","uid":"u-shell","resourceVersion":"44","creationTimestamp":"2026-11-01T10:00:00Z"},"spec":{"nodeName":"kubernites-worker","containers":[{"name":"shell","image":"busybox:1.36","stdin":true,"tty":true}]},"status":{"phase":"Running","containerStatuses":[{"name":"shell","ready":true,"state":{"running":{}}}]}}`

func shellFront(t *testing.T) (string, *livetest.Shell, *frontSeen) {
	t.Helper()
	inner, seen := stoneFront(t)
	u, _ := url.Parse(inner.URL)
	back := httputil.NewSingleHostReverseProxy(u)
	shell := &livetest.Shell{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		switch p := r.URL.Path; {
		case strings.HasSuffix(p, "/exec") || strings.HasSuffix(p, "/attach"):
			seen.add(r.Method + " " + r.URL.RequestURI())
			shell.Serve(w, r)
		case strings.HasSuffix(p, "/portforward"):
			seen.add(r.Method + " " + r.URL.RequestURI())
			shell.ServeForward(w, r)
		case len(parts) == 6 && parts[0] == "api" && parts[2] == "namespaces" && parts[4] == "pods" && r.Method == http.MethodGet:
			seen.add(r.Method + " " + r.URL.RequestURI())
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(strings.ReplaceAll(strings.ReplaceAll(shellPod, `"shell-1"`, `"`+parts[5]+`"`), `"parity"`, `"`+parts[3]+`"`)))
		default:
			back.ServeHTTP(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL, shell, seen
}

func frames(t *testing.T, s *Session, what string, ok func() bool) {
	t.Helper()
	for deadline := time.Now().Add(20 * time.Second); !ok(); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("never: %s; the Stone says %q", what, s.Hud.StoneOutput())
		}
		stoneFrame(s)
	}
}

func typedIn(shell *livetest.Shell) string {
	typed, _, _, _ := shell.Seen()
	return typed
}

func TestTheStonesTerminalTakesEveryKeyAndSaysHowItEnded(t *testing.T) {
	addr, shell, _ := shellFront(t)
	rig := connectStoneTo(t, hud.LevelSeamless, addr)
	s := rig.s
	a := askedOnTheStone(t, s, "exec -it shell-1 -n parity -- sh")
	if !strings.Contains(a.Title, "REACHING INTO WORKLOADS") {
		t.Fatalf("the Stone asked %+v", a)
	}
	stoneFrame(s, ebiten.KeyY)
	frames(t, s, "the terminal is live", s.Hud.StoneTerminal)
	if _, _, sizes, _ := shell.Seen(); len(sizes) == 0 {
		frames(t, s, "the TTY is sized", func() bool { _, _, sizes, _ := shell.Seen(); return len(sizes) > 0 })
	}
	s.Update(hud.Input{MX: -10, MY: -10, Chars: []rune("ls -a")}, 0, &Pointer{}, true)
	stoneFrame(s, ebiten.KeyEnter)
	frames(t, s, "the shell ran ls", func() bool { return strings.Contains(typedIn(shell), "ls -a\r") })
	s.Update(hud.Input{MX: -10, MY: -10, Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyC}}, 0, &Pointer{}, true)
	s.Update(hud.Input{MX: -10, MY: -10, Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyD}}, 0, &Pointer{}, true)
	s.Update(hud.Input{MX: -10, MY: -10, Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyK}}, 0, &Pointer{}, true)
	stoneFrame(s, ebiten.KeyEscape)
	stoneFrame(s, ebiten.KeyArrowUp)
	frames(t, s, "Ctrl+C, Ctrl+D, Ctrl+K, Esc and Up reached the shell", func() bool { return strings.HasSuffix(typedIn(shell), "\x03\x04\x0b\x1b\x1b[A") })
	if !s.Hud.StoneTerminal() || !s.Hud.StoneRunning() || s.Hud.Overlay() != hud.OverlayStone || !s.Hud.KeyFocus() {
		t.Fatal("a key the container should get stopped the Stone or left it")
	}
	if _, open := s.Hud.OrderOpen(); open {
		t.Fatal("a write key opened an order while the terminal held the keyboard")
	}
	s.Update(hud.Input{MX: -10, MY: -10, Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyP}}, 0, &Pointer{}, true)
	s.Update(hud.Input{MX: -10, MY: -10, Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyQ}}, 0, &Pointer{}, true)
	stoneEnds(t, s)
	if typed := typedIn(shell); strings.Contains(typed, "\x10") || strings.Contains(typed, "\x11") {
		t.Fatalf("the leave chord reached the container: %q", typed)
	}
	out := strings.Join(s.Hud.StoneOutput(), "\n")
	if !strings.Contains(out, "Left pod parity/shell-1, container shell with ctrl-p ctrl-q") {
		t.Fatalf("the Stone did not say how the session ended:\n%s", out)
	}
	if s.Hud.StoneTerminal() {
		t.Fatal("the terminal is still live")
	}
	rig.frame()
	lines := ledgerLines(t, rig)
	if len(lines) != 1 || !strings.Contains(lines[0], `"order":"kubectl exec -it shell-1 -n parity -- sh"`) || !strings.Contains(lines[0], `"object":"pod parity/shell-1"`) ||
		!strings.Contains(lines[0], `"container":"shell"`) || !strings.Contains(lines[0], `"command":"sh"`) || !strings.Contains(lines[0], `"ended":"left with ctrl-p ctrl-q"`) {
		t.Fatalf("the ledger holds %q", lines)
	}
	if strings.Contains(lines[0], "ls -a") || strings.Contains(lines[0], "ran ") {
		t.Fatalf("the ledger holds the session's keys or output: %s", lines[0])
	}
	card := s.Hud.LiveCard()
	if len(card.Writes) != 1 || !strings.Contains(card.Writes[0], "EXEC") || !strings.Contains(card.Writes[0], "parity/shell-1") || !strings.Contains(card.Writes[0], "left with ctrl-p ctrl-q") {
		t.Fatalf("the live card lists %q", card.Writes)
	}
	if !slices.ContainsFunc(s.Echoes, func(e hud.Tiding) bool {
		return strings.Contains(e.Text, "EXEC") && strings.Contains(e.Text, "container shell, left")
	}) {
		t.Fatal("the session is not in the Stone's history")
	}
}

func TestAPortForwardIsListedWithStopClosesWithTheRealmAndIsInTheLedger(t *testing.T) {
	addr, _, _ := shellFront(t)
	rig := connectStoneTo(t, hud.LevelSeamless, addr)
	s := rig.s
	askedOnTheStone(t, s, "port-forward shell-1 -n parity --address 0.0.0.0 :80")
	stoneFrame(s, ebiten.KeyY)
	stoneEnds(t, s)
	frames(t, s, "the live card lists the forward", func() bool { return len(s.Hud.LiveCard().Forwards) == 1 })
	f := s.Hud.LiveCard().Forwards[0]
	m := regexp.MustCompile(`^FORWARD :(\d+) -> parity/shell-1$`).FindStringSubmatch(f.Text)
	if m == nil || !strings.Contains(f.Tip, "127.0.0.1 only") {
		t.Fatalf("the card shows %+v", f)
	}
	if out := strings.Join(s.Hud.StoneOutput(), "\n"); !strings.Contains(out, "Kubernites listens on 127.0.0.1 only; --address 0.0.0.0 was not used.") {
		t.Fatalf("the Stone said:\n%s", out)
	}
	if !slices.ContainsFunc(s.Hud.Buttons(s.Frame(true)), func(b hud.Button) bool { return b.ID == "liveforward:stop:"+strconv.Itoa(f.ID) }) {
		t.Fatal("no STOP on the live card")
	}
	conn, err := net.Dial("tcp", "127.0.0.1:"+m[1])
	if err != nil {
		t.Fatal(err)
	}
	conn.Write([]byte("hi"))
	buf := make([]byte, 64)
	n, _ := conn.Read(buf)
	conn.Close()
	if string(buf[:n]) != "port 80 heard hi" {
		t.Fatalf("the forward answered %q", buf[:n])
	}
	s.Handle(hud.Intent{Kind: hud.IntentLiveForwardStop, Target: strconv.Itoa(f.ID)})
	if len(s.Hud.LiveCard().Forwards) != 0 {
		t.Fatalf("STOP left %+v", s.Hud.LiveCard().Forwards)
	}
	if c, err := net.Dial("tcp", "127.0.0.1:"+m[1]); err == nil {
		c.Close()
		t.Fatal("the stopped forward still listens")
	}
	lines := ledgerLines(t, rig)
	if len(lines) != 2 || !strings.Contains(lines[0], `"ended":"open on 127.0.0.1:`+m[1]+` -\u003e 80"`) || !strings.Contains(lines[1], `"ended":"stopped"`) || !strings.Contains(lines[1], `"object":"pod parity/shell-1"`) {
		t.Fatalf("the ledger holds %q", lines)
	}
	s.Hud.OpenStone("port-forward shell-1 -n parity :80")
	stoneFrame(s, ebiten.KeyEnter)
	stoneEnds(t, s)
	frames(t, s, "the second forward is listed", func() bool { return len(s.Hud.LiveCard().Forwards) == 1 })
	rig.g.Shutdown()
	lines = ledgerLines(t, rig)
	if last := lines[len(lines)-1]; !strings.Contains(last, `"ended":"closed with the realm"`) {
		t.Fatalf("closing the window left %q", lines)
	}
}

func TestClosingTheWindowEndsTheTerminalSessionCleanly(t *testing.T) {
	addr, shell, _ := shellFront(t)
	rig := connectStoneTo(t, hud.LevelSeamless, addr)
	s := rig.s
	askedOnTheStone(t, s, "exec -it shell-1 -n parity -- sh")
	stoneFrame(s, ebiten.KeyY)
	frames(t, s, "the terminal is live", s.Hud.StoneTerminal)
	rig.g.Shutdown()
	frames(t, s, "the Stone's job ended", func() bool { return !s.Hud.StoneRunning() })
	frames(t, s, "the container saw the stream close", func() bool { _, _, _, closed := shell.Seen(); return closed })
	if out := strings.Join(s.Hud.StoneOutput(), "\n"); !strings.Contains(out, "closed with the realm") {
		t.Fatalf("the Stone said:\n%s", out)
	}
	before := s.Hud.StoneOutput()
	s.Hud.OpenStone("exec -it shell-1 -n parity -- sh")
	stoneFrame(s, ebiten.KeyEnter)
	frames(t, s, "the stopped realm answered", func() bool { return !s.Hud.StoneRunning() })
	if s.Hud.StoneTerminal() || len(s.Hud.StoneOutput()) == len(before) {
		t.Fatalf("a closed realm opened a terminal: %q", s.Hud.StoneOutput())
	}
	if _, urls, _, _ := shell.Seen(); len(urls) != 1 {
		t.Fatalf("a closed realm reached the container again: %v", urls)
	}
}

func TestAForwardThatOpensAndEndsBetweenTwoFramesIsStillInTheLedger(t *testing.T) {
	addr, _, _ := shellFront(t)
	rig := connectStoneTo(t, hud.LevelSeamless, addr)
	rig.gate.Pass(write.Workloads)
	tr, ok := liveTreeOf(rig.s.Realm)
	if !ok {
		t.Fatal("no tree")
	}
	job := tr.Tree().Start("port-forward shell-1 -n parity :80", nil)
	for deadline := time.Now().Add(20 * time.Second); !job.Done(); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the forward never opened")
		}
	}
	fws := tr.Tree().Forwards()
	if len(fws) != 1 {
		t.Fatalf("forwards %+v\n%s", fws, job.Result().Output)
	}
	tr.Tree().StopForward(fws[0].ID)
	rig.frame()
	lines := ledgerLines(t, rig)
	if len(lines) != 2 || !strings.Contains(lines[0], `"ended":"open on 127.0.0.1:`) || !strings.Contains(lines[1], `"ended":"stopped"`) {
		t.Fatalf("a forward that opened and ended between two frames left %q", lines)
	}
}
