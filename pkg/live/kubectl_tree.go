package live

import (
	"fmt"
	"net/http"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func ReadOnlyConfig(c *rest.Config) *rest.Config {
	server, _, err := write.ServerURL(c)
	if err != nil {
		c.Wrap(func(http.RoundTripper) http.RoundTripper {
			return closedGate{reason: "the cluster's address could not be read: " + err.Error()}
		})
		return c
	}
	base := strings.TrimRight(server.Path, "/")
	c.Wrap(func(rt http.RoundTripper) http.RoundTripper { return OnlyTo(ReadOnly(rt, base), *server) })
	return c
}

func ReadOnlyConfigAt(connected *rest.Config) func(*rest.Config) *rest.Config {
	return func(c *rest.Config) *rest.Config {
		if why := write.Elsewhere(c, connected); why != "" {
			c.Wrap(func(http.RoundTripper) http.RoundTripper { return closedGate{reason: why} })
			return c
		}
		return ReadOnlyConfig(c)
	}
}

type closedGate struct {
	reason string
}

func (g closedGate) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		req.Body.Close()
	}
	return nil, &RefusedError{Method: req.Method, Path: req.URL.EscapedPath(), Reason: g.reason}
}

func openTree(paths []string, name string, connected *rest.Config, gate *write.Gate) *kubectltree.Tree {
	return openTreeIn(paths, name, connected, gate, "")
}

func openTreeIn(paths []string, name string, connected *rest.Config, gate *write.Gate, local string) *kubectltree.Tree {
	if connected == nil {
		return nil
	}
	wrap := ReadOnlyConfigAt(connected)
	if gate != nil {
		wrap = gate.WrapConfig
	}
	t, err := kubectltree.New(kubectltree.Config{Kubeconfig: paths, Context: name, Wrap: PinnedTo(connected, wrap), Local: local, Server: connected.Host})
	if err != nil {
		return nil
	}
	return t
}

func PinnedTo(connected *rest.Config, wrap func(*rest.Config) *rest.Config) func(*rest.Config) *rest.Config {
	pinned := rest.CopyConfig(connected)
	return func(loaded *rest.Config) *rest.Config {
		c := rest.CopyConfig(pinned)
		c.UserAgent, c.Timeout, c.DisableCompression = loaded.UserAgent, loaded.Timeout, loaded.DisableCompression
		c.ContentType, c.AcceptContentTypes = loaded.ContentType, loaded.AcceptContentTypes
		return wrap(c)
	}
}

func OpenRecordedTree(s Snapshot, kubeconfig, context string, g *write.Gate) (*Realm, error) {
	return OpenRecordedTreeIn(s, kubeconfig, context, g, "")
}

func OpenRecordedTreeIn(s Snapshot, kubeconfig, context string, g *write.Gate, local string) (*Realm, error) {
	open := OpenRecorded
	if g != nil {
		open = func(s Snapshot) (*Realm, error) { return OpenRecordedGated(s, g) }
	}
	r, err := open(s)
	if err != nil {
		return nil, err
	}
	connected, err := resolveContext([]string{kubeconfig}, context)
	if err == nil {
		r.tree = openTreeIn([]string{kubeconfig}, context, connected, g, local)
	}
	return r, nil
}

func (r *Realm) Tree() *kubectltree.Tree {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tree
}

func resolveContext(paths []string, name string) (*rest.Config, error) {
	cfg, err := loadKubeconfig(paths)
	if err != nil {
		return nil, err
	}
	if _, ok := cfg.Contexts[name]; !ok || name == "" {
		return nil, fmt.Errorf("the kubeconfig has no context %q", name)
	}
	return clientcmd.NewNonInteractiveClientConfig(*cfg, name, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
}
