package livetest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
)

type Record struct {
	mu      sync.Mutex
	seen    []string
	full    []string
	queries []Watch
}

func (r *Record) add(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := req.Method + " " + req.URL.Path
	if w := req.URL.Query().Get("watch"); w == "true" || w == "1" {
		seen += WatchMark
	} else if req.Method == http.MethodGet {
		q := map[string]string{}
		for k, v := range req.URL.Query() {
			q[k] = v[0]
		}
		r.queries = append(r.queries, Watch{Path: req.URL.Path, Version: q["resourceVersion"], Query: q, Accept: req.Header.Get("Accept")})
	}
	r.seen = append(r.seen, seen)
	full := req.Method + " " + req.URL.Path
	if req.URL.RawQuery != "" {
		full += "?" + req.URL.RawQuery
	}
	r.full = append(r.full, full)
}

func (r *Record) Full() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.full...)
}

const AccessReview = "POST /apis/authorization.k8s.io/v1/selfsubjectaccessreviews"

func (r *Record) ListsOf(path string) []Watch {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Watch
	for _, q := range r.queries {
		if q.Path == path {
			out = append(out, q)
		}
	}
	return out
}

func (r *Record) Seen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.seen...)
}

const WatchMark = "?watch"

const SelfReview = "POST /apis/authentication.k8s.io/v1/selfsubjectreviews"

func (r *Record) Writes() []string {
	var out []string
	for _, s := range r.Seen() {
		if !strings.HasPrefix(s, "GET ") && s != SelfReview {
			out = append(out, s)
		}
	}
	return out
}

func (r *Record) Reads() []string {
	var out []string
	for _, s := range r.Seen() {
		if strings.HasPrefix(s, "GET ") {
			out = append(out, s)
		}
	}
	return out
}

func Lists(s live.Snapshot) map[string]any {
	return map[string]any{
		"/api/v1/nodes": s.Nodes, "/api/v1/namespaces": s.Namespaces, "/api/v1/pods": s.Pods, "/api/v1/services": s.Services,
		"/api/v1/persistentvolumeclaims": s.PVCs, "/api/v1/persistentvolumes": s.PVs, "/api/v1/limitranges": s.LimitRanges, "/api/v1/events": s.Events,
		"/apis/discovery.k8s.io/v1/endpointslices": s.EndpointSlices, "/apis/apps/v1/deployments": s.Deployments, "/apis/apps/v1/replicasets": s.ReplicaSets,
		"/apis/apps/v1/daemonsets": s.DaemonSets, "/apis/apps/v1/statefulsets": s.StatefulSets, "/apis/batch/v1/jobs": s.Jobs, "/apis/batch/v1/cronjobs": s.CronJobs,
		"/apis/autoscaling/v2/horizontalpodautoscalers": s.HPAs, "/apis/storage.k8s.io/v1/storageclasses": s.StorageClasses,
		"/apis/scheduling.k8s.io/v1/priorityclasses": s.PriorityClasses, "/apis/networking.k8s.io/v1/ingresses": s.Ingresses,
		"/apis/gateway.networking.k8s.io/v1/gateways": s.Gateways, "/apis/apiextensions.k8s.io/v1/customresourcedefinitions": crdObjects(s.CRDs),
		"/apis/policy/v1/poddisruptionbudgets": s.PDBs,
	}
}

func crdObjects(crds []live.CRD) []map[string]any {
	out := []map[string]any{}
	for _, c := range crds {
		out = append(out, CRDObject(c, ""))
	}
	return out
}

func CRDObject(c live.CRD, version string) map[string]any {
	return map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1", "kind": "CustomResourceDefinition",
		"metadata": map[string]any{"name": c.Name, "uid": c.UID, "resourceVersion": version},
		"spec": map[string]any{"group": c.Group, "scope": c.Scope, "names": map[string]any{"kind": c.Kind, "plural": c.Plural},
			"versions": []map[string]any{{"name": c.Version, "served": true, "storage": true}}},
	}
}

func CustomItem(r live.CustomResource, version string) map[string]any {
	return map[string]any{"apiVersion": "meta.k8s.io/v1", "kind": "PartialObjectMetadata",
		"metadata": map[string]any{"name": r.Name, "namespace": r.Namespace, "uid": r.UID, "resourceVersion": version}}
}

func CustomPath(c live.CRD, namespace string) string {
	root := "/apis/" + c.Group + "/" + c.Version + "/"
	if namespace == "" {
		return root + c.Plural
	}
	return root + "namespaces/" + namespace + "/" + c.Plural
}

func customLists(s live.Snapshot) map[string][]map[string]any {
	out := map[string][]map[string]any{}
	for _, c := range s.CRDs {
		root := "/apis/" + c.Group + "/" + c.Version + "/"
		out[root+c.Plural] = []map[string]any{}
		for _, r := range s.Custom {
			if r.Group != c.Group || r.Kind != c.Kind {
				continue
			}
			item := CustomItem(r, "")
			out[root+c.Plural] = append(out[root+c.Plural], item)
			if r.Namespace != "" {
				key := root + "namespaces/" + r.Namespace + "/" + c.Plural
				out[key] = append(out[key], item)
			}
		}
	}
	return out
}

