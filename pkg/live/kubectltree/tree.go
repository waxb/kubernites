package kubectltree

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/waxb/kubernites/pkg/live/write"
)

const FieldManager = "kubernites"

var UserAgent = "kubernites-stone kubectl/" + ClientVersion() + " (" + runtime.GOOS + "/" + runtime.GOARCH + ")"

var (
	ErrNoGate    = errors.New("the Stone's kubectl runs only behind Kubernites' gate")
	ErrNoContext = errors.New("the Stone's kubectl needs the connected context")
)

type Config struct {
	Kubeconfig []string
	Context    string
	Wrap       func(*rest.Config) *rest.Config
	Local      string
}

type Tree struct {
	paths   []string
	context string
	wrap    func(*rest.Config) *rest.Config
	cache   string
	mu      sync.Mutex
	holdMu  sync.Mutex
	hold    Hold
	running context.CancelFunc
	closing bool
	fw      forwards
	local   string
}

type Held struct {
	Method    string
	URL       *url.URL
	Header    http.Header
	Body      []byte
	Err       error
	Command   []string
	Local     string
	Writing   bool
	Overwrite string
}

func (t *Tree) SetHold(fn Hold) {
	t.holdMu.Lock()
	defer t.holdMu.Unlock()
	t.hold = fn
}

func (t *Tree) holder() Hold {
	t.holdMu.Lock()
	defer t.holdMu.Unlock()
	return t.hold
}

type Result struct {
	Output   string
	Stdout   string
	Stderr   string
	Code     int
	Edit     *Edit
	Stopped  bool
	Refused  string
	Command  []string
	Requests int
	Writes   []Write
	Sessions []Session
}

type Write struct {
	Method string
	Path   string
	Name   string
	Code   int
	Status string
}

func (r Result) OK() bool {
	return r.Code == 0 && r.Refused == "" && !r.Stopped
}

func New(c Config) (*Tree, error) {
	if c.Wrap == nil {
		return nil, ErrNoGate
	}
	if c.Context == "" {
		return nil, ErrNoContext
	}
	paths := slices.DeleteFunc(slices.Clone(c.Kubeconfig), func(p string) bool { return p == "" })
	if len(paths) == 0 {
		return nil, errors.New("the Stone's kubectl needs the kubeconfig chosen at connect")
	}
	if len(paths) > 1 && !slices.Equal(paths, envKubeconfig()) {
		return nil, errors.New("the Stone's kubectl reads the kubeconfig files chosen at connect, and they are not the ones KUBECONFIG names")
	}
	quiet()
	cache, err := os.MkdirTemp("", "kubernites-stone-")
	if err != nil {
		return nil, err
	}
	local := c.Local
	if local == "" {
		if home, err := os.UserHomeDir(); err == nil {
			local = home
		}
	}
	if local == "" || !filepath.IsAbs(local) {
		os.RemoveAll(cache)
		return nil, errors.New("the Stone's kubectl needs a home directory for the local side of kubectl cp")
	}
	return &Tree{paths: paths, context: c.Context, wrap: c.Wrap, cache: cache, local: filepath.Clean(local)}, nil
}

func (t *Tree) Close() error {
	t.holdMu.Lock()
	t.closing = true
	if t.running != nil {
		t.running()
	}
	t.holdMu.Unlock()
	t.stopForwards()
	if !t.mu.TryLock() {
		return nil
	}
	defer t.mu.Unlock()
	return t.removeCache()
}

func (t *Tree) removeCache() error {
	if t.cache == "" {
		return nil
	}
	err := os.RemoveAll(t.cache)
	t.cache = ""
	return err
}

func (t *Tree) Context() string {
	return t.context
}

