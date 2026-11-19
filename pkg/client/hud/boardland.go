package hud

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

type BoardDelta struct {
	From, To int
	Earning  int
	Waits    int
	Hungry   int
	Income   float64
	CPU      int64
	Memory   int64
	Upkeep   float64
}

type deltaPiece struct {
	Text string `json:"-"`
	clr  color.RGBA
}

func BoardDeltaOf(v BoardView) BoardDelta {
	d := BoardDelta{From: v.Chain.Desired, To: v.Target}
	step := d.To - d.From
	d.Earning = step
	if step > 0 {
		d.Earning = min(step, max(v.Wall()-d.From, 0))
		d.Waits = step - d.Earning
		d.Hungry = max(0, d.Earning-min(step, max(v.fedWall()-d.From, 0)))
	}
	per := v.Facts.Per
	d.Income = float64(d.Earning) * per.Income
	d.CPU, d.Memory = int64(d.Earning)*per.CPU, int64(d.Earning)*per.Memory
	d.Upkeep = float64(d.Earning) * per.UpkeepShare
	return d
}

func signedCores(milli int64) string {
	sign := "+"
	if milli < 0 {
		sign = "-"
	}
	m := max(milli, -milli)
	return ui.Bridge(sign+shout(foodOf(float64(m))), fmt.Sprintf("%s%dm CPU", sign, m))
}

func signedMem(bytes int64) string {
	sign := "+"
	if bytes < 0 {
		sign = "-"
	}
	return ui.Bridge(sign+shout(housingOf(float64(max(bytes, -bytes)))), sign+MemQuantity(max(bytes, -bytes))+" MEMORY")
}

func payPiece(v BoardView, f Frame, step int) deltaPiece {
	if pay := payoffOf(v.Chain, f, step); pay != "" {
		return deltaPiece{pay, ui.Palette.GoldBright}
	}
	if step == 0 {
		return deltaPiece{ui.Bridge("+0 GOLD/S", "+$0/S"), ui.Palette.TextDim}
	}
	if c, ok := seasoningAhead(v, f); ok {
		return deltaPiece{fmt.Sprintf(ui.Bridge("SEASONED x%g AT %d: EACH CARRIES FASTER", "BONUS x%g AT %d READY: EACH CARRIES FASTER"), c.NextMultiplier, c.Next), ui.Palette.GoldBright}
	}
	return deltaPiece{ui.Bridge("+0 GOLD/S: KEEPING UP", "+$0/S: ENOUGH REPLICAS"), ui.Palette.TextDim}
}

func seasoningAhead(v BoardView, f Frame) (view.CrewView, bool) {
	for _, c := range f.State.Crews {
		if c.Name == v.Chain.Name && c.Next > 0 && v.Chain.Desired < c.Next && v.Target >= c.Next {
			return c, true
		}
	}
	return view.CrewView{}, false
}

func upkeepText(v float64) string {
	amount := Rate(math.Abs(v))
	if amount == "0" {
		return ""
	}
	sign := "+"
	if v < 0 {
		sign = "-"
	}
	return ui.Bridge("UPKEEP "+sign+amount+" GOLD/S", "UPKEEP "+sign+"$"+amount+"/S")
}

func DeltaPieces(v BoardView, f Frame) []deltaPiece {
	d := BoardDeltaOf(v)
	if v.Band {
		return bandPieces(v)
	}
	if d.From == d.To {
		return idlePieces(v, f)
	}
	out := []deltaPiece{{fmt.Sprintf("%d -> %d %s", d.From, d.To, ui.Bridge("HUTS", "REPLICAS")), ui.Palette.Cream}, payPiece(v, f, d.Earning)}
	if d.CPU != 0 {
		out = append(out, deltaPiece{signedCores(d.CPU), ui.Palette.Blue})
	}
	if d.Memory != 0 {
		out = append(out, deltaPiece{signedMem(d.Memory), ui.Palette.Purple})
	}
	if up := upkeepText(d.Upkeep); up != "" {
		out = append(out, deltaPiece{up, ui.Palette.RedText})
	}
	if d.Waits > 0 {
		out = append(out, deltaPiece{fmt.Sprintf(ui.Bridge("%d WAIT", "%d PENDING"), d.Waits), ui.Palette.RedText})
	}
	return out
}

