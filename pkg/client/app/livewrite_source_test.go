package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func callsIn(t *testing.T, dir string) map[string]map[string][]string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string][]string{}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				var callee string
				switch x := n.(type) {
				case *ast.CallExpr:
					switch c := x.Fun.(type) {
					case *ast.SelectorExpr:
						callee = c.Sel.Name
					case *ast.Ident:
						callee = c.Name
					}
				case *ast.Ident:
					if x.Name == "IntentLiveOrder" || x.Name == "IntentLiveScroll" {
						callee = x.Name
					}
				}
				if callee != "" {
					if out[callee] == nil {
						out[callee] = map[string][]string{}
					}
					out[callee][filepath.Base(name)] = append(out[callee][filepath.Base(name)], fn.Name.Name)
				}
				return true
			})
		}
	}
	return out
}

func TestOnlyThePlayersIntentReachesASend(t *testing.T) {
	calls := callsIn(t, ".")
	allowed := map[string][]string{
		"Send":         {"sendOrder"},
		"RunDrain":     {"startDrain"},
		"Issue":        {"confirmOrder"},
		"IssueDrain":   {"confirmOrder"},
		"sendOrder":    {"afterBuild", "confirmOrder"},
		"startDrain":   {"sendOrder", "confirmOrder"},
		"confirmOrder": {"advanceOrder"},
		"afterBuild":   {"advanceOrder"},
		"advanceOrder": {"startOrder", "liveScroll"},
		"startOrder":   {"writeIntent"},
		"liveScroll":   {"writeIntent"},
		"writeIntent":  {"Handle"},
	}
	for callee, callers := range allowed {
		for file, fns := range calls[callee] {
			for _, fn := range fns {
				if !slices.Contains(callers, fn) {
					t.Errorf("%s calls %s in %s: only %v may", fn, callee, file, callers)
				}
			}
		}
	}
	hud := callsIn(t, "../hud")
	for _, intent := range []string{"IntentLiveOrder", "IntentLiveScroll"} {
		for file, fns := range hud[intent] {
			for _, fn := range fns {
				if file != "liveorders.go" && !(file == "live.go" && fn == "") {
					t.Errorf("%s in %s makes %s: only the order buttons, their chords and the scroll may", fn, file, intent)
				}
			}
		}
	}
}
