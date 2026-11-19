package kubectltree_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/waxb/kubernites/pkg/live/livetest"
)

type seen struct {
	mu   sync.Mutex
	reqs []string
}

func (s *seen) add(r string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reqs = append(s.reqs, r)
}

func (s *seen) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.reqs...)
}

func (s *seen) writes() []string {
	var out []string
	for _, r := range s.all() {
		if !strings.HasPrefix(r, "GET ") && !strings.Contains(r, "selfsubject") {
			out = append(out, r)
		}
	}
	return out
}

func mark(r *http.Request) string {
	m := r.Method + " " + r.URL.Path
	if r.URL.Query().Get("dryRun") != "" {
		m += "?dryRun"
	}
	return m
}

const deployment = `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"web","namespace":"shop","uid":"u-web","resourceVersion":"41","generation":3,"creationTimestamp":"2026-11-01T10:00:00Z","labels":{"app":"web"},"managedFields":[{"manager":"kubectl-create","operation":"Update","apiVersion":"apps/v1","fieldsType":"FieldsV1","fieldsV1":{"f:spec":{"f:replicas":{}}}}]},"spec":{"replicas":2,"selector":{"matchLabels":{"app":"web"}},"template":{"metadata":{"labels":{"app":"web"}},"spec":{"containers":[{"name":"web","image":"nginx:1.27"}]}}},"status":{"replicas":2,"readyReplicas":2}}`

const pod = `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"web-1","namespace":"shop","uid":"u-pod","resourceVersion":"40","labels":{"app":"web"},"creationTimestamp":"2026-11-01T10:00:00Z"},"spec":{"nodeName":"keep-1","containers":[{"name":"web","image":"nginx:1.27"}]},"status":{"phase":"Running"}}`

const ttyPod = `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"shell-1","namespace":"shop","uid":"u-shell","resourceVersion":"44","creationTimestamp":"2026-11-01T10:00:00Z"},"spec":{"nodeName":"keep-1","containers":[{"name":"shell","image":"busybox:1.36","stdin":true,"tty":true}]},"status":{"phase":"Running","containerStatuses":[{"name":"shell","ready":true,"state":{"running":{}}}]}}`

const secret = `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"s","namespace":"shop","resourceVersion":"7","annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"data\":{\"password\":\"aHVudGVyMg==\"}}"}},"type":"Opaque","data":{"password":"aHVudGVyMg==","tls.crt":"Y2VydA=="}}`

const conflictBody = `{"kind":"Status","apiVersion":"v1","metadata":{},"status":"Failure","message":"Apply failed with 1 conflict: conflict with \"kubectl-create\" using apps/v1: .spec.replicas","reason":"Conflict","details":{"causes":[{"reason":"FieldManagerConflict","message":"conflict with \"kubectl-create\" using apps/v1","field":".spec.replicas"}]},"code":409}`

type fakeServer struct {
	*httptest.Server
	accepts    []string
	seen       *seen
	conflict   bool
	allow      bool
	aggregated bool
	shell      *livetest.Shell
}

func resources(gv string, list ...[3]string) string {
	var rs []string
	for _, r := range list {
		verbs := `["get","list","watch","create","update","patch","delete"]`
		rs = append(rs, fmt.Sprintf(`{"name":%q,"singularName":"","namespaced":%s,"kind":%q,"verbs":%s,"shortNames":[]}`, r[0], r[2], r[1], verbs))
	}
	return fmt.Sprintf(`{"kind":"APIResourceList","apiVersion":"v1","groupVersion":%q,"resources":[%s]}`, gv, strings.Join(rs, ","))
}

func openAPI(path, group, kind string) string {
	return fmt.Sprintf(`{"openapi":"3.0.0","info":{"title":"Kubernetes","version":"v1.37.0"},"paths":{%q:{"patch":{"parameters":[{"name":"fieldValidation","in":"query","schema":{"type":"string"}}],"x-kubernetes-action":"patch","x-kubernetes-group-version-kind":{"group":%q,"kind":%q,"version":"v1"}}}}}`, path, group, kind)
}

