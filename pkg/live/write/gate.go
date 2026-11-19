package write

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"

	"k8s.io/client-go/rest"
)

const (
	rulesReview  = "/apis/authorization.k8s.io/v1/selfsubjectrulesreviews"
	accessReview = "/apis/authorization.k8s.io/v1/selfsubjectaccessreviews"
	selfReview   = "/apis/authentication.k8s.io/v1/selfsubjectreviews"
)

var reviews = map[string]bool{rulesReview: true, accessReview: true, selfReview: true}

type Ticket string

type classesKey struct{}

func WithClasses(ctx context.Context, cs ...Class) context.Context {
	prev, _ := ctx.Value(classesKey{}).([]Class)
	return context.WithValue(ctx, classesKey{}, append(slices.Clone(prev), cs...))
}

func withDeclared(need []Class, ctx context.Context) []Class {
	declared, _ := ctx.Value(classesKey{}).([]Class)
	for _, c := range declared {
		if KnownClass(c) && !slices.Contains(need, c) {
			need = append(need, c)
		}
	}
	slices.Sort(need)
	return need
}

type Passage struct {
	Request
	DryRun bool
	Write  bool
	Code   int
	Err    error
}

type Gate struct {
	mu         sync.Mutex
	context    Context
	level      Level
	classes    map[Class]bool
	tickets    map[string]int
	stale      func() string
	observe    []func(Passage)
	client     *http.Client
	server     url.URL
	bound      *rest.Config
	local      map[string]bool
	aggregated map[string]bool
}

func New(c Choice) (*Gate, error) {
	if c.Level != Writes && c.Level != Seamless {
		return nil, errors.New("a READ ONLY connection uses the read gate, never the write gate")
	}
	if c.Context.Name == "" || c.Context.Server == "" {
		return nil, errors.New("the write gate needs the context and the API server the warning named")
	}
	g := &Gate{context: c.Context, level: c.Level, classes: map[Class]bool{}, tickets: map[string]int{}}
	for _, cl := range c.Classes {
		if KnownClass(cl) {
			g.classes[cl] = true
		}
	}
	return g, nil
}

func (g *Gate) Level() Level {
	return g.level
}

func (g *Gate) Context() Context {
	return g.context
}

func (g *Gate) Pass(c Class) {
	if !KnownClass(c) {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.classes[c] = true
}

var servedGroups = map[string]bool{
	"": true, "apps": true, "batch": true, "autoscaling": true, "policy": true, "rbac.authorization.k8s.io": true, "networking.k8s.io": true,
	"storage.k8s.io": true, "apiextensions.k8s.io": true, "admissionregistration.k8s.io": true, "coordination.k8s.io": true, "discovery.k8s.io": true,
	"events.k8s.io": true, "flowcontrol.apiserver.k8s.io": true, "node.k8s.io": true, "scheduling.k8s.io": true, "certificates.k8s.io": true,
	"authentication.k8s.io": true, "authorization.k8s.io": true, "resource.k8s.io": true, "internal.apiserver.k8s.io": true,
	"storagemigration.k8s.io": true, "apiregistration.k8s.io": true,
}

func (g *Gate) LearnLocalGroups(ctx context.Context) error {
	ans := g.do(ctx, Request{Method: http.MethodGet, Path: "/apis/apiregistration.k8s.io/v1/apiservices"}, "")
	if !ans.OK() {
		return errors.New(ans.Text())
	}
	var list struct {
		Items []struct {
			Spec struct {
				Group   string           `json:"group"`
				Service *json.RawMessage `json:"service"`
			} `json:"spec"`
		} `json:"items"`
	}
	if err := json.Unmarshal(ans.Body, &list); err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.local, g.aggregated = map[string]bool{}, map[string]bool{}
	for _, it := range list.Items {
		if it.Spec.Service != nil && string(*it.Spec.Service) != "null" {
			g.aggregated[it.Spec.Group] = true
		} else {
			g.local[it.Spec.Group] = true
		}
	}
	return nil
}

func (g *Gate) Previews(r Request) bool {
	group := parse(r.Path).group
	g.mu.Lock()
	defer g.mu.Unlock()
	return !g.aggregated[group] && (servedGroups[group] || g.local[group])
}

func (g *Gate) unpreviewable(group string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	why := "the " + group + " API group is served by an aggregated API server, which may make a change it is asked only to dry-run"
	if !g.aggregated[group] {
		why = "the " + group + " API group is not known to be served by the API server itself (its APIServices could not be read at connect), so its dry run could make the change"
	}
	return why + "; a change with no preview is never sent at WRITES. DANGEROUSLY SEAMLESS sends it as kubectl does"
}

func (g *Gate) Forget() {
	g.mu.Lock()
	defer g.mu.Unlock()
	clear(g.classes)
}

func (g *Gate) Passed(c Class) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.classes[c]
}

