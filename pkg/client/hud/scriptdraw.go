package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const placeholder = "# write a decree"

var Literal = ui.Font{Literal: true}

func (h *HUD) drawScriptorium(dst *ebiten.Image) {
	h.Script.Sync(ui.StoryNaming())
	ui.DarkPanel(dst, Overlay)
	h.header(dst, Overlay, "THE SCRIPTORIUM")
	rail := scriptRail()
	if h.liveEdit != nil {
		h.Font.Draw(dst, ui.Bridge("a living decree, read from the realm", "the live object: kubectl edit, sent with server-side apply"), Overlay.X+210, Overlay.Y+9, 1, ui.Palette.TextDim)
		ui.Well(dst, rail)
		h.drawLiveEditRail(dst, rail)
		editor := ScriptEditor()
		h.drawEditor(dst, editor)
		h.drawLiveEditAsk(dst, editor)
		h.drawLiveEditFooter(dst, scriptFooter())
		h.DrawOverlayButtons(dst, h.scriptButtons())
		return
	}
	h.Font.Draw(dst, "decrees are YAML: kubectl apply", Overlay.X+210, Overlay.Y+9, 1, ui.Palette.TextDim)
	ui.Well(dst, rail)
	h.Font.Draw(dst, "TEMPLATES", rail.X+6, rail.Y+5, 1, ui.Palette.TextDim)
	editor, beside := h.Script.EditorRect()
	h.drawEditor(dst, editor)
	h.drawBeside(dst, beside)
	h.drawScriptFooter(dst, scriptFooter())
	h.DrawOverlayButtons(dst, h.scriptButtons())
}

func (h *HUD) drawEditor(dst *ebiten.Image, r ui.Rect) {
	e := &h.Script.Editor
	ui.Fill(dst, r, consoleInk)
	ui.Stroke(dst, r, ui.Palette.BlueDark)
	ui.Fill(dst, ui.Rect{X: r.X + 1, Y: r.Y + 1, W: gutterW, H: r.H - 2}, ui.Palette.ConsoleEdge)
	rows, cols := scriptVisible(), h.Script.columns()
	e.Follow(rows, cols)
	x := r.X + gutterW + 5
	if e.Empty() && !e.Dirty {
		h.Font.Draw(dst, placeholder, x, r.Y+4, 1, ui.Palette.TextDim)
	}
	for i := 0; i < rows && e.Top+i < len(e.Lines); i++ {
		row := e.Top + i
		y := r.Y + 4 + i*ui.LineHeight
		h.drawGutter(dst, r, row, y)
		text := []rune(e.Lines[row])
		visible := string(text[min(e.Left, len(text)):min(len(text), e.Left+cols)])
		Literal.Draw(dst, visible, x, y, 1, yamlColor(e.Lines[row]))
	}
	if h.Reduce || int(h.Now*2)%2 == 0 {
		cy := r.Y + 3 + (e.Row-e.Top)*ui.LineHeight
		ui.Fill(dst, ui.Rect{X: x + (e.Col-e.Left)*ui.Advance - 1, Y: cy, W: 1, H: 9}, ui.Palette.GoldBright)
	}
	h.drawScrollbar(dst, r, len(e.Lines), rows, e.Top)
}

func (h *HUD) drawGutter(dst *ebiten.Image, r ui.Rect, row, y int) {
	clr := ui.Palette.TextDim
	if h.Script.ErrLine == row+1 {
		ui.Fill(dst, ui.Rect{X: r.X + 1, Y: y - 1, W: gutterW, H: ui.LineHeight}, ui.Palette.RedDark)
		clr = ui.Palette.Text
	}
	label := fmt.Sprint(row + 1)
	h.Font.Draw(dst, label, r.X+gutterW-len(label)*ui.Advance, y, 1, clr)
}

func yamlColor(line string) color.RGBA {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "#"):
		return ui.Palette.TextDim
	case strings.HasPrefix(trimmed, "kind:") || strings.HasPrefix(trimmed, "apiVersion:") || strings.HasPrefix(trimmed, "decree:") || strings.HasPrefix(trimmed, "book:"):
		return ui.Palette.Gold
	}
	return ui.Palette.Text
}

func (h *HUD) drawScriptFooter(dst *ebiten.Image, r ui.Rect) {
	s := h.Script
	ui.Well(dst, r)
	width := ui.MaxChars(r.W-140, 1)
	switch {
	case s.Warn != "":
		h.Font.Draw(dst, ui.Clip(s.Warn+" (Y/N)", width), r.X+6, r.Y+7, 1, ui.Palette.GoldBright)
	case s.Confirm != "":
		h.Font.Draw(dst, "Replace your draft with "+s.Confirm+"? (Y/N)", r.X+6, r.Y+7, 1, ui.Palette.GoldBright)
	case len(s.Result) > 0 && !s.OK:
		h.Font.Draw(dst, ui.Clip(strings.Join(s.Result, "  "), width), r.X+6, r.Y+7, 1, ui.Palette.RedText)
	case len(s.Result) > 0 && h.Now-s.ResultAt < resultFade:
		fade := uint8(255 * min(1, (resultFade-(h.Now-s.ResultAt))/1.5))
		h.Font.Draw(dst, ui.Clip(strings.Join(s.Result, "  "), width), r.X+6, r.Y+7, 1, ui.Tint(ui.Palette.Green, fade))
	default:
		h.Font.Draw(dst, "Tab indents. Ctrl+Enter applies. Esc closes.", r.X+6, r.Y+7, 1, ui.Palette.TextDim)
	}
}
