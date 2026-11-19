package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	guildKey     = "guild"
	guildRateX   = 82
	GuildValueX  = 124
	GuildBarXAt  = 154
	guildBarW    = 50
	guildTierX   = 212
	shareWarn    = .6
	shareGood    = .9
	nextLines    = 2
	StateNoDepot = "NO DEPOT"
	StateNoCrew  = "NO CREW"
	StateStart   = "STARTING"
	StateNeck    = "BOTTLENECK"
)

type GuildLine struct {
	Kind  string
	Name  string
	Trade string
	Rate  float64
	Value string
	Share float64
	Crew  int
	State string
}

func CarriedShare(arrival, served, waiting float64) float64 {
	if arrival <= 0 {
		if waiting < .5 {
			return 1
		}
		return 0
	}
	return min(max(served/arrival, 0), 1)
}

func Guilds(f Frame) []GuildLine {
	fv := f.State.Freight
	worst := worstKind(fv)
	out := []GuildLine{}
	for _, k := range fv.Kinds {
		if !k.Open && k.Waiting < .5 {
			continue
		}
		g := GuildLine{Kind: k.Kind, Name: strings.ToUpper(k.CrewName), Trade: k.CrewName, Rate: k.ServedRate, Value: BoxGold(k.Value * 1.25),
			Share: CarriedShare(k.ArrivalRate, k.ServedRate, k.Waiting), Crew: k.Crew, State: crewWord(k, f.State.Bottleneck, worst, raised(f.State.Crews, k.CrewName))}
		out = append(out, g)
	}
	if s := fv.Seals; s.DepotOpen || s.ArrivalRate > 0 {
		out = append(out, GuildLine{Kind: "sealed", Name: strings.ToUpper(s.CrewName), Trade: s.CrewName, Rate: s.ServedRate, Value: "x1.5",
			Share: CarriedShare(s.ArrivalRate, s.ServedRate, s.Waiting), Crew: s.Crew, State: sealsWord(s)})
	}
	return out
}

func BoxGold(v float64) string {
	switch {
	case v >= 1000:
		return ui.Bridge(Compact(v), "$"+Compact(v))
	case v >= 10:
		return Money(fmt.Sprintf("%.0f", v))
	}
	return Money(fmt.Sprintf("%.1f", v))
}

func raised(crews []view.CrewView, name string) bool {
	for _, c := range crews {
		if c.Name == name && c.Replicas > 0 {
			return true
		}
	}
	return false
}

func crewWord(k view.KindView, neck, worst string, raised bool) string {
	switch {
	case !k.DepotOpen && !k.Door:
		return StateNoDepot
	case k.DepotOpen && k.Crew == 0 && raised:
		return StateStart
	case k.DepotOpen && k.Crew == 0:
		return StateNoCrew
	case k.Kind == worst && (neck == "crew" || neck == "labor" || neck == "crowded"):
		return StateNeck
	}
	return ""
}

func sealsWord(s view.SealsView) string {
	switch {
	case !s.DepotOpen:
		return StateNoDepot
	case s.Crew == 0:
		return StateNoCrew
	}
	return ""
}

func worstKind(fv view.FreightView) string {
	worst, top := "", 0.0
	for _, k := range fv.Kinds {
		if l := k.Losses.Spoiled + k.Losses.QuayFull; l > top {
			worst, top = k.Kind, l
		}
	}
	return worst
}

func Cause(f Frame) string { return CauseForms(f)[0] }

func CauseForms(f Frame) []string {
	st := f.State
	switch st.Bottleneck {
	case "labor":
		return []string{"Keeps out of food: no room to feed more"}
	case "crowded":
		return []string{"Huts crowd one Keep while others idle"}
	}
	if lines := StateLossLines(st); len(lines) > 0 && st.Bottleneck != "" {
		return []string{lines[0].Text}
	}
	ready, total := hearthCount(f.Pods)
	boxes := plural(int(st.Freight.Waiting+.5), "box", "boxes")
	workers := "huts"
	if total == 1 {
		workers = "hut"
	}
	return []string{
		fmt.Sprintf("%d of %d %s ready, %s on the quay", ready, total, workers, boxes),
		fmt.Sprintf("%d/%d %s ready, %s on the quay", ready, total, workers, boxes),
		fmt.Sprintf("%d/%d %s ready, %s waiting", ready, total, workers, boxes),
		fmt.Sprintf("%d/%d ready, %d waiting", ready, total, int(st.Freight.Waiting+.5)),
	}
}