func (g *Gate) Classes() []Class {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []Class
	for _, c := range AllClasses {
		if g.classes[c] {
			out = append(out, c)
		}
	}
	return out
}

func (g *Gate) Issue(r Request) Ticket {
	return g.issue(r, 1)
}

func (g *Gate) issue(r Request, n int) Ticket {
	g.mu.Lock()
	defer g.mu.Unlock()
	key := r.Key()
	g.tickets[key] = n
	return Ticket(key)
}

func (g *Gate) Revoke(t Ticket) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.tickets, string(t))
}

func (g *Gate) StaleWhen(fn func() string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.stale = fn
}

func (g *Gate) Stale() string {
	g.mu.Lock()
	fn := g.stale
	g.mu.Unlock()
	if fn == nil {
		return ""
	}
	return fn()
}

func (g *Gate) Observe(fn func(Passage)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.observe = append(g.observe, fn)
}

func (g *Gate) WrapConfig(cfg *rest.Config) *rest.Config {
	out := rest.CopyConfig(cfg)
	base, ok := "", false
	var origin *url.URL
	if u, _, err := rest.DefaultServerUrlFor(out); err == nil {
		base, ok, origin = strings.TrimRight(u.Path, "/"), true, u
	}
	other := g.elsewhere(out)
	out.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return gated{g: g, next: rt, base: base, known: ok, other: other, origin: origin}
	})
	return out
}

func (g *Gate) elsewhere(cfg *rest.Config) string {
	g.mu.Lock()
	bound := g.bound
	g.mu.Unlock()
	if bound != nil {
		return Elsewhere(cfg, bound)
	}
	if !sameServer(cfg.Host, g.context.Server) {
		return "the server " + cfg.Host + " is not " + g.context.Server + ", the one the warning named"
	}
	return ""
}

func (g *Gate) Client(cfg *rest.Config) (*http.Client, error) {
	g.mu.Lock()
	if g.client != nil || g.bound != nil {
		g.mu.Unlock()
		return nil, errors.New("the write gate is bound to one connection, once")
	}
	g.bound = rest.CopyConfig(cfg)
	g.mu.Unlock()
	wrapped := g.WrapConfig(cfg)
	wrapped.UserAgent = UserAgent(g.level)
	server, _, err := rest.DefaultServerUrlFor(wrapped)
	if err != nil {
		return nil, err
	}
	client, err := rest.HTTPClientFor(wrapped)
	if err != nil {
		return nil, err
	}
	g.mu.Lock()
	g.client, g.server = client, *server
	g.mu.Unlock()
	return client, nil
}

type gated struct {
	g      *Gate
	next   http.RoundTripper
	base   string
	known  bool
	other  string
	origin *url.URL
}

func (t gated) RoundTrip(req *http.Request) (*http.Response, error) {
	other := t.other
	if other == "" && t.origin != nil && !SameOrigin(req.URL, t.origin) {
		other = "the request is for " + req.URL.Scheme + "://" + req.URL.Host + ", not the connected API server"
	}
	if other != "" {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, &Refusal{Method: req.Method, Path: req.URL.EscapedPath(), Why: NotAtLevel, Level: t.g.level, Detail: other}
	}
	out, seen, err := t.g.admit(req, t.base, t.known)
	if err != nil {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, err
	}
	resp, err := t.next.RoundTrip(out)
	t.g.tell(seen, resp, err)
	return resp, err
}

type kind int

const (
	kindRead kind = iota
	kindReview
	kindDryRun
	kindWrite
)

type seen struct {
	Request
	kind kind
}

