package live

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/waxb/kubernites/pkg/live/write"
)

const (
	userAgent    = "kubernites-engineer-mode (read-only)"
	maxBodyBytes = 256 << 20
	dialTimeout  = 30 * time.Second
)

type ContextInfo struct {
	Name      string
	Cluster   string
	Server    string
	User      string
	Namespace string
	Identity  string `json:",omitempty"`
	Facets    string `json:"-"`
	ClusterID string `json:"-"`
}

func (c ContextInfo) Remembered() write.Context {
	return write.Context{Name: c.Name, Server: c.Server, User: c.User, Identity: c.Identity, ClusterID: c.ClusterID}
}

const maxIdentityFile = 1 << 20

const execExtension = "client.authentication.k8s.io/exec"

var IdentityFacets = []string{"certificate authority", "server certificate name", "proxy", "credentials", "impersonation"}

const ServerVerification = "server verification"

var namedFacets = []string{"certificate authority", ServerVerification, "server certificate name", "proxy", "credentials", "impersonation"}

type resolved struct {
	cluster   clientcmdapi.Cluster
	user      clientcmdapi.AuthInfo
	tokenFile []byte
	unread    []string
}

func readOnce(path string) ([]byte, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if st.Size() > maxIdentityFile {
		return nil, fmt.Errorf("%s is larger than %d bytes", path, maxIdentityFile)
	}
	return os.ReadFile(path)
}

func resolveEntry(cfg *clientcmdapi.Config, c *clientcmdapi.Context) resolved {
	var r resolved
	if cl, ok := cfg.Clusters[c.Cluster]; ok && cl != nil {
		r.cluster = *cl.DeepCopy()
	}
	if u, ok := cfg.AuthInfos[c.AuthInfo]; ok && u != nil {
		r.user = *u.DeepCopy()
	}
	inline := func(path *string, data *[]byte) {
		if *path == "" {
			return
		}
		if len(*data) == 0 {
			b, err := readOnce(*path)
			if err != nil {
				r.unread = append(r.unread, *path)
				return
			}
			*data = b
		}
		*path = ""
	}
	inline(&r.cluster.CertificateAuthority, &r.cluster.CertificateAuthorityData)
	inline(&r.user.ClientCertificate, &r.user.ClientCertificateData)
	inline(&r.user.ClientKey, &r.user.ClientKeyData)
	if path := r.user.TokenFile; path != "" {
		b, err := readOnce(path)
		if err != nil {
			r.unread = append(r.unread, path)
		} else {
			r.tokenFile = b
			if r.user.Token == "" {
				r.user.Token = string(b)
			}
			r.user.TokenFile = ""
		}
	}
	return r
}

func extensionBytes(obj runtime.Object) []byte {
	if obj == nil {
		return nil
	}
	if u, ok := obj.(*runtime.Unknown); ok {
		return u.Raw
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return []byte(err.Error())
	}
	return b
}

func (r resolved) facets() map[string][]byte {
	out := map[string][]byte{}
	for _, f := range namedFacets {
		out[f] = nil
	}
	add := func(facet, label string, b []byte) {
		out[facet] = fmt.Appendf(out[facet], "%s %d\n", label, len(b))
		out[facet] = append(out[facet], b...)
	}
	cl, u := r.cluster, r.user
	add("certificate authority", "certificate-authority-data", cl.CertificateAuthorityData)
	add(ServerVerification, "insecure-skip-tls-verify", []byte(strconv.FormatBool(cl.InsecureSkipTLSVerify)))
	add("server certificate name", "tls-server-name", []byte(cl.TLSServerName))
	add("proxy", "proxy-url", []byte(cl.ProxyURL))
	if su, err := url.Parse(cl.Server); err == nil && su.User != nil {
		add("credentials", "server-userinfo", []byte(su.User.String()))
	} else if info := write.ServerUserinfo(cl.Server); info != "" {
		add("credentials", "server-userinfo", []byte(strings.TrimSuffix(info, "@")))
	}
	add("credentials", "client-certificate-data", u.ClientCertificateData)
	add("credentials", "client-key-data", u.ClientKeyData)
	add("credentials", "token", []byte(u.Token))
	add("credentials", "token-file", r.tokenFile)
	add("credentials", "username", []byte(u.Username))
	add("credentials", "password", []byte(u.Password))
	if p := u.AuthProvider; p != nil {
		add("credentials", "auth-provider", []byte(p.Name))
		keys := make([]string, 0, len(p.Config))
		for k := range p.Config {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			add("credentials", "auth-provider-config", []byte(k+"="+p.Config[k]))
		}
	}
	if e := u.Exec; e != nil {
		add("credentials", "exec", []byte(e.Command))
		add("credentials", "exec-api-version", []byte(e.APIVersion))
		for _, a := range e.Args {
			add("credentials", "arg", []byte(a))
		}
		for _, v := range e.Env {
			add("credentials", "env", []byte(v.Name+"="+v.Value))
		}
		add("credentials", "provide-cluster-info", []byte(strconv.FormatBool(e.ProvideClusterInfo)))
		add("credentials", "exec-extension", extensionBytes(cl.Extensions[execExtension]))
	}
	for _, path := range r.unread {
		add("credentials", "unread", []byte(path))
	}
	add("impersonation", "act-as", []byte(u.Impersonate))
	add("impersonation", "act-as-uid", []byte(u.ImpersonateUID))
	for _, g := range u.ImpersonateGroups {
		add("impersonation", "act-as-group", []byte(g))
	}
	keys := make([]string, 0, len(u.ImpersonateUserExtra))
	for k := range u.ImpersonateUserExtra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range u.ImpersonateUserExtra[k] {
			add("impersonation", "act-as-user-extra", []byte(k+"="+v))
		}
	}
	return out
}