func group(name string) string {
	return fmt.Sprintf(`{"name":%q,"versions":[{"groupVersion":"%s/v1","version":"v1"}],"preferredVersion":{"groupVersion":"%s/v1","version":"v1"}}`, name, name, name)
}

func newFake(t *testing.T) *fakeServer {
	f := &fakeServer{seen: &seen{}}
	f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.Close)
	return f
}

func newFakeHandler(w http.ResponseWriter, r *http.Request) {
	(&fakeServer{seen: &seen{}}).serve(w, r)
}

func (f *fakeServer) serve(w http.ResponseWriter, r *http.Request) {
	{
		body, _ := io.ReadAll(r.Body)
		f.seen.add(mark(r))
		w.Header().Set("Content-Type", "application/json")
		p := r.URL.Path
		switch {
		case f.shell != nil && (strings.HasSuffix(p, "/exec") || strings.HasSuffix(p, "/attach")):
			f.shell.Serve(w, r)
		case f.shell != nil && strings.HasSuffix(p, "/portforward"):
			f.shell.ServeForward(w, r)
		case strings.HasSuffix(p, "/pods/shell-1"):
			fmt.Fprint(w, ttyPod)
		case strings.Contains(p, "/proxy"):
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "proxied to "+p+"\n")
		case p == "/openapi/v3":
			fmt.Fprint(w, `{"paths":{"apis/apps/v1":{"serverRelativeURL":"/openapi/v3/apis/apps/v1?hash=fake"},"api/v1":{"serverRelativeURL":"/openapi/v3/api/v1?hash=fake"}}}`)
		case p == "/openapi/v3/apis/apps/v1":
			fmt.Fprint(w, openAPI("/apis/apps/v1/namespaces/{namespace}/deployments/{name}", "apps", "Deployment"))
		case p == "/openapi/v3/api/v1":
			fmt.Fprint(w, openAPI("/api/v1/namespaces/{namespace}/configmaps/{name}", "", "ConfigMap"))
		case p == "/version":
			fmt.Fprint(w, `{"major":"1","minor":"37","gitVersion":"v1.37.0"}`)
		case p == "/api":
			fmt.Fprint(w, `{"kind":"APIVersions","versions":["v1"],"serverAddressByClientCIDRs":[]}`)
		case f.aggregated && p == "/apis":
			fmt.Fprintf(w, `{"kind":"APIGroupList","apiVersion":"v1","groups":[%s,%s,%s,%s]}`, group("apps"), group("authorization.k8s.io"), group("authentication.k8s.io"), group("metrics.example.io"))
		case f.aggregated && p == "/apis/metrics.example.io/v1":
			fmt.Fprint(w, resources("metrics.example.io/v1", [3]string{"gauges", "Gauge", "true"}))
		case f.aggregated && p == "/apis/apiregistration.k8s.io/v1/apiservices":
			fmt.Fprint(w, `{"kind":"APIServiceList","apiVersion":"apiregistration.k8s.io/v1","items":[{"spec":{"group":"metrics.example.io","version":"v1","service":{"name":"gauges","namespace":"kube-system"}}},{"spec":{"group":"apps","version":"v1"}}]}`)
		case f.aggregated && strings.HasSuffix(p, "/gauges/g"):
			fmt.Fprint(w, `{"apiVersion":"metrics.example.io/v1","kind":"Gauge","metadata":{"name":"g","namespace":"shop","resourceVersion":"3"},"spec":{"a":0}}`)
		case p == "/apis":
			fmt.Fprintf(w, `{"kind":"APIGroupList","apiVersion":"v1","groups":[%s,%s,%s]}`, group("apps"), group("authorization.k8s.io"), group("authentication.k8s.io"))
		case p == "/api/v1":
			fmt.Fprint(w, resources("v1", [3]string{"pods", "Pod", "true"}, [3]string{"namespaces", "Namespace", "false"}, [3]string{"nodes", "Node", "false"}, [3]string{"configmaps", "ConfigMap", "true"}, [3]string{"secrets", "Secret", "true"}, [3]string{"events", "Event", "true"}, [3]string{"services", "Service", "true"}))
		case p == "/apis/apps/v1":
			fmt.Fprint(w, resources("apps/v1", [3]string{"deployments", "Deployment", "true"}, [3]string{"deployments/scale", "Scale", "true"}, [3]string{"replicasets", "ReplicaSet", "true"}))
		case p == "/apis/authorization.k8s.io/v1":
			fmt.Fprint(w, resources("authorization.k8s.io/v1", [3]string{"selfsubjectaccessreviews", "SelfSubjectAccessReview", "false"}, [3]string{"selfsubjectrulesreviews", "SelfSubjectRulesReview", "true"}))
		case p == "/apis/authentication.k8s.io/v1":
			fmt.Fprint(w, resources("authentication.k8s.io/v1", [3]string{"selfsubjectreviews", "SelfSubjectReview", "false"}))
		case r.Method == http.MethodPost && strings.HasSuffix(p, "/selfsubjectaccessreviews"):
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"apiVersion":"authorization.k8s.io/v1","kind":"SelfSubjectAccessReview","spec":{},"status":{"allowed":%v}}`, f.allow)
		case r.Method == http.MethodPatch && f.conflict && (strings.Contains(string(body), `"replicas":5`) || strings.Contains(string(body), "replicas: 5")):
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, conflictBody)
		case strings.HasSuffix(p, "/secrets/s") || strings.HasSuffix(p, "/secrets") && r.URL.Query().Get("watch") == "":
			f.accepts = append(f.accepts, r.Header.Get("Accept"))
			if strings.HasSuffix(p, "/secrets/s") {
				fmt.Fprint(w, secret)
			} else {
				fmt.Fprintf(w, `{"apiVersion":"v1","kind":"SecretList","metadata":{"resourceVersion":"42"},"items":[%s]}`, secret)
			}
		case strings.HasSuffix(p, "/secrets"):
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"type":"ADDED","object":%s}`+"\n", secret)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case strings.HasSuffix(p, "/deployments/web/scale"):
			fmt.Fprint(w, `{"apiVersion":"autoscaling/v1","kind":"Scale","metadata":{"name":"web","namespace":"shop","resourceVersion":"43"},"spec":{"replicas":3},"status":{"replicas":2,"selector":"app=web"}}`)
		case strings.HasSuffix(p, "/deployments/web"):
			fmt.Fprint(w, deployment)
		case strings.HasSuffix(p, "/deployments"):
			fmt.Fprintf(w, `{"apiVersion":"apps/v1","kind":"DeploymentList","metadata":{"resourceVersion":"42"},"items":[%s,%s]}`, deployment, strings.ReplaceAll(deployment, `"web"`, `"api"`))
		case strings.HasSuffix(p, "/pods/web-1/log"):
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "serving on :8080\n")
		case strings.HasSuffix(p, "/pods/web-1"):
			fmt.Fprint(w, pod)
		case strings.HasSuffix(p, "/pods"):
			fmt.Fprintf(w, `{"apiVersion":"v1","kind":"PodList","metadata":{"resourceVersion":"42"},"items":[%s]}`, pod)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404,"message":"not found"}`)
		}
	}
}

func kubeconfig(t *testing.T, server, context, namespace string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := fmt.Sprintf("apiVersion: v1\nkind: Config\ncurrent-context: %s\nclusters:\n- name: fake\n  cluster:\n    server: %s\ncontexts:\n- name: %s\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: %s\nusers:\n- name: fake-user\n  user:\n    token: not-a-real-token\n", context, server, context, namespace)
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func jsonOf(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("not json: %v\n%s", err, s)
	}
	return m
}