func envKubeconfig() []string {
	out := []string{}
	for _, p := range filepath.SplitList(os.Getenv("KUBECONFIG")) {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

var quietOnce sync.Once

type fatalExit struct {
	msg  string
	code int
}

func quiet() {
	quietOnce.Do(func() {
		os.Setenv("KUBERC", "off")
		os.Setenv("KUBECTL_KUBERC", "false")
		cmdutil.BehaviorOnFatal(func(msg string, code int) { panic(fatalExit{msg: msg, code: code}) })
		klog.OsExit = func(code int) { panic(fatalExit{code: code}) }
		klog.SetLogger(logr.Discard())
	})
}

type Hold func(context.Context, Held) bool

func (t *Tree) Run(ctx context.Context, line string) Result {
	return t.run(ctx, line, hooks{})
}

type hooks struct {
	stdin  *os.File
	sink   Sink
	hold   Hold
	wrote  func(Write)
	reveal bool
	term   *Terminal
	opened func(*Session)
}

type run struct {
	stdin    bool
	asking   sync.Mutex
	declared []write.Class
	hold     Hold
	local    string
	writing  bool
	command  []string
	reveal   bool
	refused  string
	edit     *editAsk
	requests int
	writes   []Write
	wrote    func(Write)
	term     *Terminal
	sessions []*Session
	onOpen   func(*Session)
	mu       sync.Mutex
}

func (r *run) opened(s *Session) {
	r.mu.Lock()
	r.sessions = append(r.sessions, s)
	tell := r.onOpen
	r.mu.Unlock()
	if tell != nil {
		tell(s)
	}
}

func (r *run) declare(c write.Class) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !slices.Contains(r.declared, c) {
		r.declared = append(r.declared, c)
	}
}

func (r *run) classes() []write.Class {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.declared)
}

func (r *run) localSide(path string, writing bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.local, r.writing = path, writing
}

func (r *run) held(h Held) Held {
	r.mu.Lock()
	defer r.mu.Unlock()
	h.Command, h.Local, h.Writing = slices.Clone(r.command), r.local, r.writing
	return h
}

func (r *run) ask(ctx context.Context, h Held) bool {
	if r.hold == nil {
		return false
	}
	r.asking.Lock()
	defer r.asking.Unlock()
	if ctx.Err() != nil {
		return false
	}
	return r.hold(ctx, r.held(h))
}

func (r *run) refuse(reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.refused == "" {
		r.refused = reason
	}
}

func (r *run) count() {
	r.mu.Lock()
	r.requests++
	r.mu.Unlock()
}

func (r *run) note(w Write) {
	r.mu.Lock()
	r.writes = append(r.writes, w)
	tell := r.wrote
	r.mu.Unlock()
	if tell != nil {
		tell(w)
	}
}

type Sink func(text string, stderr bool)

func (t *Tree) run(ctx context.Context, line string, h hooks) Result {
	args, err := Words(line)
	if err != nil {
		return Result{Output: "error: " + err.Error() + "\n", Stderr: "error: " + err.Error() + "\n", Code: 1}
	}
	if len(args) > 0 && args[0] == "kubectl" {
		args = args[1:]
	}
	return t.runAs(ctx, args, h)
}

