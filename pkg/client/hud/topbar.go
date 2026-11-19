package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var speeds = []float64{1, 2, 5, 20}

func Compact(v float64) string {
	return view.Compact(v)
}

func Clock(realmTime, phase float64) string {
	period := view.DefaultConfig().DayPeriod
	day := int(realmTime/period+view.StartPhase) + 1
	minutes := int(phase * 24 * 60)
	return fmt.Sprintf("DAY %d %02d:%02d", day, minutes/60, minutes%60)
}

func ninesColor(slo view.SLOView) color.RGBA {
	if slo.Target > 0 && slo.Samples >= 1 && slo.Current < slo.Target {
		return ui.Palette.RedText
	}
	return ui.Palette.Green
}

func Nines(current float64) string {
	return view.FreshShare(current)
}

func (h *HUD) TopButtons(f Frame) []Button {
	out := append(h.pinnedButtons(f), Button{ID: "top:menu", Rect: ui.Rect{X: Width - 52, Y: 2, W: 48, H: 14}, Label: "MENU", Tip: "Save, sound and the title screen"})
	x := Width - 58
	if f.Revealed("speed") && f.Shown(SurfaceSpeed) {
		for i := len(speeds) - 1; i >= 0; i-- {
			if f.State.Tutorial && speeds[i] > chapterTopSpeed {
				continue
			}
			w := 22 + 6*len(fmt.Sprint(speeds[i]))/2
			x -= w + 2
			out = append(out, Button{ID: fmt.Sprintf("speed:%g", speeds[i]), Rect: ui.Rect{X: x, Y: 2, W: w, H: 14},
				Label: fmt.Sprintf("%gX", speeds[i]), Tip: "Realm time flows faster", Selected: f.State.Speed == speeds[i] && !f.Paused})
		}
	}
	x -= 24
	out = append(out, Button{ID: "top:pause", Rect: ui.Rect{X: x, Y: 2, W: 22, H: 14}, Label: "II", Tip: "Pause the realm [Space]", Selected: f.Paused})
	if b, ok := ledgerTopButton(f, x-ledgerTopW-4); ok {
		out = append(out, b)
		x = b.Rect.X
	}
	if b, ok := h.helpTopButton(f, x-20); ok {
		out = append(out, b)
	}
	return out
}

var meterPages = map[string]string{MeterGold: "gold", MeterWisdom: "wisdom", MeterHands: "food", MeterLore: "housing"}

type Stat struct {
	icon, Text string `json:"-"`
	clr        color.RGBA
	Tip        string `json:"-"`
	X          int    `json:"-"`
	Id         string `json:"-"`
	gauge      bool
	fill       float64
	Meter      string `json:"-"`
	slot       string
	Rate       string `json:"-"`
	rateSlot   string
	RateClr    color.RGBA `json:"-"`
	Upkeep     string     `json:"-"`
	UpkeepClr  color.RGBA `json:"-"`
	Net        string     `json:"-"`
	netClr     color.RGBA
	Floor      string `json:"-"`
}

var meterLabels = map[string]string{
	MeterGold: "GOLD", MeterWisdom: "WISDOM", MeterRenown: "RENOWN", MeterNines: "NINES",
	MeterHands: "FOOD", MeterLore: "HOUSING", MeterCodex: "CODEX",
}

func statLabel(st Stat) string {
	if name, ok := meterLabels[st.Meter]; ok {
		return name + " " + st.Text
	}
	return ""
}

func shownStats(stats []Stat, f Frame) []Stat {
	out := make([]Stat, 0, len(stats))
	for _, st := range stats {
		if st.Meter == "" || f.Shown(st.Meter) {
			out = append(out, st)
		}
	}
	return out
}

func (h *HUD) rawTopStats(f Frame) []Stat {
	if h.live != nil {
		out := h.liveStats()
		h.layoutStats(out)
		return out
	}
	s := f.State
	out := []Stat{}
	if !s.Tutorial {
		out = append(out, GoldStat(f))
	}
	if f.Revealed("wisdom") {
		out = append(out, Stat{icon: "wisdom", Text: FitAmount(s.Wisdom, countChars), slot: digits(countChars), clr: ui.Palette.Purple, Tip: WisdomTip(f), Meter: MeterWisdom})
	}
	out = append(out, supplyStats(f)...)
	if f.Revealed("renown") {
		out = append(out, Stat{icon: "renown", Text: FitAmount(s.Renown, renownChars), slot: digits(renownChars), clr: ui.Palette.Cream, Tip: RenownTip(f), Meter: MeterRenown})
	}
	if f.Revealed("slo") {
		out = append(out, Stat{icon: "alert", Text: "NINES " + Nines(s.SLO.Current), slot: "NINES " + Nines(1), clr: ninesColor(s.SLO), Tip: NinesTip(f), Meter: MeterNines})
	}
	out = shownStats(out, f)
	if st, ok := codexStat(f); ok {
		out = append(out, st)
	}
	return h.squeezeStats(out, f)
}

