package kubectltree_test

import (
	"io"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

var listening = regexp.MustCompile(`^127\.0\.0\.1:(\d+) -> 80$`)

func otherAddresses(t *testing.T) []string {
	t.Helper()
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	out := []string{"[::1]"}
	for _, a := range addrs {
		ip, _, err := net.ParseCIDR(a.String())
		if err != nil || ip.IsLoopback() {
			continue
		}
		if ip.To4() != nil {
			out = append(out, ip.String())
		} else if !ip.IsLinkLocalUnicast() {
			out = append(out, "["+ip.String()+"]")
		}
	}
	return out
}

func echoes(t *testing.T, addr, say, want string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	io.WriteString(conn, say)
	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil || string(buf[:n]) != want {
		t.Fatalf("%s answered %q (%v), want %q", addr, buf[:n], err, want)
	}
}

func refused(t *testing.T, addr string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err == nil {
		conn.Close()
		t.Fatalf("something listens on %s", addr)
	}
}

func openForward(t *testing.T, tree *kubectltree.Tree, line string) (kubectltree.Forward, string, kubectltree.Result) {
	t.Helper()
	res := run(t, tree, line)
	if res.Code != 0 || !strings.Contains(res.Output, "Forwarding from 127.0.0.1:") {
		t.Fatalf("%s answered %d:\n%s", line, res.Code, res.Output)
	}
	fws := tree.Forwards()
	f := fws[len(fws)-1]
	if len(f.Listening) != 1 || !listening.MatchString(f.Listening[0]) {
		t.Fatalf("the forward listens on %v", f.Listening)
	}
	return f, listening.FindStringSubmatch(f.Listening[0])[1], res
}

func TestPortForwardListensOn127001OnlyWhateverAddressIsTypedAndStaysOpenUntilStop(t *testing.T) {
	tree, _, _ := shellTree(t, write.Seamless, write.Workloads)
	for _, line := range []string{"port-forward web-1 -n shop :80", "port-forward pod/web-1 -n shop --address 0.0.0.0 :80", "port-forward web-1 -n shop --address localhost,::,10.0.0.1 :80"} {
		f, port, res := openForward(t, tree, line)
		if strings.Contains(line, "--address") && !strings.Contains(res.Output, "Kubernites listens on 127.0.0.1 only") {
			t.Errorf("%s did not say the address was not used:\n%s", line, res.Output)
		}
		if strings.Contains(res.Output, "[::1]") {
			t.Errorf("%s listened on ::1:\n%s", line, res.Output)
		}
		echoes(t, "127.0.0.1:"+port, "hello", "port 80 heard hello")
		for _, host := range otherAddresses(t) {
			refused(t, host+":"+port)
		}
		if f.Pod != "web-1" || f.Namespace != "shop" || len(res.Sessions) != 1 || res.Sessions[0].Channel != "portforward" || !res.Sessions[0].Opened {
			t.Fatalf("the forward %+v, sessions %+v", f, res.Sessions)
		}
		if !tree.StopForward(f.ID) {
			t.Fatal("STOP found no forward")
		}
		refused(t, "127.0.0.1:"+port)
		ended := tree.TakeEndedForwards()
		if len(ended) != 1 || ended[0].ID != f.ID || ended[0].Ended != kubectltree.EndedLeft || ended[0].Connections != 1 {
			t.Fatalf("ended %+v", ended)
		}
	}
	if len(tree.Forwards()) != 0 {
		t.Fatalf("still open: %+v", tree.Forwards())
	}
}

func TestEveryForwardClosesWithTheRealm(t *testing.T) {
	tree, _, shell := shellTree(t, write.Seamless, write.Workloads)
	_, a, _ := openForward(t, tree, "port-forward web-1 -n shop :80")
	_, b, _ := openForward(t, tree, "port-forward web-1 -n shop :80")
	if len(tree.Forwards()) != 2 {
		t.Fatalf("open %+v", tree.Forwards())
	}
	tree.Close()
	refused(t, "127.0.0.1:"+a)
	refused(t, "127.0.0.1:"+b)
	ended := tree.TakeEndedForwards()
	if len(ended) != 2 || ended[0].Ended != kubectltree.EndedClosed || ended[1].Ended != kubectltree.EndedClosed || len(tree.Forwards()) != 0 {
		t.Fatalf("ended %+v open %+v", ended, tree.Forwards())
	}
	until(t, "the cluster saw both streams close", func() bool { _, _, _, closed := shell.Seen(); return closed })
}

func TestAtWritesAPortForwardShowsWhatItOpens(t *testing.T) {
	tree, _, asks := gatedTree(t, write.Writes, false)
	res := run(t, tree, "port-forward web-1 -n shop 8080:80")
	if res.Code == 0 || len(*asks) != 1 || (*asks)[0].Class != write.Workloads || len(tree.Forwards()) != 0 {
		t.Fatalf("cancelled: %d %+v %+v\n%s", res.Code, *asks, tree.Forwards(), res.Output)
	}
}
