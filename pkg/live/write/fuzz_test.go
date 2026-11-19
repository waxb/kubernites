package write

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var fuzzMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions, http.MethodConnect, http.MethodTrace, "get", "LIST", "EVICT"}

var (
	secretPath  = regexp.MustCompile(`^/(api/[^/]+|apis/[^/]+/[^/]+)/(watch/)?(namespaces/[^/]+/)?secrets(/|$)`)
	channelPath = regexp.MustCompile(`^/(api/[^/]+|apis/[^/]+/[^/]+)/((watch/)?(namespaces/[^/]+/)?[^/]+/[^/]+/(exec|attach|portforward|proxy)(/|$)|(watch/)?(namespaces/[^/]+/)?proxy(/|$))`)
	connectPath = regexp.MustCompile(`^/api/v1/(namespaces/[^/]+/pods/[^/]+/(exec|attach|portforward)|namespaces/[^/]+/(pods|services)/[^/]+/proxy(/.*)?|nodes/[^/]+/proxy(/.*)?)$`)
	channelOnly = regexp.MustCompile(`/(exec|attach|portforward)$`)
	nsLevel     = regexp.MustCompile(`^/(api/[^/]+|apis/[^/]+/[^/]+)/(watch/)?namespaces/[^/]+/(exec|attach|portforward)(/|$)`)
	ephemeral   = regexp.MustCompile(`^/(api/[^/]+|apis/[^/]+/[^/]+)/(watch/)?(namespaces/[^/]+/)?pods/[^/]+/ephemeralcontainers(/|$)`)
	reviewPaths = []string{rulesReview, accessReview, selfReview}
)

func fuzzRequest(method, path, query, body string) (*http.Request, bool) {
	req, err := http.NewRequest("GET", "https://127.0.0.1:6443/", nil)
	if err != nil {
		return nil, false
	}
	u, err := url.Parse("https://127.0.0.1:6443" + path)
	if err != nil || u.Host != "127.0.0.1:6443" {
		return nil, false
	}
	u.RawQuery = query
	req.URL, req.Method = u, method
	if body != "" {
		req.Body = http.NoBody
		r, _ := http.NewRequest(http.MethodPost, "https://x/", strings.NewReader(body))
		req.Body, req.ContentLength = r.Body, int64(len(body))
	}
	return req, true
}

