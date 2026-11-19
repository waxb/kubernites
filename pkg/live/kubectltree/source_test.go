package kubectltree_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

type parsed struct {
	name string
	file *ast.File
}

func sources(t *testing.T) []parsed {
	t.Helper()
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var out []parsed
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, parsed{name: name, file: f})
	}
	return out
}

func selector(n ast.Node) string {
	sel, ok := n.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	if id, ok := sel.X.(*ast.Ident); ok {
		return id.Name + "." + sel.Sel.Name
	}
	return "." + sel.Sel.Name
}

func TestEveryConfigTheTreeHandsToKubectlGoesThroughTheGate(t *testing.T) {
	calls := map[string][]string{}
	wraps := 0
	for _, p := range sources(t) {
		for _, imp := range p.file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range []string{"os/exec", "k8s.io/utils/exec", "k8s.io/client-go/kubernetes", "k8s.io/client-go/dynamic", "k8s.io/client-go/transport"} {
				if path == bad || strings.HasPrefix(path, bad+"/") {
					t.Errorf("%s imports %s", p.name, path)
				}
			}
		}
		for _, decl := range p.file.Decls {
			fn := ""
			if d, ok := decl.(*ast.FuncDecl); ok {
				fn = d.Name.Name
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CallExpr:
					name := selector(x.Fun)
					calls[name] = append(calls[name], fn)
					switch name {
					case "cmdutil.NewFactory":
						if len(x.Args) != 1 || selector(x.Args[0].(*ast.CallExpr).Fun) != "cmdutil.NewMatchVersionFlags" {
							t.Errorf("%s: %s builds a factory from something other than the tree's flags", p.name, fn)
						} else if id, ok := x.Args[0].(*ast.CallExpr).Args[0].(*ast.Ident); !ok || id.Name != "cf" {
							t.Errorf("%s: %s builds a factory from something other than the tree's flags", p.name, fn)
						}
					case "cmdconfig.NewCmdConfig":
						if id, ok := x.Args[0].(*ast.Ident); !ok || id.Name != "cf" {
							t.Errorf("%s: %s gives kubectl config other flags", p.name, fn)
						}
					}
					if strings.HasSuffix(name, ".WithWrapConfigFn") || name == "rest.InClusterConfig" || name == "rest.CopyConfig" || name == "rest.AnonymousClientConfig" ||
						strings.HasPrefix(name, "clientcmd.Build") || strings.HasPrefix(name, "clientcmd.New") && name != "clientcmd.NewDefaultClientConfigLoadingRules" ||
						name == ".ClientConfig" || name == "http.Get" || name == "http.Post" || name == "http.Head" || strings.HasPrefix(name, "kubectlcmd.NewDefault") || name == "rest.HTTPClientFor" || name == "rest.RESTClientFor" {
						t.Errorf("%s: %s calls %s", p.name, fn, name)
					}
				case *ast.CompositeLit:
					switch selector(x.Type) {
					case "rest.Config", "http.Client", "http.Transport", "genericclioptions.ConfigFlags":
						t.Errorf("%s: %s builds a %s by hand", p.name, fn, selector(x.Type))
					case "kubectlcmd.KubectlOptions":
						for _, el := range x.Elts {
							kv := el.(*ast.KeyValueExpr)
							key := kv.Key.(*ast.Ident).Name
							if key == "PluginHandler" {
								t.Errorf("%s: %s gives kubectl a plugin handler", p.name, fn)
							}
							if key == "ConfigFlags" {
								if id, ok := kv.Value.(*ast.Ident); !ok || id.Name != "cf" {
									t.Errorf("%s: %s gives kubectl flags that are not the tree's", p.name, fn)
								}
							}
						}
					}
				case *ast.AssignStmt:
					for i, lhs := range x.Lhs {
						s := selector(lhs)
						switch {
						case strings.HasSuffix(s, ".WrapConfigFn"):
							wraps++
							lit, ok := x.Rhs[i].(*ast.FuncLit)
							if fn != "flags" || !ok || !callsGateFirst(lit) {
								t.Errorf("%s: %s sets WrapConfigFn without calling the gate first", p.name, fn)
							}
						case strings.HasSuffix(s, ".WrapTransport"), strings.HasSuffix(s, ".Transport"), strings.HasSuffix(s, ".Dial"), strings.HasSuffix(s, ".Proxy"):
							t.Errorf("%s: %s sets %s", p.name, fn, s)
						case s == "http.DefaultClient" || s == "http.DefaultTransport":
							t.Errorf("%s: %s sets %s", p.name, fn, s)
						}
					}
				case *ast.SelectorExpr:
					if s := selector(x); s == "http.DefaultClient" || s == "http.DefaultTransport" {
						t.Errorf("%s: %s uses %s", p.name, fn, s)
					}
				}
				return true
			})
		}
	}
	if wraps != 1 {
		t.Errorf("WrapConfigFn is set %d times, want once", wraps)
	}
	if c := calls["genericclioptions.NewConfigFlags"]; len(c) != 1 || c[0] != "flags" {
		t.Errorf("ConfigFlags are made in %v, want only in flags", c)
	}
	if c := calls["kubectlcmd.NewKubectlCommand"]; len(c) != 1 || c[0] != "runAs" {
		t.Errorf("the kubectl tree is made in %v, want only in runAs", c)
	}
	if c := calls["t.flags"]; len(c) != 1 || c[0] != "runAs" {
		t.Errorf("the tree's flags are made in %v, want only in runAs", c)
	}
}

func callsGateFirst(lit *ast.FuncLit) bool {
	for _, st := range lit.Body.List {
		as, ok := st.(*ast.AssignStmt)
		if !ok {
			return false
		}
		if sel, ok := as.Lhs[0].(*ast.SelectorExpr); ok {
			if _, call := as.Rhs[0].(*ast.CallExpr); !call && selector(sel) == "c."+sel.Sel.Name {
				continue
			}
			return false
		}
		call, ok := as.Rhs[0].(*ast.CallExpr)
		if !ok {
			return false
		}
		id, ok := call.Fun.(*ast.Ident)
		lhs, lok := as.Lhs[0].(*ast.Ident)
		arg, aok := call.Args[0].(*ast.Ident)
		return ok && id.Name == "gate" && len(call.Args) == 1 && lok && lhs.Name == "c" && aok && arg.Name == "c"
	}
	return false
}

func TestTheGateIsTheTreesOwn(t *testing.T) {
	found := false
	for _, p := range sources(t) {
		for _, decl := range p.file.Decls {
			d, ok := decl.(*ast.FuncDecl)
			if !ok || d.Name.Name != "flags" {
				continue
			}
			ast.Inspect(d, func(n ast.Node) bool {
				as, ok := n.(*ast.AssignStmt)
				if ok && len(as.Lhs) == len(as.Rhs) {
					for i, lhs := range as.Lhs {
						if id, ok := lhs.(*ast.Ident); ok && id.Name == "gate" && selector(as.Rhs[i]) == "t.wrap" {
							found = true
						}
					}
				}
				return true
			})
		}
	}
	if !found {
		t.Fatal("flags does not take the gate from the tree")
	}
}
