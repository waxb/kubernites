package kubectltree

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	clientforward "k8s.io/client-go/tools/portforward"
	"k8s.io/kubectl/pkg/cmd/portforward"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	forwarddial "github.com/waxb/kubernites/pkg/live/forwarding"
	"github.com/waxb/kubernites/pkg/live/write"
)

const ForwardAddress = "127.0.0.1"

type Forward struct {
	ID          int
	Target      string
	Namespace   string
	Pod         string
	Ports       []string
	Listening   []string
	Connections int
	Ended       Ended
	Err         string
}

type forwarding struct {
	mu    sync.Mutex
	info  Forward
	stop  chan struct{}
	done  chan struct{}
	once  sync.Once
	abort func()
	ports io.Closer
	conns io.Closer
	gone  bool
}

func (f *forwarding) closeListeners() {
	f.mu.Lock()
	ports := f.ports
	f.mu.Unlock()
	if ports != nil {
		ports.Close()
	}
}

func (f *forwarding) closeConnections() {
	f.mu.Lock()
	conns := f.conns
	f.mu.Unlock()
	if conns != nil {
		conns.Close()
	}
}

type portsCloser struct {
	pf *clientforward.PortForwarder
}

func (c portsCloser) Close() error {
	c.pf.Close()
	return nil
}

func (f *forwarding) snapshot() Forward {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.info
	out.Ports, out.Listening = slices.Clone(f.info.Ports), slices.Clone(f.info.Listening)
	return out
}

func (f *forwarding) halt(how Ended) {
	f.once.Do(func() {
		f.mu.Lock()
		if f.info.Ended == "" {
			f.info.Ended = how
		}
		f.mu.Unlock()
		close(f.stop)
	})
}

type forwards struct {
	mu    sync.Mutex
	next  int
	open  []*forwarding
	ended []Forward
}

func (t *Tree) Forwards() []Forward {
	t.fw.mu.Lock()
	defer t.fw.mu.Unlock()
	out := make([]Forward, 0, len(t.fw.open))
	for _, f := range t.fw.open {
		if snap := f.snapshot(); len(snap.Listening) > 0 {
			out = append(out, snap)
		}
	}
	return out
}

func (t *Tree) TakeEndedForwards() []Forward {
	t.fw.mu.Lock()
	defer t.fw.mu.Unlock()
	out := t.fw.ended
	t.fw.ended = nil
	return out
}

func (t *Tree) StopForward(id int) bool {
	t.fw.mu.Lock()
	var target *forwarding
	for _, f := range t.fw.open {
		if f.info.ID == id {
			target = f
		}
	}
	t.fw.mu.Unlock()
	if target == nil {
		return false
	}
	t.letGo(target, EndedLeft)
	return true
}

func (t *Tree) letGo(f *forwarding, how Ended) {
	listening := len(f.snapshot().Listening) > 0
	if listening {
		f.closeListeners()
	}
	f.halt(how)
	f.closeConnections()
	if !listening && f.abort != nil {
		f.abort()
	}
	t.release(f, nil)
}

func (t *Tree) stopForwards() {
	t.fw.mu.Lock()
	open := slices.Clone(t.fw.open)
	t.fw.mu.Unlock()
	for _, f := range open {
		t.letGo(f, EndedClosed)
	}
}

func (t *Tree) addForward(info Forward) *forwarding {
	t.fw.mu.Lock()
	defer t.fw.mu.Unlock()
	t.fw.next++
	info.ID = t.fw.next
	f := &forwarding{info: info, stop: make(chan struct{}), done: make(chan struct{})}
	t.fw.open = append(t.fw.open, f)
	return f
}

func (t *Tree) endForward(f *forwarding, err error) {
	t.release(f, err)
	close(f.done)
}

func (t *Tree) release(f *forwarding, err error) {
	f.mu.Lock()
	if f.gone {
		f.mu.Unlock()
		return
	}
	f.gone = true
	switch {
	case f.info.Ended != "":
	case err == nil:
		f.info.Ended = EndedExited
	default:
		f.info.Ended, f.info.Err = EndedFailed, write.Redact(err.Error())
	}
	f.mu.Unlock()
	t.fw.mu.Lock()
	t.fw.open = slices.DeleteFunc(t.fw.open, func(o *forwarding) bool { return o == f })
	t.fw.ended = append(t.fw.ended, f.snapshot())
	t.fw.mu.Unlock()
}

