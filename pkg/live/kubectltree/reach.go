package kubectltree

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/moby/term"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/remotecommand"
	watchtools "k8s.io/client-go/tools/watch"
	uexec "k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/cmd/attach"
	"k8s.io/kubectl/pkg/cmd/debug"
	"k8s.io/kubectl/pkg/cmd/exec"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/waxb/kubernites/pkg/live/write"
)

type Ended string

const (
	EndedExited Ended = "exited"
	EndedLeft   Ended = "left"
	EndedClosed Ended = "closed"
	EndedFailed Ended = "failed"
)

type Session struct {
	Verb      string
	Channel   string
	Namespace string
	Pod       string
	Container string
	Command   []string
	Stdin     bool
	TTY       bool
	Opened    bool
	Ended     Ended
	Code      int
	Err       string
}

const noTTYLine = "Unable to use a TTY - input is not a terminal or the right kind of file\n"

type reach struct {
	tree    *Tree
	verb    string
	ctx     context.Context
	f       cmdutil.Factory
	ios     genericiooptions.IOStreams
	term    *Terminal
	run     *run
	stdin   bool
	tty     bool
	nudge   bool
	mu      sync.Mutex
	session *Session
}

func (t *Tree) reachInto(root *cobra.Command, cf *genericclioptions.ConfigFlags, ios genericiooptions.IOStreams, r *run, ctx context.Context) {
	f := cmdutil.NewFactory(cmdutil.NewMatchVersionFlags(cf))
	if c := child(root, "exec"); c != nil {
		c.Run = (&reach{verb: "exec", ctx: ctx, f: f, ios: ios, term: r.term, run: r}).execRun
	}
	if c := child(root, "attach"); c != nil {
		c.Run = (&reach{verb: "attach", ctx: ctx, f: f, ios: ios, term: r.term, run: r}).attachRun
	}
	if c := child(root, "debug"); c != nil {
		root.RemoveCommand(c)
		root.AddCommand((&reach{verb: "debug", ctx: ctx, f: f, ios: ios, term: r.term, run: r}).debugCommand(c))
	}
	if c := child(root, "run"); c != nil {
		create := c.Run
		c.Run = (&reach{verb: "run", ctx: ctx, f: f, ios: ios, term: r.term, run: r}).runRun(create)
	}
	if c := child(root, "cp"); c != nil {
		c.Run = (&reach{tree: t, verb: "cp", ctx: ctx, f: f, ios: ios, run: r}).cpRun
		lockFlag(c.Flags().Lookup("retries"), ReasonRetries, r)
	}
	if c := child(root, "port-forward"); c != nil {
		c.Run = (&reach{tree: t, verb: "port-forward", ctx: ctx, f: f, ios: ios, run: r}).forwardRun
	}
}

func flagBool(c *cobra.Command, name string) bool {
	v, _ := c.Flags().GetBool(name)
	return v
}

func flagString(c *cobra.Command, name string) string {
	v, _ := c.Flags().GetString(name)
	return v
}

func (s *reach) needsTerminal() {
	if s.stdin && s.term == nil {
		cmdutil.CheckErr(fmt.Errorf("kubectl %s -i needs the Stone's terminal", s.verb))
	}
}

func (s *reach) execRun(cmd *cobra.Command, args []string) {
	o := &exec.ExecOptions{StreamOptions: exec.StreamOptions{IOStreams: s.ios}, Executor: s}
	o.ContainerName = flagString(cmd, "container")
	o.Stdin, o.TTY, o.Quiet = flagBool(cmd, "stdin"), flagBool(cmd, "tty"), flagBool(cmd, "quiet")
	o.FilenameOptions.Filenames, _ = cmd.Flags().GetStringSlice("filename")
	s.stdin, s.tty = o.Stdin, o.Stdin && o.TTY
	s.needsTerminal()
	cmdutil.CheckErr(o.Complete(s.f, cmd, args, cmd.ArgsLenAtDash()))
	cmdutil.CheckErr(o.Validate())
	cmdutil.CheckErr(o.Run())
}

func (s *reach) Execute(u *url.URL, cfg *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, q remotecommand.TerminalSizeQueue) error {
	return s.ExecuteWithContext(s.ctx, u, cfg, stdin, stdout, stderr, tty, q)
}

