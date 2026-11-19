package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	watchAdded    = "ADDED"
	watchModified = "MODIFIED"
	watchDeleted  = "DELETED"
	watchBookmark = "BOOKMARK"
	watchError    = "ERROR"
	backoffFirst  = 800 * time.Millisecond
	backoffMost   = 30 * time.Second
	StaleAfter    = 20 * time.Second
	watchSpan     = 300
	shortWatch    = time.Second
	MetricsEvery  = 15 * time.Second
	maxEventBytes = 16 << 20
	anyVersion    = "0"
	ListEvery     = 10 * time.Second
	listSpacing   = time.Second
)

var (
	errGone       = errors.New("the watch's resource version is too old for the server (410 Gone)")
	errShortWatch = errors.New("the watch closed at once with nothing in it")
)

type wire struct {
	list   lister
	stream func(ctx context.Context, path string, query url.Values, accept string) (io.ReadCloser, error)
	usage  func(ctx context.Context) Usage
	now    func() time.Time
	sleep  func(ctx context.Context, d time.Duration) bool
}

func wireOf(c *Conn) *wire {
	return &wire{
		list: c,
		stream: func(ctx context.Context, path string, query url.Values, accept string) (io.ReadCloser, error) {
			return openWatch(ctx, c, path, query, accept)
		},
		usage: c.Usage,
		now:   time.Now,
		sleep: sleepFor,
	}
}

func sleepFor(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func openWatch(ctx context.Context, c *Conn, path string, query url.Values, accept string) (io.ReadCloser, error) {
	u := c.server
	u.Path = strings.TrimRight(c.server.Path, "/") + path
	u.RawPath = ""
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if accept == "" {
		accept = "application/json"
	}
	req.Header.Set("Accept", accept)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_, err := answer(resp, nil)
		return nil, err
	}
	return resp.Body, nil
}

type watchEvent struct {
	Type   string          `json:"type"`
	Object json.RawMessage `json:"object"`
}

type kindHealth struct {
	failing time.Time
	err     string
}

type Freshness struct {
	Watching   bool
	StaleSince time.Time
	Why        string
}

func goneStatus(code int32, reason metav1.StatusReason) bool {
	return code == http.StatusGone || reason == metav1.StatusReasonExpired || reason == metav1.StatusReasonGone
}

func isGone(err error) bool {
	var se *StatusError
	return errors.As(err, &se) && goneStatus(int32(se.Code), metav1.StatusReason(se.Reason))
}

func watchTimeout(name string) string {
	return strconv.Itoa(watchSpan + int(hash64(name)%watchSpan))
}

func (r *Realm) StartWatching() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wire == nil || r.watching || r.stopped {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.watching, r.cancel = true, cancel
	r.reads = readBook{ctx: ctx, custom: map[string]customWatch{}, listing: map[string]Listed{}}
	for _, k := range kindReads {
		m, ok := r.snap.Lists[k.name]
		if !ok {
			continue
		}
		go r.watchKind(ctx, k, m)
	}
	r.reconcileCustom(true)
	go r.readMetrics(ctx)
}

func (r *Realm) Stop() {
	r.mu.Lock()
	r.stopped = true
	if r.cancel != nil {
		r.cancel()
	}
	tree := r.tree
	r.mu.Unlock()
	if tree != nil {
		tree.Close()
	}
}

func (r *Realm) watchKind(ctx context.Context, k kindRead, m ListMark) {
	version, ns := m.Version, m.Namespace
	read, listing := version != "", r.listsOnly(k.name)
	var refusal *Refusal
	delay := backoffFirst
	for ctx.Err() == nil {
		if listing || version == "" {
			since := ""
			if listing {
				since = version
			}
			v, err := r.relist(ctx, k, ns, since)
			if err == nil {
				version, read, delay = v, true, backoffFirst
				if refusal != nil {
					r.refused(ctx, *refusal)
					refusal = nil
				}
				if listing && !r.wire.sleep(ctx, r.listEvery(k.name)) {
					return
				}
				continue
			}
			switch {
			case ctx.Err() != nil:
				return
			case isGone(err) && since != "":
				version = ""
				continue
			case !read && k.custom && IsForbidden(err) && k.namespaced && ns == "":
				ns = r.home()
				refusal = &Refusal{List: k.name, Namespace: ns, Message: messageOf(err)}
				continue
			case !read && k.custom && IsForbidden(err):
				r.unread(ctx, k.name, refusal, err)
				return
			case !(k.custom && IsNotFound(err)):
				r.failed(ctx, k.name, err)
			}
			if !r.wire.sleep(ctx, delay) {
				return
			}
			delay = r.later(delay, listing)
			continue
		}
		began := r.wire.now()
		got, err := r.watchOnce(ctx, k, ns, &version)
		switch {
		case ctx.Err() != nil:
			return
		case IsForbidden(err):
			listing = true
			r.fallBack(ctx, k.name)
			continue
		case errors.Is(err, errGone):
			version = ""
			continue
		case err == nil && (got > 0 || r.wire.now().Sub(began) >= shortWatch):
			delay = backoffFirst
			continue
		case err == nil:
			err = errShortWatch
		}
		r.failed(ctx, k.name, err)
		if !r.wire.sleep(ctx, delay) {
			return
		}
		delay = r.later(delay, listing)
	}
}

