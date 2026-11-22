package live

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var liveClosure = []string{
	"github.com/waxb/kubernites/pkg/kubectl", "github.com/waxb/kubernites/pkg/lexicon", "github.com/waxb/kubernites/pkg/live",
	"github.com/waxb/kubernites/pkg/live/forwarding", "github.com/waxb/kubernites/pkg/live/kubectltree", "github.com/waxb/kubernites/pkg/live/write", "github.com/waxb/kubernites/pkg/sim/clock", "github.com/waxb/kubernites/pkg/sim/limitranger", "github.com/waxb/kubernites/pkg/sim/podutil",
	"github.com/waxb/kubernites/pkg/sim/rolloututil", "github.com/waxb/kubernites/pkg/sim/store", "github.com/waxb/kubernites/pkg/sim/volumeutil",
	"github.com/waxb/kubernites/pkg/soulstone", "github.com/waxb/kubernites/pkg/view",
}

func TestTheLivePathReachesNoSimulationAndNoTrigger(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", ".").Output()
	if err != nil {
		t.Fatal(err)
	}
	var mine []string
	for _, p := range strings.Fields(string(out)) {
		if strings.HasPrefix(p, "github.com/waxb/") || strings.HasPrefix(p, "kubernites/") {
			mine = append(mine, p)
		}
	}
	slices.Sort(mine)
	if !slices.Equal(mine, liveClosure) {
		t.Fatalf("the live package reaches %v; only %v may be reached, none of them a simulation", mine, liveClosure)
	}
}

type callSite struct {
	file, fn, callee string
	args             []string
}

func liveCalls(t *testing.T) []callSite {
	t.Helper()
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var out []callSite
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fd, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				callee := ""
				switch fn := call.Fun.(type) {
				case *ast.Ident:
					callee = fn.Name
				case *ast.SelectorExpr:
					callee = fn.Sel.Name
				}
				var args []string
				for _, a := range call.Args {
					args = append(args, exprText(fset, a))
				}
				out = append(out, callSite{file: name, fn: fd.Name.Name, callee: callee, args: args})
				return true
			})
		}
	}
	return out
}

func exprText(fset *token.FileSet, e ast.Expr) string {
	var b strings.Builder
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			b.WriteString(x.Name + ".")
		case *ast.UnaryExpr:
			b.WriteString(x.Op.String())
		}
		return true
	})
	return b.String()
}

func TestOnlyTheMirrorCanRaiseACalamityInALiveRealm(t *testing.T) {
	allowed := map[string][]string{
		"reckon":         {"Pump"},
		"newTroubleBook": {"OpenContext"},
		"signals":        {"Pump", "OpenContext", "Signals"},
		"feed":           {"ingest"},
		"swap":           {"relist"},
		"mirrorOne":      {"ingest", "mirrorSwap"},
		"mirrorSwap":     {"relist"},
		"Mirror":         {"mirrorOne"},
		"Unmirror":       {"mirrorOne"},
		"Restore":        {"NewMirror"},
		"ingest":         {"follow"},
		"follow":         {"watchOnce", "Replay"},
		"relist":         {"watchKind"},
		"write":          {"age"},
	}
	for _, c := range liveCalls(t) {
		fns, watched := allowed[c.callee]
		if !watched {
			continue
		}
		if !slices.Contains(fns, c.fn) {
			t.Errorf("%s: %s calls %s; only %v may", c.file, c.fn, c.callee, fns)
		}
		if (c.callee == "reckon" || c.callee == "newTroubleBook") && (len(c.args) == 0 || c.args[0] != "signals.&r.snap.") {
			t.Errorf("%s: %s feeds the trouble book from %q, not from the mirror", c.file, c.fn, c.args)
		}
	}
	fset := token.NewFileSet()
	files, _ := filepath.Glob("*.go")
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fd, func(n ast.Node) bool {
				as, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for i, lhs := range as.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "book" && sel.Sel.Name != "snap" && sel.Sel.Name != "mirror" {
						continue
					}
					if fd.Name.Name == "OpenContext" && sel.Sel.Name == "book" && i < len(as.Rhs) && strings.HasPrefix(exprText(fset, as.Rhs[i]), "newTroubleBook.signals.&r.snap.") {
						continue
					}
					t.Errorf("%s: %s assigns %s: the trouble book, the mirror and its snapshot change only through the watch", name, fd.Name.Name, sel.Sel.Name)
				}
				return true
			})
		}
	}
}