func (h *HUD) firstFit(forms []string, width int) string {
	for _, text := range forms {
		if w, _ := h.Font.Measure(text, 1); w <= width {
			return text
		}
	}
	return forms[len(forms)-1]
}

func hearthCount(pods []model.PodView) (int, int) {
	ready, total := 0, 0
	for _, p := range pods {
		if p.System {
			continue
		}
		total++
		if p.State == model.HearthReady {
			ready++
		}
	}
	return ready, total
}

func NextLine(f Frame) string {
	if _, ok := f.current(); ok || NextShown(f) && BoardShown(f) || f.State.Calm > 0 || f.Introducing() {
		return ""
	}
	return f.State.Advice
}

func nextRect(f Frame) ui.Rect {
	rows := cardRows(f)
	return ui.Rect{X: CardBox.X + 6, Y: CardBox.Y + 6 + rows*ui.LineHeight + 4, W: CardBox.W - 12, H: nextLines * ui.LineHeight}
}

func (h *HUD) realmCardButtons(f Frame) []Button {
	if h.live != nil && !f.cardSelection() {
		return h.liveCardButtons()
	}
	if h.live != nil || f.cardSelection() || gateMode(f) {
		return nil
	}
	out := h.guildButtons(f)
	if NextLine(f) == "" {
		return out
	}
	return append(out, Button{ID: "card:next", Rect: nextRect(f), Tip: "Show me where [N]"})
}

func (h *HUD) guildButtons(f Frame) []Button {
	out := []Button{}
	y := CardBox.Y + 6 + ui.LineHeight + 2
	for _, g := range Guilds(f) {
		if c, ok := chainNamed(f, g.Trade); ok {
			out = append(out, Button{ID: guildKey + ":" + g.Trade, Rect: ui.Rect{X: CardBox.X + 6, Y: y, W: CardBox.W - 12, H: ui.LineHeight},
				Tip: openTip(c)})
		}
		y += ui.LineHeight
	}
	return out
}

func (h *HUD) pressGuild(rest string, f Frame) []Intent {
	c, ok := chainNamed(f, rest)
	if !ok {
		return nil
	}
	sel := CrewSelection(c)
	return []Intent{{Kind: IntentChoose, Target: sel.Kind, Command: sel.ID}, {Kind: IntentSound, Command: "click"}}
}

func init() { prefixHandlers[guildKey] = (*HUD).pressGuild }

func (h *HUD) drawRealmCard(dst *ebiten.Image, f Frame) {
	if h.live != nil {
		h.drawLiveCard(dst)
		return
	}
	if gateMode(f) {
		h.drawGateCard(dst, f)
		return
	}
	x, y := CardBox.X+8, CardBox.Y+6
	h.Font.DrawShadow(dst, "YOUR REALM", x, y, 1, ui.Palette.Gold)
	if f.Revealed("slo") {
		clr := ui.Palette.Green
		if f.State.SLO.Current < f.State.SLO.Target {
			clr = ui.Palette.RedText
		}
		text := "NINES " + Nines(f.State.SLO.Current)
		w, _ := h.Font.Measure(text, 1)
		h.Font.Draw(dst, text, CardBox.X+CardBox.W-8-w, y, 1, clr)
	}
	y += ui.LineHeight + 2
	for _, g := range Guilds(f) {
		if h.Hovering && h.Hover.ID == guildKey+":"+g.Trade {
			ui.Fill(dst, ui.Rect{X: CardBox.X + 6, Y: y - 1, W: CardBox.W - 12, H: ui.LineHeight}, rowHover)
		}
		h.drawGuild(dst, g, x, y)
		y += ui.LineHeight
	}
	drawClipped(dst, h.firstFit(CauseForms(f), CardBox.W-16), x, y, CardBox.W-16, ui.Palette.TextDim)
	h.drawNext(dst, f)
}

