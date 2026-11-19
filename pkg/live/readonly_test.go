package live

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type recorder struct {
	seen []string
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.seen = append(r.seen, req.Method+" "+req.URL.EscapedPath())
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}")), Request: req}, nil
}

type closeSpy struct {
	io.Reader
	closed bool
}

func (c *closeSpy) Close() error {
	c.closed = true
	return nil
}

var everyMethod = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
	http.MethodOptions, http.MethodConnect, http.MethodTrace, "LIST", "WATCH", "EVICT", "get", "post",
}

var readablePaths = []string{
	"/api", "/api/v1", "/apis", "/apis/apps", "/apis/apps/v1", "/version", "/openapi/v2", "/openapi/v3", "/healthz", "/livez", "/readyz",
	"/api/v1/nodes", "/api/v1/nodes/kind-worker", "/api/v1/nodes/kind-worker/status",
	"/api/v1/namespaces", "/api/v1/namespaces/shop", "/api/v1/namespaces/shop/status",
	"/api/v1/pods", "/api/v1/namespaces/shop/pods", "/api/v1/namespaces/shop/pods/web-1", "/api/v1/namespaces/shop/pods/web-1/log",
	"/api/v1/namespaces/shop/pods/proxy", "/api/v1/namespaces/proxy/pods/exec", "/api/v1/namespaces/secrets/pods",
	"/api/v1/namespaces/shop/pods/web-1/status", "/api/v1/namespaces/shop/pods/web-1/ephemeralcontainers",
	"/api/v1/services", "/api/v1/namespaces/shop/services/web", "/api/v1/namespaces/shop/events", "/api/v1/events",
	"/api/v1/configmaps", "/api/v1/persistentvolumes", "/api/v1/persistentvolumeclaims", "/api/v1/namespaces/shop/serviceaccounts/default",
	"/api/v1/watch/namespaces/shop/pods", "/api/v1/watch/pods",
	"/apis/apps/v1/deployments", "/apis/apps/v1/namespaces/shop/deployments/web", "/apis/apps/v1/namespaces/shop/deployments/web/scale",
	"/apis/apps/v1/replicasets", "/apis/apps/v1/daemonsets", "/apis/apps/v1/statefulsets", "/apis/batch/v1/jobs", "/apis/batch/v1/cronjobs",
	"/apis/discovery.k8s.io/v1/endpointslices", "/apis/autoscaling/v2/horizontalpodautoscalers", "/apis/storage.k8s.io/v1/storageclasses",
	"/apis/networking.k8s.io/v1/ingresses", "/apis/gateway.networking.k8s.io/v1/gateways", "/apis/apiextensions.k8s.io/v1/customresourcedefinitions",
	"/apis/metrics.k8s.io/v1beta1/nodes", "/apis/metrics.k8s.io/v1/namespaces/shop/pods", "/apis/events.k8s.io/v1/events",
}

var closedPaths = []string{
	"/api/v1/secrets", "/api/v1/namespaces/shop/secrets", "/api/v1/namespaces/shop/secrets/vault-key", "/api/v1/watch/secrets",
	"/api/v1/watch/namespaces/shop/secrets", "/apis/example.com/v1/secrets",
	"/api/v1/namespaces/shop/pods/web-1/exec", "/api/v1/namespaces/shop/pods/web-1/attach", "/api/v1/namespaces/shop/pods/web-1/portforward",
	"/api/v1/namespaces/shop/pods/web-1/proxy", "/api/v1/namespaces/shop/pods/web-1/proxy/healthz", "/api/v1/namespaces/shop/services/web:80/proxy/",
	"/api/v1/nodes/kind-worker/proxy", "/api/v1/nodes/kind-worker/proxy/configz", "/api/v1/proxy/namespaces/shop/pods/web-1",
	"/", "/metrics", "/logs", "/logs/kube-apiserver.log", "/debug/pprof/", "/apis/../api/v1/secrets", "/api/v1/namespaces/shop/./secrets",
	"/api/v1//namespaces/shop/pods/web-1/exec", "/api/v1/namespaces/shop/pods/web-1/../web-1/exec",
}

var writePaths = []string{
	"/api/v1/namespaces", "/api/v1/namespaces/shop", "/api/v1/namespaces/shop/pods", "/api/v1/namespaces/shop/pods/web-1",
	"/api/v1/namespaces/shop/pods/web-1/eviction", "/api/v1/namespaces/shop/pods/web-1/binding", "/api/v1/namespaces/shop/bindings",
	"/api/v1/namespaces/shop/pods/web-1/exec", "/api/v1/namespaces/shop/pods/web-1/ephemeralcontainers", "/api/v1/namespaces/shop/pods/web-1/status",
	"/api/v1/nodes/kind-worker", "/api/v1/nodes/kind-worker/status", "/api/v1/namespaces/shop/configmaps", "/api/v1/namespaces/shop/secrets",
	"/api/v1/namespaces/shop/serviceaccounts/default/token", "/api/v1/namespaces/shop/services/web",
	"/apis/apps/v1/namespaces/shop/deployments", "/apis/apps/v1/namespaces/shop/deployments/web", "/apis/apps/v1/namespaces/shop/deployments/web/scale",
	"/apis/apps/v1/namespaces/shop/deployments/web/status", "/apis/apps/v1/namespaces/shop/statefulsets/ledger/scale",
	"/apis/batch/v1/namespaces/batch/jobs", "/apis/autoscaling/v2/namespaces/shop/horizontalpodautoscalers/web",
	"/apis/authentication.k8s.io/v1/tokenreviews", "/apis/authentication.k8s.io/v1beta1/selfsubjectreviews",
	"/apis/authentication.k8s.io/v1alpha1/selfsubjectreviews", "/apis/authentication.k8s.io/v1/selfsubjectreviews/x",
	"/apis/authentication.k8s.io/v1/selfsubjectreviews/", "/apis/authentication.k8s.io/v1/namespaces/shop/selfsubjectreviews",
	"/apis/authentication.k8s.io/v1/selfsubjectreviews/../tokenreviews",
	"/apis/authorization.k8s.io/v1/subjectaccessreviews", "/apis/authorization.k8s.io/v1/namespaces/shop/localsubjectaccessreviews",
	"/apis/authorization.k8s.io/v1beta1/selfsubjectrulesreviews", "/apis/authorization.k8s.io/v1/selfsubjectrulesreviews/x",
	"/apis/authorization.k8s.io/v1/selfsubjectrulesreviews/", "/apis/authorization.k8s.io/v1/namespaces/shop/selfsubjectrulesreviews",
	"/apis/authorization.k8s.io/v1/selfsubjectrulesreviews/../../../../api/v1/namespaces/shop/pods",
	"/apis/certificates.k8s.io/v1/certificatesigningrequests/csr-1/approval", "/apis/rbac.authorization.k8s.io/v1/clusterrolebindings",
	"/apis/apiextensions.k8s.io/v1/customresourcedefinitions", "/apis/coordination.k8s.io/v1/namespaces/kube-node-lease/leases/kind-worker",
	"/apis/policy/v1/namespaces/shop/poddisruptionbudgets",
}