func (g *Gate) admit(req *http.Request, base string, known bool) (*http.Request, seen, error) {
	path := req.URL.EscapedPath()
	refuse := func(why Why, class Class, detail string) (*http.Request, seen, error) {
		return nil, seen{}, &Refusal{Method: req.Method, Path: path, Why: why, Class: class, Level: g.level, Detail: detail}
	}
	if !known {
		return refuse(NotAtLevel, "", "the API server's address cannot be read")
	}
	switch req.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return refuse(NotAtLevel, "", "only GET, POST, PUT, PATCH and DELETE reach a cluster")
	}
	if path != req.URL.Path || unusualPath(path) {
		return refuse(NotAtLevel, "", "the path is not a plain API path")
	}
	if base != "" && path != base && !strings.HasPrefix(path, base+"/") {
		return refuse(NotAtLevel, "", "the path is outside the cluster's API")
	}
	query, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return refuse(NotAtLevel, "", "the query cannot be read")
	}
	if req.Header.Get("Content-Encoding") != "" {
		return refuse(NotAtLevel, "", "a compressed body cannot be checked")
	}
	rest := strings.TrimPrefix(path, base)
	if rest == "" {
		rest = "/"
	}
	t := parse(rest)
	if !t.api && !readableRoots[t.root] {
		return refuse(NotAtLevel, "", "the path is not part of the cluster's API")
	}
	body, err := readBody(req)
	if err != nil {
		return refuse(NotAtLevel, "", err.Error())
	}
	r := Request{Method: req.Method, Path: rest, Query: query, Body: body}
	upgrade := upgrading(req.Header)
	k := kindWrite
	switch {
	case req.Method == http.MethodPost && reviews[rest] && !upgrade:
		k = kindReview
	case req.Method == http.MethodGet && !upgrade && !t.legacyProxy && !(t.resource != "" && channels[t.sub()]):
		k = kindRead
	case !t.api:
		return refuse(NotAtLevel, "", "only reads reach this path")
	}
	_, connects := t.connect(req.Method)
	if t.reaches(upgrade) && !connects {
		return refuse(NotAtLevel, "", "only the exec, attach, port-forward and proxy of a Pod and the proxy of a Service or Node reach into a workload")
	}
	need := withDeclared(classesOf(r, t, upgrade), req.Context())
	honoured := r.DryRun() && !connects && g.Previews(r)
	ticketed := false
	if k == kindWrite && g.level == Writes && !honoured {
		g.mu.Lock()
		key := r.Key()
		if g.tickets[key] > 0 {
			ticketed = true
			g.tickets[key]--
			if g.tickets[key] == 0 {
				delete(g.tickets, key)
			}
		}
		g.mu.Unlock()
	}
	if k == kindWrite && g.level == Writes && !connects && !g.Previews(r) {
		return refuse(NotAtLevel, "", g.unpreviewable(t.group))
	}
	if k == kindWrite {
		if why := g.Stale(); why != "" {
			return refuse(Stale, "", why)
		}
	}
	g.mu.Lock()
	missing := Class("")
	for _, c := range need {
		if !g.classes[c] {
			missing = c
			break
		}
	}
	if missing != "" && k != kindReview {
		g.mu.Unlock()
		return refuse(NeedsClass, missing, "")
	}
	if k == kindWrite && honoured {
		k = kindDryRun
	}
	if k == kindWrite && g.level == Writes && !ticketed {
		g.mu.Unlock()
		return refuse(NeedsConfirm, "", "")
	}
	g.mu.Unlock()
	out := req.Clone(req.Context())
	if body != nil {
		out.Body = io.NopCloser(bytes.NewReader(body))
		out.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
		out.ContentLength = int64(len(body))
	}
	return out, seen{Request: r, kind: k}, nil
}

func (g *Gate) tell(s seen, resp *http.Response, err error) {
	g.mu.Lock()
	fns := slices.Clone(g.observe)
	g.mu.Unlock()
	if len(fns) == 0 {
		return
	}
	p := Passage{Request: s.Request, DryRun: s.kind == kindDryRun, Write: s.kind == kindWrite, Err: err}
	if resp != nil {
		p.Code = resp.StatusCode
	}
	for _, fn := range fns {
		fn(p)
	}
}