func idlePieces(v BoardView, f Frame) []deltaPiece {
	per := v.Facts.Per
	out := []deltaPiece{{ui.Bridge("EACH HUT", "PER REPLICA"), ui.Palette.Cream}, payPiece(v, f, 1)}
	if per.CPU <= 0 && per.Memory <= 0 {
		out = append(out, deltaPiece{ui.Bridge("PLEDGES NOTHING: ANY KEEP TAKES IT", "REQUESTS NOTHING (BestEffort)"), ui.Palette.TextDim})
	}
	if up := upkeepText(per.UpkeepShare); up != "" {
		out = append(out, deltaPiece{up, ui.Palette.RedText})
	}
	return out
}

func (h *HUD) drawBoardDeltas(dst *ebiten.Image, v BoardView, f Frame) {
	h.drawPieces(dst, DeltaPieces(v, f), v.box.X+cbPad, v.box.Y+cbDeltaY, v.box.X+v.box.W-cbPad)
}

func landStatus(v BoardView) (string, color.RGBA) {
	if n := v.Waiting(); n > 0 {
		return fmt.Sprintf(ui.Bridge("%d WAITING", "%d PENDING"), n), ui.Palette.RedText
	}
	if !v.Facts.Bounded() {
		return ui.Bridge("ROOM FOR MANY", "NO REQUESTS: ANY NODE"), ui.Palette.Green
	}
	at := max(v.Target, v.Chain.Desired)
	room, seats := max(v.fedWall()-at, 0), max(v.Wall()-at, 0)
	switch {
	case room > 0 && seats > room:
		return fmt.Sprintf(ui.Bridge("ROOM FOR %d MORE BY PLEDGE; AT THEIR REAL USE, %d", "%d MORE FIT BY REQUESTS; AT REAL USE, %d"), seats, room), ui.Palette.GoldBright
	case room > 0:
		return fmt.Sprintf(ui.Bridge("ROOM FOR %d MORE", "ROOM FOR %d MORE"), room), ui.Palette.Green
	case seats > 0:
		return fmt.Sprintf(ui.Bridge("ROOM FOR %d MORE BY PLEDGE; AT THEIR REAL USE, 0", "%d MORE FIT BY REQUESTS; AT REAL USE, 0"), seats), ui.Palette.GoldBright
	}
	return ui.Bridge("LAND FULL", "NODES FULL"), ui.Palette.GoldBright
}

func (h *HUD) landLayout(v BoardView) (ui.Rect, int) {
	right := v.land.X + v.land.W
	if len(v.Buy) > 0 {
		right = v.Buy[0].Rect.X - landGap
	}
	status, _ := landStatus(v)
	statusX := right - h.Font.Width(status, 1)
	bar := ui.Rect{X: v.land.X + landLabelW, Y: v.land.Y + 2, W: statusX - landGap - v.land.X - landLabelW, H: v.land.H - 6}
	if bar.W < landMinBar {
		bar.W = 0
	}
	return bar, statusX
}

func (h *HUD) drawLandStrip(dst *ebiten.Image, v BoardView) {
	ty := v.land.Y + (v.land.H-ui.GlyphHeight)/2
	h.Font.Draw(dst, ui.Bridge("LAND", "NODES"), v.land.X, ty, 1, ui.Palette.Cream)
	bar, statusX := h.landLayout(v)
	if bar.W > 0 {
		h.drawLandBar(dst, v, bar)
	}
	status, clr := landStatus(v)
	h.Font.Draw(dst, status, statusX, ty, 1, clr)
}

