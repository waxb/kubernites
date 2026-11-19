package write

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

const maxBody = 16 << 20

type Request struct {
	Method string
	Path   string
	Query  url.Values
	Body   []byte
}

func (r Request) Key() string {
	q := url.Values{}
	for k, v := range r.Query {
		if k != "dryRun" {
			q[k] = v
		}
	}
	sum := sha256.Sum256([]byte(r.Method + "\x00" + r.Path + "\x00" + q.Encode() + "\x00" + string(r.Body)))
	return hex.EncodeToString(sum[:])
}

func (r Request) DryRun() bool {
	if r.Method == http.MethodGet {
		return false
	}
	if r.Method == http.MethodDelete && len(r.Body) > 0 {
		return bodyDryRun(r.Body)
	}
	return allAll(r.Query["dryRun"])
}

func allAll(vals []string) bool {
	if len(vals) == 0 {
		return false
	}
	for _, v := range vals {
		if v != "All" {
			return false
		}
	}
	return true
}

func bodyDryRun(body []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(body))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return false
	}
	found := 0
	var vals []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return false
		}
		key, ok := tok.(string)
		if !ok {
			return false
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return false
		}
		if key != "dryRun" {
			continue
		}
		found++
		if err := json.Unmarshal(raw, &vals); err != nil {
			return false
		}
	}
	if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
		return false
	}
	if _, err := dec.Token(); err != io.EOF {
		return false
	}
	return found == 1 && allAll(vals)
}

type target struct {
	api         bool
	root        string
	group       string
	version     string
	namespace   string
	resource    string
	name        string
	subresource string
	legacyProxy bool
	watch       bool
	trailing    bool
}

var readableRoots = map[string]bool{"version": true, "openapi": true, "healthz": true, "livez": true, "readyz": true}

func parse(rest string) target {
	segs := strings.Split(strings.Trim(rest, "/"), "/")
	t := target{root: segs[0], trailing: len(rest) > 1 && strings.HasSuffix(rest, "/")}
	var res []string
	switch segs[0] {
	case "api":
		t.api = true
		if len(segs) >= 2 {
			t.version = segs[1]
		}
		if len(segs) > 2 {
			res = segs[2:]
		}
	case "apis":
		t.api = true
		if len(segs) >= 2 {
			t.group = segs[1]
		}
		if len(segs) >= 3 {
			t.version = segs[2]
		}
		if len(segs) > 3 {
			res = segs[3:]
		}
	default:
		return t
	}
	if len(res) > 0 && res[0] == "watch" {
		res = res[1:]
		t.watch = true
	}
	if len(res) == 0 {
		return t
	}
	if res[0] == "proxy" {
		t.legacyProxy = true
		return t
	}
	if res[0] == "namespaces" && len(res) >= 3 && res[2] != "status" && res[2] != "finalize" {
		t.namespace = res[1]
		res = res[2:]
	}
	t.resource = res[0]
	if t.resource == "proxy" {
		t.legacyProxy = true
	}
	if len(res) >= 2 {
		t.name = res[1]
	}
	if len(res) >= 3 {
		t.subresource = strings.Join(res[2:], "/")
	}
	return t
}

func (t target) sub() string {
	first, _, _ := strings.Cut(t.subresource, "/")
	return first
}

var channels = map[string]bool{"exec": true, "attach": true, "portforward": true, "proxy": true}

func (t target) reaches(upgrade bool) bool {
	return upgrade || t.legacyProxy || (t.resource != "" && channels[t.sub()])
}

func (t target) connect(method string) (string, bool) {
	if !t.api || t.group != "" || t.version != "v1" || t.watch || t.legacyProxy || t.name == "" {
		return "", false
	}
	switch {
	case t.resource == "pods" && t.namespace != "" && (t.subresource == "exec" || t.subresource == "attach" || t.subresource == "portforward"):
		if t.trailing || (method != http.MethodGet && method != http.MethodPost) {
			return "", false
		}
		return t.subresource, true
	case (t.resource == "pods" || t.resource == "services") && t.namespace != "" && t.sub() == "proxy":
		return "proxy", true
	case t.resource == "nodes" && t.namespace == "" && t.sub() == "proxy":
		return "proxy", true
	}
	return "", false
}

type destroyKey struct {
	group    string
	resource string
}

var destroyKinds = map[destroyKey]bool{
	{"", "namespaces"}:        true,
	{"", "nodes"}:             true,
	{"", "persistentvolumes"}: true,
	{"apiextensions.k8s.io", "customresourcedefinitions"}:                 true,
	{"rbac.authorization.k8s.io", "roles"}:                                true,
	{"rbac.authorization.k8s.io", "rolebindings"}:                         true,
	{"rbac.authorization.k8s.io", "clusterroles"}:                         true,
	{"rbac.authorization.k8s.io", "clusterrolebindings"}:                  true,
	{"admissionregistration.k8s.io", "mutatingwebhookconfigurations"}:     true,
	{"admissionregistration.k8s.io", "validatingwebhookconfigurations"}:   true,
	{"admissionregistration.k8s.io", "validatingadmissionpolicies"}:       true,
	{"admissionregistration.k8s.io", "validatingadmissionpolicybindings"}: true,
	{"admissionregistration.k8s.io", "mutatingadmissionpolicies"}:         true,
	{"admissionregistration.k8s.io", "mutatingadmissionpolicybindings"}:   true,
}

