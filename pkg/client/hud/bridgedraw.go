package hud

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	bridgeInset     = 10
	bridgeLabelW    = 9
	bridgeYAMLRows  = 12
	bridgeRuleRows  = 3
	bridgeBendRows  = 3
	bridgeFooterH   = 24
	bridgeSection   = 4
	bridgeNoYAML    = "Nothing in the YAML changes: this command only reads what the API server holds."
	bridgeNextTime  = "The next time this comes up, the Stone offers the command written out for you."
	bridgeNextK8s   = "The next time this comes up, the console offers the command written out for you."
	bridgeYoursNote = "(your own command)"
	bridgeExample   = "(an example: do it yourself and your own command shows here)"
)

type tokenRow struct {
	story, kube []lexicon.CommandToken
}

func tokenRows(tokens []lexicon.CommandToken, cells int) []tokenRow {
	out := []tokenRow{}
	cur, used := tokenRow{}, 0
	for _, t := range tokens {
		w := tokenWidth(t) + 1
		if used+w > cells && used > 0 {
			out = append(out, cur)
			cur, used = tokenRow{}, 0
		}
		cur.story, cur.kube = append(cur.story, t), append(cur.kube, t)
		used += w
	}
	if used > 0 {
		out = append(out, cur)
	}
	return out
}

func tokenWidth(t lexicon.CommandToken) int {
	return max(len([]rune(t.Story)), len([]rune(t.Kube)))
}

func bridgeBody() ui.Rect {
	return ui.Rect{X: Overlay.X + bridgeInset, Y: Overlay.Y + 44, W: Overlay.W - 2*bridgeInset, H: Overlay.H - 44 - bridgeFooterH}
}

func bridgeFooter() ui.Rect {
	return ui.Rect{X: Overlay.X + bridgeInset, Y: Overlay.Y + Overlay.H - bridgeFooterH, W: Overlay.W - 2*bridgeInset, H: bridgeFooterH - 4}
}

func (h *HUD) bridgeButtons(f Frame) []Button {
	out := []Button{closeButton(Overlay)}
	v := h.Kube.bridge.view
	if v.Page != "" && bookMet(f, v.Page) {
		foot := bridgeFooter()
		out = append(out, Button{ID: bridgeBook, Rect: ui.Rect{X: foot.X + foot.W - cardAlmanacW, Y: foot.Y + 2, W: cardAlmanacW, H: 16}, Label: AlmanacLabel,
			Tip: "Open this page in the Almanac: its Kubernetes side keeps this block"})
	}
	return out
}

func (h *HUD) drawBridge(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, Overlay)
	v := h.Kube.bridge.view
	Literal.DrawShadow(dst, bridgeLabel, Overlay.X+bridgeInset, Overlay.Y+6, 2, ui.Palette.Gold)
	if ui.Naming() {
		Literal.Draw(dst, v.K8sTitle, Overlay.X+bridgeInset, Overlay.Y+28, 1, ui.Palette.GoldBright)
	} else {
		title := h.Font.Width(v.Title, 1)
		h.Font.Draw(dst, v.Title, Overlay.X+bridgeInset, Overlay.Y+28, 1, ui.Palette.GoldBright)
		Literal.Draw(dst, "is "+v.K8sTitle+" in Kubernetes", Overlay.X+bridgeInset+title+ui.Advance, Overlay.Y+28, 1, ui.Palette.Blue)
	}
	body := bridgeBody()
	y := h.drawBridgeCommand(dst, v, body, body.Y)
	rule := Literal.WrapLines(v.Rule, body.W, 1, bridgeRuleRows)
	bends := bridgeBends(v, body.W)
	tail := sectionHeight(len(rule)) + ui.LineHeight
	if len(bends) > 0 {
		tail += sectionHeight(len(bends))
	}
	y = h.drawBridgeYAML(dst, v, body, y, body.Y+body.H-tail)
	y = drawSection(dst, "THE RULE", rule, body.X, y, ui.Palette.Cream)
	if len(bends) > 0 {
		y = drawSection(dst, "WHERE THE STORY BENDS", bends, body.X, y, ui.Palette.Parchment)
	}
	Literal.Draw(dst, bridgeNext(), body.X, y, 1, ui.Palette.TextDim)
	Literal.Draw(dst, "READ MORE AT KUBERNETES.IO", body.X, bridgeFooter().Y-ui.LineHeight+4, 1, ui.Palette.Gold)
	foot := bridgeFooter()
	buttons := h.bridgeButtons(f)
	room := foot.W
	if len(buttons) > 1 {
		room -= cardAlmanacW + 8
	}
	Literal.Draw(dst, Literal.Clip(v.Doc, ui.MaxChars(room, 1)), foot.X, foot.Y+6, 1, ui.Palette.Blue)
	h.DrawOverlayButtons(dst, buttons)
}