func send(t *testing.T, base, method, path string, header http.Header) (error, *recorder, *closeSpy) {
	t.Helper()
	return sendRaw(t, base, method, base+path, header)
}

func sendRaw(t *testing.T, base, method, full string, header http.Header) (error, *recorder, *closeSpy) {
	t.Helper()
	rec := &recorder{}
	body := &closeSpy{Reader: strings.NewReader(`{"kind":"Pod"}`)}
	u, err := url.Parse("https://127.0.0.1:6443" + full)
	if err != nil {
		t.Fatal(err)
	}
	if header == nil {
		header = http.Header{}
	}
	req := &http.Request{Method: method, URL: u, Header: header, Body: body}
	_, err = ReadOnly(rec, base).RoundTrip(req)
	return err, rec, body
}

func assertRefused(t *testing.T, base, method, path string, header http.Header) {
	t.Helper()
	assertRefusedRaw(t, base, method, base+path, header)
}

func assertRefusedRaw(t *testing.T, base, method, full string, header http.Header) {
	t.Helper()
	err, rec, body := sendRaw(t, base, method, full, header)
	var refused *RefusedError
	if !errors.As(err, &refused) || !errors.Is(err, ErrReadOnly) {
		t.Errorf("%s %s went through (err %v)", method, full, err)
	}
	if len(rec.seen) != 0 {
		t.Errorf("%s %s left the process: %v", method, full, rec.seen)
	}
	if !body.closed {
		t.Errorf("%s %s: the refused request's body was not closed", method, full)
	}
}

func assertPassed(t *testing.T, base, method, path string) {
	t.Helper()
	err, rec, _ := send(t, base, method, path, nil)
	if err != nil || len(rec.seen) != 1 {
		t.Errorf("%s %s%s was refused: %v", method, base, path, err)
	}
}

func TestOnlyReadsAndAccessReviewsLeaveTheProcess(t *testing.T) {
	for _, base := range []string{"", "/k8s/clusters/c-7"} {
		for _, path := range readablePaths {
			assertPassed(t, base, http.MethodGet, path)
			for _, m := range everyMethod {
				if m != http.MethodGet {
					assertRefused(t, base, m, path, nil)
				}
			}
		}
		for _, path := range closedPaths {
			for _, m := range everyMethod {
				assertRefused(t, base, m, path, nil)
			}
		}
		for _, path := range writePaths {
			for _, m := range everyMethod {
				if m != http.MethodGet {
					assertRefused(t, base, m, path, nil)
				}
			}
		}
		for _, path := range []string{rulesReview, accessReview, selfReview} {
			assertPassed(t, base, http.MethodPost, path)
			for _, m := range everyMethod {
				if m != http.MethodPost && m != http.MethodGet {
					assertRefused(t, base, m, path, nil)
				}
			}
		}
	}
}

func TestARequestOutsideTheBasePathIsRefused(t *testing.T) {
	assertRefusedRaw(t, "/k8s/clusters/c-7", http.MethodGet, "/api/v1/pods", nil)
	assertRefusedRaw(t, "/k8s/clusters/c-7", http.MethodPost, rulesReview, nil)
	assertRefusedRaw(t, "/k8s/clusters/c-7", http.MethodGet, "/k8s/clusters/c-8/api/v1/pods", nil)
	assertRefusedRaw(t, "/k8s/clusters/c-7", http.MethodGet, "/k8s/clusters/c-70/api/v1/pods", nil)
}

func TestAConnectionUpgradeIsRefusedEvenOnAGet(t *testing.T) {
	for _, h := range []http.Header{
		{"Upgrade": {"websocket"}, "Connection": {"Upgrade"}},
		{"Upgrade": {"SPDY/3.1"}},
		{"Connection": {"keep-alive, Upgrade"}},
	} {
		assertRefused(t, "", http.MethodGet, "/api/v1/namespaces/shop/pods/web-1/log", h)
	}
}

func TestAnEncodedPathIsRefused(t *testing.T) {
	for _, p := range []string{"/api/v1/namespaces/shop/pods/web-1%2Fexec", "/api/v1/namespaces/shop/%73ecrets", "/api/v1/namespaces/shop/pods/web-1/%2e%2e/x/exec"} {
		assertRefused(t, "", http.MethodGet, p, nil)
	}
}
