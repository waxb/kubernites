package write

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	gatePackage   = "pkg/live/write"
	embeddingRoot = "KUBERNITES_EMBEDDING_ROOT"
)

var selfReviewPost = map[string]string{"pkg/live/whoami.go": "WhoAmI"}

var buildsTheGate = map[string]string{"pkg/client/app/livechoices.go": "gateFor"}

var choosesTheLevel = map[string]string{"pkg/client/app/livechoices.go": "confirmedChoice"}

var issuesTickets = map[string]string{
	"pkg/client/app/livewrite.go": "confirmOrder",
	"pkg/live/stone_hold.go":      "HoldStoneWrite",
}

var passesClasses = map[string][]string{
	"pkg/client/app/livewrite.go": {"advanceOrder", "PassForCapture"},
	"pkg/live/stone_hold.go":      {"HoldStoneWrite"},
}

var embedsKubectl = map[string]string{
	"pkg/live/kubectltree/tree.go":    "the Stone's kubectl tree, built with ConfigFlags.WrapConfigFn set to the gate's wrap",
	"pkg/live/kubectltree/prune.go":   "the tree's pruning: kubectl config, auth can-i and version rebuilt on the same gated flags",
	"pkg/live/kubectltree/edit.go":    "kubectl edit reads the object for the Scriptorium through the same gated flags",
	"pkg/live/kubectltree/reach.go":   "kubectl exec, attach, debug and run -i stream into the Stone's terminal through the same gated flags",
	"pkg/live/kubectltree/forward.go": "kubectl port-forward listens on 127.0.0.1 through the same gated flags and stays open until STOP",
	"pkg/live/kubectltree/cp.go":      "kubectl cp's tar through exec, through the same gated flags, its relative local paths under ~/Downloads/Kubernites, never ~/.kube or the connected kubeconfig",
}

var writingClients = []string{
	"k8s.io/client-go/kubernetes", "k8s.io/client-go/dynamic", "k8s.io/client-go/scale",
	"k8s.io/cli-runtime/pkg/resource", "k8s.io/kubectl",
}

var restClientBuilders = map[string]bool{"RESTClientFor": true, "UnversionedRESTClientFor": true, "NewRESTClient": true, "RESTClientForConfigAndClient": true}

func repositoryRoots(t *testing.T) (string, []string) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(data), "module github.com/waxb/kubernites\n") {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("the open module's go.mod was not found")
		}
		dir = parent
	}
	roots := []string{dir}
	if extra := os.Getenv(embeddingRoot); extra != "" {
		abs, err := filepath.Abs(extra)
		if err != nil {
			t.Fatal(err)
		}
		if info, err := os.Stat(abs); err != nil || !info.IsDir() {
			t.Fatalf("%s=%s is not a directory", embeddingRoot, extra)
		}
		roots = append(roots, abs)
	}
	for i, r := range roots {
		real, err := filepath.EvalSymlinks(r)
		if err != nil {
			t.Fatal(err)
		}
		roots[i] = real
	}
	return roots[0], roots
}

type sourceFile struct {
	rel  string
	file *ast.File
	fset *token.FileSet
}

