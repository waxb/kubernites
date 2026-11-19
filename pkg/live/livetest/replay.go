package livetest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
)

type Exchange struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Query       string `json:"query,omitempty"`
	Accept      string `json:"accept,omitempty"`
	Status      int    `json:"status"`
	ContentType string `json:"contentType,omitempty"`
	Body        string `json:"body"`
}

type Conflict struct {
	Manager     string   `json:"manager"`
	Subresource string   `json:"subresource,omitempty"`
	Fields      []string `json:"fields"`
}

type Ask struct {
	Command     string `json:"command"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Query       string `json:"query,omitempty"`
	Body        string `json:"body"`
	ContentType string `json:"contentType,omitempty"`
	Code        int    `json:"code"`
	Message     string `json:"message,omitempty"`
}

type Recording struct {
	Context   string            `json:"context"`
	Exchanges []Exchange        `json:"exchanges"`
	Conflicts []Conflict        `json:"conflicts,omitempty"`
	Ask       *Ask              `json:"ask,omitempty"`
	Sessions  map[string]string `json:"sessions,omitempty"`
}

func queryKey(raw string) string {
	q, err := url.ParseQuery(raw)
	if err != nil {
		return raw
	}
	q.Del("timeout")
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		for _, v := range q[k] {
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, "&")
}

type Recorder struct {
	mu   sync.Mutex
	Next http.RoundTripper
	Rec  *Recording
}

func (r *Recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.Next.RoundTrip(req)
	if err != nil || req.Method != http.MethodGet || req.URL.Query().Get("watch") != "" || resp.StatusCode == http.StatusSwitchingProtocols || req.Header.Get("Sec-Websocket-Protocol") != "" {
		return resp, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Rec.Exchanges = append(r.Rec.Exchanges, Exchange{Method: req.Method, Path: req.URL.Path, Query: queryKey(req.URL.RawQuery), Accept: req.Header.Get("Accept"),
		Status: resp.StatusCode, ContentType: resp.Header.Get("Content-Type"), Body: string(body)})
	return resp, nil
}

func ReadRecording(path string) (Recording, error) {
	var rec Recording
	data, err := os.ReadFile(path)
	if err != nil {
		return rec, err
	}
	err = json.Unmarshal(data, &rec)
	return rec, err
}

func ServeRecording(rec Recording) *httptest.Server {
	srv, _ := ServeRecordingSeen(rec)
	return srv
}

func ServeRecordingSeen(rec Recording) (*httptest.Server, *Record) {
	seen := &Record{}
	byKey, byPath := map[string]Exchange{}, map[string]Exchange{}
	for _, e := range rec.Exchanges {
		byKey[e.Method+" "+e.Path+"?"+e.Query+" "+e.Accept] = e
		byPath[e.Method+" "+e.Path+"?"+e.Query] = e
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.add(r)
		q := queryKey(r.URL.RawQuery)
		e, ok := byKey[r.Method+" "+r.URL.Path+"?"+q+" "+r.Header.Get("Accept")]
		if !ok {
			e, ok = byPath[r.Method+" "+r.URL.Path+"?"+q]
		}
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404,"message":"not in the recording"}`)
			return
		}
		if e.ContentType != "" {
			w.Header().Set("Content-Type", e.ContentType)
		}
		w.WriteHeader(e.Status)
		io.WriteString(w, e.Body)
	})), seen
}
