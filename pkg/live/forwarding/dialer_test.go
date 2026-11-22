package forwarding_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"k8s.io/client-go/rest"

	"github.com/waxb/kubernites/pkg/live/forwarding"
)

var allowedImports = []string{
	"bufio", "context", "encoding/json", "errors", "fmt", "io", "net", "net/http", "net/url", "strings", "sync", "time",
	"k8s.io/apimachinery/pkg/util/portforward", "k8s.io/client-go/rest", "k8s.io/client-go/tools/portforward",
	"k8s.io/client-go/transport/spdy", "k8s.io/client-go/transport/websocket", "k8s.io/streaming/pkg/httpstream", "k8s.io/streaming/pkg/httpstream/spdy",
}

func TestTheForwardDialerIsKubectlsOwnAndBuildsNothingByHandButItsClient(t *testing.T) {
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	clients, requests, wrapped := 0, 0, 0
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			if p := strings.Trim(imp.Path.Value, `"`); !slices.Contains(allowedImports, p) {
				t.Errorf("%s imports %s", name, p)
			}
		}
		owned := map[string]bool{}
		for _, decl := range f.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv != nil && len(fd.Recv.List) == 1 && len(fd.Recv.List[0].Names) == 1 {
				if star, ok := fd.Recv.List[0].Type.(*ast.StarExpr); ok && star.X.(*ast.Ident).Name == "Held" {
					owned[fd.Recv.List[0].Names[0].Name] = true
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CompositeLit:
				sel, _ := x.Type.(*ast.SelectorExpr)
				if sel == nil {
					return true
				}
				typ := sel.X.(*ast.Ident).Name + "." + sel.Sel.Name
				switch typ {
				case "http.Client":
					clients++
					kv, ok := x.Elts[0].(*ast.KeyValueExpr)
					if len(x.Elts) != 1 || !ok || kv.Key.(*ast.Ident).Name != "Transport" || kv.Value.(*ast.Ident).Name != "transport" {
						t.Errorf("%s builds an http.Client from something other than spdy.RoundTripperFor's transport", name)
					}
				case "http.Transport", "http.Request", "rest.Config":
					t.Errorf("%s builds a %s by hand", name, typ)
				}
			case *ast.CallExpr:
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
					switch sel.Sel.Name {
					case "NewRequestWithContext":
						requests++
						if m, ok := x.Args[1].(*ast.SelectorExpr); !ok || m.Sel.Name != "MethodGet" {
							t.Errorf("%s builds a request other than the websocket tunnel's GET", name)
						}
					case "HTTPWrappersForConfig":
						wrapped++
						if id, ok := x.Args[0].(*ast.Ident); !ok || id.Name != "cfg" {
							t.Errorf("%s wraps a transport with something other than the forward's config", name)
						}
					case "NewRequest", "Get", "Post", "Do", "HTTPClientFor", "RESTClientFor", "CopyConfig", "NewTransport", "RoundTripperFor":
						header := false
						if h, ok := sel.X.(*ast.SelectorExpr); ok && h.Sel.Name == "Header" {
							header = true
						}
						if !(sel.Sel.Name == "RoundTripperFor" && selectorPackage(sel) == "websocket") && !(sel.Sel.Name == "Get" && header) {
							t.Errorf("%s calls %s", name, sel.Sel.Name)
						}
					}
				}
			case *ast.AssignStmt:
				for _, lhs := range x.Lhs {
					if sel, ok := lhs.(*ast.SelectorExpr); ok && !owned[selectorPackage(sel)] {
						t.Errorf("%s sets %s", name, sel.Sel.Name)
					}
				}
			}
			return true
		})
	}
	if clients != 1 {
		t.Errorf("the package builds %d http.Clients, want the one kubectl's dialer builds", clients)
	}
	if requests != 1 || wrapped != 1 {
		t.Errorf("the package builds %d requests and wraps %d transports by hand, want the websocket tunnel's GET and the SPDY upgrade wrapped in the config's own wrappers", requests, wrapped)
	}
}

func selectorPackage(sel *ast.SelectorExpr) string {
	if id, ok := sel.X.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func TestEveryDialOfAForwardPassesTheConfigsWrappedTransport(t *testing.T) {
	var mu sync.Mutex
	var server, wrapped []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		server = append(server, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)
	cfg := &rest.Config{Host: srv.URL}
	cfg.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return roundTrip(func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			wrapped = append(wrapped, r.Method+" "+r.URL.Path)
			mu.Unlock()
			return rt.RoundTrip(r)
		})
	})
	u, _ := url.Parse(srv.URL + "/api/v1/namespaces/shop/pods/web-1/portforward")
	for _, websockets := range []bool{true, false} {
		mu.Lock()
		server, wrapped = nil, nil
		mu.Unlock()
		d, _, err := forwarding.Dialer("POST", u, cfg, websockets)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := d.Dial("portforward.k8s.io"); err == nil {
			t.Fatal("a server that refuses every upgrade was dialled")
		}
		mu.Lock()
		if len(server) == 0 || !slices.Equal(server, wrapped) {
			t.Errorf("websockets %v: the server saw %v and the wrapped transport %v", websockets, server, wrapped)
		}
		mu.Unlock()
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