func FuzzTheGateNeverPassesWhatItsLevelDoesNotAllow(f *testing.F) {
	seeds := []struct {
		level, classes, method uint8
		path, query, body      string
		ticket, stale, upgrade bool
	}{
		{1, 0, 3, scalePath, "fieldManager=kubernites", `{"spec":{"replicas":3}}`, false, false, false},
		{1, 0, 3, scalePath, "dryRun=All", `{}`, false, false, false},
		{1, 0, 3, scalePath, "dryRun=All&dryRun=None", `{}`, false, false, false},
		{1, 0, 3, scalePath, "dryRun=all", `{}`, false, false, false},
		{1, 0, 3, scalePath, "", `{}`, true, false, false},
		{2, 0, 4, "/api/v1/namespaces/shop/pods/web-1", "gracePeriodSeconds=0", "", false, false, false},
		{2, 0, 4, "/api/v1/namespaces/shop/pods", "", "", false, false, false},
		{2, 0, 4, "/api/v1/namespaces/kube-system", "", "", false, false, false},
		{2, 0, 0, "/api/v1/namespaces/shop/secrets/key", "", "", false, false, false},
		{2, 0, 0, "/api/v1/namespaces/shop/pods/web-1/exec", "command=sh", "", false, false, true},
		{2, 0, 1, "/api/v1/namespaces/shop/pods/web-1/exec", "command=sh", "", false, false, false},
		{2, 7, 1, "/api/v1/namespaces/shop/pods/web-1/eviction", "", `{"deleteOptions":{"gracePeriodSeconds":0}}`, false, true, false},
		{1, 7, 4, "/api/v1/namespaces/shop/pods/web-1", "dryRun=All", `{"preconditions":{"uid":"u"}}`, false, false, false},
		{1, 7, 4, "/api/v1/namespaces/shop/pods/web-1", "", `{"dryRun":["All"]}`, false, false, false},
		{1, 7, 4, "/api/v1/namespaces/shop/pods/web-1", "dryRun=All", `{"DryRun":["All"]}`, false, false, false},
		{2, 0, 1, accessReview, "", `{}`, false, true, false},
		{1, 0, 1, "/apis/authorization.k8s.io/v1/selfsubjectaccessreviews/../../../../api/v1/namespaces/shop/pods", "", `{}`, false, false, false},
		{2, 0, 0, "/metrics", "", "", false, false, false},
		{0, 7, 4, "/api/v1/namespaces/shop/pods/web-1", "", "", true, false, false},
		{2, 0, 0, "/api/v1/namespaces/shop/services/web:80/proxy/", "", "", false, false, false},
		{2, 0, 0, "/api/v1/proxy/nodes/n", "", "", false, false, false},
		{1, 2, 1, "/api/v1/namespaces/shop/pods/web-1/exec", "command=sh&stdin=true&tty=true", "", false, false, true},
		{1, 2, 1, "/api/v1/namespaces/shop/pods/web-1/exec", "command=sh&dryRun=All", "", false, false, true},
		{1, 2, 0, "/api/v1/namespaces/shop/pods/web-1/attach", "container=app&stdin=true", "", true, false, true},
		{2, 2, 1, "/api/v1/namespaces/shop/pods/web-1/portforward", "", "", false, false, true},
		{1, 2, 1, "/api/v1/namespaces/shop/pods/web-1/portforward", "", "", true, true, true},
		{2, 2, 0, "/api/v1/namespaces/shop/services/web:80/proxy/admin", "", "", false, false, false},
		{1, 2, 1, "/api/v1/nodes/kind-worker/proxy/run/shop/web-1/app", "cmd=id&dryRun=All", `{}`, false, false, false},
		{1, 2, 3, "/api/v1/namespaces/shop/pods/web-1/ephemeralcontainers", "dryRun=All", `{}`, false, false, false},
		{2, 2, 0, "/api/v1/pods", "", "", false, false, true},
		{2, 7, 1, "/apis/apps/v1/namespaces/shop/deployments/web/exec", "", "", true, false, true},
		{2, 7, 0, "/api/v1/watch/namespaces/shop/pods/web-1/exec", "", "", true, false, true},
		{2, 7, 0, "/api/v1/namespaces/shop/proxy", "", "", true, false, false},
	}
	for _, s := range seeds {
		f.Add(s.level, s.classes, s.method, s.path, s.query, s.body, s.ticket, s.stale, s.upgrade)
	}
	f.Fuzz(func(t *testing.T, level, classes, method uint8, path, query, body string, ticket, stale, upgrade bool) {
		l := Level(level % 4)
		var passed []Class
		for i, c := range AllClasses {
			if classes&(1<<i) != 0 {
				passed = append(passed, c)
			}
		}
		g, err := New(choice(l, passed...))
		if l != Writes && l != Seamless {
			if err == nil {
				t.Fatalf("a write gate was built at %v", l)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		m := fuzzMethods[int(method)%len(fuzzMethods)]
		req, ok := fuzzRequest(m, path, query, body)
		if !ok {
			return
		}
		if upgrade {
			upgradeHeaders(req, path)
		}
		if stale {
			g.StaleWhen(func() string { return "a watch broke" })
		}
		q, _ := url.ParseQuery(query)
		if ticket {
			g.Issue(Request{Method: m, Path: req.URL.EscapedPath(), Query: q, Body: []byte(body)})
		}
		rec := &recorder{}
		_, err = gated{g: g, next: rec, known: true}.RoundTrip(req)
		if err != nil {
			if len(rec.seen) != 0 {
				t.Fatalf("refused (%v) yet the server saw %v", err, rec.seen)
			}
			if !strings.Contains(err.Error(), "refused by the write gate") {
				t.Fatalf("a refusal that does not say so: %v", err)
			}
			return
		}
		if len(rec.seen) != 1 {
			t.Fatalf("passed, but the server saw %v", rec.seen)
		}
		p := req.URL.EscapedPath()
		if !slices.Contains([]string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}, m) {
			t.Fatalf("%s passed", m)
		}
		if p != req.URL.Path || strings.Contains(p, "//") || strings.Contains(p, "/./") || strings.Contains(p, "/../") || strings.HasSuffix(p, "/..") || strings.HasSuffix(p, "/.") {
			t.Fatalf("an odd path passed: %q (%q)", p, req.URL.Path)
		}
		if !strings.HasPrefix(p, "/api/") && p != "/api" && !strings.HasPrefix(p, "/apis") && m != http.MethodGet {
			t.Fatalf("%s %s outside the API passed", m, p)
		}
		review := m == http.MethodPost && slices.Contains(reviewPaths, p) && !upgrade
		reaches := upgrade || (channelPath.MatchString(p) && !nsLevel.MatchString(p))
		connect := connectPath.MatchString(p)
		if reaches && !connect {
			t.Fatalf("%s %s reached into something that is not a Pod's channel or a proxy", m, p)
		}
		if reaches && channelOnly.MatchString(p) && m != http.MethodGet && m != http.MethodPost {
			t.Fatalf("%s %s opened a channel with a method that opens none", m, p)
		}
		read := m == http.MethodGet && !reaches
		dry := !reaches && m != http.MethodGet && len(q["dryRun"]) > 0 && !slices.ContainsFunc(q["dryRun"], func(v string) bool { return v != "All" })
		if m == http.MethodDelete && body != "" && !reaches {
			dry = serverReadsDryRun(body)
		}
		if secretPath.MatchString(p) && !slices.Contains(passed, Secrets) {
			t.Fatalf("%s %s touched Secrets without their warning", m, p)
		}
		if reaches && !slices.Contains(passed, Workloads) {
			t.Fatalf("%s %s reached into a workload without its warning", m, p)
		}
		if ephemeral.MatchString(p) && m != http.MethodGet && !slices.Contains(passed, Workloads) {
			t.Fatalf("%s %s added an ephemeral container without the workloads warning", m, p)
		}
		if m == http.MethodDelete && slices.Contains(q["gracePeriodSeconds"], "0") && !slices.Contains(passed, Destroy) {
			t.Fatalf("a force delete %s passed without its warning", p)
		}
		if review || read {
			return
		}
		if stale {
			t.Fatalf("%s %s?%s passed while the realm was stale", m, p, query)
		}
		if l == Writes && !dry {
			if !ticket {
				t.Fatalf("at WRITES %s %s?%s passed without a ticket", m, p, query)
			}
			req2, _ := fuzzRequest(m, path, query, body)
			if upgrade {
				upgradeHeaders(req2, path)
			}
			if _, err := (gated{g: g, next: rec, known: true}).RoundTrip(req2); err == nil {
				t.Fatalf("at WRITES a ticket was used twice: %s %s", m, p)
			}
		}
	})
}

func upgradeHeaders(req *http.Request, path string) {
	if len(path)%2 == 0 {
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "SPDY/3.1")
		return
	}
	req.Header.Set("Sec-Websocket-Protocol", "v5.channel.k8s.io")
}

func serverReadsDryRun(body string) bool {
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &top); err != nil {
		return false
	}
	raw, ok := top["dryRun"]
	if !ok || strings.Count(body, "dryRun") > 1 {
		return false
	}
	var vals []string
	if json.Unmarshal(raw, &vals) != nil || len(vals) == 0 {
		return false
	}
	for _, v := range vals {
		if v != "All" {
			return false
		}
	}
	return true
}