func (s *reach) ExecuteWithContext(ctx context.Context, u *url.URL, cfg *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, _ remotecommand.TerminalSizeQueue) error {
	if s.stdin {
		query := u.Query()
		if s.tty {
			query.Set("tty", "true")
			query.Del("stderr")
		}
		u.RawQuery = query.Encode()
		return s.stream(ctx, u, cfg, nil, nil)
	}
	return s.stream(ctx, u, cfg, stdout, stderr)
}

func (s *reach) attachRun(cmd *cobra.Command, args []string) {
	o := attach.NewAttachOptions(s.ios)
	o.ContainerName = flagString(cmd, "container")
	o.Stdin, o.TTY, o.Quiet = flagBool(cmd, "stdin"), flagBool(cmd, "tty"), flagBool(cmd, "quiet")
	o.DetachKeys = flagString(cmd, "detach-keys")
	o.AttachFunc = s.attachFunc
	s.stdin = o.Stdin
	s.needsTerminal()
	cmdutil.CheckErr(o.Complete(s.f, cmd, args))
	cmdutil.CheckErr(o.Validate())
	cmdutil.CheckErr(o.Run())
}

func (s *reach) attachFunc(o *attach.AttachOptions, c *corev1.Container, _ bool, _ remotecommand.TerminalSizeQueue) func() error {
	return func() error {
		s.stdin = o.Stdin && s.term != nil
		s.tty = s.stdin && c.TTY
		s.nudge = s.tty
		client, err := s.f.RESTClient()
		if err != nil {
			return err
		}
		req := client.Get().Resource("pods").Name(o.Pod.Name).Namespace(o.Pod.Namespace).SubResource("attach")
		req.VersionedParams(&corev1.PodAttachOptions{Container: c.Name, Stdin: s.stdin, Stdout: o.Out != nil, Stderr: !s.tty, TTY: s.tty}, scheme.ParameterCodec)
		if s.stdin {
			return s.stream(s.ctx, req.URL(), o.Config, nil, nil)
		}
		return s.stream(s.ctx, req.URL(), o.Config, o.Out, o.ErrOut)
	}
}

func (s *reach) debugCommand(kubectl *cobra.Command) *cobra.Command {
	o := debug.NewDebugOptions(s.ios)
	c := &cobra.Command{Use: kubectl.Use, DisableFlagsInUseLine: true, Short: kubectl.Short, Long: kubectl.Long, Example: kubectl.Example, GroupID: kubectl.GroupID}
	c.Run = func(cmd *cobra.Command, args []string) {
		s.run.declare(write.Workloads)
		o.AttachFunc = func(ctx context.Context, getter genericclioptions.RESTClientGetter, cmdPath, ns, pod, container string) error {
			return s.debugAttach(ctx, getter, cmdPath, ns, pod, container, o)
		}
		s.stdin = flagBool(cmd, "stdin")
		s.needsTerminal()
		cmdutil.CheckErr(o.Complete(s.f, cmd, args))
		cmdutil.CheckErr(o.Validate())
		cmdutil.CheckErr(o.Run(s.f, cmd))
	}
	o.AddFlags(c)
	return c
}

func (s *reach) debugAttach(ctx context.Context, getter genericclioptions.RESTClientGetter, cmdPath, ns, podName, container string, d *debug.DebugOptions) error {
	pod, err := s.waitForContainer(ctx, ns, podName, container, d.Quiet)
	if err != nil {
		return err
	}
	if st := containerStatus(pod, container); st != nil && st.State.Terminated != nil {
		return fmt.Errorf("the debug container %s has already ended (%s): read it with kubectl logs %s -n %s -c %s", container, st.State.Terminated.Reason, podName, ns, container)
	}
	return s.attachTo(getter, cmdPath, pod, container, d.Interactive, d.TTY, d.Quiet)
}

func (s *reach) attachTo(getter genericclioptions.RESTClientGetter, cmdPath string, pod *corev1.Pod, container string, stdin, tty, quiet bool) error {
	cfg, err := getter.ToRESTConfig()
	if err != nil {
		return err
	}
	opts := &attach.AttachOptions{StreamOptions: exec.StreamOptions{IOStreams: s.ios, Stdin: stdin, TTY: tty, Quiet: quiet}, CommandName: cmdPath + " attach", Attach: &attach.DefaultRemoteAttach{}}
	opts.Config, opts.AttachFunc = cfg, s.attachFunc
	opts.Namespace, opts.Pod, opts.PodName, opts.ContainerName = pod.Namespace, pod, pod.Name, container
	if err := opts.Run(); err != nil {
		return fmt.Errorf("couldn't attach to pod/%s: %w", pod.Name, err)
	}
	return nil
}

