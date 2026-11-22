package kubectltree_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	streamhttp "k8s.io/streaming/pkg/httpstream"
	streamspdy "k8s.io/streaming/pkg/httpstream/spdy"

	"github.com/waxb/kubernites/pkg/live/write"
)

func talkingPod(w http.ResponseWriter, r *http.Request, quit chan struct{}) {
	if _, err := streamhttp.Handshake(r, w, []string{"portforward.k8s.io"}); err != nil {
		return
	}
	conn := streamspdy.NewResponseUpgrader().UpgradeResponse(w, r, func(st streamhttp.Stream, replySent <-chan struct{}) error {
		if st.Headers().Get("streamType") == "data" {
			go func() {
				<-replySent
				for {
					select {
					case <-quit:
						return
					case <-time.After(20 * time.Millisecond):
					}
					if _, err := st.Write([]byte("tick\n")); err != nil {
						return
					}
				}
			}()
		}
		return nil
	})
	if conn == nil {
		return
	}
	defer conn.Close()
	select {
	case <-conn.CloseChan():
	case <-quit:
	}
}

func TestNoBytesFlowThroughAForwardAfterStopOrClose(t *testing.T) {
	for _, how := range []string{"STOP", "closing the realm"} {
		quit := make(chan struct{})
		var base *fakeServer
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/portforward") && r.Header.Get("Upgrade") != "websocket" {
				talkingPod(w, r, quit)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/portforward") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			base.serve(w, r)
		}))
		base = &fakeServer{Server: srv, seen: &seen{}}
		path := kubeconfig(t, srv.URL, "stone", "shop")
		g := gateAt(t, write.Seamless, base, path)
		g.Pass(write.Workloads)
		tree := newTree(t, path, g.WrapConfig)
		res := run(t, tree, "port-forward web-1 -n shop :80")
		fws := tree.Forwards()
		if res.Code != 0 || len(fws) != 1 || len(fws[0].Listening) != 1 {
			t.Fatalf("the forward did not open: %+v\n%s", fws, res.Output)
		}
		addr, _, _ := strings.Cut(fws[0].Listening[0], " -> ")
		var got atomic.Int64
		var conns []net.Conn
		for range 4 {
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			conns = append(conns, conn)
			go func() {
				chunk := make([]byte, 64<<10)
				for {
					if _, err := conn.Write(chunk); err != nil {
						return
					}
				}
			}()
			go func() {
				buf := make([]byte, 4096)
				for {
					n, err := conn.Read(buf)
					got.Add(int64(n))
					if err != nil {
						return
					}
				}
			}()
		}
		time.Sleep(3 * time.Second)
		if how == "STOP" {
			tree.StopForward(fws[0].ID)
		} else {
			tree.Close()
		}
		time.Sleep(500 * time.Millisecond)
		before := got.Load()
		time.Sleep(2 * time.Second)
		after := got.Load()
		close(quit)
		for _, c := range conns {
			c.Close()
		}
		srv.CloseClientConnections()
		srv.Close()
		if after > before {
			t.Errorf("after %s, connections made before it still carried %d bytes from the pod to this computer in 2 s", how, after-before)
		}
	}
}

func TestLocalClientsOfAStoppedForwardSeeTheirConnectionsEnd(t *testing.T) {
	for _, how := range []string{"STOP", "closing the realm"} {
		quit := make(chan struct{})
		var base *fakeServer
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/portforward") && r.Header.Get("Upgrade") != "websocket" {
				talkingPod(w, r, quit)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/portforward") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			base.serve(w, r)
		}))
		base = &fakeServer{Server: srv, seen: &seen{}}
		path := kubeconfig(t, srv.URL, "stone", "shop")
		g := gateAt(t, write.Seamless, base, path)
		g.Pass(write.Workloads)
		tree := newTree(t, path, g.WrapConfig)
		res := run(t, tree, "port-forward web-1 -n shop :80")
		fws := tree.Forwards()
		if res.Code != 0 || len(fws) != 1 || len(fws[0].Listening) != 1 {
			t.Fatalf("the forward did not open: %+v\n%s", fws, res.Output)
		}
		addr, _, _ := strings.Cut(fws[0].Listening[0], " -> ")
		ended := make(chan struct{}, 3)
		var conns []net.Conn
		for range 3 {
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			conns = append(conns, conn)
			go func() {
				chunk := make([]byte, 64<<10)
				for {
					if _, err := conn.Write(chunk); err != nil {
						return
					}
				}
			}()
			go func() {
				buf := make([]byte, 4096)
				for {
					if _, err := conn.Read(buf); err != nil {
						ended <- struct{}{}
						return
					}
				}
			}()
		}
		time.Sleep(3 * time.Second)
		if how == "STOP" {
			tree.StopForward(fws[0].ID)
		} else {
			tree.Close()
		}
		deadline := time.After(5 * time.Second)
		for range conns {
			select {
			case <-ended:
			case <-deadline:
				t.Fatalf("after %s a local client's connection to the forward is still open", how)
			}
		}
		close(quit)
		for _, c := range conns {
			c.Close()
		}
		srv.CloseClientConnections()
		srv.Close()
	}
}

func silentPod(w http.ResponseWriter, r *http.Request, quit, asked chan struct{}) {
	if _, err := streamhttp.Handshake(r, w, []string{"portforward.k8s.io"}); err != nil {
		return
	}
	conn := streamspdy.NewResponseUpgrader().UpgradeResponse(w, r, func(st streamhttp.Stream, replySent <-chan struct{}) error {
		select {
		case asked <- struct{}{}:
		default:
		}
		<-quit
		return nil
	})
	if conn == nil {
		return
	}
	defer conn.Close()
	select {
	case <-conn.CloseChan():
	case <-quit:
	}
}

func TestALocalClientWhoseStreamAwaitsItsReplySeesItsConnectionEndAtStop(t *testing.T) {
	for _, how := range []string{"STOP", "closing the realm"} {
		quit := make(chan struct{})
		asked := make(chan struct{}, 1)
		var base *fakeServer
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/portforward") && r.Header.Get("Upgrade") != "websocket" {
				silentPod(w, r, quit, asked)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/portforward") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			base.serve(w, r)
		}))
		base = &fakeServer{Server: srv, seen: &seen{}}
		path := kubeconfig(t, srv.URL, "stone", "shop")
		g := gateAt(t, write.Seamless, base, path)
		g.Pass(write.Workloads)
		tree := newTree(t, path, g.WrapConfig)
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
		ended := make(chan struct{})
		go func() {
			buf := make([]byte, 4096)
			for {
				if _, err := conn.Read(buf); err != nil {
					close(ended)
					return
				}
			}
		}()
		select {
		case <-asked:
		case <-time.After(10 * time.Second):
			t.Fatal("the local connection's stream never reached the pod")
		}
		if how == "STOP" {
			tree.StopForward(fws[0].ID)
		} else {
			tree.Close()
		}
		select {
		case <-ended:
		case <-time.After(5 * time.Second):
			t.Errorf("after %s a local client whose stream awaited its reply still has its connection open", how)
		}
		close(quit)
		conn.Close()
		srv.CloseClientConnections()
		srv.Close()
	}
}
