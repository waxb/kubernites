package live

import (
	"context"
	"slices"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const CustomWatchCap = 64

type Listed struct {
	List    string
	Every   time.Duration
	PastCap bool
}

type customWatch struct {
	crd    CRD
	cancel context.CancelFunc
	idle   bool
}

type readBook struct {
	ctx     context.Context
	custom  map[string]customWatch
	listing map[string]Listed
}

func (r *Realm) reconcileCustom(first bool) {
	if r.reads.ctx == nil || r.reads.ctx.Err() != nil {
		return
	}
	want := map[string]CRD{}
	for _, c := range r.snap.CRDs {
		if !c.native() {
			want[c.List()] = c
		}
	}
	running := make([]string, 0, len(r.reads.custom))
	for name := range r.reads.custom {
		running = append(running, name)
	}
	sort.Strings(running)
	for _, name := range running {
		cw := r.reads.custom[name]
		if c, ok := want[name]; ok && c == cw.crd {
			continue
		}
		if cw.cancel != nil {
			cw.cancel()
		}
		delete(r.reads.custom, name)
		delete(r.reads.listing, name)
		delete(r.health, name)
		delete(r.snap.Lists, name)
		r.dirty = true
	}
	streams := 0
	for name, cw := range r.reads.custom {
		if _, listed := r.reads.listing[name]; !cw.idle && !listed {
			streams++
		}
	}
	for _, c := range r.customOrder(want) {
		name := c.List()
		if _, ok := r.reads.custom[name]; ok {
			continue
		}
		m, marked := r.snap.Lists[name]
		if first && !marked {
			r.reads.custom[name] = customWatch{crd: c, idle: true}
			continue
		}
		if !first {
			m = ListMark{}
		}
		if streams >= CustomWatchCap {
			r.reads.listing[name] = Listed{List: name, PastCap: true}
		} else {
			streams++
		}
		ctx, cancel := context.WithCancel(r.reads.ctx)
		r.reads.custom[name] = customWatch{crd: c, cancel: cancel}
		go r.watchKind(ctx, customKind(c), m)
	}
}

func (r *Realm) customOrder(want map[string]CRD) []CRD {
	held := map[string]bool{}
	for _, cr := range r.snap.Custom {
		held[cr.Group+"/"+cr.Kind] = true
	}
	out := make([]CRD, 0, len(want))
	for _, c := range want {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := held[out[i].Group+"/"+out[i].Kind], held[out[j].Group+"/"+out[j].Kind]
		if a != b {
			return a
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (r *Realm) listsOnly(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.reads.listing[name]
	return ok
}

func (r *Realm) fallBack(ctx context.Context, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx.Err() != nil || r.reads.listing == nil {
		return
	}
	if _, ok := r.reads.listing[name]; !ok {
		r.reads.listing[name] = Listed{List: name}
		r.dirty = true
	}
}

func (r *Realm) listEvery(name string) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.every(r.reads.listing[name])
}

func (r *Realm) every(l Listed) time.Duration {
	if !l.PastCap {
		return ListEvery
	}
	past := 0
	for _, o := range r.reads.listing {
		if o.PastCap {
			past++
		}
	}
	return max(ListEvery, time.Duration(past)*listSpacing)
}

func (r *Realm) Listing() []Listed {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Listed, 0, len(r.reads.listing))
	for _, l := range r.reads.listing {
		l.Every = r.every(l)
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PastCap != out[j].PastCap {
			return out[j].PastCap
		}
		return out[i].List < out[j].List
	})
	return out
}

func (r *Realm) home() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ns := r.snap.Context.Namespace; ns != "" {
		return ns
	}
	return corev1.NamespaceDefault
}

func (r *Realm) refused(ctx context.Context, f Refusal) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx.Err() != nil || slices.ContainsFunc(r.snap.Refused, func(o Refusal) bool { return o.List == f.List }) {
		return
	}
	r.snap.Refused = append(r.snap.Refused, f)
	r.dirty = true
}

func (r *Realm) unread(ctx context.Context, name string, f *Refusal, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx.Err() != nil {
		return
	}
	if cw, ok := r.reads.custom[name]; ok {
		cw.idle = true
		r.reads.custom[name] = cw
	}
	delete(r.reads.listing, name)
	if f == nil {
		f = &Refusal{List: name, Message: messageOf(err)}
	}
	f.Namespace = ""
	if !slices.Contains(r.snap.Denied, name) {
		r.snap.Denied = append(r.snap.Denied, name)
		r.snap.Refused = append(r.snap.Refused, *f)
	}
	r.dirty = true
}
