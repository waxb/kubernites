package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlaySiege = "siege"
	SiegeReveal  = "siege"
	siegeLabel   = "RUSH TIDE"
	siegeTip     = "A Rush Tide gathers at sea [G]"
	stripBarW    = 132
	SiegeRows    = 9
	SiegeInset   = 36
	floodTail    = "expect the quay to flood."
)

var (
	SiegeBox   = ui.Rect{X: Width/2 - 210, Y: 112, W: 420, H: 160 + boxVigH + (SiegeRows-5)*ui.LineHeight}
	SiegeStrip = ui.Rect{X: boardX, Y: TopH + 4 + BoardMaxH + boardGap, W: BoardW, H: 30}
)

func siegeKnown(f Frame) bool { return f.Revealed(SiegeReveal) && !f.State.CalmRealm }

func SiegeActive(f Frame) bool { return f.State.Siege.Phase != "" }

func Duration(seconds float64) string {
	s := max(int(seconds+.5), 0)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func CrowdText(crowd float64) string {
	return fmt.Sprintf("For 3 minutes, up to %.3g more boxes land each second.", crowd)
}

func SiegeButton(f Frame) Button {
	sg := f.State.Siege
	b := Button{ID: "cmd:siege", Label: siegeLabel, Icon: "icon/siege", Tip: siegeTip}
	switch {
	case SiegeActive(f):
		b.Selected, b.Tip = true, "The Rush Tide floods the quay"
	case sg.RetryIn > 0:
		b.Note, b.Tip = Duration(sg.RetryIn), "the next Rush Tide in "+Duration(sg.RetryIn)
	case !sg.Available:
		b.Tip = sg.Reason
	}
	return b
}

func SiegeReadiness(f Frame) []Line {
	sounding, first := 0, ""
	for _, h := range f.Horns {
		if h.Sounding {
			sounding++
			if first == "" {
				first = fmt.Sprintf(" (%s %d-%d)", h.Target, h.Min, h.Max)
			}
		}
	}
	horns := Line{Text: fmt.Sprintf("Your Horns: %d sounding%s", sounding, first), Clr: ui.Palette.Green}
	if sounding == 0 {
		horns = Line{Text: silentHorns(len(f.Horns)), Clr: ui.Palette.RedText}
	}
	keeps := Line{Text: fmt.Sprintf("Your Keeps: %d Ready, %s in all", f.KeepsReady, foodOfCores(keepFood(f.Metrics.Nodes))), Clr: ui.Palette.Green}
	if f.Joining {
		keeps = Line{Text: "A Keep is joining: wait for Ready", Clr: ui.Palette.RedText}
	}
	sg := f.State.Siege
	hands := Line{Text: fmt.Sprintf("The tide needs %s spare; you have %s", foodOfCores(sg.Hands), foodCount(sg.Spare*view.RationsPerCPU)), Clr: ui.Palette.Green}
	if sg.Spare < sg.Hands {
		hands.Clr = ui.Palette.RedText
	}
	out := []Line{horns, keeps, hands}
	if sg.Crowded != "" && sg.Phase == "" {
		return append(out, Line{Text: CapitalFirst(sg.Crowded) + ": " + floodTail, Clr: ui.Palette.RedText})
	}
	if sg.Forecast != "" && sg.Phase == "" {
		out = append(out, Line{Text: sg.Forecast, Clr: ui.Palette.RedText})
	}
	return out
}

func silentHorns(n int) string {
	switch n {
	case 0:
		return "No War Horn stands"
	case 1:
		return "Your War Horn is silent"
	}
	return fmt.Sprintf("Your %d War Horns are silent", n)
}

func keepFood(nodes map[string]view.NodeMetrics) float64 {
	var milli int64
	for _, n := range nodes {
		milli += n.CPUAllocatableMilli
	}
	return float64(milli) / 1000
}

func siegeShort(f Frame) bool {
	sg := f.State.Siege
	return sg.Phase == "" && (sg.Spare < sg.Hands || sg.Crowded != "")
}

func (h *HUD) SiegeButtons(f Frame) []Button {
	r := SiegeBox
	y := r.Y + r.H - 28
	answer := Button{ID: "siege:answer", Rect: ui.Rect{X: r.X + r.W - 196, Y: y, W: 160, H: 18}, Label: "ANSWER THE CALL",
		Tip: "The ships arrive after ten heartbeats"}
	switch {
	case f.Joining:
		answer.Disabled, answer.Tip = true, "wait for Ready"
	case !f.State.Siege.Available:
		answer.Disabled, answer.Tip = true, f.State.Siege.Reason
	case siegeShort(f):
		answer.Tip = f.State.Siege.Forecast
	}
	return []Button{{ID: "siege:cancel", Rect: ui.Rect{X: r.X + 36, Y: y, W: 110, H: 18}, Label: "NOT YET", Tip: "The tide can wait [Esc]"}, answer,
		closeButtonAs(r, "siege:close", "The tide can wait [Esc]")}
}

func (h *HUD) pressSiege(rest string, f Frame) []Intent {
	switch rest {
	case "cancel", "close":
		h.OverlayID = OverlayNone
	case "answer":
		h.OverlayID = OverlayNone
		return []Intent{act(view.Action{Action: "answerSiege"})}
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) OpenSiege(f Frame) []Intent {
	if SiegeActive(f) {
		return nil
	}
	h.OverlayID = OverlaySiege
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) DrawSiege(dst *ebiten.Image, f Frame) {
	r := SiegeBox
	ui.DarkPanel(dst, r)
	h.Font.DrawCentered(dst, "THE RUSH TIDE", ui.Rect{X: r.X, Y: r.Y + 8, W: r.W, H: 16}, 2, ui.Palette.GoldBright)
	sg := f.State.Siege
	lines := []Line{
		{Text: CrowdText(sg.Crowd), Clr: ui.Palette.Text},
		{Text: fmt.Sprintf("Deliver %s fresh. Reward: %s threads + a Seal.", Nines(sg.Target), Compact(sg.Reward)), Clr: ui.Palette.Text},
	}
	lines = append(lines, SiegeReadiness(f)...)
	row := 0
	for _, l := range lines {
		for _, text := range h.Font.WrapLines(l.Text, r.W-2*SiegeInset, 1, SiegeRows-row) {
			h.Font.Draw(dst, text, r.X+SiegeInset, r.Y+34+row*ui.LineHeight, 1, l.Clr)
			row++
		}
	}
	h.DrawVig(dst, SiegeVig(f), SiegeVigRect(), 1)
	buttons := h.SiegeButtons(f)
	h.drawButton(dst, buttons[2])
	h.drawButton(dst, buttons[0])
	ui.Stroke(dst, buttons[0].Rect.Inset(-2), ui.Palette.Cream)
	if buttons[1].Disabled || siegeShort(f) {
		h.drawButton(dst, buttons[1])
		return
	}
	h.drawPrimary(dst, buttons[1])
}

func (h *HUD) DrawSiegeStrip(dst *ebiten.Image, f Frame) {
	if !SiegeActive(f) {
		return
	}
	r := SiegeStripRect(f)
	if !h.Lib.NineSlice(dst, "panel-dark", r) {
		ui.DarkPanel(dst, r)
	}
	sg := f.State.Siege
	seals := fmt.Sprintf("SEALS %d/%d", sg.Seals, sg.SealsGoal)
	sw, _ := h.Font.Measure(seals, 1)
	h.Font.Draw(dst, seals, r.X+r.W-8-sw, r.Y+5, 1, ui.Palette.Gold)
	if sg.Phase == "countdown" {
		h.Font.Draw(dst, "THE TIDE GATHERS "+Duration(sg.TimeLeft), r.X+8, r.Y+5, 1, ui.Palette.GoldBright)
		h.Font.Draw(dst, "Ten heartbeats to the flood of boxes", r.X+8, r.Y+17, 1, ui.Palette.TextDim)
		return
	}
	title := "RUSH TIDE " + Duration(sg.TimeLeft)
	h.Font.Draw(dst, title, r.X+8, r.Y+5, 1, ui.Palette.GoldBright)
	if sg.Extra > 0 {
		tw, _ := h.Font.Measure(title, 1)
		h.Font.Draw(dst, fmt.Sprintf("+%.1f/S", sg.Extra), r.X+8+tw+8, r.Y+5, 1, ui.Palette.Text)
	}
	fill := ui.Palette.Green
	if sg.Served < sg.Target {
		fill = ui.Palette.RedText
	}
	h.drawMiniBar(dst, ui.Rect{X: r.X + 8, Y: r.Y + 19, W: stripBarW, H: 6}, sg.Served, fill)
	score := fmt.Sprintf("%.1f%% / %s", sg.Served*100, strings.Replace(Nines(sg.Target), ".0%", "%", 1))
	h.Font.Draw(dst, score, r.X+stripBarW+14, r.Y+17, 1, ui.Palette.Text)
	if waiting := waitingHuts(f); waiting > 0 {
		label := ui.Bridge(fmt.Sprintf("%d UNHOUSED", waiting), fmt.Sprintf("%d PENDING", waiting))
		lw, _ := h.Font.Measure(label, 1)
		h.Font.Draw(dst, label, r.X+r.W-8-lw, r.Y+17, 1, ui.Palette.RedText)
	}
}

func waitingHuts(f Frame) int {
	total := 0
	for _, c := range f.State.Crews {
		total += c.Pending
	}
	return total
}

func SiegeStripRect(f Frame) ui.Rect {
	r := SiegeStrip
	if BoardShown(f) {
		r.Y = TopH + 4 + FreightRect(f).H + boardGap
	}
	return r
}
