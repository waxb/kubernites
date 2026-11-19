package hud

import (
	"strings"
)

const maxEditorLines = 400

type Editor struct {
	Lines []string
	Row   int
	Col   int
	Top   int
	Left  int
	Dirty bool
}

func NewEditor(text string) Editor {
	lines := strings.Split(strings.ReplaceAll(text, "\t", "  "), "\n")
	return Editor{Lines: lines}
}

func (e *Editor) Text() string {
	return strings.Join(e.Lines, "\n")
}

func (e *Editor) Empty() bool {
	return strings.TrimSpace(e.Text()) == ""
}

func (e *Editor) line() []rune {
	return []rune(e.Lines[e.Row])
}

func (e *Editor) setLine(row int, text string) {
	next := append([]string(nil), e.Lines...)
	next[row] = text
	e.Lines = next
	e.Dirty = true
}

func (e *Editor) Insert(r rune) {
	if r < 32 || r == 127 {
		return
	}
	cur := e.line()
	e.setLine(e.Row, string(cur[:e.Col])+string(r)+string(cur[e.Col:]))
	e.Col++
}

func (e *Editor) InsertText(text string) {
	for _, r := range text {
		e.Insert(r)
	}
}

func (e *Editor) Tab() {
	e.InsertText("  ")
}

func indentOf(text string) string {
	return text[:len(text)-len(strings.TrimLeft(text, " "))]
}

func (e *Editor) Newline() {
	if len(e.Lines) >= maxEditorLines {
		return
	}
	cur := e.line()
	head, tail := string(cur[:e.Col]), string(cur[e.Col:])
	indent := indentOf(head)
	if strings.HasSuffix(strings.TrimSpace(head), ":") {
		indent += "  "
	}
	next := make([]string, 0, len(e.Lines)+1)
	next = append(next, e.Lines[:e.Row]...)
	next = append(next, head, indent+strings.TrimLeft(tail, " "))
	next = append(next, e.Lines[e.Row+1:]...)
	e.Lines, e.Dirty = next, true
	e.Row, e.Col = e.Row+1, len([]rune(indent))
}

func (e *Editor) Backspace() {
	if e.Col > 0 {
		cur := e.line()
		e.setLine(e.Row, string(cur[:e.Col-1])+string(cur[e.Col:]))
		e.Col--
		return
	}
	if e.Row == 0 {
		return
	}
	prev := []rune(e.Lines[e.Row-1])
	joined := string(prev) + e.Lines[e.Row]
	next := append(append([]string(nil), e.Lines[:e.Row-1]...), joined)
	e.Lines, e.Dirty = append(next, e.Lines[e.Row+1:]...), true
	e.Row, e.Col = e.Row-1, len(prev)
}

func (e *Editor) Delete() {
	cur := e.line()
	if e.Col < len(cur) {
		e.setLine(e.Row, string(cur[:e.Col])+string(cur[e.Col+1:]))
		return
	}
	if e.Row+1 < len(e.Lines) {
		e.Row, e.Col = e.Row+1, 0
		e.Backspace()
	}
}

func (e *Editor) Move(dRow, dCol int) {
	if dCol != 0 {
		e.Col += dCol
		switch {
		case e.Col < 0 && e.Row > 0:
			e.Row--
			e.Col = len(e.line())
		case e.Col > len(e.line()) && e.Row+1 < len(e.Lines):
			e.Row, e.Col = e.Row+1, 0
		}
	}
	e.Row = min(max(e.Row+dRow, 0), len(e.Lines)-1)
	e.Col = min(max(e.Col, 0), len(e.line()))
}

func (e *Editor) Home() { e.Col = 0 }

func (e *Editor) End() { e.Col = len(e.line()) }

func (e *Editor) ScrollBy(step, visible int) {
	e.Top = min(max(e.Top+step, 0), max(len(e.Lines)-visible, 0))
}

func (e *Editor) Follow(visibleRows, visibleCols int) {
	switch {
	case e.Row < e.Top:
		e.Top = e.Row
	case e.Row >= e.Top+visibleRows:
		e.Top = e.Row - visibleRows + 1
	}
	switch {
	case e.Col < e.Left:
		e.Left = e.Col
	case e.Col >= e.Left+visibleCols:
		e.Left = e.Col - visibleCols + 1
	}
}
