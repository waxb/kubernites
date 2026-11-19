package livetest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

const ListVersion = "1000"

type Watch struct {
	Path    string
	Version string
	Query   map[string]string
	Accept  string
}

type Feed struct {
	mu      sync.Mutex
	watches []Watch
	queues  map[string]chan []byte
	gone    map[string]int
	broken  map[string]int
	fail    map[string]bool
	lists   map[string]any
	open    map[string]int
}

func NewFeed() *Feed {
	return &Feed{queues: map[string]chan []byte{}, gone: map[string]int{}, broken: map[string]int{}, fail: map[string]bool{}, lists: map[string]any{}, open: map[string]int{}}
}

func (f *Feed) SetList(path string, items any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lists[path] = items
}

func (f *Feed) listed(path string) bool {
	_, ok := f.list(path)
	return ok
}

func (f *Feed) list(path string) (any, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items, ok := f.lists[path]
	return items, ok
}

func (f *Feed) queue(path string) chan []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	q, ok := f.queues[path]
	if !ok {
		q = make(chan []byte, 256)
		f.queues[path] = q
	}
	return q
}

func (f *Feed) Send(path, typ string, obj any) {
	data, err := json.Marshal(map[string]any{"type": typ, "object": obj})
	if err != nil {
		panic(err)
	}
	f.queue(path) <- data
}

func (f *Feed) Close(path string) {
	f.queue(path) <- nil
}

func (f *Feed) Gone(path string, times int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gone[path] = times
}

func (f *Feed) Break(path string, times int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.broken[path] = times
}

func (f *Feed) Fail(path string, on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail[path] = on
}

func (f *Feed) Watches() []Watch {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Watch(nil), f.watches...)
}

func (f *Feed) Streaming(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for path, n := range f.open {
		if n > 0 && strings.HasPrefix(path, prefix) {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func (f *Feed) WatchesOf(path string) []Watch {
	var out []Watch
	for _, w := range f.Watches() {
		if w.Path == path {
			out = append(out, w)
		}
	}
	return out
}

func (f *Feed) serve(w http.ResponseWriter, r *http.Request) {
	q := map[string]string{}
	for k, v := range r.URL.Query() {
		q[k] = v[0]
	}
	path := r.URL.Path
	f.mu.Lock()
	f.watches = append(f.watches, Watch{Path: path, Version: q["resourceVersion"], Query: q, Accept: r.Header.Get("Accept")})
	gone, broken, fail := f.gone[path], f.broken[path], f.fail[path]
	if gone > 0 {
		f.gone[path]--
	}
	if broken > 0 {
		f.broken[path]--
	}
	f.mu.Unlock()
	switch {
	case fail:
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"kind":"Status","status":"Failure","reason":"InternalError","code":500,"message":"the watch cache is not ready"}`)
		return
	case gone > 0:
		w.WriteHeader(http.StatusGone)
		fmt.Fprint(w, `{"kind":"Status","status":"Failure","reason":"Expired","code":410,"message":"too old resource version"}`)
		return
	}
	f.mu.Lock()
	f.open[path]++
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.open[path]--
		f.mu.Unlock()
	}()
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}
	if broken > 0 {
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, err := hj.Hijack()
			if err == nil {
				conn.Close()
			}
		}
		return
	}
	queue := f.queue(path)
	for {
		select {
		case <-r.Context().Done():
			return
		case data := <-queue:
			if data == nil {
				return
			}
			_, _ = w.Write(append(data, '\n'))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}