type Options struct {
	User        string
	Forbid      func(path string) bool
	ForbidWatch func(path string) bool
	Stall       func(path string) bool
	Metrics     bool
	Feed        *Feed
	Deny        func(attrs map[string]any) string
	Answer      func(r *http.Request, body []byte) (int, string, bool)
}

func page(w http.ResponseWriter, r *http.Request, all any) {
	data, _ := json.Marshal(all)
	var items []json.RawMessage
	_ = json.Unmarshal(data, &items)
	from, _ := strconv.Atoi(r.URL.Query().Get("continue"))
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		limit = len(items)
	}
	from = min(max(from, 0), len(items))
	to := min(from+limit, len(items))
	meta := map[string]any{"resourceVersion": ListVersion}
	if to < len(items) {
		meta["continue"] = strconv.Itoa(to)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"metadata": meta, "items": items[from:to]})
}

func inNamespace(lists map[string]any, path string) any {
	parts := strings.Split(path, "/namespaces/")
	if len(parts) != 2 {
		return nil
	}
	ns, resource, ok := strings.Cut(parts[1], "/")
	if !ok || strings.Contains(resource, "/") {
		return nil
	}
	all, ok := lists[parts[0]+"/"+resource]
	if !ok {
		return nil
	}
	data, err := json.Marshal(all)
	if err != nil {
		return nil
	}
	var items []map[string]any
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	out := []map[string]any{}
	for _, it := range items {
		if m, ok := it["metadata"].(map[string]any); ok && m["namespace"] == ns {
			out = append(out, it)
		}
	}
	return out
}

func Serve(t testing.TB, s live.Snapshot) (*httptest.Server, *Record) {
	return ServeAs(t, s, Options{User: "kind-admin@example.com"})
}

func ServeAs(t testing.TB, s live.Snapshot, o Options) (*httptest.Server, *Record) {
	t.Helper()
	lists, custom := Lists(s), customLists(s)
	rec := &Record{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method+" "+r.URL.Path == SelfReview:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"apiVersion": "authentication.k8s.io/v1", "kind": "SelfSubjectReview",
				"status": map[string]any{"userInfo": map[string]any{"username": o.User, "groups": []string{"system:authenticated"}}}})
		case r.Method+" "+r.URL.Path == AccessReview:
			var review struct {
				Spec struct {
					Attrs map[string]any `json:"resourceAttributes"`
				} `json:"spec"`
			}
			_ = json.NewDecoder(r.Body).Decode(&review)
			reason := ""
			if o.Deny != nil {
				reason = o.Deny(review.Spec.Attrs)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"apiVersion": "authorization.k8s.io/v1", "kind": "SelfSubjectAccessReview", "status": map[string]any{"allowed": reason == "", "reason": reason}})
		case r.Method != http.MethodGet && o.Answer != nil:
			body, _ := io.ReadAll(r.Body)
			code, text, ok := o.Answer(r, body)
			if !ok {
				code, text = http.StatusCreated, `{}`
			}
			w.WriteHeader(code)
			fmt.Fprint(w, text)
		case r.Method != http.MethodGet:
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{}`)
		case watching(r) && o.ForbidWatch != nil && o.ForbidWatch(r.URL.Path):
			forbid(w, r, o.User, "watch")
		case o.Feed != nil && watching(r):
			o.Feed.serve(w, r)
		case o.Stall != nil && o.Stall(r.URL.Path):
			<-r.Context().Done()
			return
		case o.Metrics && strings.HasPrefix(r.URL.Path, metricsRoot):
			serveMetrics(w, r, s)
		case o.Forbid != nil && o.Forbid(r.URL.Path):
			forbid(w, r, o.User, "list")
		case o.Feed != nil && o.Feed.listed(r.URL.Path):
			items, _ := o.Feed.list(r.URL.Path)
			page(w, r, items)
		case r.URL.Path == "/version":
			fmt.Fprint(w, `{"gitVersion":"v1.37.0"}`)
		case strings.HasSuffix(r.URL.Path, "/log"):
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "opening the ledger\nthe ledger is missing\n")
		case lists[r.URL.Path] != nil:
			page(w, r, lists[r.URL.Path])
		case custom[r.URL.Path] != nil:
			page(w, r, custom[r.URL.Path])
		case inNamespace(lists, r.URL.Path) != nil:
			page(w, r, inNamespace(lists, r.URL.Path))
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"kind":"Status","status":"Failure","reason":"NotFound","code":404}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func watching(r *http.Request) bool {
	return r.Method == http.MethodGet && (r.URL.Query().Get("watch") == "true" || r.URL.Query().Get("watch") == "1")
}

func forbid(w http.ResponseWriter, r *http.Request, user, verb string) {
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]any{"kind": "Status", "status": "Failure", "reason": "Forbidden", "code": 403,
		"message": fmt.Sprintf("%s is forbidden: User %q cannot %s it", r.URL.Path, user, verb)})
}

const Context = "fake-ctx"

func Kubeconfig(t testing.TB, server string) string {
	return KubeconfigIn(t, server, "")
}

func KubeconfigIn(t testing.TB, server, namespace string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: fake
  cluster:
    server: %s
contexts:
- name: %s
  context:
    cluster: fake
    user: fake-user
    namespace: %q
users:
- name: fake-user
  user:
    token: not-a-real-token
`, server, Context, namespace)
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func Snapshot(t testing.TB, path string) live.Snapshot {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s, err := live.ReadSnapshot(f)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
