package kubectltree_test

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/httpstream/wsstream"
	streamhttp "k8s.io/streaming/pkg/httpstream"
	streamspdy "k8s.io/streaming/pkg/httpstream/spdy"

	"github.com/waxb/kubernites/pkg/client/termview"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

type stall struct {
	once    sync.Once
	release chan struct{}
}

func (s *stall) free() {
	s.once.Do(func() { close(s.release) })
}

func (s *stall) exec(w http.ResponseWriter, r *http.Request) {
	if !wsstream.IsWebSocketRequest(r) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	channels := []wsstream.ChannelType{wsstream.ReadChannel, wsstream.WriteChannel, wsstream.WriteChannel, wsstream.WriteChannel, wsstream.ReadChannel}
	conn := wsstream.NewConn(map[string]wsstream.ChannelProtocolConfig{
		"v5.channel.k8s.io": {Binary: true, Channels: channels},
		"v4.channel.k8s.io": {Binary: true, Channels: channels},
	})
	_, streams, err := conn.Open(w, r)
	if err != nil {
		return
	}
	defer conn.Close()
	flood := []byte(strings.Repeat("\x1b[6n\x1b[c", 512))
	for {
		select {
		case <-s.release:
			return
		default:
		}
		if _, err := streams[1].Write(flood); err != nil {
			return
		}
	}
}

func (s *stall) forward(w http.ResponseWriter, r *http.Request) {
	if wsstream.IsWebSocketRequest(r) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := streamhttp.Handshake(r, w, []string{"portforward.k8s.io"}); err != nil {
		return
	}
	streams := make(chan streamhttp.Stream, 64)
	conn := streamspdy.NewResponseUpgrader().UpgradeResponse(w, r, func(st streamhttp.Stream, _ <-chan struct{}) error {
		streams <- st
		return nil
	})
	if conn == nil {
		return
	}
	defer conn.Close()
	var held []streamhttp.Stream
	for {
		select {
		case st := <-streams:
			held = append(held, st)
		case <-s.release:
			for _, st := range held {
				go io.Copy(io.Discard, st)
			}
			for {
				select {
				case st := <-streams:
					go io.Copy(io.Discard, st)
				case <-conn.CloseChan():
					return
				}
			}
		case <-conn.CloseChan():
			return
		}
	}
}

func stallTree(t *testing.T) (*kubectltree.Tree, *stall) {
	t.Helper()
	s := &stall{release: make(chan struct{})}
	var base *fakeServer
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/exec"):
			s.exec(w, r)
		case strings.HasSuffix(r.URL.Path, "/portforward"):
			s.forward(w, r)
		default:
			base.serve(w, r)
		}
	}))
	base = &fakeServer{Server: srv, seen: &seen{}}
	t.Cleanup(srv.Close)
	t.Cleanup(s.free)
	path := kubeconfig(t, srv.URL, "stone", "shop")
	g := gateAt(t, write.Seamless, base, path)
	g.Pass(write.Workloads)
	return newTree(t, path, g.WrapConfig), s
}

func TestCtrlPCtrlQLeavesWhileTheContainerReadsNoInput(t *testing.T) {
	tree, s := stallTree(t)
	var term *kubectltree.Terminal
	view := termview.New(154, 10, func(b []byte) { term.Type(b) })
	term = kubectltree.NewTerminal(154, 10, view)
	defer view.Close()
	job := tree.StartIn("exec -it web-1 -n shop -- sh", nil, term)
	defer func() {
		s.free()
		job.Stop()
		for deadline := time.Now().Add(15 * time.Second); !job.Done() && time.Now().Before(deadline); {
			time.Sleep(20 * time.Millisecond)
		}
	}()
	until(t, "the terminal is live", term.Live)
	time.Sleep(3 * time.Second)
	view.Key(termview.Key{Rune: 'p', Ctrl: true})
	view.Key(termview.Key{Rune: 'q', Ctrl: true})
	deadline := time.Now().Add(15 * time.Second)
	for !job.Done() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !job.Done() {
		t.Fatalf("ctrl-p ctrl-q did not leave within 15 s: the container floods terminal queries and reads no input, so every key stays in the container's terminal")
	}
	if ss := job.Result().Sessions; len(ss) == 0 || ss[0].Ended != kubectltree.EndedLeft {
		t.Fatalf("the session ended %+v", ss)
	}
}

func TestStopClosesAForwardWhileThePodReadsNothing(t *testing.T) {
	tree, s := stallTree(t)
	res := run(t, tree, "port-forward web-1 -n shop :80")
	fws := tree.Forwards()
	if res.Code != 0 || len(fws) != 1 || len(fws[0].Listening) != 1 {
		t.Fatalf("the forward did not open: %+v\n%s", fws, res.Output)
	}
	addr, _, _ := strings.Cut(fws[0].Listening[0], " -> ")
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var wrote atomic.Int64
	go func() {
		chunk := make([]byte, 64<<10)
		for {
			n, err := conn.Write(chunk)
			wrote.Add(int64(n))
			if err != nil {
				return
			}
		}
	}()
	last := int64(-1)
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(time.Second) {
		now := wrote.Load()
		if now == last {
			break
		}
		last = now
	}
	stopped := make(chan struct{})
	go func() {
		tree.StopForward(fws[0].ID)
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		listening := false
		if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
			listening = true
			c.Close()
		}
		t.Errorf("STOP (the live card's button, on the game's thread) did not return within 10 s after the pod stopped reading %d bytes; %s still listening: %v", wrote.Load(), addr, listening)
	}
	s.free()
	conn.Close()
	select {
	case <-stopped:
	case <-time.After(30 * time.Second):
		t.Fatal("STOP never returned, even after the pod read again")
	}
}

func TestStopAndCloseShutAForwardsPortAtOnceWhileThePodReadsNothing(t *testing.T) {
	for _, how := range []string{"STOP", "closing the realm"} {
		tree, s := stallTree(t)
		res := run(t, tree, "port-forward web-1 -n shop :80")
		fws := tree.Forwards()
		if res.Code != 0 || len(fws) != 1 || len(fws[0].Listening) != 1 {
			t.Fatalf("the forward did not open: %+v\n%s", fws, res.Output)
		}
		addr, _, _ := strings.Cut(fws[0].Listening[0], " -> ")
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			chunk := make([]byte, 64<<10)
			for {
				if _, err := conn.Write(chunk); err != nil {
					return
				}
			}
		}()
		time.Sleep(3 * time.Second)
		began := time.Now()
		if how == "STOP" {
			tree.StopForward(fws[0].ID)
		} else {
			tree.Close()
		}
		if took := time.Since(began); took > time.Second {
			t.Errorf("%s took %v on the game's thread", how, took)
		}
		if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
			c.Close()
			t.Errorf("after %s, %s still listens", how, addr)
		}
		if len(tree.Forwards()) != 0 {
			t.Errorf("after %s the live card still lists %+v", how, tree.Forwards())
		}
		ended := tree.TakeEndedForwards()
		if len(ended) != 1 || ended[0].Ended == "" {
			t.Errorf("after %s the forward's ledger line is %+v", how, ended)
		}
		s.free()
		conn.Close()
	}
}