func (t *Tree) runAs(ctx context.Context, args []string, h hooks) Result {
	t.mu.Lock()
	defer t.mu.Unlock()
	defer func() {
		t.holdMu.Lock()
		closing := t.closing
		t.holdMu.Unlock()
		if closing {
			t.removeCache()
		}
	}()
	res := Result{Command: slices.Clone(args)}
	t.holdMu.Lock()
	closing := t.closing
	t.holdMu.Unlock()
	if t.cache == "" || closing {
		return refusedResult(res, "the Stone's kubectl is closed")
	}
	if len(args) > 0 && strings.HasPrefix(args[0], "__") {
		return refusedResult(res, "kubectl "+args[0]+" is not part of the Stone: shell completion would run plugins")
	}
	if src, dst, ok := MigrationWouldWrite(); ok {
		return refusedResult(res, fmt.Sprintf("the Stone's kubectl will not run: client-go would copy %s to %s, and Kubernites never writes a kubeconfig", src, dst))
	}
	local := clientSide(args)
	if local && templated(args) && makesSecrets(args) {
		return refusedResult(res, ReasonClientSecret)
	}
	out := &streams{sink: h.sink, held: local}
	r := &run{command: slices.Clone(args), reveal: h.reveal, wrote: h.wrote, term: h.term, onOpen: h.opened, stdin: h.stdin != nil}
	ios := genericiooptions.IOStreams{In: bytes.NewReader(nil), Out: out.writer(&out.stdout, false), ErrOut: out.writer(&out.stderr, true)}
	if h.stdin != nil {
		saved := os.Stdin
		os.Stdin = h.stdin
		defer func() { os.Stdin = saved }()
	}
	if h.term != nil {
		ios.ErrOut = dropLine{w: ios.ErrOut, line: noTTYLine}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	t.holdMu.Lock()
	t.running = cancel
	t.holdMu.Unlock()
	defer func() {
		t.holdMu.Lock()
		t.running = nil
		t.holdMu.Unlock()
	}()
	hold := h.hold
	if hold == nil {
		hold = t.holder()
	}
	r.hold = hold
	cf := t.flags(ctx, ios, r, hold)
	root := kubectlcmd.NewKubectlCommand(kubectlcmd.KubectlOptions{ConfigFlags: cf, IOStreams: ios, Arguments: append([]string{"kubectl"}, args...)})
	t.prune(root, cf, ios, r)
	t.reachInto(root, cf, ios, r, ctx)
	guardFiles(root, r)
	if c, _, err := root.Find(args); err == nil && strings.HasPrefix(c.Name(), "__") {
		return refusedResult(res, "kubectl "+c.Name()+" is not part of the Stone: shell completion would run plugins")
	}
	root.SetArgs(append([]string{}, args...))
	root.SetIn(ios.In)
	root.SetOut(ios.Out)
	root.SetErr(ios.ErrOut)
	res.Code = execute(ctx, root, ios.ErrOut)
	res.Stopped = ctx.Err() != nil
	r.mu.Lock()
	res.Refused, res.Requests, res.Writes = r.refused, r.requests, slices.Clone(r.writes)
	for _, ss := range r.sessions {
		res.Sessions = append(res.Sessions, *ss)
	}
	r.mu.Unlock()
	if r.edit != nil && res.Code == 0 {
		res.Edit, res.Code = fetchEdit(cf, r.edit, ios.ErrOut)
	}
	if local {
		out.release()
	}
	res.Output, res.Stdout, res.Stderr = out.all.String(), out.stdout.String(), out.stderr.String()
	return res
}

func refusedResult(res Result, reason string) Result {
	res.Refused = reason
	res.Code = 1
	res.Output = "error: " + reason + "\n"
	res.Stderr = res.Output
	return res
}

func execute(ctx context.Context, root *cobra.Command, errOut io.Writer) (code int) {
	defer func() {
		if p := recover(); p != nil {
			f, ok := p.(fatalExit)
			if !ok {
				fmt.Fprintf(errOut, "error: kubectl stopped: %v\n", p)
				code = 1
				return
			}
			if f.msg != "" {
				msg := f.msg
				if !strings.HasSuffix(msg, "\n") {
					msg += "\n"
				}
				fmt.Fprint(errOut, msg)
			}
			code = max(f.code, 1)
		}
	}()
	root.SilenceErrors = true
	if !root.SilenceUsage {
		root.SilenceUsage = true
		root.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
			c.SilenceUsage = false
			return err
		})
	}
	if err := root.ExecuteContext(ctx); err != nil {
		cmdutil.CheckErr(err)
	}
	return 0
}

func keep(buf *bytes.Buffer, p []byte) {
	buf.Write(p)
	if buf.Len() <= MaxOutput {
		return
	}
	tail := append([]byte(nil), buf.Bytes()[buf.Len()-MaxOutput*3/4:]...)
	buf.Reset()
	buf.Write(tail)
}