func (h *HUD) squeezeStats(out []Stat, f Frame) []Stat {
	end := h.layoutStats(out)
	if len(out) == 0 || out[0].Net == "" || h.clockFits(f, end) {
		return out
	}
	out[0].Upkeep, out[0].Net = "", ""
	out[0].Rate, out[0].RateClr = FitRate(NetGoldRate(f.State)), ui.Palette.GoldBright
	if out[0].Rate[0] == '-' {
		out[0].RateClr = ui.Palette.RedText
	}
	if end := h.layoutStats(out); out[0].Floor != "" && !h.clockFits(f, end) {
		out[0].Floor = ""
		h.layoutStats(out)
	}
	return out
}

func (h *HUD) layoutStats(stats []Stat) int {
	x := goldX
	for i := range stats {
		stats[i].X = x
		x += h.StatWidth(stats[i]) + statGap
	}
	return x
}

func (h *HUD) clockFits(f Frame, end int) bool {
	buttons := h.TopButtons(f)
	right := buttons[len(buttons)-1].Rect.X - clockGap
	need := h.Font.Width(clockDay, 1)
	return right-(end-statGap+clockGap) >= need
}

func supplyStats(f Frame) []Stat {
	hands, lore := SupplyOf(f.Metrics.Nodes)
	out := []Stat{}
	if f.Revealed("hands") {
		out = append(out, Stat{icon: "food", Text: SupplyCount(Rations(hands)), slot: supplyCountSlot(), clr: SupplyColor(hands), Tip: HandsTip(f), Meter: MeterHands})
	}
	if f.Revealed("lore") {
		out = append(out, Stat{icon: "housing", Text: SupplyCount(Rooms(lore)), slot: supplyCountSlot(), clr: SupplyColor(lore), Tip: LoreTip(f), Meter: MeterLore})
	}
	return out
}

func (h *HUD) StatButtons(f Frame) []Button {
	out := []Button{}
	for _, st := range h.TopStats(f) {
		id := "info:" + st.Text
		if st.Id != "" {
			id = st.Id
		}
		out = append(out, Button{ID: id, Rect: ui.Rect{X: st.X, Y: 1, W: h.StatWidth(st), H: TopH - 3}, Label: statLabel(st), Tip: st.Tip, Page: meterPages[st.Meter]})
	}
	return out
}

func (h *HUD) DrawTopBar(dst *ebiten.Image, f Frame) {
	ui.Fill(dst, TopBar, ui.Palette.ConsoleEdge)
	ui.Fill(dst, ui.Rect{X: 0, Y: TopH - 2, W: Width, H: 1}, ui.Palette.Gold)
	ui.Fill(dst, ui.Rect{X: 0, Y: TopH - 1, W: Width, H: 1}, ui.Palette.Ink)
	h.Font.DrawShadow(dst, "KUBERNITES", 8, 5, 1, ui.Palette.GoldBright)
	for _, st := range h.TopStats(f) {
		if st.Meter == "" {
			h.drawStat(dst, st)
			continue
		}
		h.withAppear(dst, st.Meter, f, func(d *ebiten.Image) { h.drawStat(d, st) })
	}
	h.drawGoldPulse(dst)
	if clock, slot := h.ClockSlot(f); clock != "" {
		h.Font.DrawShadow(dst, clock, slot.X+slot.W-h.Font.Width(clock, 1), 5, 1, ui.Palette.Text)
	}
	for _, b := range h.TopButtons(f) {
		if b.ID != PinnedResume {
			h.drawButton(dst, b)
		}
	}
}

func (h *HUD) drawButton(dst *ebiten.Image, b Button) {
	b = h.liveGate(b)
	state := ui.ButtonState{Hovered: h.Hovering && h.Hover.ID == b.ID, Disabled: b.Disabled, Selected: b.Selected}
	face := b.Rect
	if b.Face.W > 0 {
		face = b.Face
	}
	ui.Button(dst, h.Font, face, b.Label, state)
}

func (h *HUD) drawStat(dst *ebiten.Image, st Stat) {
	x := st.X
	if st.icon != "" {
		if name, ok := meterLabels[st.Meter]; ok && !strings.Contains(st.Text, name) {
			ui.Caption(ui.Name(name), ui.Rect{X: x, Y: 5, W: statIconW - 6, H: ui.CapHeight})
		}
		h.Lib.Icon(dst, st.icon, x, 1)
		x += statIconW
	}
	h.Font.DrawShadow(dst, st.Text, x, 5, 1, st.clr)
	x += h.Font.Width(st.slot, 1)
	if st.rateSlot != "" {
		h.Font.DrawShadow(dst, st.Rate, x+ui.Advance, 5, 1, st.RateClr)
		x += ui.Advance + h.Font.Width(st.rateSlot, 1)
	}
	if st.Upkeep != "" {
		h.Font.DrawShadow(dst, st.Upkeep, x+ui.Advance, 5, 1, st.UpkeepClr)
		x += ui.Advance + h.Font.Width(st.rateSlot, 1)
	}
	if st.Net != "" {
		h.Font.DrawShadow(dst, st.Net, x+ui.Advance, 5, 1, st.netClr)
		x += 2*ui.Advance + h.Font.Width(st.rateSlot, 1)
	}
	if st.Floor != "" {
		h.Font.DrawShadow(dst, st.Floor, x+ui.Advance, 5, 1, ui.Palette.TextDim)
	}
	if st.gauge {
		h.drawMiniBar(dst, ui.Rect{X: x + gaugeGap, Y: 6, W: gaugeW, H: 5}, st.fill, st.clr)
	}
}