func (h *HUD) drawGuild(dst *ebiten.Image, g GuildLine, x, y int) {
	drawClipped(dst, ui.Settle(g.Name), x, y, guildRateX-ui.Advance, ui.Palette.Text)
	h.Font.Draw(dst, fmt.Sprintf("%.1f/s", g.Rate), x+guildRateX, y, 1, ui.Palette.Cream)
	h.Font.Draw(dst, g.Value, x+GuildValueX, y, 1, ui.Palette.Gold)
	bar := GuildBarX(g.Value)
	if g.State != "" {
		h.Font.Draw(dst, g.State, x+bar, y, 1, ui.Palette.RedText)
		return
	}
	h.drawMiniBar(dst, ui.Rect{X: x + bar, Y: y + 2, W: guildBarW - (bar - GuildBarXAt), H: 5}, g.Share, ShareColor(g.Share))
	h.Font.Draw(dst, GuildCount(g), x+guildTierX, y, 1, ui.Palette.Gold)
}

func GuildCount(g GuildLine) string {
	return fmt.Sprintf(ui.Bridge("%d CARRY", "%d SERVE"), g.Crew)
}

func GuildBarX(value string) int {
	return max(GuildBarXAt, GuildValueX+ui.Font{}.Span(value, 1))
}

func ShareColor(share float64) color.RGBA {
	switch {
	case share >= shareGood:
		return ui.Palette.Green
	case share >= shareWarn:
		return ui.Palette.GoldBright
	}
	return ui.Palette.RedText
}

func (h *HUD) drawMiniBar(dst *ebiten.Image, r ui.Rect, value float64, fill color.RGBA) {
	if !h.Lib.NineSlice(dst, "bar-mini", r) {
		ui.Fill(dst, r, ui.Palette.ConsoleEdge)
		ui.Stroke(dst, r, ui.Palette.Ink)
	}
	inner := r.Inset(1)
	inner.W = int(float64(inner.W) * min(max(value, 0), 1))
	if inner.W > 0 {
		ui.Fill(dst, inner, fill)
	}
}

func (h *HUD) drawNext(dst *ebiten.Image, f Frame) {
	text := NextLine(f)
	if text == "" {
		return
	}
	r := nextRect(f)
	hot := h.Hovering && h.Hover.ID == "card:next"
	for i, row := range h.Font.WrapLines("NEXT: "+text, r.W-4, 1, nextLines) {
		y := r.Y + i*ui.LineHeight
		h.Font.Draw(dst, row, r.X+2, y, 1, ui.Palette.Blue)
		if hot {
			w, _ := h.Font.Measure(row, 1)
			ui.Fill(dst, ui.Rect{X: r.X + 2, Y: y + ui.CapHeight + 2, W: w, H: 1}, ui.Palette.Blue)
		}
	}
}

var nextTargets = []struct{ word, page, overlay string }{
	{"Stone Keep", PageBuild, ""}, {"land", PageBuild, ""}, {"endow", "", OverlaySoulvault},
	{"Keep", PageBuild, ""},
}

func (h *HUD) FollowNext(f Frame) []Intent {
	advice := f.State.Advice
	for _, t := range nextTargets {
		if advice == "" || !strings.Contains(advice, t.word) {
			continue
		}
		if t.page != "" {
			h.PageID, h.OverlayID = t.page, OverlayNone
		} else {
			h.OverlayID = t.overlay
		}
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	if strings.Contains(advice, "berth") {
		return []Intent{{Kind: IntentHarbor}, {Kind: IntentSound, Command: "click"}}
	}
	if cmd := AdviceCommand(f); cmd != "" {
		h.OpenStone(cmd)
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func AdviceCommand(f Frame) string {
	advice := f.State.Advice
	if advice == "" {
		return f.SpeakCommand()
	}
	for _, c := range f.State.Commands {
		if name := c.Action.Name; name != "" && strings.Contains(advice, name) {
			return c.Command
		}
		if kind := c.Action.Kind; kind != "" && strings.Contains(advice, kind) {
			return c.Command
		}
	}
	return ""
}

func (h *HUD) pressCard(_ string, f Frame) []Intent {
	return h.FollowNext(f)
}
