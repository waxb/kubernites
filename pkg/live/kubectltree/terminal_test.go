package kubectltree_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

type screen struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *screen) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *screen) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func shellTree(t *testing.T, level write.Level, passed ...write.Class) (*kubectltree.Tree, *fakeServer, *livetest.Shell) {
	t.Helper()
	f := newFake(t)
	f.shell = &livetest.Shell{}
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, level, f, path)
	for _, c := range passed {
		g.Pass(c)
	}
	return newTree(t, path, g.WrapConfig), f, f.shell
}

func until(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for deadline := time.Now().Add(15 * time.Second); !ok(); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("never: %s", what)
		}
	}
}

func TestExecItRunsAShellInTheStonesTerminal(t *testing.T) {
	tree, _, shell := shellTree(t, write.Seamless, write.Workloads)
	out := &screen{}
	term := kubectltree.NewTerminal(100, 30, out)
	job := tree.StartIn("exec -it web-1 -n shop -- sh", nil, term)
	until(t, "the terminal is live", term.Live)
	until(t, "the prompt", func() bool { return strings.Contains(out.String(), "$ ") })
	term.Type([]byte("ls -la\r"))
	until(t, "the command ran", func() bool { return strings.Contains(out.String(), "ran ls -la") })
	term.Resize(120, 40)
	until(t, "the resize", func() bool { _, _, sizes, _ := shell.Seen(); return len(sizes) == 2 })
	term.Type([]byte("exit 3\r"))
	until(t, "the job ended", job.Done)
	res := job.Result()
	typed, urls, sizes, _ := shell.Seen()
	if len(urls) != 1 || !strings.Contains(urls[0], "GET /api/v1/namespaces/shop/pods/web-1/exec?") || !strings.Contains(urls[0], "tty=true") || !strings.Contains(urls[0], "stdin=true") || strings.Contains(urls[0], "stderr") || !strings.Contains(urls[0], "container=web") {
		t.Fatalf("the cluster saw %v", urls)
	}
	if strings.Join(sizes, " ") != "100x30 120x40" {
		t.Fatalf("the TTY was sized %v", sizes)
	}
	if typed != "ls -la\rexit 3\r" {
		t.Fatalf("the container got %q", typed)
	}
	if len(res.Sessions) != 1 {
		t.Fatalf("sessions %+v", res.Sessions)
	}
	s := res.Sessions[0]
	if s.Verb != "exec" || s.Pod != "web-1" || s.Namespace != "shop" || s.Container != "web" || strings.Join(s.Command, " ") != "sh" || !s.TTY || !s.Opened || s.Ended != kubectltree.EndedExited || s.Code != 3 {
		t.Fatalf("the session %+v", s)
	}
	if strings.Contains(res.Output, "Unable to use a TTY") {
		t.Fatalf("the Stone said it has no terminal:\n%s", res.Output)
	}
	if len(res.Writes) != 0 {
		t.Fatalf("an exec was reported as a write: %+v", res.Writes)
	}
	if term.Live() {
		t.Fatal("the terminal is still live after the session")
	}
}

func TestCtrlPCtrlQLeavesTheTerminalAndNeverReachesTheContainer(t *testing.T) {
	tree, _, shell := shellTree(t, write.Seamless, write.Workloads)
	out := &screen{}
	term := kubectltree.NewTerminal(80, 24, out)
	job := tree.StartIn("exec -it web-1 -n shop -- sh", nil, term)
	until(t, "the prompt", func() bool { return strings.Contains(out.String(), "$ ") })
	term.Type([]byte("\x10a\x03\x04"))
	until(t, "ctrl-p then a letter reached the shell", func() bool { typed, _, _, _ := shell.Seen(); return strings.Contains(typed, "\x04") })
	term.Type([]byte("ok\x10\x11never"))
	until(t, "the job ended", job.Done)
	typed, _, _, _ := shell.Seen()
	if typed != "\x10a\x03\x04ok" {
		t.Fatalf("the container got %q", typed)
	}
	s := job.Result().Sessions[0]
	if s.Ended != kubectltree.EndedLeft || !s.Opened {
		t.Fatalf("the session %+v", s)
	}
	if job.Result().Code != 0 {
		t.Fatalf("leaving answered %d:\n%s", job.Result().Code, job.Result().Output)
	}
}

