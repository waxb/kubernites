package hud

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayVoyage = "voyage"

	voyageRow     = 40
	voyageTop     = 52
	voyageLookW   = 52
	voyageActW    = 118
	voyageTextW   = 470
	voyageSubline = "Each isle stands on its own: a storm over one leaves the others standing. Charter one to raise Keeps there."
)

type VoyageRow struct {
	ID     string
	Name   string
	Zone   string
	Owned  bool
	Center [2]float64
	Keeps  int
}

func charterTermsOf(f Frame) view.CharterView {
	return f.State.Charter
}

func voyageOpen(f Frame) bool {
	return f.Revealed("keeps") && len(f.Site.Islands) > 1
}

func VoyageRows(f Frame) []VoyageRow {
	site := f.Site
	keeps := make([]int, len(site.Islands)+1)
	for _, s := range site.Structures {
		if s.Kind != view.StructKeep {
			continue
		}
		if is, ok := site.IslandAt(s.X+s.W/2, s.Y+s.H/2); ok {
			keeps[islandIndex(site, is.ID)+1]++
		}
	}
	out := make([]VoyageRow, 0, len(site.Islands))
	for i, is := range site.Islands {
		out = append(out, VoyageRow{ID: is.ID, Name: is.Name, Zone: is.Zone, Owned: is.Owned,
			Center: [2]float64{float64(is.Center.X), float64(is.Center.Y)}, Keeps: keeps[i+1]})
	}
	return out
}

func islandIndex(site view.Site, id string) int {
	for i, is := range site.Islands {
		if is.ID == id {
			return i
		}
	}
	return -1
}

func voyageVisible() int { return (Overlay.H - voyageTop - 8) / voyageRow }

func voyageRowRect(i int) ui.Rect {
	return ui.Rect{X: Overlay.X + 8, Y: Overlay.Y + voyageTop + i*voyageRow, W: Overlay.W - 16, H: voyageRow - 4}
}

func voyageLines(r VoyageRow, terms view.CharterView) []Line {
	width := ui.MaxChars(voyageTextW, 1)
	name := strings.ToUpper(r.Name)
	zone := "zone " + r.Zone
	head := ui.Clip(name, max(width-len(zone)-2, 8))
	status, clr := charterStatus(r, terms)
	return []Line{
		{Text: head, Clr: ui.Palette.Gold},
		{Text: ui.Clip(zone, width), Clr: ui.Palette.Blue},
		{Text: ui.Clip(status, width), Clr: clr},
	}
}

func charterStatus(r VoyageRow, terms view.CharterView) (string, color.RGBA) {
	switch {
	case r.Owned && r.Keeps == 0:
		return "CHARTERED: yours to build Keeps on", ui.Palette.Green
	case r.Owned:
		return "CHARTERED   " + count(r.Keeps, "Keep"), ui.Palette.Green
	case !terms.Available && terms.Reason != "":
		return "UNCHARTERED: " + charterWhy(terms.Reason), ui.Palette.RedText
	}
	return "UNCHARTERED", ui.Palette.TextDim
}

func charterWhy(reason string) string {
	switch {
	case strings.Contains(reason, "Grimoire"):
		return "learn the Charters page"
	case strings.Contains(reason, "Feudal"):
		return "comes with the Feudal Age"
	case strings.Contains(reason, "gold"):
		return strings.ToLower(reason[:1]) + reason[1:]
	}
	return reason
}

func voyageStart(h *HUD, f Frame) int {
	return min(h.voyageList, max(len(VoyageRows(f))-voyageVisible(), 0))
}

func voyageButtons(f Frame, start int) []Button {
	rows := VoyageRows(f)
	terms := charterTermsOf(f)
	out := []Button{}
	for i, r := range rows[start:min(len(rows), start+voyageVisible())] {
		rect := voyageRowRect(i)
		y := rect.Y + (rect.H-18)/2
		look := Button{ID: "voyage:look:" + r.ID, Rect: ui.Rect{X: rect.X + rect.W - voyageActW - voyageLookW - 10, Y: y, W: voyageLookW, H: 18},
			Label: "LOOK", Tip: "Sail the camera to " + r.Name}
		out = append(out, look, charterButton(r, terms, ui.Rect{X: rect.X + rect.W - voyageActW - 4, Y: y, W: voyageActW, H: 18}))
	}
	return append(out, closeButton(Overlay))
}

func charterButton(r VoyageRow, terms view.CharterView, rect ui.Rect) Button {
	if r.Owned {
		return Button{ID: "voyage:owned:" + r.ID, Rect: rect, Label: "CHARTERED", Selected: true, Tip: r.Name + " is yours: its Keeps join zone " + r.Zone}
	}
	b := Button{ID: "voyage:charter:" + r.ID, Rect: rect, Label: "CHARTER " + GoldWord(terms.Cost),
		Tip: "Charter " + r.Name + ": a new zone for your Keeps (" + r.Zone + ")"}
	if terms.Cost <= 0 {
		b.Label = "CHARTER"
	}
	if !terms.Available {
		b.Disabled = true
		if terms.Reason != "" {
			b.Tip = terms.Reason
		}
	}
	return b
}

func (h *HUD) voyageOverlayButtons(f Frame) []Button {
	return voyageButtons(f, voyageStart(h, f))
}

func (h *HUD) pressVoyage(rest string, f Frame) []Intent {
	verb, id, _ := strings.Cut(rest, ":")
	for _, r := range VoyageRows(f) {
		if r.ID != id {
			continue
		}
		switch verb {
		case "look":
			h.OverlayID = OverlayNone
			return h.focus(r.Center)
		case "charter":
			return []Intent{act(view.Action{Action: "charterIsland", ID: r.ID})}
		}
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	return nil
}

func (h *HUD) toggleVoyage() {
	h.toggleOverlay(OverlayVoyage)
}

func voyageColor(owned bool) color.RGBA {
	if owned {
		return ui.Palette.Green
	}
	return ui.Palette.Muted
}

func (h *HUD) drawVoyage(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, Overlay)
	h.header(dst, Overlay, "THE VOYAGE")
	h.drawRows(dst, h.Font.WrapLines(voyageSubline, Overlay.W-20, 1, 2), Overlay.X+10, Overlay.Y+26, ui.Palette.TextDim)
	rows := VoyageRows(f)
	start := voyageStart(h, f)
	for i, r := range rows[start:min(len(rows), start+voyageVisible())] {
		h.drawIsle(dst, voyageRowRect(i), r, f)
	}
	h.DrawOverlayButtons(dst, h.voyageOverlayButtons(f))
}

func (h *HUD) drawIsle(dst *ebiten.Image, rect ui.Rect, r VoyageRow, f Frame) {
	ui.Well(dst, rect)
	cx, cy := rect.X+14, rect.Y+rect.H/2
	for dy := -5; dy <= 5; dy++ {
		w := 2 * (6 - iabs(dy))
		ui.Fill(dst, ui.Rect{X: cx - w, Y: cy + dy, W: 2 * w, H: 1}, voyageColor(r.Owned))
	}
	lines := voyageLines(r, charterTermsOf(f))
	x := rect.X + 30
	h.Font.Draw(dst, lines[0].Text, x, rect.Y+7, 1, lines[0].Clr)
	h.Font.Draw(dst, lines[1].Text, x+h.Font.Span(lines[0].Text, 2), rect.Y+7, 1, lines[1].Clr)
	h.Font.Draw(dst, lines[2].Text, x, rect.Y+20, 1, lines[2].Clr)
}