func containerStatus(pod *corev1.Pod, name string) *corev1.ContainerStatus {
	for _, list := range [][]corev1.ContainerStatus{pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses, pod.Status.EphemeralContainerStatuses} {
		for i := range list {
			if list[i].Name == name {
				return &list[i]
			}
		}
	}
	return nil
}

func (s *reach) waitForContainer(ctx context.Context, ns, name, container string, quiet bool) (*corev1.Pod, error) {
	return s.waitForPod(ctx, ns, name, 0, func(p *corev1.Pod) (bool, error) {
		st := containerStatus(p, container)
		if st == nil {
			return false, nil
		}
		if st.State.Running != nil || st.State.Terminated != nil {
			return true, nil
		}
		if !quiet && st.State.Waiting != nil && st.State.Waiting.Message != "" {
			fmt.Fprintf(s.ios.ErrOut, "Warning: container %s: %s\n", container, st.State.Waiting.Message)
		}
		return false, nil
	})
}

func (s *reach) waitForPod(ctx context.Context, ns, name string, timeout time.Duration, ready func(*corev1.Pod) (bool, error)) (*corev1.Pod, error) {
	cs, err := s.f.KubernetesClientSet()
	if err != nil {
		return nil, err
	}
	ctx, cancel := watchtools.ContextWithOptionalTimeout(ctx, timeout)
	defer cancel()
	selector := fields.OneTermEqualSelector("metadata.name", name).String()
	pods := cs.CoreV1().Pods(ns)
	lw := &cache.ListWatch{
		ListFunc: func(o metav1.ListOptions) (runtime.Object, error) {
			o.FieldSelector = selector
			return pods.List(ctx, o)
		},
		WatchFunc: func(o metav1.ListOptions) (watch.Interface, error) {
			o.FieldSelector = selector
			return pods.Watch(ctx, o)
		},
	}
	ev, err := watchtools.UntilWithSync(ctx, lw, &corev1.Pod{}, nil, func(ev watch.Event) (bool, error) {
		if ev.Type == watch.Deleted {
			return false, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, name)
		}
		p, ok := ev.Object.(*corev1.Pod)
		if !ok {
			return false, fmt.Errorf("watch did not return a pod: %v", ev.Object)
		}
		return ready(p)
	})
	if ev == nil {
		return nil, err
	}
	p, _ := ev.Object.(*corev1.Pod)
	return p, err
}

func (s *reach) runRun(create func(*cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		stdin, tty, remove := flagBool(cmd, "stdin"), flagBool(cmd, "tty"), flagBool(cmd, "rm")
		attachFlag := cmd.Flags().Lookup("attach")
		attaching := flagBool(cmd, "attach") || stdin && (attachFlag == nil || !attachFlag.Changed)
		if !attaching {
			create(cmd, args)
			return
		}
		s.run.declare(write.Workloads)
		if tty && !stdin {
			cmdutil.CheckErr(errors.New("-i/--stdin is required for containers with -t/--tty=true"))
		}
		if dry := flagString(cmd, "dry-run"); dry != "" && dry != "none" {
			cmdutil.CheckErr(errors.New("--dry-run=[server|client] can't be used with attached containers options (--attach, --stdin, or --tty)"))
		}
		if len(args) == 0 {
			cmdutil.CheckErr(errors.New("NAME is required for run"))
		}
		s.stdin = stdin
		s.needsTerminal()
		cmd.Flags().Set("attach", "false")
		cmd.Flags().Set("rm", "false")
		create(cmd, args)
		ns, _, err := s.f.ToRawKubeConfigLoader().Namespace()
		cmdutil.CheckErr(err)
		timeout, _ := cmd.Flags().GetDuration("pod-running-timeout")
		pod, err := s.waitForPod(s.ctx, ns, args[0], timeout, func(p *corev1.Pod) (bool, error) {
			switch p.Status.Phase {
			case corev1.PodRunning, corev1.PodSucceeded, corev1.PodFailed:
				return true, nil
			}
			return false, nil
		})
		if err == nil {
			err = s.attachTo(s.f, cmd.Parent().CommandPath(), pod, args[0], stdin, tty, flagBool(cmd, "quiet"))
		}
		if remove {
			s.remove(ns, args[0])
		}
		cmdutil.CheckErr(err)
	}
}