func Upgrading(h http.Header) bool {
	return upgrading(h)
}

func upgrading(h http.Header) bool {
	for k, vals := range h {
		switch strings.ToLower(k) {
		case "upgrade", "sec-websocket-protocol", "sec-websocket-key", "sec-websocket-version", "sec-websocket-extensions":
			return true
		case "connection":
			for _, v := range vals {
				if strings.Contains(strings.ToLower(v), "upgrade") {
					return true
				}
			}
		}
	}
	return false
}

func ClassesOf(r Request) []Class {
	return classesOf(r, parse(r.Path), false)
}

func classesOf(r Request, t target, upgrade bool) []Class {
	var out []Class
	add := func(c Class) {
		if !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	if t.resource == "secrets" {
		add(Secrets)
	}
	if upgrade || t.legacyProxy || (t.resource != "" && channels[t.sub()]) {
		add(Workloads)
	}
	if r.Method != http.MethodGet && t.resource == "pods" && t.sub() == "ephemeralcontainers" {
		add(Workloads)
	}
	if destroys(r, t) {
		add(Destroy)
	}
	slices.Sort(out)
	return out
}

func destroys(r Request, t target) bool {
	switch {
	case r.Method == http.MethodDelete:
		if destroyKinds[destroyKey{t.group, t.resource}] && t.subresource == "" {
			return true
		}
		if t.name == "" || r.Query.Get("labelSelector") != "" || r.Query.Get("fieldSelector") != "" {
			return true
		}
		return forced(r.Query, r.Body, "")
	case r.Method == http.MethodPost && t.resource == "pods" && t.sub() == "eviction":
		return forced(r.Query, r.Body, "deleteOptions")
	case (r.Method == http.MethodPut || r.Method == http.MethodPatch) && t.group == "" && t.resource == "namespaces" && t.sub() == "finalize":
		return true
	}
	return false
}

func forced(q url.Values, body []byte, field string) bool {
	for _, g := range q["gracePeriodSeconds"] {
		if n, err := strconv.ParseInt(strings.TrimSpace(g), 10, 64); err != nil || n == 0 {
			return true
		}
	}
	if q.Get("ignoreStoreReadErrorWithClusterBreakingPotential") != "" && q.Get("ignoreStoreReadErrorWithClusterBreakingPotential") != "false" {
		return true
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return false
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		return true
	}
	if field != "" {
		inner, ok := doc[field]
		if !ok || string(bytes.TrimSpace(inner)) == "null" {
			return false
		}
		doc = nil
		if err := json.Unmarshal(inner, &doc); err != nil {
			return true
		}
	}
	if raw, ok := doc["gracePeriodSeconds"]; ok {
		var n *int64
		if err := json.Unmarshal(raw, &n); err != nil {
			return true
		}
		if n != nil && *n == 0 {
			return true
		}
	}
	if raw, ok := doc["ignoreStoreReadErrorWithClusterBreakingPotential"]; ok {
		var b *bool
		if err := json.Unmarshal(raw, &b); err != nil || (b != nil && *b) {
			return true
		}
	}
	return false
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

var errTooLarge = errors.New("the request body is too large to check")

func readBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	data, err := io.ReadAll(io.LimitReader(req.Body, maxBody+1))
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	if len(data) > maxBody {
		return nil, errTooLarge
	}
	if data == nil {
		data = []byte{}
	}
	return data, nil
}

type Opening struct {
	Channel   string
	Kind      string
	Namespace string
	Name      string
	Container string
	Command   []string
	Path      string
	Stdin     bool
	TTY       bool
}

func Opens(r Request) (Opening, bool) {
	t := parse(r.Path)
	channel, ok := t.connect(r.Method)
	if !ok {
		return Opening{}, false
	}
	o := Opening{Channel: channel, Namespace: t.namespace, Name: t.name, Container: r.Query.Get("container"), Command: slices.Clone(r.Query["command"])}
	switch t.resource {
	case "pods":
		o.Kind = "Pod"
	case "services":
		o.Kind = "Service"
	case "nodes":
		o.Kind = "Node"
	}
	if channel == "proxy" {
		_, rest, _ := strings.Cut(t.subresource, "/")
		o.Path = "/" + rest
		if t.trailing && rest != "" {
			o.Path += "/"
		}
	}
	o.Stdin = truthy(r.Query.Get("stdin"))
	o.TTY = truthy(r.Query.Get("tty"))
	return o, true
}

func Connects(r Request) bool {
	_, ok := parse(r.Path).connect(r.Method)
	return ok
}

func truthy(v string) bool {
	return v == "true" || v == "1"
}