func everySource(t *testing.T) []sourceFile {
	t.Helper()
	top, roots := repositoryRoots(t)
	var out []sourceFile
	seen := map[string]bool{}
	for i, root := range roots {
		before := len(out)
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if d.IsDir() {
				if path != root && (name == "testdata" || name == "vendor" || name == "node_modules" || strings.HasPrefix(name, ".") || name == "web" || name == "bin") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || seen[path] {
				return nil
			}
			seen[path] = true
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(top, path)
			out = append(out, sourceFile{rel: filepath.ToSlash(rel), file: f, fset: fset})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 {
			t.Logf("%d sources from the embedding root", len(out)-before)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out
}

func inGate(rel string) bool {
	return strings.HasPrefix(rel, gatePackage+"/") && strings.Count(strings.TrimPrefix(rel, gatePackage+"/"), "/") == 0
}

func funcAt(f *ast.File, pos token.Pos) string {
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Pos() <= pos && pos <= fn.End() {
			return fn.Name.Name
		}
	}
	return ""
}

func importName(f *ast.File, path string) string {
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != path {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return path[strings.LastIndex(path, "/")+1:]
	}
	return ""
}

func getMethod(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.SelectorExpr:
		return x.Sel.Name == "MethodGet"
	case *ast.BasicLit:
		v, _ := strconv.Unquote(x.Value)
		return v == "GET"
	}
	return false
}

func TestOnlyTheWriteGateBuildsARequestThatIsNotAGet(t *testing.T) {
	files := everySource(t)
	if len(files) < 100 {
		t.Fatalf("only %d source files were read", len(files))
	}
	for _, sf := range files {
		if inGate(sf.rel) {
			continue
		}
		f := sf.file
		where := func(n ast.Node) string { return sf.rel + ":" + strconv.Itoa(sf.fset.Position(n.Pos()).Line) }
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			for _, w := range writingClients {
				if (p == w || strings.HasPrefix(p, w+"/")) && embedsKubectl[sf.rel] == "" {
					t.Errorf("%s imports %s, a client that can write, outside the write gate", sf.rel, p)
				}
			}
		}
		restName := importName(f, "k8s.io/client-go/rest")
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				sel, ok := x.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				switch sel.Sel.Name {
				case "NewRequest", "NewRequestWithContext":
					i := 0
					if sel.Sel.Name == "NewRequestWithContext" {
						i = 1
					}
					if len(x.Args) <= i || getMethod(x.Args[i]) {
						return true
					}
					if m, ok := x.Args[i].(*ast.SelectorExpr); ok && m.Sel.Name == "MethodPost" && selfReviewPost[sf.rel] == funcAt(f, x.Pos()) && funcAt(f, x.Pos()) != "" {
						return true
					}
					t.Errorf("%s builds a request that is not a GET", where(x))
				case "Post", "PostForm", "Patch", "Head":
					t.Errorf("%s calls .%s", where(x), sel.Sel.Name)
				case "Put":
					if id, ok := sel.X.(*ast.Ident); !ok || !strings.HasSuffix(strings.ToLower(id.Name), "pool") {
						t.Errorf("%s calls .Put", where(x))
					}
				}
				if id, ok := sel.X.(*ast.Ident); ok && restName != "" && id.Name == restName && restClientBuilders[sel.Sel.Name] {
					t.Errorf("%s builds a REST client (%s.%s), which can write", where(x), id.Name, sel.Sel.Name)
				}
			case *ast.CompositeLit:
				if sel, ok := x.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Request" {
					if id, ok := sel.X.(*ast.Ident); ok && id.Name == "http" {
						for _, el := range x.Elts {
							if kv, ok := el.(*ast.KeyValueExpr); ok {
								if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "Method" && !getMethod(kv.Value) {
									t.Errorf("%s builds an http.Request whose method is not GET", where(x))
								}
							}
						}
					}
				}
			case *ast.AssignStmt:
				for i, lhs := range x.Lhs {
					if sel, ok := lhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "Method" && (i >= len(x.Rhs) || !getMethod(x.Rhs[i])) {
						t.Errorf("%s sets a request's method", where(x))
					}
				}
			}
			return true
		})
	}
}

func TestOnlyTheConfirmedChoiceBuildsTheGateAndOnlyCONFIRMIssuesATicket(t *testing.T) {
	for _, sf := range everySource(t) {
		if inGate(sf.rel) {
			continue
		}
		f := sf.file
		name := importName(f, "github.com/waxb/kubernites/pkg/live/write")
		if name == "" {
			continue
		}
		where := func(n ast.Node) string { return sf.rel + ":" + strconv.Itoa(sf.fset.Position(n.Pos()).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				sel, ok := x.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				fn := funcAt(f, x.Pos())
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == name && sel.Sel.Name == "New" && buildsTheGate[sf.rel] != fn {
					t.Errorf("%s builds the write gate in %s: only %v may, from the picker's confirmed choice", where(x), fn, buildsTheGate)
				}
				if strings.HasPrefix(sel.Sel.Name, "Issue") && issuesTickets[sf.rel] != fn {
					t.Errorf("%s issues a ticket in %s: only CONFIRM may (%v)", where(x), fn, issuesTickets)
				}
				if sel.Sel.Name == "Pass" && !slices.Contains(passesClasses[sf.rel], fn) {
					t.Errorf("%s passes a class warning in %s: only its warning's I UNDERSTAND may (%v)", where(x), fn, passesClasses)
				}
			case *ast.CompositeLit:
				if sel, ok := x.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Choice" {
					if id, ok := sel.X.(*ast.Ident); ok && id.Name == name && choosesTheLevel[sf.rel] != funcAt(f, x.Pos()) {
						t.Errorf("%s makes a write.Choice in %s: only %v may", where(x), funcAt(f, x.Pos()), choosesTheLevel)
					}
				}
			}
			return true
		})
	}
}
