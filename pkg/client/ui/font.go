package ui

import (
	"image/color"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	GlyphWidth  = 5
	GlyphHeight = 9
	CapHeight   = 7
	Advance     = GlyphWidth + 1
	LineHeight  = 11
)

type Font struct {
	Literal bool
}

func glyphFor(r rune) [GlyphHeight]byte {
	if g, ok := glyphs[r]; ok {
		return g
	}
	if g, ok := glyphs[unicode.ToUpper(r)]; ok {
		return g
	}
	return glyphs['?']
}

func (f Font) Measure(text string, scale int) (int, int) {
	text = f.named(text)
	scale = max(scale, 1)
	width := 0
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if n := cells(line); n > 0 && n*Advance-1 > width {
			width = n*Advance - 1
		}
	}
	return width * scale, len(lines) * LineHeight * scale
}

func (f Font) Width(text string, scale int) int {
	w, _ := f.Measure(text, scale)
	return w
}

func (f Font) Span(text string, gap int) int {
	w := f.Width(text, 1)
	if w > 0 {
		w++
	}
	return w + gap*Advance
}

func (f Font) Draw(dst *ebiten.Image, text string, x, y, scale int, clr color.Color) {
	text = f.named(text)
	record(dst, text, x, y, scale)
	drawText(dst, text, x, y, scale, clr)
}

func drawText(dst *ebiten.Image, text string, x, y, scale int, clr color.Color) {
	drawRunes(dst, text, x, y, scale, clr, nil)
}

func drawRunes(dst *ebiten.Image, text string, x, y, scale int, clr, silhouette color.Color) {
	scale = max(scale, 1)
	originX := x
	runes := []rune(text)
	for i, r := range runes {
		switch {
		case r == '\n':
			x = originX
			y += LineHeight * scale
			continue
		case r == gapRune:
			continue
		case iconStart(r) && i+1 < len(runes) && runes[i+1] == r+1:
			drawIcon(dst, r, x, y, scale, silhouette)
		case iconRune(r):
		default:
			drawGlyph(dst, glyphFor(r), x, y, scale, clr)
		}
		x += Advance * scale
	}
}

func drawGlyph(dst *ebiten.Image, pattern [GlyphHeight]byte, x, y, scale int, clr color.Color) {
	for row, bits := range pattern {
		for col := 0; col < GlyphWidth; col++ {
			if bits&(1<<uint(GlyphWidth-1-col)) == 0 {
				continue
			}
			vector.FillRect(dst, float32(x+col*scale), float32(y+row*scale), float32(scale), float32(scale), clr, false)
		}
	}
}

func (f Font) DrawShadow(dst *ebiten.Image, text string, x, y, scale int, clr color.Color) {
	f.DrawShadowAs(dst, text, x, y, scale, clr, Palette.Shadow)
}

func (f Font) DrawShadowAs(dst *ebiten.Image, text string, x, y, scale int, clr, shadow color.Color) {
	text = f.named(text)
	record(dst, text, x, y, scale)
	drawRunes(dst, text, x+scale, y+scale, scale, shadow, shadow)
	drawText(dst, text, x, y, scale, clr)
}

func (f Font) DrawCentered(dst *ebiten.Image, text string, r Rect, scale int, clr color.Color) {
	w, _ := f.Measure(text, scale)
	f.DrawShadow(dst, text, r.X+(r.W-w)/2, r.Y+(r.H-CapHeight*scale)/2, scale, clr)
}

func MaxChars(width, scale int) int {
	scale = max(scale, 1)
	return max((width/scale+1)/Advance, 1)
}

func Clip(text string, maxChars int) string {
	return Font{}.Clip(text, maxChars)
}

func (f Font) Clip(text string, maxChars int) string {
	runes := []rune(f.named(text))
	if cells(string(runes)) <= maxChars {
		return f.settled(string(runes))
	}
	if maxChars < 4 {
		return f.settled(string(runes[:cutCells(runes, maxChars)]))
	}
	return f.settled(ellipsis(string(runes[:cutCells(runes, maxChars-3)])))
}

func ClipWords(text string, maxChars int) string {
	f := Font{}
	runes := []rune(f.named(text))
	if cells(string(runes)) <= maxChars || maxChars < 4 {
		return f.Clip(text, maxChars)
	}
	head := runes[:cutCells(runes, maxChars-3)]
	if len(head) < len(runes) && runes[len(head)] != ' ' {
		if space := strings.LastIndex(string(head), " "); space >= len(string(head))/2 {
			head = []rune(string(head)[:space])
		}
	}
	return f.settled(ellipsis(string(head)))
}

