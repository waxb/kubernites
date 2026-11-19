package ui

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func greedyNamer(s string) string {
	return strings.NewReplacer("PORTERS", "PORTER PODS", "PORTER", "PORTER POD", "Hearth", "Pod").Replace(s)
}

func TestNamedTextIsNeverNamedTwice(t *testing.T) {
	SetNamer(greedyNamer)
	defer SetNamer(nil)
	var f Font
	once := Name("PORTERS, CLASS OF 1.0")
	if once != "PORTER PODS, CLASS OF 1.0" {
		t.Fatalf("Name = %q", once)
	}
	if twice := Name(once); twice != once {
		t.Errorf("named text was named again: %q", twice)
	}
	clipped := Clip("PORTERS, CLASS OF 1.0", 12)
	if clipped != "PORTER PO..." {
		t.Fatalf("Clip = %q", clipped)
	}
	if w, _ := f.Measure(clipped, 1); w != 12*Advance-1 {
		t.Errorf("a clipped label grew when drawn: %d", w)
	}
	for _, line := range f.Wrap("PORTERS PORTERS PORTERS", 12*Advance, 1) {
		if got := Name(line); got != line {
			t.Errorf("wrapped line %q was named again: %q", line, got)
		}
	}
	for _, line := range f.WrapLines("PORTERS PORTERS PORTERS PORTERS", 12*Advance, 1, 1) {
		if got := Name(line); got != line {
			t.Errorf("wrapped line %q was named again: %q", line, got)
		}
	}
}

func TestSettledTextStaysAsIs(t *testing.T) {
	SetNamer(greedyNamer)
	defer SetNamer(nil)
	if got := Name(Settle("PORTER POD")); got != "PORTER POD" {
		t.Errorf("settled text was renamed: %q", got)
	}
	SetNamer(nil)
	if got := Settle("PORTER"); got != "PORTER" || Name(got) != "PORTER" {
		t.Errorf("settle without a namer = %q", got)
	}
}

func TestNamerMemoryIsBounded(t *testing.T) {
	SetNamer(greedyNamer)
	defer SetNamer(nil)
	for i := 0; i < namedMemory+10; i++ {
		Name(strings.Repeat("x", i%50) + string(rune('a'+i%26)) + strings.Repeat("y", i/26))
	}
	if n := current.Load().size(); n > namedMemory {
		t.Errorf("memory grew to %d", n)
	}
	if got := Name("a Hearth"); got != "a Pod" {
		t.Errorf("naming after a reset = %q", got)
	}
}

func TestLiteralClipKeepsCommands(t *testing.T) {
	SetNamer(greedyNamer)
	defer SetNamer(nil)
	literal := Font{Literal: true}
	if got := literal.Clip("kubectl get Hearth", 30); got != "kubectl get Hearth" {
		t.Errorf("literal clip renamed: %q", got)
	}
	if got := literal.Clip("kubectl get Hearth", 10); got != "kubectl..." {
		t.Errorf("literal clip = %q", got)
	}
	if got := (Font{}).Clip("a Hearth", 30); got != "a Pod" {
		t.Errorf("clip = %q", got)
	}
}

func TestBridgePicksTheNamesMode(t *testing.T) {
	SetNamer(nil)
	if Naming() {
		t.Fatal("no namer is story mode")
	}
	if got := Bridge("Professions are images.", "Images are pulled."); got != "Professions are images." {
		t.Errorf("story bridge = %q", got)
	}
	SetNamer(greedyNamer)
	defer SetNamer(nil)
	if !Naming() {
		t.Fatal("a namer is Kubernetes mode")
	}
	got := Bridge("Hearths are Pods.", "PORTER Pods run here.")
	if got != "PORTER Pods run here." || Name(got) != got {
		t.Errorf("kubernetes bridge = %q named %q", got, Name(got))
	}
}

func TestRecorderSeesDrawnText(t *testing.T) {
	SetNamer(greedyNamer)
	defer SetNamer(nil)
	var runs []TextRun
	stop := RecordText(func(r TextRun) { runs = append(runs, r) })
	dst := ebiten.NewImage(200, 60)
	defer dst.Deallocate()
	var f Font
	f.Draw(dst, "  a Hearth", 10, 5, 1, Palette.Text)
	f.DrawShadow(dst, "ab\nabcd", 20, 30, 2, Palette.Text)
	Font{Literal: true}.Draw(dst, "Hearth", 0, 0, 1, Palette.Text)
	f.Draw(dst, "   ", 0, 0, 1, Palette.Text)
	stop()
	f.Draw(dst, "late", 0, 0, 1, Palette.Text)
	want := []TextRun{
		{Text: "a Pod", Rect: Rect{X: 10 + 2*Advance, Y: 5, W: 5*Advance - 1, H: GlyphHeight}},
		{Text: "ab\nabcd", Rect: Rect{X: 20, Y: 30, W: (4*Advance - 1) * 2, H: (LineHeight + GlyphHeight) * 2}},
		{Text: "Hearth", Rect: Rect{X: 0, Y: 0, W: 6*Advance - 1, H: GlyphHeight}},
	}
	if len(runs) != len(want) {
		t.Fatalf("runs = %+v", runs)
	}
	for i := range want {
		want[i].Spelled = want[i].Text
		if runs[i] != want[i] {
			t.Errorf("run %d = %+v, want %+v", i, runs[i], want[i])
		}
	}
}

func TestTextWidthUsesTheNamer(t *testing.T) {
	SetNamer(greedyNamer)
	defer SetNamer(nil)
	var f Font
	if got := f.Width("PORTER", 1); got != 10*Advance-1 {
		t.Errorf("width = %d", got)
	}
	if got := (Font{Literal: true}).Width("PORTER", 2); got != (6*Advance-1)*2 {
		t.Errorf("literal width = %d", got)
	}
}

func TestSpanPlacesTextAfterNamedText(t *testing.T) {
	SetNamer(greedyNamer)
	defer SetNamer(nil)
	var f Font
	if got := f.Span("PORTER", 2); got != 12*Advance {
		t.Errorf("span = %d", got)
	}
	if got := (Font{Literal: true}).Span("PORTER", 1); got != 7*Advance {
		t.Errorf("literal span = %d", got)
	}
	if got := f.Span("", 2); got != 2*Advance {
		t.Errorf("empty span = %d", got)
	}
}
