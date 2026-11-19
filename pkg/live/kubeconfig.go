package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		info := ContextInfo{Name: name, Cluster: c.Cluster, User: c.AuthInfo, Namespace: c.Namespace}
		if cl, ok := cfg.Clusters[c.Cluster]; ok {
			info.Server = cl.Server
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
	rc, err := clientcmd.NewNonInteractiveClientConfig(*cfg, name, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("context %s: %w", name, err)
	}
	return dialConfig(rc, ContextInfo{Name: name, Cluster: c.Cluster, User: c.AuthInfo, Namespace: c.Namespace, Server: rc.Host})
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
	rc, err := clientcmd.NewNonInteractiveClientConfig(*cfg, name, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("context %s: %w", name, err)
	}
	info := ContextInfo{Name: name, Cluster: c.Cluster, User: c.AuthInfo, Namespace: c.Namespace, Server: rc.Host}
	want := g.Context()
	if want.Name != info.Name || want.Server != info.Server || want.User != info.User {
		return nil, fmt.Errorf("context %s changed since the warning named it (server %s, user %s): choose it again", name, info.Server, info.User)
	}
	server, _, err := rest.DefaultServerUrlFor(rc)
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
	server, _, err := rest.DefaultServerUrlFor(rc)
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
