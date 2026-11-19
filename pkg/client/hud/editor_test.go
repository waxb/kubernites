package hud

import "testing"

func TestEditorEditing(t *testing.T) {
	e := NewEditor("spec:\n\tname: x")
	if e.Lines[1] != "  name: x" {
		t.Fatalf("tabs become two spaces: %q", e.Lines[1])
	}
	e.End()
	e.Newline()
	if e.Row != 1 || e.Col != 2 || e.Lines[1] != "  " {
		t.Fatalf("newline after a map key indents: row %d col %d %q", e.Row, e.Col, e.Lines)
	}
	e.InsertText("app: y")
	e.Newline()
	if e.Col != 2 || e.Lines[2] != "  " {
		t.Fatalf("newline keeps indentation: col %d %q", e.Col, e.Lines)
	}
	e.Insert('\n')
	e.Tab()
	if e.Lines[2] != "    " {
		t.Fatalf("tab inserts two spaces: %q", e.Lines[2])
	}
	e.Home()
	e.Backspace()
	if e.Row != 1 || e.Lines[1] != "  app: y    " {
		t.Fatalf("backspace at column 0 joins lines: row %d %q", e.Row, e.Lines)
	}
	e.End()
	e.Delete()
	if e.Lines[1] != "  app: y      name: x" {
		t.Fatalf("delete at the end joins the next line: %q", e.Lines)
	}
	e.Home()
	e.Delete()
	e.Move(-10, 0)
	e.Home()
	e.Backspace()
	if e.Row != 0 || !e.Dirty {
		t.Fatal("top of the file")
	}
}

func TestEditorNavigation(t *testing.T) {
	e := NewEditor("spec:\n  app: y\nkind: x")
	e.Row, e.Col = 0, 0
	e.Move(0, -1)
	e.End()
	e.Move(0, 1)
	if e.Row != 1 || e.Col != 0 {
		t.Fatalf("right at the end of a line wraps: %d,%d", e.Row, e.Col)
	}
	e.Row, e.Col = 30, 90
	e.Follow(10, 20)
	if e.Top != 21 || e.Left != 71 {
		t.Fatalf("follow = %d,%d", e.Top, e.Left)
	}
	e.Row, e.Col = 0, 0
	e.Follow(10, 20)
	if e.Top != 0 || e.Left != 0 {
		t.Fatal("follow back")
	}
	big := NewEditor(string(make([]byte, 0)))
	for i := 0; i < maxEditorLines+5; i++ {
		big.Newline()
	}
	if len(big.Lines) != maxEditorLines {
		t.Fatalf("line cap = %d", len(big.Lines))
	}
	big.ScrollBy(1000, 10)
	if big.Top != maxEditorLines-10 {
		t.Fatalf("scroll = %d", big.Top)
	}
	blank := NewEditor("  \n")
	if !blank.Empty() {
		t.Fatal("whitespace is empty")
	}
}
