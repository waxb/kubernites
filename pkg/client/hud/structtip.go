package hud

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	tipMark        = "\x01"
	tipRowSep      = "\x02"
	tipTitleK      = "T"
	tipRowK        = "R"
	tipGoldK       = "G"
	tipDimK        = "D"
	tipLabelW      = 6
	structTipWidth = 340
)

type TipPart struct{ Kind, A, B string }

func TipTitle(text string) TipPart       { return TipPart{Kind: tipTitleK, A: text} }
func TipRow(label, value string) TipPart { return TipPart{Kind: tipRowK, A: label, B: value} }
func TipGold(text string) TipPart        { return TipPart{Kind: tipGoldK, A: text} }
func TipDim(text string) TipPart         { return TipPart{Kind: tipDimK, A: text} }

func StructTip(parts ...TipPart) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, p.Kind+p.A+tipRowSep+p.B)
	}
	return tipMark + strings.Join(out, "\n")
}

func TipText(tip string) string {
	rest, structured := strings.CutPrefix(tip, tipMark)
	if !structured {
		return ui.Name(tip)
	}
	lines := []string{}
	for _, raw := range strings.Split(rest, "\n") {
		if raw == "" {
			continue
		}
		a, b, _ := strings.Cut(raw[1:], tipRowSep)
		lines = append(lines, strings.TrimSpace(ui.Name(a)+" "+ui.Name(b)))
	}
	return strings.Join(lines, "\n")
}

type tipStyle struct {
	clr    color.RGBA
	label  int
	labelC color.RGBA
	plain  bool
}

func (h *HUD) layoutTip(tip string) ([]string, []tipStyle) {
	rest, structured := strings.CutPrefix(tip, tipMark)
	if !structured {
		lines := h.Font.Wrap(tip, TipWidth, 1)
		styles := make([]tipStyle, len(lines))
		for i, l := range lines {
			styles[i] = tipStyle{clr: ui.Palette.Cream, plain: true}
			if strings.HasPrefix(l, "+") {
				styles[i].clr = ui.Palette.GoldBright
			}
		}
		return lines, styles
	}
	parts := []TipPart{}
	labelW := 0
	for _, raw := range strings.Split(rest, "\n") {
		if raw == "" {
			continue
		}
		a, b, _ := strings.Cut(raw[1:], tipRowSep)
		a, b = ui.Name(a), ui.Name(b)
		p := TipPart{Kind: raw[:1], A: a, B: b}
		parts = append(parts, p)
		if p.Kind == tipRowK {
			labelW = max(labelW, (Literal.Width(p.A, 1)+1)/ui.Advance+1)
		}
	}
	labelW = max(labelW, tipLabelW)
	chars := ui.MaxChars(structTipWidth, 1)
	lines, styles := []string{}, []tipStyle{}
	add := func(text string, st tipStyle) {
		lines = append(lines, text)
		styles = append(styles, st)
	}
	for _, p := range parts {
		switch p.Kind {
		case tipRowK:
			label := p.A
			pad := strings.Repeat(" ", max(labelW-len([]rune(label)), 0))
			for i, row := range Literal.Wrap(p.B, structTipWidth-labelW*ui.Advance, 1) {
				if i == 0 {
					add(label+pad+row, tipStyle{clr: ui.Palette.Cream, label: labelW, labelC: ui.Palette.TextDim})
					continue
				}
				add(strings.Repeat(" ", labelW)+row, tipStyle{clr: ui.Palette.Cream, label: labelW, labelC: ui.Palette.TextDim})
			}
		default:
			clr := map[string]color.RGBA{tipTitleK: ui.Palette.GoldBright, tipGoldK: ui.Palette.Gold, tipDimK: ui.Palette.TextDim}[p.Kind]
			for _, row := range Literal.Wrap(p.A, chars*ui.Advance, 1) {
				add(row, tipStyle{clr: clr})
			}
		}
	}
	return lines, styles
}

func (h *HUD) tipSize(tip string) []string {
	lines, _ := h.layoutTip(tip)
	return lines
}

func (h *HUD) drawTipLines(dst *ebiten.Image, r ui.Rect, lines []string, styles []tipStyle) {
	for i, l := range lines {
		st := styles[i]
		y := r.Y + 4 + i*ui.LineHeight
		if st.plain {
			h.Font.Draw(dst, l, r.X+5, y, 1, st.clr)
			continue
		}
		if st.label > 0 && len([]rune(l)) >= st.label {
			runes := []rune(l)
			Literal.Draw(dst, string(runes[:st.label]), r.X+5, y, 1, st.labelC)
			Literal.Draw(dst, string(runes[st.label:]), r.X+5+st.label*ui.Advance, y, 1, st.clr)
			continue
		}
		Literal.Draw(dst, l, r.X+5, y, 1, st.clr)
	}
}

func (h *HUD) tipFont(tip string) ui.Font {
	if strings.HasPrefix(tip, tipMark) {
		return Literal
	}
	return h.Font
}