func bridgeBends(v view.BridgeView, w int) []string {
	out := []string{}
	for _, b := range v.Bends {
		text := b.Bends
		if ui.Naming() {
			text = ui.Name(text)
		}
		out = append(out, Literal.WrapLines(text, w, 1, bridgeBendRows)...)
	}
	return out
}

func bridgeNext() string { return ui.Bridge(bridgeNextTime, bridgeNextK8s) }

func commandLabelsK8s() (string, string) { return "STONE", "KUBECTL" }

func sectionHeight(rows int) int {
	return ui.LineHeight + rows*ui.LineHeight + bridgeSection
}

func drawSection(dst *ebiten.Image, title string, rows []string, x, y int, clr color.RGBA) int {
	Literal.Draw(dst, title, x, y, 1, ui.Palette.Gold)
	y += ui.LineHeight
	for _, row := range rows {
		Literal.Draw(dst, row, x, y, 1, clr)
		y += ui.LineHeight
	}
	return y + bridgeSection
}

func (h *HUD) drawBridgeCommand(dst *ebiten.Image, v view.BridgeView, body ui.Rect, y int) int {
	note := bridgeExample
	if v.Yours {
		note = bridgeYoursNote
	}
	Literal.Draw(dst, "THE COMMAND", body.X, y, 1, ui.Palette.Gold)
	Literal.Draw(dst, note, body.X+12*ui.Advance, y, 1, ui.Palette.TextDim)
	y += ui.LineHeight
	cells := ui.MaxChars(body.W, 1) - bridgeLabelW
	if ui.Naming() {
		_, kubeLabel := commandLabelsK8s()
		for _, row := range tokenRows(v.Tokens, cells) {
			Literal.Draw(dst, kubeLabel, body.X, y, 1, ui.Palette.TextDim)
			x := body.X + bridgeLabelW*ui.Advance
			for _, t := range row.story {
				Literal.Draw(dst, t.Kube, x, y, 1, ui.Palette.Blue)
				x += (tokenWidth(t) + 1) * ui.Advance
			}
			y += ui.LineHeight + 2
		}
		return y + bridgeSection
	}
	for _, row := range tokenRows(v.Tokens, cells) {
		stone, kube := commandLabelsK8s()
		Literal.Draw(dst, stone, body.X, y, 1, ui.Palette.TextDim)
		Literal.Draw(dst, kube, body.X, y+ui.LineHeight, 1, ui.Palette.TextDim)
		x := body.X + bridgeLabelW*ui.Advance
		for _, t := range row.story {
			story, kube := ui.Palette.Text, ui.Palette.TextDim
			if t.Mapped() {
				story, kube = ui.Palette.GoldBright, ui.Palette.Blue
			}
			Literal.Draw(dst, t.Story, x, y, 1, story)
			Literal.Draw(dst, t.Kube, x, y+ui.LineHeight, 1, kube)
			x += (tokenWidth(t) + 1) * ui.Advance
		}
		y += 2*ui.LineHeight + 2
	}
	return y + bridgeSection
}

func (h *HUD) drawBridgeYAML(dst *ebiten.Image, v view.BridgeView, body ui.Rect, y, limit int) int {
	if len(v.YAML) == 0 {
		return drawSection(dst, "THE YAML", []string{bridgeNoYAML}, body.X, y, ui.Palette.TextDim)
	}
	Literal.Draw(dst, v.YAMLHeading(), body.X, y, 1, ui.Palette.Gold)
	half := body.W / 2
	y += ui.LineHeight
	named := ui.Naming()
	if named {
		half = 0
	}
	if !named {
		Literal.Draw(dst, "the story's script", body.X, y, 1, ui.Palette.TextDim)
	}
	Literal.Draw(dst, "the Kubernetes YAML", body.X+half, y, 1, ui.Palette.TextDim)
	y += ui.LineHeight
	rows := min(len(v.YAML), bridgeYAMLRows, max(1, (limit-y)/ui.LineHeight))
	cells := ui.MaxChars(max(half, body.W)-8, 1)
	if !named {
		cells = ui.MaxChars(half-8, 1)
	}
	for i := range rows {
		story, kube := yamlRow(v.StoryYAML, i, rows, len(v.YAML)), yamlRow(v.YAML, i, rows, len(v.YAML))
		if !named {
			Literal.Draw(dst, Literal.Clip(story, cells), body.X, y, 1, ui.Palette.Text)
		}
		Literal.Draw(dst, Literal.Clip(kube, cells), body.X+half, y, 1, ui.Palette.Blue)
		y += ui.LineHeight
	}
	return y + bridgeSection
}

func yamlRow(lines []string, i, rows, total int) string {
	if i == rows-1 && total > rows {
		return "  ..."
	}
	if i < len(lines) {
		return strings.TrimRight(lines[i], " ")
	}
	return ""
}

func init() {
	Overlays[OverlayBridge] = OverlaySpec{(*HUD).bridgeButtons, (*HUD).drawBridge}
}