func (s *reach) remove(ns, name string) {
	client, err := s.f.RESTClient()
	if err == nil {
		err = client.Delete().Namespace(ns).Resource("pods").Name(name).Do(context.Background()).Error()
	}
	if err != nil {
		fmt.Fprintf(s.ios.ErrOut, "error: %v\n", err)
		return
	}
	fmt.Fprintf(s.ios.Out, "pod %q deleted from %s namespace\n", name, ns)
}

func (s *reach) stream(ctx context.Context, u *url.URL, cfg *rest.Config, stdout, stderr io.Writer) error {
	session := s.begin(u)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	opts := remotecommand.StreamOptions{Stdout: stdout, Stderr: stderr}
	var keys *leaveReader
	if s.stdin {
		s.term.open(s.tty, s.nudge)
		s.term.stopWith(cancel)
		defer s.term.end()
		keys = &leaveReader{r: term.NewEscapeProxy(s.term, mustKeys(LeaveKeys)), cancel: cancel}
		opts.Stdin, opts.Stdout, opts.Tty = keys, s.term, s.tty
		if !s.tty {
			opts.Stderr = s.term
		} else {
			opts.TerminalSizeQueue = s.term
		}
	}
	err := s.executor(u, cfg, ctx, opts)
	if s.stdin {
		if failed := s.term.Failed(); failed != nil {
			err = failed
		}
	}
	s.finish(session, keys, err)
	if s.left(keys) {
		return nil
	}
	return err
}

func (s *reach) left(keys *leaveReader) bool {
	return keys != nil && (keys.left() || s.term.Left())
}

func (s *reach) executor(u *url.URL, cfg *rest.Config, ctx context.Context, opts remotecommand.StreamOptions) error {
	spdy, err := remotecommand.NewSPDYExecutor(cfg, "POST", u)
	if err != nil {
		return err
	}
	var e remotecommand.Executor = spdy
	if !cmdutil.RemoteCommandWebsockets.IsDisabled() {
		ws, err := remotecommand.NewWebSocketExecutor(cfg, "GET", u.String())
		if err != nil {
			return err
		}
		e, err = remotecommand.NewFallbackExecutor(ws, spdy, func(err error) bool {
			return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
		})
		if err != nil {
			return err
		}
	}
	return e.StreamWithContext(ctx, opts)
}

func mustKeys(keys string) []byte {
	b, err := term.ToBytes(keys)
	if err != nil {
		panic(err)
	}
	return b
}

type leaveReader struct {
	r      io.Reader
	cancel context.CancelFunc
	mu     sync.Mutex
	gone   bool
}

func (l *leaveReader) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	if errors.Is(err, term.EscapeError{}) {
		l.mu.Lock()
		l.gone = true
		l.mu.Unlock()
		l.cancel()
		return n, io.EOF
	}
	return n, err
}

func (l *leaveReader) left() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.gone
}

func (s *reach) begin(u *url.URL) *Session {
	path := u.EscapedPath()
	if i := strings.Index(path, "/api/"); i > 0 {
		path = path[i:]
	}
	o, _ := write.Opens(write.Request{Method: "POST", Path: path, Query: u.Query()})
	session := &Session{Verb: s.verb, Channel: o.Channel, Namespace: o.Namespace, Pod: o.Name, Container: o.Container, Command: o.Command, Stdin: s.stdin, TTY: s.stdin && s.tty}
	s.mu.Lock()
	s.session = session
	s.mu.Unlock()
	s.run.opened(session)
	return session
}

func (s *reach) finish(session *Session, keys *leaveReader, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session.Opened = err == nil || s.stdin && s.term.Live()
	var code uexec.ExitError
	switch {
	case s.left(keys):
		session.Ended = EndedLeft
	case s.ctx.Err() != nil:
		session.Ended = EndedClosed
	case err == nil:
		session.Ended = EndedExited
	case errors.As(err, &code):
		session.Ended, session.Code = EndedExited, code.ExitStatus()
		session.Opened = true
	default:
		session.Ended, session.Err = EndedFailed, write.Redact(err.Error())
	}
}

type dropLine struct {
	w    io.Writer
	line string
}

func (d dropLine) Write(p []byte) (int, error) {
	if string(p) == d.line {
		return len(p), nil
	}
	if i := bytes.Index(p, []byte(d.line)); i >= 0 && (i == 0 || p[i-1] == '\n') {
		if _, err := d.w.Write(append(append([]byte{}, p[:i]...), p[i+len(d.line):]...)); err != nil {
			return 0, err
		}
		return len(p), nil
	}
	return d.w.Write(p)
}
