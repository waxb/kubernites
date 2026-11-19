package live

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/waxb/kubernites/pkg/live/write"
)

var ErrReadOnly = errors.New("this realm is read-only: you are watching a real cluster")

type RefusedError struct {
	Method string
	Path   string
	Reason string
}

func (e *RefusedError) Error() string {
	return fmt.Sprintf("refused before it left Kubernites: %s %s (%s); %v", e.Method, e.Path, e.Reason, ErrReadOnly)
}

func (e *RefusedError) Unwrap() error {
	return ErrReadOnly
}

const (
	rulesReview  = "/apis/authorization.k8s.io/v1/selfsubjectrulesreviews"
	accessReview = "/apis/authorization.k8s.io/v1/selfsubjectaccessreviews"
	selfReview   = "/apis/authentication.k8s.io/v1/selfsubjectreviews"
)

var allowedPosts = map[string]bool{rulesReview: true, accessReview: true, selfReview: true}

var readableRoots = map[string]bool{"version": true, "openapi": true, "healthz": true, "livez": true, "readyz": true}

var closedSubresources = map[string]bool{"exec": true, "attach": true, "portforward": true, "proxy": true}

type readOnly struct {
	next http.RoundTripper
	base string
}

func ReadOnly(next http.RoundTripper, base string) http.RoundTripper {
	return readOnly{next: next, base: strings.TrimRight(base, "/")}
}

func (t readOnly) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := Check(req, t.base); err != nil {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, err
	}
	return t.next.RoundTrip(req)
}

func Check(req *http.Request, base string) error {
	path := req.URL.EscapedPath()
	refuse := func(reason string) error {
		return &RefusedError{Method: req.Method, Path: path, Reason: reason}
	}
	if req.Header.Get("Upgrade") != "" || strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade") || write.Upgrading(req.Header) {
		return refuse("a connection upgrade opens a channel into the cluster")
	}
	if path != req.URL.Path || unusualPath(path) {
		return refuse("the path is not a plain API path")
	}
	base = strings.TrimRight(base, "/")
	if base != "" && path != base && !strings.HasPrefix(path, base+"/") {
		return refuse("the path is outside the cluster's API")
	}
	rest := strings.TrimPrefix(path, base)
	switch req.Method {
	case http.MethodGet:
		if reason := unreadable(rest); reason != "" {
			return refuse(reason)
		}
		return nil
	case http.MethodPost:
		if allowedPosts[rest] {
			return nil
		}
		return refuse("only an access review or a question of who you are may be posted")
	}
	return refuse("only reads may leave Kubernites")
}

func unusualPath(path string) bool {
	if path == "" || path[0] != '/' || strings.Contains(path, "//") {
		return true
	}
	for _, seg := range strings.Split(path, "/") {
		if seg == "." || seg == ".." {
			return true
		}
	}
	return false
}

func unreadable(rest string) string {
	segs := strings.Split(strings.Trim(rest, "/"), "/")
	var res []string
	switch segs[0] {
	case "api":
		if len(segs) <= 2 {
			return ""
		}
		res = segs[2:]
	case "apis":
		if len(segs) <= 3 {
			return ""
		}
		res = segs[3:]
	default:
		if readableRoots[segs[0]] {
			return ""
		}
		return "the path is not part of the read API"
	}
	if res[0] == "watch" {
		res = res[1:]
	}
	if len(res) == 0 {
		return ""
	}
	if res[0] == "proxy" {
		return "a proxy reaches into the workloads"
	}
	resource, named := res[0], res[1:]
	if res[0] == "namespaces" && len(res) >= 3 {
		resource, named = res[2], res[3:]
	}
	if resource == "secrets" {
		return "Secrets are never read"
	}
	if len(named) >= 2 && closedSubresources[named[1]] {
		return "the " + named[1] + " subresource reaches into the workloads"
	}
	return ""
}
