package live

import (
	"net/http"
	"net/url"

	"github.com/waxb/kubernites/pkg/live/write"
)

type onlyTo struct {
	next   http.RoundTripper
	server url.URL
}

func OnlyTo(next http.RoundTripper, server url.URL) http.RoundTripper {
	return onlyTo{next: next, server: server}
}

func (t onlyTo) RoundTrip(req *http.Request) (*http.Response, error) {
	if !write.SameOrigin(req.URL, &t.server) {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, &RefusedError{Method: req.Method, Path: req.URL.EscapedPath(), Reason: "the request is for " + req.URL.Scheme + "://" + req.URL.Host + ", not the connected API server"}
	}
	return t.next.RoundTrip(req)
}
