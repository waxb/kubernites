package hud

import (
	"fmt"
	"image/color"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/termview"
	"github.com/waxb/kubernites/pkg/client/ui"
)

type StoneScreen interface {
	Live() bool
	Resize(cols, rows int)
	Row(y int) []termview.Cell
	Cursor() (int, int, bool)
	Key(termview.Key)
	Type(string)
}

type StoneSession struct {
	Verb      string
	Kind      string
	Namespace string
	Name      string
	Container string
	Command   []string
	TTY       bool
	Opened    bool
	Ended     string
	Code      int
	Err       string
}

const (
	SessionExited = "exited"
	SessionLeft   = "left"
	SessionClosed = "closed"
	SessionFailed = "failed"
)

type screened interface {
	Screen() StoneScreen
}

const SessionHeldLine = "The session ended: keys typed now are dropped. Enter returns to the Stone's prompt."

func (r *stoneRun) screen() StoneScreen {
	if r == nil {
		return nil
	}
	s, ok := r.job.(screened)
	if !ok {
		return nil
	}
	if sc := s.Screen(); sc != nil && sc.Live() {
		return sc
	}
	return nil
}

func (h *HUD) StoneTerminal() bool {
	return h.kube.screen() != nil
}

func (h *HUD) TerminalCells() (int, int) {
	out := h.stoneOutput()
	return max((out.W-8)/ui.Advance, 1), max((out.H-6)/ui.LineHeight, 1)
}

func (h *HUD) holdTerminal() {
	sc := h.kube.screen()
	if sc == nil {
		return
	}
	h.OverlayID = OverlayStone
	h.Stone.away = false
	sc.Resize(h.TerminalCells())
}

var terminalKeyNames = map[ebiten.Key]string{
	ebiten.KeyEnter: "enter", ebiten.KeyNumpadEnter: "enter", ebiten.KeyTab: "tab", ebiten.KeyBackspace: "backspace", ebiten.KeyEscape: "escape",
	ebiten.KeyArrowUp: "up", ebiten.KeyArrowDown: "down", ebiten.KeyArrowLeft: "left", ebiten.KeyArrowRight: "right",
	ebiten.KeyHome: "home", ebiten.KeyEnd: "end", ebiten.KeyPageUp: "pgup", ebiten.KeyPageDown: "pgdown",
	ebiten.KeyInsert: "insert", ebiten.KeyDelete: "delete",
	ebiten.KeyF1: "f1", ebiten.KeyF2: "f2", ebiten.KeyF3: "f3", ebiten.KeyF4: "f4", ebiten.KeyF5: "f5", ebiten.KeyF6: "f6",
	ebiten.KeyF7: "f7", ebiten.KeyF8: "f8", ebiten.KeyF9: "f9", ebiten.KeyF10: "f10", ebiten.KeyF11: "f11", ebiten.KeyF12: "f12",
}

var controlRunes = map[ebiten.Key]rune{
	ebiten.KeySpace: ' ', ebiten.KeyBracketLeft: '[', ebiten.KeyBackslash: '\\', ebiten.KeyBracketRight: ']',
	ebiten.KeyDigit6: '^', ebiten.KeyMinus: '_',
}

var labelledControl = map[rune]rune{' ': ' ', '[': '[', '\\': '\\', ']': ']', '^': '^', '_': '_', '6': '^', '-': '_'}

var layoutKeys = []ebiten.Key{
	ebiten.KeyA, ebiten.KeyB, ebiten.KeyC, ebiten.KeyD, ebiten.KeyE, ebiten.KeyF, ebiten.KeyG, ebiten.KeyH, ebiten.KeyI,
	ebiten.KeyJ, ebiten.KeyK, ebiten.KeyL, ebiten.KeyM, ebiten.KeyN, ebiten.KeyO, ebiten.KeyP, ebiten.KeyQ, ebiten.KeyR,
	ebiten.KeyS, ebiten.KeyT, ebiten.KeyU, ebiten.KeyV, ebiten.KeyW, ebiten.KeyX, ebiten.KeyY, ebiten.KeyZ,
	ebiten.KeyDigit0, ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4, ebiten.KeyDigit5,
	ebiten.KeyDigit6, ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9, ebiten.KeyMinus, ebiten.KeyEqual,
	ebiten.KeyBracketLeft, ebiten.KeyBracketRight, ebiten.KeyBackslash, ebiten.KeySemicolon, ebiten.KeyQuote,
	ebiten.KeyComma, ebiten.KeyPeriod, ebiten.KeySlash, ebiten.KeyBackquote, ebiten.KeyIntlBackslash,
}

func (in Input) labelled(letter rune) bool {
	for _, k := range layoutKeys {
		if label := []rune(in.Layout(k)); len(label) == 1 && unicode.ToLower(label[0]) == letter {
			return true
		}
	}
	return false
}

func (in Input) ChordRune(k ebiten.Key) (rune, bool) {
	if in.Layout != nil {
		if label := []rune(in.Layout(k)); len(label) == 1 && label[0] < utf8.RuneSelf {
			r := unicode.ToLower(label[0])
			if r >= 'a' && r <= 'z' {
				return r, true
			}
			if us := rune('a' + int(k-ebiten.KeyA)); k >= ebiten.KeyA && k <= ebiten.KeyZ && !in.labelled(us) {
				return us, true
			}
			c, ok := labelledControl[r]
			return c, ok
		}
	}
	if k >= ebiten.KeyA && k <= ebiten.KeyZ {
		return rune('a' + int(k-ebiten.KeyA)), true
	}
	r, ok := controlRunes[k]
	return r, ok
}