func ellipsis(head string) string {
	word := strings.TrimRight(head, " ")
	return strings.TrimRight(word, clipTrim) + head[len(word):] + "..."
}

func (f Font) WrapLines(text string, maxWidth, scale, maxLines int) []string {
	lines := f.Wrap(text, maxWidth, scale)
	if maxLines <= 0 {
		return nil
	}
	if len(lines) <= maxLines {
		return lines
	}
	out := append([]string(nil), lines[:maxLines]...)
	last := []rune(out[maxLines-1])
	limit := MaxChars(maxWidth, scale) - 3
	if cells(string(last)) > limit {
		last = last[:cutCells(last, limit)]
		if cut := strings.LastIndex(string(last), " "); cut > 0 && last[len(last)-1] != ' ' {
			last = []rune(string(last)[:cut])
		}
	}
	out[maxLines-1] = f.settled(ellipsis(string(last)))
	return out
}

func (f Font) Wrap(text string, maxWidth, scale int) []string {
	text = f.named(text)
	maxChars := MaxChars(maxWidth, scale)
	out := []string{}
	for _, paragraph := range strings.Split(text, "\n") {
		for _, line := range wrapParagraph(paragraph, maxChars) {
			out = append(out, f.settled(line))
		}
	}
	return out
}

const clipTrim = ".,;:"

var phraseGap = regexp.MustCompile(`\s{2,}`)

func wordBreak(runes []rune, cut, maxChars int, fill bool) int {
	if at, ok := breakAfter(runes, cut, maxChars, fill, func(i int) bool { return runes[i] == ',' }); ok {
		return at
	}
	head := string(runes[:cut])
	if slash := strings.LastIndex(head, "/"); slash > 0 && slash+1 < len(head) {
		return len([]rune(head[:slash+1]))
	}
	if at, ok := breakAfter(runes, cut, maxChars, fill, func(i int) bool {
		return runes[i] == '-' || runes[i] == ':' || runes[i] == '.' && !(unicode.IsDigit(runes[i-1]) && unicode.IsDigit(runes[i+1]))
	}); ok {
		return at
	}
	return cut
}

func breakAfter(runes []rune, cut, maxChars int, fill bool, at func(int) bool) (int, bool) {
	var breaks []int
	for i := 1; i < cut; i++ {
		if at(i) {
			breaks = append(breaks, i+1)
		}
	}
	if len(breaks) == 0 {
		return 0, false
	}
	fitting := slices.DeleteFunc(slices.Clone(breaks), func(at int) bool { return cells(string(runes[at:])) > maxChars })
	if len(fitting) == 0 {
		return breaks[len(breaks)-1], true
	}
	if fill {
		return slices.Max(fitting), true
	}
	half := len(runes) / 2
	return slices.MinFunc(fitting, func(a, b int) int { return max(a-half, half-a) - max(b-half, half-b) }), true
}

func wrapParagraph(paragraph string, maxChars int) []string {
	out, line := []string{}, ""
	for _, phrase := range phraseGap.Split(paragraph, -1) {
		phrase = strings.Join(strings.Fields(phrase), " ")
		if phrase == "" {
			continue
		}
		if line != "" && cells(line+" "+phrase) <= maxChars {
			line += " " + phrase
			continue
		}
		if line != "" {
			out = append(out, line)
		}
		rows := wrapWords(phrase, maxChars)
		out, line = append(out, rows[:len(rows)-1]...), rows[len(rows)-1]
	}
	if line != "" || len(out) == 0 {
		out = append(out, line)
	}
	return out
}

func wrapWords(paragraph string, maxChars int) []string {
	words := strings.Fields(paragraph)
	if len(words) == 0 {
		return []string{""}
	}
	out := []string{}
	line := ""
	for i, word := range words {
		for cells(word) > maxChars {
			if line != "" {
				out = append(out, line)
				line = ""
			}
			runes := []rune(word)
			cut := wordBreak(runes, max(cutCells(runes, maxChars), 1), maxChars, i < len(words)-1)
			out = append(out, string(runes[:cut]))
			word = string(runes[cut:])
		}
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if cells(candidate) > maxChars {
			out = append(out, line)
			line = word
			continue
		}
		line = candidate
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