func TestAttachItToAContainerWithATTY(t *testing.T) {
	tree, _, shell := shellTree(t, write.Seamless, write.Workloads)
	out := &screen{}
	term := kubectltree.NewTerminal(80, 24, out)
	job := tree.StartIn("attach -it shell-1 -n shop", nil, term)
	until(t, "the prompt", func() bool { return strings.Contains(out.String(), "$ ") })
	term.Type([]byte("exit\r"))
	until(t, "the job ended", job.Done)
	_, urls, sizes, _ := shell.Seen()
	if len(urls) != 1 || !strings.Contains(urls[0], "/pods/shell-1/attach?") || !strings.Contains(urls[0], "tty=true") || !strings.Contains(urls[0], "container=shell") || strings.Join(sizes, " ") != "81x25 80x24" {
		t.Fatalf("the cluster saw %v, sizes %v", urls, sizes)
	}
	s := job.Result().Sessions[0]
	if s.Verb != "attach" || s.Channel != "attach" || !s.TTY || s.Ended != kubectltree.EndedExited || s.Code != 0 {
		t.Fatalf("the session %+v\n%s", s, job.Result().Output)
	}
}

func TestClosingTheTreeEndsTheSessionCleanly(t *testing.T) {
	tree, _, shell := shellTree(t, write.Seamless, write.Workloads)
	out := &screen{}
	term := kubectltree.NewTerminal(80, 24, out)
	job := tree.StartIn("exec -it web-1 -n shop -- sh", nil, term)
	until(t, "the prompt", func() bool { return strings.Contains(out.String(), "$ ") })
	tree.Close()
	until(t, "the job ended", job.Done)
	until(t, "the cluster saw the stream close", func() bool { _, _, _, closed := shell.Seen(); return closed })
	if s := job.Result().Sessions[0]; s.Ended != kubectltree.EndedClosed {
		t.Fatalf("the session %+v", s)
	}
	again := tree.StartIn("exec -it web-1 -n shop -- sh", nil, kubectltree.NewTerminal(80, 24, io.Discard))
	until(t, "the closed tree answered", again.Done)
	if r := again.Result(); r.Refused == "" || len(r.Sessions) != 0 {
		t.Fatalf("a closed tree opened %+v", r)
	}
}

func TestExecWithoutATerminalRunsACommandIntoTheStonesLinesAndRefusesStdin(t *testing.T) {
	tree, _, shell := shellTree(t, write.Seamless, write.Workloads)
	res := run(t, tree, "exec -it web-1 -n shop -- sh")
	if res.Code == 0 || !strings.Contains(res.Output, "needs the Stone's terminal") {
		t.Fatalf("exec -it without a terminal answered %d:\n%s", res.Code, res.Output)
	}
	if _, urls, _, _ := shell.Seen(); len(urls) != 0 {
		t.Fatalf("the cluster saw %v", urls)
	}
}

func TestAtWritesExecItAsksTheWarningThenConfirmsWhatItOpens(t *testing.T) {
	f := newFake(t)
	f.shell = &livetest.Shell{}
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, write.Writes, f, path)
	tree := newTree(t, path, g.WrapConfig)
	var asks []string
	var mu sync.Mutex
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return holdAll(ctx, g, f.URL, h, func(why string) { mu.Lock(); asks = append(asks, why); mu.Unlock() })
	})
	out := &screen{}
	term := kubectltree.NewTerminal(80, 24, out)
	job := tree.StartIn("exec -it web-1 -n shop -- sh", nil, term)
	until(t, "the prompt", func() bool { return strings.Contains(out.String(), "$ ") })
	term.Type([]byte("exit\r"))
	until(t, "the job ended", job.Done)
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(asks, " | ") != "class workloads | connect exec pod shop/web-1 container web command sh" {
		t.Fatalf("the Stone asked %q", asks)
	}
}

func holdAll(ctx context.Context, g *write.Gate, server string, h kubectltree.Held, note func(string)) bool {
	return live.HoldStoneWrite(ctx, g, server, h, func(a live.StoneAsk) bool {
		switch {
		case a.Why == write.NeedsClass:
			note("class " + string(a.Class))
		case a.Connect:
			o := a.Opens
			note(fmt.Sprintf("connect %s pod %s/%s container %s command %s", o.Channel, o.Namespace, o.Name, o.Container, strings.Join(o.Command, " ")))
		default:
			preview := "previewed"
			if !a.Preview.OK() {
				preview = "no preview: " + a.Preview.Text()
			}
			note("write " + a.Order.Request.Method + " " + a.Order.Request.Path + " " + preview)
		}
		return true
	})
}