func (r *Realm) later(delay time.Duration, listing bool) time.Duration {
	if listing {
		return min(2*delay, ListEvery)
	}
	return min(2*delay, backoffMost)
}

func (r *Realm) watchOnce(ctx context.Context, k kindRead, ns string, version *string) (int, error) {
	q := url.Values{"watch": {"true"}, "resourceVersion": {*version}, "allowWatchBookmarks": {"true"}, "timeoutSeconds": {watchTimeout(k.name)}}
	body, err := r.wire.stream(ctx, k.path(ns), q, k.accept)
	if err != nil {
		if isGone(err) {
			return 0, errGone
		}
		return 0, err
	}
	defer body.Close()
	r.healthy(k.name)
	return r.follow(k, io.LimitReader(body, maxEventBytes<<6), version)
}

func (r *Realm) follow(k kindRead, body io.Reader, version *string) (int, error) {
	dec := json.NewDecoder(body)
	n := 0
	for {
		var ev watchEvent
		if err := dec.Decode(&ev); err != nil {
			if errors.Is(err, io.EOF) {
				return n, nil
			}
			return n, err
		}
		switch ev.Type {
		case watchError:
			var st metav1.Status
			_ = json.Unmarshal(ev.Object, &st)
			if goneStatus(st.Code, st.Reason) {
				return n, errGone
			}
			return n, &StatusError{Code: int(st.Code), Reason: string(st.Reason), Message: st.Message}
		case watchBookmark:
			var o struct {
				Metadata metav1.ObjectMeta `json:"metadata"`
			}
			if json.Unmarshal(ev.Object, &o) == nil && o.Metadata.ResourceVersion != "" {
				*version = o.Metadata.ResourceVersion
			}
		case watchAdded, watchModified, watchDeleted:
			if len(ev.Object) > maxEventBytes {
				return n, fmt.Errorf("a %s event of %d bytes is larger than the realm reads", k.name, len(ev.Object))
			}
			v, err := r.ingest(k, ev.Type, ev.Object)
			if err != nil {
				return n, err
			}
			if v != "" {
				*version = v
			}
			n++
		}
	}
}

func (r *Realm) ingest(k kindRead, typ string, raw []byte) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	obj, err := k.feed(&r.snap, typ, raw)
	if err != nil {
		return "", err
	}
	r.mirrorOne(typ, obj)
	r.dirty = true
	if k.name == crdKind.name {
		r.reconcileCustom(false)
	}
	return obj.GetResourceVersion(), nil
}

func (r *Realm) relist(ctx context.Context, k kindRead, ns, since string) (string, error) {
	var fresh Snapshot
	rd := &reading{l: r.wire.list, s: &fresh, since: since}
	if err := k.read(ctx, rd, k.path(ns)); err != nil {
		return "", err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	before := k.items(&r.snap)
	k.swap(&r.snap, &fresh)
	r.mirrorSwap(before, k.items(&r.snap))
	r.dirty = true
	delete(r.health, k.name)
	version := rd.version
	if version == "" {
		version = anyVersion
	}
	if r.snap.Lists == nil {
		r.snap.Lists = map[string]ListMark{}
	}
	r.snap.Lists[k.name] = ListMark{Version: version, Namespace: ns}
	if k.name == crdKind.name {
		r.reconcileCustom(false)
	}
	return version, nil
}

func (r *Realm) readMetrics(ctx context.Context) {
	for r.wire.sleep(ctx, MetricsEvery) {
		u := r.wire.usage(ctx)
		if ctx.Err() != nil {
			return
		}
		r.mu.Lock()
		r.usage = u
		r.mu.Unlock()
	}
}

func (r *Realm) failed(ctx context.Context, name string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx.Err() != nil {
		return
	}
	h := r.health[name]
	if h.failing.IsZero() {
		h.failing = r.wire.now()
	}
	h.err = err.Error()
	r.health[name] = h
}

func (r *Realm) healthy(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.health, name)
}

func (r *Realm) Freshness() Freshness {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wire == nil {
		return Freshness{Watching: r.replayed, StaleSince: r.staleAt, Why: r.staleWhy}
	}
	if !r.watching {
		return Freshness{}
	}
	out := Freshness{Watching: true}
	now := r.wire.now()
	names := make([]string, 0, len(r.health))
	for name := range r.health {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		h := r.health[name]
		if h.failing.IsZero() || now.Sub(h.failing) < StaleAfter {
			continue
		}
		if out.StaleSince.IsZero() || h.failing.Before(out.StaleSince) {
			out.StaleSince, out.Why = h.failing, name+": "+h.err
		}
	}
	return out
}
