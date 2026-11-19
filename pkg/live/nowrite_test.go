package live

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestAWriteForcedThroughTheConnectionIsRefusedBeforeTheServer(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = append(seen, r.Method+" "+r.URL.Path) }))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\nclusters:\n- name: c\n  cluster:\n    server: " + srv.URL + "\ncontexts:\n- name: x\n  context:\n    cluster: c\n    user: u\nusers:\n- name: u\n  user:\n    token: t\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	conn, err := Dial([]string{path}, "x")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req, _ := http.NewRequest(m, srv.URL+"/api/v1/namespaces/shop/pods/web-1", strings.NewReader(`{}`))
		if _, err := conn.client.Do(req); !errors.Is(err, ErrReadOnly) {
			t.Errorf("%s through the live connection: %v", m, err)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/namespaces/shop/secrets", nil)
	if _, err := conn.client.Do(req); !errors.Is(err, ErrReadOnly) {
		t.Errorf("reading Secrets through the live connection: %v", err)
	}
	if len(seen) != 0 {
		t.Fatalf("the server saw %v", seen)
	}
}

var connReads = map[string]bool{"read": true, "getJSON": true, "getMeta": true, "getText": true, "podLogs": true, "Usage": true, "WhoAmI": true}

const (
	askFile = "whoami.go"
	askFunc = "WhoAmI"
)

func TestTheLivePackageCanOnlyBuildReads(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	methods := []string{}
	posts, asks := 0, map[string]bool{}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if writingClient(p) || p != kubectlTree && reachesWritingClient(t, p, map[string]bool{}) {
				t.Errorf("%s imports %s, a client that can write", name, p)
			}
		}
		for _, decl := range f.Decls {
			fn := ""
			if d, ok := decl.(*ast.FuncDecl); ok {
				fn = d.Name.Name
				if d.Recv != nil && receiverName(d.Recv) == "Conn" {
					methods = append(methods, fn)
				}
			}
			asking := name == askFile && fn == askFunc
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.Ident:
					if asking && (x.Name == "selfReview" || x.Name == "selfReviewBody") {
						asks[x.Name] = true
					}
				case *ast.SelectorExpr:
					if id, ok := x.X.(*ast.Ident); ok && id.Name == "http" && name != "readonly.go" {
						switch x.Sel.Name {
						case "MethodPost":
							if asking {
								posts++
								break
							}
							t.Errorf("%s uses http.%s outside %s", name, x.Sel.Name, askFunc)
						case "MethodPut", "MethodPatch", "MethodDelete", "MethodConnect", "Post", "PostForm":
							t.Errorf("%s uses http.%s", name, x.Sel.Name)
						}
					}
					switch x.Sel.Name {
					case "Post", "PostForm", "Put", "Patch", "Delete", "Create", "Update", "UpdateStatus", "Evict", "DeleteCollection":
						if name != "mirror.go" {
							t.Errorf("%s calls .%s", name, x.Sel.Name)
						}
					}
				case *ast.CallExpr:
					if sel, ok := x.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "NewRequestWithContext" || sel.Sel.Name == "NewRequest") {
						arg := x.Args[0]
						if sel.Sel.Name == "NewRequestWithContext" {
							arg = x.Args[1]
						}
						m, ok := arg.(*ast.SelectorExpr)
						switch {
						case ok && m.Sel.Name == "MethodGet":
						case ok && m.Sel.Name == "MethodPost" && asking:
						default:
							t.Errorf("%s builds a request that is not a GET", name)
						}
					}
				case *ast.BasicLit:
					for _, verb := range []string{`"POST"`, `"PUT"`, `"PATCH"`, `"DELETE"`} {
						if x.Value == verb {
							t.Errorf("%s spells the method %s", name, verb)
						}
					}
				}
				return true
			})
		}
	}
	sort.Strings(methods)
	for _, m := range methods {
		if !connReads[m] {
			t.Errorf("Conn has a method %s beyond the reads %v", m, connReads)
		}
	}
	if posts != 1 || !asks["selfReview"] || !asks["selfReviewBody"] {
		t.Errorf("%s in %s posts %d times (path named %v, body named %v): it may post only the fixed question of who you are", askFunc, askFile, posts, asks["selfReview"], asks["selfReviewBody"])
	}
}

const (
	modulePath  = "github.com/waxb/kubernites/"
	kubectlTree = modulePath + "pkg/live/kubectltree"
)

func writingClient(p string) bool {
	for _, w := range []string{"k8s.io/client-go/kubernetes", "k8s.io/client-go/dynamic", "k8s.io/kubectl", "k8s.io/cli-runtime"} {
		if p == w || strings.HasPrefix(p, w+"/") {
			return true
		}
	}
	return false
}

func reachesWritingClient(t *testing.T, p string, seen map[string]bool) bool {
	if !strings.HasPrefix(p, modulePath) || seen[p] {
		return false
	}
	seen[p] = true
	files, err := filepath.Glob(filepath.Join("..", "..", strings.TrimPrefix(p, modulePath), "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			q := strings.Trim(imp.Path.Value, `"`)
			if writingClient(q) || reachesWritingClient(t, q, seen) {
				return true
			}
		}
	}
	return false
}

func receiverName(fl *ast.FieldList) string {
	if len(fl.List) == 0 {
		return ""
	}
	switch t := fl.List[0].Type.(type) {
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}