func TerminalKeys(in Input) ([]termview.Key, string) {
	meta := in.Ctrl && !in.Control
	if meta {
		return nil, ""
	}
	var keys []termview.Key
	for _, k := range in.Keys {
		if name, ok := terminalKeyNames[k]; ok {
			keys = append(keys, termview.Key{Name: name, Ctrl: in.Control, Alt: in.Alt, Shift: in.Shift})
			continue
		}
		if !in.Control {
			continue
		}
		if r, ok := in.ChordRune(k); ok {
			keys = append(keys, termview.Key{Rune: r, Ctrl: true, Alt: in.Alt})
		}
	}
	if in.Control {
		return keys, ""
	}
	text := string(in.Chars)
	if in.Alt && text != "" {
		for _, r := range in.Chars {
			keys = append(keys, termview.Key{Rune: r, Alt: true})
		}
		text = ""
	}
	return keys, text
}

func (h *HUD) terminalKeys(sc StoneScreen, in Input) []Intent {
	keys, text := TerminalKeys(in)
	if text != "" {
		sc.Type(text)
	}
	for _, k := range keys {
		sc.Key(k)
	}
	return nil
}

var terminalInk = color.RGBA{6, 9, 11, 255}

func firstGlyph(text string) string {
	r, size := utf8.DecodeRuneInString(text)
	if size == 0 || r == utf8.RuneError && size <= 1 {
		return " "
	}
	return text[:size]
}

func (h *HUD) drawTerminal(dst *ebiten.Image, sc StoneScreen) {
	out := h.stoneOutput()
	ui.Fill(dst, out, terminalInk)
	ui.Stroke(dst, out, ui.Palette.Gold)
	cols, rows := h.TerminalCells()
	x0, y0 := out.X+4, out.Y+3
	for y := 0; y < rows; y++ {
		cells := sc.Row(y)
		py := y0 + y*ui.LineHeight
		run, runX, runClr := "", 0, color.RGBA{}
		flush := func() {
			if strings.TrimSpace(run) != "" {
				Literal.Draw(dst, run, x0+runX*ui.Advance, py, 1, runClr)
			}
			run = ""
		}
		for x := 0; x < min(cols, len(cells)); x++ {
			c := cells[x]
			fg, bg, hasBg := ui.Palette.Cream, terminalInk, c.HasBg
			if c.HasFg {
				fg = c.Fg
			}
			if c.HasBg {
				bg = c.Bg
			}
			if c.Reverse {
				fg, bg, hasBg = bg, fg, true
			}
			if c.Faint {
				fg = color.RGBA{fg.R / 2, fg.G / 2, fg.B / 2, 255}
			}
			if hasBg {
				ui.Fill(dst, ui.Rect{X: x0 + x*ui.Advance - 1, Y: py - 1, W: ui.Advance, H: ui.LineHeight}, bg)
			}
			if c.Underline {
				ui.Fill(dst, ui.Rect{X: x0 + x*ui.Advance - 1, Y: py + ui.GlyphHeight, W: ui.Advance, H: 1}, fg)
			}
			if run == "" || fg != runClr {
				flush()
				runX, runClr = x, fg
			}
			run += firstGlyph(c.Text)
		}
		flush()
	}
	if cx, cy, shown := sc.Cursor(); shown && (h.Reduce || int(h.Now*2)%2 == 0) && cx < cols && cy < rows {
		ui.Fill(dst, ui.Rect{X: x0 + cx*ui.Advance - 1, Y: y0 + cy*ui.LineHeight + ui.GlyphHeight - 1, W: ui.Advance, H: 2}, ui.Palette.GoldBright)
	}
	hint := ui.Bridge("Every key goes to the worker: ctrl-p ctrl-q leaves", "Every key goes to the container: ctrl-p ctrl-q leaves")
	h.Font.Draw(dst, hint, out.X+4, out.Y+out.H+4, 1, ui.Palette.TextDim)
}

func SessionEndLine(s StoneSession) (string, bool) {
	if !s.Opened && s.Ended != SessionLeft {
		return "", false
	}
	object := s.Name
	if s.Namespace != "" {
		object = s.Namespace + "/" + s.Name
	}
	kind := map[string][2]string{"Pod": {"hut", "pod"}, "Service": {"depot", "service"}, "Node": {"Keep", "node"}}[s.Kind]
	if kind[0] == "" {
		kind = [2]string{"hut", "pod"}
	}
	where := ui.Bridge(kind[0]+" "+object, kind[1]+" "+object)
	if s.Container != "" {
		where += ui.Bridge(", worker ", ", container ") + s.Container
	}
	switch s.Ended {
	case SessionLeft:
		return ui.Bridge("Left "+where+" with ctrl-p ctrl-q; the worker carries on.", "Left "+where+" with ctrl-p ctrl-q; the session ended, the container runs on."), true
	case SessionClosed:
		return ui.Bridge("The session with "+where+" closed with the realm.", "The session to "+where+" closed with the realm."), true
	case SessionExited:
		if s.Code != 0 {
			return fmt.Sprintf(ui.Bridge("The session with %s ended: its command exited with code %d.", "The session to %s ended: the command exited with code %d."), where, s.Code), true
		}
		return ui.Bridge("The session with "+where+" ended: its command exited.", "The session to "+where+" ended: the command exited."), true
	case SessionFailed:
		return ui.Bridge("The session with "+where+" broke off.", "The session to "+where+" broke off."), true
	}
	return "", false
}

func (h *HUD) StoneScreenText() []string {
	sc := h.kube.screen()
	if sc == nil {
		return nil
	}
	cols, rows := h.TerminalCells()
	out := make([]string, rows)
	for y := range rows {
		line := ""
		for x, c := range sc.Row(y) {
			if x >= cols {
				break
			}
			line += c.Text
		}
		out[y] = line
	}
	return out
}