type streams struct {
	mu     sync.Mutex
	all    bytes.Buffer
	stdout bytes.Buffer
	stderr bytes.Buffer
	sink   Sink
	held   bool
	local  bytes.Buffer
}

func (s *streams) release() {
	s.mu.Lock()
	text := maskLocal(s.local.String())
	s.held = false
	s.mu.Unlock()
	if text != "" {
		streamWriter{s: s, own: &s.stdout}.Write([]byte(text))
	}
}

type streamWriter struct {
	s      *streams
	own    *bytes.Buffer
	stderr bool
}

func (s *streams) writer(own *bytes.Buffer, stderr bool) io.Writer {
	return streamWriter{s: s, own: own, stderr: stderr}
}

func (w streamWriter) Write(p []byte) (int, error) {
	w.s.mu.Lock()
	defer w.s.mu.Unlock()
	if w.s.held && !w.stderr {
		keep(&w.s.local, p)
		return len(p), nil
	}
	keep(&w.s.all, p)
	keep(w.own, p)
	if w.s.sink != nil {
		w.s.sink(string(p), w.stderr)
	}
	return len(p), nil
}

func (t *Tree) flags(ctx context.Context, ios genericiooptions.IOStreams, r *run, hold Hold) *genericclioptions.ConfigFlags {
	cf := genericclioptions.NewConfigFlags(true).WithDiscoveryBurst(300).WithDiscoveryQPS(50.0).WithWarningPrinter(ios)
	*cf.CacheDir = t.cache
	cf.KubeConfig = nil
	if len(t.paths) == 1 {
		cf.KubeConfig = &t.paths[0]
	}
	*cf.Context = t.context
	gate := t.wrap
	cf.WrapConfigFn = func(c *rest.Config) *rest.Config {
		c.UserAgent = UserAgent
		c.ContentType = "application/json"
		c = gate(c)
		c.Wrap(func(rt http.RoundTripper) http.RoundTripper { return stopper{next: rt, ctx: ctx, run: r, hold: hold} })
		return c
	}
	return cf
}

func MigrationWouldWrite() (string, string, bool) {
	for dst, src := range clientcmd.NewDefaultClientConfigLoadingRules().MigrationRules {
		if _, err := os.Stat(dst); !errors.Is(err, os.ErrNotExist) {
			continue
		}
		if fi, err := os.Stat(src); err == nil && !fi.IsDir() {
			return src, dst, true
		}
	}
	return "", "", false
}

type stopper struct {
	next http.RoundTripper
	ctx  context.Context
	run  *run
	hold Hold
}

const maxHolds = 3

func (s stopper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := s.ctx.Err(); err != nil {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, err
	}
	s.run.count()
	if declared := s.run.classes(); len(declared) > 0 {
		req = req.WithContext(write.WithClasses(req.Context(), declared...))
	}
	masked := underSecrets(req) && !(s.run.reveal && req.Method == http.MethodGet)
	if masked {
		req = preferJSON(req)
	}
	ctx, cancel := context.WithCancel(req.Context())
	stop := context.AfterFunc(s.ctx, cancel)
	resp, err := s.next.RoundTrip(req.WithContext(ctx))
	for holds := 0; err != nil && s.hold != nil && holds < maxHolds && s.ctx.Err() == nil; holds++ {
		body, ok := replayable(req)
		if !ok || !s.run.ask(s.ctx, Held{Method: req.Method, URL: req.URL, Header: req.Header.Clone(), Body: body, Err: err}) {
			break
		}
		again := req.Clone(ctx)
		again.Body = nil
		if body != nil {
			again.Body = io.NopCloser(bytes.NewReader(body))
		}
		resp, err = s.next.RoundTrip(again)
	}
	if err != nil {
		stop()
		cancel()
		return nil, err
	}
	if w, ok := change(req, resp); ok {
		s.run.note(w)
	}
	if masked {
		resp = maskResponse(resp, req.Method == http.MethodGet && req.URL.Query().Get("watch") != "")
	}
	resp.Body = &cancelBody{ReadCloser: resp.Body, done: func() { stop(); cancel() }}
	return resp, nil
}

