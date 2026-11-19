package ui

import (
	"slices"
	"strings"
	"testing"
)

func TestMeasureAndMaxChars(t *testing.T) {
	var f Font
	w, h := f.Measure("abc", 1)
	if w != 3*Advance-1 || h != LineHeight {
		t.Fatalf("measure = %d,%d", w, h)
	}
	w, h = f.Measure("ab\nabcd", 2)
	if w != (4*Advance-1)*2 || h != 2*LineHeight*2 {
		t.Fatalf("multi = %d,%d", w, h)
	}
	if MaxChars(60, 1) != 10 || MaxChars(0, 1) != 1 {
		t.Fatalf("maxchars = %d", MaxChars(60, 1))
	}
}

func TestGlyphFallbacks(t *testing.T) {
	if glyphFor('a') == glyphFor('A') {
		t.Fatal("lower case should differ from upper case")
	}
	if glyphFor('é') != glyphs['?'] {
		t.Fatal("unknown rune should fall back to ?")
	}
	for r := 'a'; r <= 'z'; r++ {
		if _, ok := glyphs[r]; !ok {
			t.Fatalf("missing lower-case glyph %q", r)
		}
	}
}

func TestWrapAndClip(t *testing.T) {
	var f Font
	lines := f.Wrap("the quick brown fox\n\njumps", 6*Advance, 1)
	want := []string{"the", "quick", "brown", "fox", "", "jumps"}
	if len(lines) != len(want) {
		t.Fatalf("lines = %q", lines)
	}
	long := f.Wrap("abcdefghijkl", 5*Advance, 1)
	if len(long) != 3 || long[0] != "abcde" {
		t.Fatalf("long = %q", long)
	}
	if Clip("hello world", 5) != "he..." || Clip("hi", 5) != "hi" || Clip("hey", 1) != "h" || Clip("hello", 3) != "hel" {
		t.Fatalf("clip wrong: %q", Clip("hello world", 5))
	}
}

func TestFontCellHasDescenders(t *testing.T) {
	if GlyphHeight != 9 || CapHeight != 7 || LineHeight != 11 {
		t.Fatalf("cell = %d cap %d line %d", GlyphHeight, CapHeight, LineHeight)
	}
	for _, r := range "gpqyj,;" {
		g := glyphFor(r)
		if g[7] == 0 && g[8] == 0 {
			t.Errorf("%q has no descender", r)
		}
	}
	for _, r := range "ABCabcdefhiklmnorstuvwxz0123456789" {
		g := glyphFor(r)
		if g[7] != 0 || g[8] != 0 {
			t.Errorf("%q must sit on the baseline", r)
		}
	}
}

func TestWrapBreaksAtPhrasesBeforeWords(t *testing.T) {
	var f Font
	for _, c := range []struct {
		text  string
		chars int
		want  []string
	}{
		{"EXTRA HUTS 25%  HUTS MAY BE DOWN 25%", 33, []string{"EXTRA HUTS 25%", "HUTS MAY BE DOWN 25%"}},
		{"SURGE 25%  UNAVAILABLE 25%", 33, []string{"SURGE 25% UNAVAILABLE 25%"}},
		{"REVISION 3  HISTORY 2 OF 10  NEWEST FIRST", 24, []string{"REVISION 3", "HISTORY 2 OF 10", "NEWEST FIRST"}},
		{"KEEPS 1  READY 1  HUTS 1", 17, []string{"KEEPS 1 READY 1", "HUTS 1"}},
		{"ONE  A PHRASE FAR TOO LONG FOR ONE LINE", 12, []string{"ONE", "A PHRASE FAR", "TOO LONG FOR", "ONE LINE"}},
		{"the quick brown fox", 9, []string{"the quick", "brown fox"}},
	} {
		if got := f.Wrap(c.text, c.chars*Advance, 1); !slices.Equal(got, c.want) {
			t.Errorf("Wrap(%q, %d chars) = %q, want %q", c.text, c.chars, got, c.want)
		}
	}
}

func TestWrapLines(t *testing.T) {
	var f Font
	lines := f.WrapLines("one two three four five six seven", 10*Advance, 1, 2)
	if len(lines) != 2 || lines[1][len(lines[1])-3:] != "..." {
		t.Fatalf("wrap lines = %q", lines)
	}
	if got := f.WrapLines("short", 60, 1, 2); len(got) != 1 || got[0] != "short" {
		t.Fatalf("short = %q", got)
	}
	if got := f.WrapLines("anything", 60, 1, 0); len(got) != 0 {
		t.Fatalf("zero lines = %q", got)
	}
}

func TestRedTextToken(t *testing.T) {
	if Palette.RedText.R != 240 || Palette.RedText.G != 128 || Palette.RedText.B != 112 || Palette.ChipBorder != Palette.Parchment {
		t.Fatalf("tokens = %v %v", Palette.RedText, Palette.ChipBorder)
	}
}

func TestScroll(t *testing.T) {
	s := Scroll{}.By(5, 20, 10)
	if s.Offset != 5 {
		t.Fatalf("offset = %d", s.Offset)
	}
	if s.By(50, 20, 10).Offset != 10 || s.By(-50, 20, 10).Offset != 0 {
		t.Fatal("clamp wrong")
	}
	if (Scroll{Offset: 9}).Clamp(5, 10).Offset != 0 {
		t.Fatal("short content should clamp to zero")
	}
	lines := []string{"a", "b", "c", "d"}
	if got := (Scroll{Offset: 2}).Window(lines, 5); len(got) != 2 || got[0] != "c" {
		t.Fatalf("window = %q", got)
	}
	if VisibleLines(110, 1) != 10 || VisibleLines(1, 1) != 1 {
		t.Fatal("visible lines wrong")
	}
}

func TestRectHelpers(t *testing.T) {
	r := Rect{10, 10, 20, 20}
	if !r.Contains(10, 10) || r.Contains(30, 30) {
		t.Fatal("contains wrong")
	}
	if r.Inset(2) != (Rect{12, 12, 16, 16}) || r.Offset(1, 2) != (Rect{11, 12, 20, 20}) {
		t.Fatal("inset/offset wrong")
	}
	if (Rect{0, 0, 2, 2}).Inset(5).W != 0 {
		t.Fatal("inset should not go negative")
	}
	if lighten(Palette.Ink, 300).R != 255 || Tint(Palette.Ink, 9).A != 9 {
		t.Fatal("color helpers wrong")
	}
}

func TestAClippedLastLineEndsOnAWholeWord(t *testing.T) {
	var f Font
	lines := f.WrapLines("aaaa bbbb cccc dddd eeee ffff gggg", 14*Advance, 1, 2)
	if got := lines[len(lines)-1]; got != "dddd eeee..." {
		t.Fatalf("the last line is %q, want it cut after a whole word", got)
	}
}

func TestALongURLWrapsAtItsSlashesNotInsideAWord(t *testing.T) {
	var f Font
	url := "https://github.com/kelseyhightower/kubernetes-the-hard-way/blob/master/docs/08-bootstrapping-kubernetes-controllers.md"
	lines := f.Wrap(url, 44*Advance, 1)
	if strings.Join(lines, "") != url {
		t.Fatalf("the URL lost characters: %q", lines)
	}
	for _, l := range lines[:len(lines)-1] {
		if !strings.HasSuffix(l, "/") {
			t.Errorf("the URL breaks inside a word: %q", lines)
		}
	}
	if long := f.Wrap("abcdefghijkl", 5*Advance, 1); len(long) != 3 || long[0] != "abcde" {
		t.Fatalf("a word with no slash = %q", long)
	}
}
