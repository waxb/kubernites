package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	LensNone     = ""
	LensLabor    = "labor"
	LensUnhoused = "unhoused"
	LensMidden   = "midden"
	LensCodex    = "codex"
	LensHeraldry = "heraldry"
	lensPrefix   = "lens"
	legendH      = 14
)

var LensOrder = []string{LensLabor, LensUnhoused, LensMidden, LensCodex, LensHeraldry}

type lensInfo struct {
	label, icon, legend string
}

var lensInfos = map[string]lensInfo{
	LensLabor:    {"FOOD", "icon/overlay-labor", "Food heat: food eaten on each Keep (pledged while the Augur is blind)"},
	LensUnhoused: {"UNHOUSED", "icon/overlay-unhoused", "Unhoused huts ringed; Keeps tinted by the food already pledged"},
	LensMidden:   {"CELLARS", "icon/overlay-midden", "The cellars: how full each Keep's ground is (ephemeral storage)"},
	LensCodex:    {"CODEX", "icon/codex", "The Codex: its size against its quota"},
	LensHeraldry: {"HERALDRY", "icon/overlay-heraldry", "The Heraldry view: each fief's ground and huts in its colour, and every Keep flies a pennant with the crest of each fief that lives on it"},
}

func LensAvailable(mode string, f Frame) bool {
	if mode == LensHeraldry {
		return f.Revealed(heraldryReveal)
	}
	if !f.Revealed(LensReveal) {
		return false
	}
	switch mode {
	case LensLabor:
		return len(f.Metrics.Nodes) > 0
	case LensUnhoused:
		return f.Revealed("summon")
	case LensMidden:
		return len(middenNodes(f)) > 0
	case LensCodex:
		return f.Caesar.Codex.Known()
	}
	return false
}

func NextLens(current string, f Frame) string {
	start := -1
	for i, m := range LensOrder {
		if m == current {
			start = i
		}
	}
	for i := start + 1; i < len(LensOrder); i++ {
		if LensAvailable(LensOrder[i], f) {
			return LensOrder[i]
		}
	}
	return LensNone
}

func (h *HUD) Lens() string { return h.lens }

func (h *HUD) SetLens(mode string) { h.lens = mode }

func (h *HUD) cycleLens(f Frame) []Intent {
	h.lens = NextLens(h.lens, f)
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) lensButtons(f Frame, modes []string) []Button {
	out := []Button{}
	for _, m := range modes {
		if !LensAvailable(m, f) {
			continue
		}
		info := lensInfos[m]
		out = append(out, Button{ID: lensPrefix + ":" + m, Rect: FootRect(len(out)), Label: "MAP: " + info.label, Tip: info.legend + " [O]", Selected: h.lens == m})
	}
	return out
}

func (h *HUD) pressLens(mode string, f Frame) []Intent {
	if mode == lensCycle {
		return h.cycleLens(f)
	}
	if h.lens == mode {
		h.lens = LensNone
	} else {
		h.lens = mode
	}
	h.OverlayID = OverlayNone
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func legendText(mode string) string {
	return "MAP: " + lensInfos[mode].label + "   O next, Esc clears"
}

func LegendRect(mode string) ui.Rect {
	var font ui.Font
	w, _ := font.Measure(legendText(mode), 1)
	w += 16 + 22
	return ui.Rect{X: Width/2 - w/2, Y: ConsoleY - legendH - 8, W: w, H: legendH + 2}
}

func (h *HUD) LegendShown(f Frame) bool {
	return h.lens != LensNone && h.OverlayID == OverlayNone && f.Placing == ""
}

func (h *HUD) legendButtons(f Frame) []Button {
	if !h.LegendShown(f) {
		return nil
	}
	return []Button{{ID: lensPrefix + ":" + h.lens, Rect: LegendRect(h.lens), Tip: lensInfos[h.lens].legend + ". Click to clear."}}
}

func (h *HUD) drawLegend(dst *ebiten.Image, f Frame) {
	if !h.LegendShown(f) {
		return
	}
	r := LegendRect(h.lens)
	ui.Fill(dst, r.Offset(1, 2), ui.Palette.Shadow)
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 230))
	ui.Stroke(dst, r, ui.Palette.Gold)
	h.icon(dst, lensInfos[h.lens].icon, r.X+3, r.Y)
	h.Font.Draw(dst, legendText(h.lens), r.X+22, r.Y+5, 1, ui.Palette.Cream)
}

func lensFromID(id string) (string, bool) {
	mode, ok := strings.CutPrefix(id, lensPrefix+":")
	return mode, ok
}