var selfReviews = []string{"/selfsubjectaccessreviews", "/selfsubjectrulesreviews", "/selfsubjectreviews"}

func change(req *http.Request, resp *http.Response) (Write, bool) {
	if req.Method == http.MethodGet || req.Method == http.MethodHead {
		return Write{}, false
	}
	if write.Connects(write.Request{Method: req.Method, Path: apiPath(req.URL.Path)}) {
		return Write{}, false
	}
	for _, s := range selfReviews {
		if req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, s) {
			return Write{}, false
		}
	}
	body, _ := replayable(req)
	if (write.Request{Method: req.Method, Path: req.URL.Path, Query: req.URL.Query(), Body: body}).DryRun() {
		return Write{}, false
	}
	w := Write{Method: req.Method, Path: req.URL.Path, Code: resp.StatusCode, Status: resp.Status}
	if req.Method == http.MethodPost {
		var obj struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		}
		if json.Unmarshal(body, &obj) == nil {
			w.Name = obj.Metadata.Name
		}
	}
	return w, true
}

func apiPath(path string) string {
	for _, root := range []string{"/api/", "/apis/"} {
		if i := strings.Index(path, root); i > 0 {
			return path[i:]
		}
	}
	return path
}

func replayable(req *http.Request) ([]byte, bool) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, true
	}
	if req.GetBody == nil {
		return nil, false
	}
	rc, err := req.GetBody()
	if err != nil {
		return nil, false
	}
	defer rc.Close()
	body, err := io.ReadAll(rc)
	return body, err == nil
}

type cancelBody struct {
	io.ReadCloser
	once sync.Once
	done func()
}

func (b *cancelBody) Close() error {
	err := b.ReadCloser.Close()
	b.once.Do(b.done)
	return err
}

func lockFlag(f *pflag.Flag, reason string, r *run) {
	if f == nil {
		return
	}
	f.Value = lockedValue{Value: f.Value, reason: reason, run: r}
}

type lockedValue struct {
	pflag.Value
	reason string
	run    *run
}

func (v lockedValue) Set(string) error {
	v.run.refuse(v.reason)
	return errors.New(v.reason)
}

type guardedFiles struct {
	pflag.Value
	run   *run
	keyed bool
}

func (v guardedFiles) Set(s string) error {
	for _, part := range strings.Split(s, ",") {
		p := strings.ToLower(strings.TrimSpace(part))
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			v.run.refuse(webFiles)
			return errors.New(webFiles)
		}
		if p == "-" && !v.run.stdin {
			v.run.refuse(noStdin)
			return errors.New(noStdin)
		}
		local := strings.TrimSpace(part)
		if v.keyed {
			if _, path, ok := strings.Cut(local, "="); ok {
				local = path
			}
		}
		if err := plainFiles(local); err != nil {
			v.run.refuse(err.Error())
			return err
		}
	}
	return v.Value.Set(s)
}

const maxWalk = 10000

func plainFiles(path string) error {
	if path == "" {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil
	}
	refuse := func(p string) error {
		return fmt.Errorf("the Stone reads only regular files and directories: %s is not one, and reading it could wait forever", p)
	}
	if st.Mode().IsRegular() {
		return nil
	}
	if !st.IsDir() {
		return refuse(path)
	}
	seen := 0
	return filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || p == path {
			return nil
		}
		if seen++; seen > maxWalk {
			return fmt.Errorf("the Stone reads at most %d entries of a directory: %s holds more", maxWalk, path)
		}
		st, err := os.Stat(p)
		if err != nil {
			return nil
		}
		if !st.Mode().IsRegular() && !st.IsDir() {
			return refuse(p)
		}
		return nil
	})
}

const (
	webFiles = "the Stone reads manifests from files: a web address would be fetched outside Kubernites' gate"
	noStdin  = "the Stone has no standard input: write the manifest in the Scriptorium"
)

const MaxOutput = 4 << 20
