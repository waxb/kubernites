package live

import (
	"net/http"
	"testing"
)

func TestAtReadOnlyNothingReachesIntoAWorkload(t *testing.T) {
	ws := http.Header{"Sec-Websocket-Protocol": {"v5.channel.k8s.io"}}
	tunnel := http.Header{"Sec-Websocket-Protocol": {"SPDY/3.1+portforward.k8s.io"}}
	spdy := http.Header{"Connection": {"Upgrade"}, "Upgrade": {"SPDY/3.1"}}
	cases := []struct {
		method, path string
		header       http.Header
	}{
		{http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/exec?command=sh&stdin=true&tty=true", spdy},
		{http.MethodGet, "/api/v1/namespaces/shop/pods/web-1/exec?command=sh&stdin=true&tty=true", ws},
		{http.MethodGet, "/api/v1/namespaces/shop/pods/web-1/exec?command=id&stdout=true", nil},
		{http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/attach?stdin=true&tty=true", spdy},
		{http.MethodGet, "/api/v1/namespaces/shop/pods/web-1/attach?stdin=true&tty=true", ws},
		{http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/portforward", spdy},
		{http.MethodGet, "/api/v1/namespaces/shop/pods/web-1/portforward", tunnel},
		{http.MethodGet, "/api/v1/namespaces/shop/pods/web-1:8080/proxy/healthz", nil},
		{http.MethodGet, "/api/v1/namespaces/shop/services/web:80/proxy/admin", nil},
		{http.MethodGet, "/api/v1/nodes/kind-worker/proxy/configz", nil},
		{http.MethodPost, "/api/v1/nodes/kind-worker/proxy/run/shop/web-1/app?cmd=id", nil},
		{http.MethodPatch, "/api/v1/namespaces/shop/pods/web-1/ephemeralcontainers", nil},
		{http.MethodPatch, "/api/v1/namespaces/shop/pods/web-1/ephemeralcontainers?dryRun=All", nil},
		{http.MethodPost, "/api/v1/namespaces/shop/pods", nil},
		{http.MethodPost, "/api/v1/namespaces/default/pods?dryRun=All", nil},
	}
	for _, base := range []string{"", "/k8s/clusters/c-7"} {
		for _, c := range cases {
			assertRefused(t, base, c.method, c.path, c.header)
		}
	}
}