func (h *HUD) drawLandBar(dst *ebiten.Image, v BoardView, bar ui.Rect) {
	cpu, mem := v.Facts.LandTotals()
	ui.Fill(dst, bar, railDark)
	if cpu[1] <= 0 {
		return
	}
	d := BoardDeltaOf(v)
	mine := int64(placedCount(v.Pods))
	cpuBar := ui.Rect{X: bar.X, Y: bar.Y, W: bar.W, H: bar.H - memBarH - 1}
	fillShares(dst, cpuBar, float64(cpu[1]), []landShare{{float64(cpu[0] - mine*v.Facts.Per.CPU), othersFill},
		{float64(mine * v.Facts.Per.CPU), ui.Palette.Green}, {float64(max(d.CPU, 0)), previewAdd}})
	memBar := ui.Rect{X: bar.X, Y: bar.Y + bar.H - memBarH, W: bar.W, H: memBarH}
	fillShares(dst, memBar, float64(mem[1]), []landShare{{float64(mem[0] - mine*v.Facts.Per.Memory), othersFill},
		{float64(mine * v.Facts.Per.Memory), ui.Palette.Purple}, {float64(max(d.Memory, 0)), previewAdd}})
	h.drawKeepSeams(dst, v, bar)
	ui.Stroke(dst, bar.Inset(-1), ui.Palette.Ink)
}

func (h *HUD) drawKeepSeams(dst *ebiten.Image, v BoardView, bar ui.Rect) {
	cpu, _ := v.Facts.LandTotals()
	at := int64(0)
	for _, k := range v.Facts.Keeps[:max(len(v.Facts.Keeps)-1, 0)] {
		at += k.CPU[1]
		x := bar.X + shareWidth(float64(at), float64(cpu[1]), bar.W)
		ui.Fill(dst, ui.Rect{X: x, Y: bar.Y - 1, W: 1, H: bar.H + 2}, ui.Palette.Ink)
	}
}

type landShare struct {
	amount float64
	clr    color.RGBA
}

var othersFill = color.RGBA{104, 110, 104, 255}

func fillShares(dst *ebiten.Image, r ui.Rect, total float64, shares []landShare) {
	x := r.X
	for _, s := range shares {
		w := min(shareWidth(max(s.amount, 0), total, r.W), r.X+r.W-x)
		if w > 0 {
			ui.Fill(dst, ui.Rect{X: x, Y: r.Y, W: w, H: r.H}, s.clr)
		}
		x += max(w, 0)
	}
}

func bandPieces(v BoardView) []deltaPiece {
	p, hn := v.horn, v.Chain.Horn
	out := []deltaPiece{{ui.Bridge("WAR HORN", "HPA"), ui.Palette.Cream},
		{fmt.Sprintf(ui.Bridge("FLOOR %d CEILING %d", "MIN %d MAX %d"), p.Min, p.Max), ui.Palette.Blue}}
	if hn != nil {
		out = append(out, deltaPiece{fmt.Sprintf(ui.Bridge("NOW %d, CPU %s OF %d%%", "NOW %d, CPU %s OF %d%%"), int(hornNow(v.Chain)), hornUse(hn), hn.Goal), ui.Palette.TextDim})
	}
	if hn != nil && p != FirstHorn(v.Chain) {
		out = append(out, deltaPiece{ui.Bridge("RETUNE TO ORDER IT", "PATCH HPA TO APPLY"), ui.Palette.GoldBright})
	}
	if hn != nil && v.Facts.Bounded() && v.Facts.Limit == view.FlowMemory {
		out = append(out, deltaPiece{ui.Bridge("HOUSING FILLS BEFORE FOOD: THE HORN READS FOOD", "MEMORY BINDS BEFORE CPU: THE HPA READS CPU"), ui.Palette.RedText})
	}
	if wall := v.Wall(); v.Facts.Bounded() && p.Max > wall {
		out = append(out, deltaPiece{fmt.Sprintf(ui.Bridge("THE KEEPS HOLD %d: PAST IT, HUTS WAIT UNHOUSED", "NODES FIT %d: PAST IT, PODS STAY PENDING"), max(wall, 0)), ui.Palette.RedText})
	}
	return out
}

func hornUse(hn *model.Horn) string {
	if !hn.Measured {
		return ui.Bridge("NOT READ YET", "<UNKNOWN>")
	}
	return fmt.Sprintf("%d%%", hn.Utilization)
}
