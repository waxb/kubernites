package gfx

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var (
	previewDirs    = []string{"../hud", "../app", "../livescenes"}
	legacyHelpers  = map[string]string{"BuildingName": "PreviewNames", "BuildingNames": "PreviewNames", "HearthName": "HearthLookNames", "HearthQoSNames": "HearthLookNames", "DepotName": "PreviewNames", "DepotNames": "PreviewNames"}
	legacyLiterals = []string{"building/", "hearth/", "portrait/keep"}
)

type legacyScan struct {
	fset *token.FileSet
	out  []string
}

func (l *legacyScan) flag(n ast.Node, msg string) {
	p := l.fset.Position(n.Pos())
	l.out = append(l.out, fmt.Sprintf("%s:%d: %s", filepath.Base(p.Filename), p.Line, msg))
}

func (l *legacyScan) visit(n ast.Node) bool {
	switch e := n.(type) {
	case *ast.BasicLit:
		l.literal(e)
	case *ast.SelectorExpr:
		if pkg, ok := e.X.(*ast.Ident); ok && pkg.Name == "gfx" && legacyHelpers[e.Sel.Name] != "" {
			l.flag(e, "gfx."+e.Sel.Name+" names the Dev sprite; use gfx."+legacyHelpers[e.Sel.Name])
		}
	case *ast.BinaryExpr:
		l.kindPortrait(e)
	}
	return true
}

func (l *legacyScan) literal(e *ast.BasicLit) {
	if e.Kind != token.STRING {
		return
	}
	text, err := strconv.Unquote(e.Value)
	if err != nil {
		return
	}
	for _, prefix := range legacyLiterals {
		if strings.HasPrefix(text, prefix) {
			l.flag(e, fmt.Sprintf("raw sprite name %q; ask gfx for the aged look", text))
		}
	}
}

func (l *legacyScan) kindPortrait(e *ast.BinaryExpr) {
	lit, ok := e.X.(*ast.BasicLit)
	sel, isSel := e.Y.(*ast.SelectorExpr)
	if e.Op != token.ADD || !ok || !isSel || lit.Value != strconv.Quote("portrait/") || sel.Sel.Name != "Kind" {
		return
	}
	l.flag(e, "a building portrait by kind; use gfx.BuildPortrait or gfx.StructurePortrait")
}

func scanLegacy(t *testing.T, dir string) []string {
	t.Helper()
	fset := token.NewFileSet()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	scan := &legacyScan{fset: fset}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(f, scan.visit)
	}
	return scan.out
}

func TestPreviewSitesUseAgedLooks(t *testing.T) {
	out := []string{}
	for _, dir := range previewDirs {
		out = append(out, scanLegacy(t, dir)...)
	}
	sort.Strings(out)
	if len(out) > 0 {
		t.Fatalf("hud and app must show buildings through the aged helpers:\n%s", strings.Join(out, "\n"))
	}
}

func TestLegacyScanCatchesEveryForm(t *testing.T) {
	src := `package x
func f(b B) {
	_ = "building/codex/ready"
	_ = gfx.BuildingNames(s)
	_ = "portrait/" + b.Kind
	_ = "portrait/keep-stone"
	_ = gfx.StructurePortrait(s, 2)
	_ = "portrait/" + sprite
}`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	scan := &legacyScan{fset: fset}
	ast.Inspect(f, scan.visit)
	if len(scan.out) != 4 {
		t.Fatalf("want 4 findings, got %d: %v", len(scan.out), scan.out)
	}
}