func (r resolved) clusterID() string {
	parts := r.hashedFacets()
	h := sha256.New()
	server := write.RedactServer(r.cluster.Server)
	fmt.Fprintf(h, "server %d\n%s", len(server), server)
	for _, f := range []string{"certificate authority", "server certificate name"} {
		fmt.Fprintf(h, "%s %d\n", f, len(parts[f]))
		h.Write(parts[f])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (r resolved) hashedFacets() map[string][]byte {
	parts := r.facets()
	parts["certificate authority"] = append(parts["certificate authority"], parts[ServerVerification]...)
	delete(parts, ServerVerification)
	return parts
}

func (r resolved) identity() (string, string) {
	h := sha256.New()
	hashed := r.hashedFacets()
	for _, f := range IdentityFacets {
		sum := sha256.Sum256(hashed[f])
		fmt.Fprintf(h, "%s %x\n", f, sum)
	}
	parts := r.facets()
	var marks []string
	for _, f := range namedFacets {
		if f == ServerVerification {
			mark := "on"
			if r.cluster.InsecureSkipTLSVerify {
				mark = "off"
			}
			marks = append(marks, f+"="+mark)
			continue
		}
		sum := sha256.Sum256(parts[f])
		marks = append(marks, f+"="+hex.EncodeToString(sum[:8]))
	}
	return hex.EncodeToString(h.Sum(nil)), strings.Join(marks, "\n")
}

func ChangedFacets(was, now string) []string {
	if was == "" || now == "" {
		return nil
	}
	old := map[string]string{}
	for _, line := range strings.Split(was, "\n") {
		k, v, _ := strings.Cut(line, "=")
		old[k] = v
	}
	var out []string
	for _, line := range strings.Split(now, "\n") {
		k, v, _ := strings.Cut(line, "=")
		if old[k] == v {
			continue
		}
		if k == ServerVerification {
			k += " " + v
		}
		out = append(out, k)
	}
	return out
}

func (r resolved) restConfig(name string, c *clientcmdapi.Context) (*rest.Config, error) {
	if len(r.unread) > 0 {
		return nil, fmt.Errorf("context %s: cannot read %s", name, strings.Join(r.unread, ", "))
	}
	one := clientcmdapi.NewConfig()
	one.Clusters[c.Cluster] = r.cluster.DeepCopy()
	one.AuthInfos[c.AuthInfo] = r.user.DeepCopy()
	one.Contexts[name] = c.DeepCopy()
	one.CurrentContext = name
	rc, err := clientcmd.NewNonInteractiveClientConfig(*one, name, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("context %s: %w", name, err)
	}
	return rc, nil
}

type IdentityChanged struct {
	Context string
	Facets  string
}

func (e *IdentityChanged) Error() string {
	return "context " + e.Context + " changed since its level was chosen: its kubeconfig entries now name other credentials, another certificate authority, server certificate name, proxy or impersonation. Nothing was sent; choose it again"
}

func KubeconfigPaths(explicit string, env func(string) string, home string) []string {
	if explicit != "" {
		return []string{explicit}
	}
	if list := env("KUBECONFIG"); list != "" {
		out := []string{}
		for _, p := range filepath.SplitList(list) {
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	if home == "" {
		return nil
	}
	return []string{filepath.Join(home, ".kube", "config")}
}

func loadKubeconfig(paths []string) (*clientcmdapi.Config, error) {
	if len(paths) == 0 {
		return nil, errors.New("no kubeconfig file was named")
	}
	rules := &clientcmd.ClientConfigLoadingRules{Precedence: paths}
	cfg, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("read the kubeconfig: %w", err)
	}
	return cfg, nil
}

func ListContexts(paths []string) ([]ContextInfo, error) {
	cfg, err := loadKubeconfig(paths)
	if err != nil {
		return nil, err
	}
	out := make([]ContextInfo, 0, len(cfg.Contexts))
	for name, c := range cfg.Contexts {
		r := resolveEntry(cfg, c)
		identity, facets := r.identity()
		info := ContextInfo{Name: name, Cluster: c.Cluster, User: c.AuthInfo, Namespace: c.Namespace, Identity: identity, Facets: facets, ClusterID: r.clusterID()}
		if cl, ok := cfg.Clusters[c.Cluster]; ok {
			info.Server = write.RedactServer(cl.Server)
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

type Conn struct {
	Context ContextInfo
	client  *http.Client
	server  url.URL
	rest    *rest.Config
}

func Dial(paths []string, name string) (*Conn, error) {
	cfg, err := loadKubeconfig(paths)
	if err != nil {
		return nil, err
	}
	c, ok := cfg.Contexts[name]
	if !ok || name == "" {
		return nil, fmt.Errorf("the kubeconfig has no context %q", name)
	}
	r := resolveEntry(cfg, c)
	rc, err := r.restConfig(name, c)
	if err != nil {
		return nil, err
	}
	identity, facets := r.identity()
	return dialConfig(rc, ContextInfo{Name: name, Cluster: c.Cluster, User: c.AuthInfo, Namespace: c.Namespace, Server: write.RedactServer(rc.Host), Identity: identity, Facets: facets, ClusterID: r.clusterID()})
}

func DialGated(paths []string, name string, g *write.Gate) (*Conn, error) {
	if g == nil {
		return nil, errors.New("a writable connection needs the write gate")
	}
	cfg, err := loadKubeconfig(paths)
	if err != nil {
		return nil, err
	}
	c, ok := cfg.Contexts[name]
	if !ok || name == "" {
		return nil, fmt.Errorf("the kubeconfig has no context %q", name)
	}
	r := resolveEntry(cfg, c)
	rc, err := r.restConfig(name, c)
	if err != nil {
		return nil, err
	}
	identity, facets := r.identity()
	info := ContextInfo{Name: name, Cluster: c.Cluster, User: c.AuthInfo, Namespace: c.Namespace, Server: write.RedactServer(rc.Host), Identity: identity, Facets: facets, ClusterID: r.clusterID()}
	want := g.Context()
	if want.Name != info.Name || want.Server != info.Server || want.User != info.User {
		return nil, fmt.Errorf("context %s changed since the warning named it (server %s, user %s): choose it again", name, info.Server, info.User)
	}
	if want.Identity != info.Identity {
		return nil, &IdentityChanged{Context: name, Facets: facets}
	}
	server, _, err := write.ServerURL(rc)
	if err != nil {
		return nil, err
	}
	client, err := g.Client(rc)
	if err != nil {
		return nil, err
	}
	return &Conn{Context: info, client: client, server: *server, rest: rest.CopyConfig(rc)}, nil
}

func dialConfig(rc *rest.Config, info ContextInfo) (*Conn, error) {
	server, _, err := write.ServerURL(rc)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(server.Path, "/")
	connected := rest.CopyConfig(rc)
	rc = rest.CopyConfig(rc)
	rc.UserAgent = userAgent
	rc.Wrap(func(rt http.RoundTripper) http.RoundTripper { return OnlyTo(ReadOnly(rt, base), *server) })
	client, err := rest.HTTPClientFor(rc)
	if err != nil {
		return nil, err
	}
	client.Transport = OnlyTo(ReadOnly(client.Transport, base), *server)
	return &Conn{Context: info, client: client, server: *server, rest: connected}, nil
}

type StatusError struct {
	Code    int
	Reason  string
	Message string
}

func (e *StatusError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("the API server answered %d %s", e.Code, e.Reason)
}

func IsForbidden(err error) bool {
	var se *StatusError
	return errors.As(err, &se) && se.Code == http.StatusForbidden
}

func IsNotFound(err error) bool {
	var se *StatusError
	return errors.As(err, &se) && se.Code == http.StatusNotFound
}

func (c *Conn) read(ctx context.Context, path string, query url.Values, accept string) ([]byte, error) {
	u := c.server
	u.Path = strings.TrimRight(c.server.Path, "/") + path
	u.RawPath = ""
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	return answer(c.client.Do(req))
}

func answer(resp *http.Response, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var st metav1.Status
		_ = json.Unmarshal(body, &st)
		return nil, &StatusError{Code: resp.StatusCode, Reason: string(st.Reason), Message: st.Message}
	}
	return body, nil
}

func (c *Conn) getJSON(ctx context.Context, path string, query url.Values, into any) error {
	body, err := c.read(ctx, path, query, "application/json")
	if err != nil {
		return err
	}
	return json.Unmarshal(body, into)
}

const metadataOnly = "application/json;as=PartialObjectMetadataList;g=meta.k8s.io;v=v1"

func (c *Conn) getMeta(ctx context.Context, path string, query url.Values, into any) error {
	body, err := c.read(ctx, path, query, metadataOnly)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, into)
}

func (c *Conn) getText(ctx context.Context, path string, query url.Values) (string, error) {
	body, err := c.read(ctx, path, query, "text/plain, */*")
	return string(body), err
}