var forwardingFrom = regexp.MustCompile(`^Forwarding from (\S+) -> (\d+)`)

type forwardOut struct {
	f      *forwarding
	mu     sync.Mutex
	stone  io.Writer
	notice bytes.Buffer
}

func (w *forwardOut) detach() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stone = nil
}

func (w *forwardOut) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.notice.Write(p)
	for {
		line, err := w.notice.ReadString('\n')
		if err != nil {
			w.notice.Reset()
			w.notice.WriteString(line)
			break
		}
		if w.f != nil {
			w.f.mu.Lock()
			switch m := forwardingFrom.FindStringSubmatch(line); {
			case m != nil:
				w.f.info.Listening = append(w.f.info.Listening, m[1]+" -> "+m[2])
			case strings.HasPrefix(line, "Handling connection for "):
				w.f.info.Connections++
			}
			w.f.mu.Unlock()
		}
		if w.stone != nil {
			io.WriteString(w.stone, line)
		}
	}
	return len(p), nil
}

func (s *reach) forwardRun(cmd *cobra.Command, args []string) {
	typed, _ := cmd.Flags().GetStringSlice("address")
	changed := cmd.Flags().Changed("address")
	out := &forwardOut{stone: s.ios.Out}
	errOut := &forwardOut{stone: s.ios.ErrOut}
	o := portforward.NewDefaultPortForwardOptions(genericiooptions.IOStreams{In: s.ios.In, Out: out, ErrOut: errOut})
	cmdutil.CheckErr(o.Complete(s.f, cmd, args))
	o.Address = []string{ForwardAddress}
	cmdutil.CheckErr(o.Validate())
	if changed && (len(typed) != 1 || typed[0] != ForwardAddress) {
		fmt.Fprintf(s.ios.ErrOut, "Kubernites listens on %s only; --address %s was not used.\n", ForwardAddress, strings.Join(typed, ","))
	}
	pod, err := o.PodClient.Pods(o.Namespace).Get(s.ctx, o.PodName, metav1.GetOptions{})
	cmdutil.CheckErr(err)
	if pod.Status.Phase != corev1.PodRunning {
		cmdutil.CheckErr(fmt.Errorf("unable to forward port because pod is not running. Current status=%v", pod.Status.Phase))
	}
	u := o.RESTClient.Get().Resource("pods").Namespace(o.Namespace).Name(pod.Name).SubResource("portforward").URL()
	t := s.tree
	t.holdMu.Lock()
	closing := t.closing
	t.holdMu.Unlock()
	if closing {
		cmdutil.CheckErr(errors.New("the Stone's kubectl is closed"))
	}
	t.holdMu.Lock()
	abort := t.running
	t.holdMu.Unlock()
	registered := t.addForward(Forward{Target: args[0], Namespace: o.Namespace, Pod: pod.Name, Ports: slices.Clone(o.Ports)})
	registered.abort = abort
	out.f, errOut.f = registered, registered
	o.StopChannel = registered.stop
	ended := make(chan error, 1)
	session := &Session{Verb: "port-forward", Channel: "portforward", Namespace: o.Namespace, Pod: pod.Name}
	s.run.opened(session)
	dialer, held, err := forwarddial.Dialer("POST", u, o.Config, !cmdutil.PortForwardWebsockets.IsDisabled())
	cmdutil.CheckErr(err)
	registered.mu.Lock()
	registered.conns = held
	registered.mu.Unlock()
	pf, err := clientforward.NewOnAddressesForStreaming(dialer, o.Address, o.Ports, registered.stop, o.ReadyChannel, out, errOut)
	cmdutil.CheckErr(err)
	registered.mu.Lock()
	registered.ports = portsCloser{pf: pf}
	registered.mu.Unlock()
	go func() {
		err := pf.ForwardPorts()
		out.detach()
		errOut.detach()
		t.endForward(registered, err)
		ended <- err
	}()
	select {
	case <-o.ReadyChannel:
		out.detach()
		errOut.detach()
		session.Opened, session.Ended = true, ""
		fmt.Fprintf(s.ios.ErrOut, "The forward stays open in the background: the live card lists it with STOP.\n")
	case err := <-ended:
		session.Ended = EndedFailed
		if err != nil {
			session.Err = write.Redact(err.Error())
		}
		cmdutil.CheckErr(err)
	case <-s.ctx.Done():
		t.letGo(registered, EndedClosed)
		session.Ended = EndedClosed
	}
}
