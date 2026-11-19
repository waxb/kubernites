package hud

import (
	"fmt"
	"math"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	cbW        = 512
	cbPad      = 6
	cbTitleY   = 5
	cbHeadY    = 18
	cbChipH    = 12
	cbGaugeY   = 34
	cbYardY    = 54
	cbYardH    = 62
	cbSliderY  = 120
	cbSliderH  = 16
	cbDeltaY   = 141
	cbUseY     = 155
	cbLandY    = 169
	cbLandH    = 15
	cbFlagW    = 10
	cbHistW    = 6
	cbCloseW   = 30
	cbTrackGap = 6
	cbGrab     = 4
	cbMaxHist  = 5
	cbPlotFull = 26
	cbPlotMini = 9
)

const (
	stepTemplate    = "+1 +0.00 GOLD/S"
	stepTemplateK8s = "+1 +$0.00/S"
	stepIconCells   = 2
)

type BoardView struct {
	Chain  model.CrewChain `json:"-"`
	stall  string
	Facts  CrewFacts       `json:"-"`
	Pods   []model.PodView `json:"-"`
	Slots  []Slot          `json:"-"`
	Target int             `json:"-"`
	Top    int             `json:"-"`
	Band   bool            `json:"-"`
	horn   HornPick
	box    ui.Rect
	Track  ui.Rect `json:"-"`
	yard   ui.Rect
	land   ui.Rect
	chips  []Button
	Steps  []Button `json:"-"`
	Buy    []Button `json:"-"`
}

func depotSelected(f Frame) bool {
	return f.Selection.Kind == "structure"
}

func stallName(f Frame, c model.CrewChain) string {
	if depotSelected(f) {
		return c.Depot
	}
	return ""
}

func (h *HUD) BoardView(f Frame) (BoardView, bool) {
	c, ok := ChainOf(f)
	if !ok || !h.ChainShown(f) || !crewKept(c) || h.CrewMode(c) != CrewModeChain || h.depotBoardOn(f, c) {
		return BoardView{}, false
	}
	v := BoardView{Chain: c, stall: stallName(f, c), Facts: CrewFactsOf(f, c), Pods: crewPods(c, f), Band: c.Horn != nil, horn: h.Crew.Horn}
	v.Top, v.Target = h.sliderRange(v), h.boardTarget(v)
	v.Slots = BoardSlots(v.Pods, v.Target, max(v.Wall()-placedCount(v.Pods), 0))
	height := cbLandY + cbLandH + cbPad
	v.box = f.placed(PanelCrew, ui.Rect{X: CardBox.X, Y: ConsoleY - crewFloat - height, W: cbW, H: height})
	v.chips = h.foremanChips(v, f)
	v.Steps, v.Track = h.sliderRow(v, f)
	v.yard = ui.Rect{X: v.Track.X, Y: v.box.Y + cbYardY, W: v.Track.W, H: cbYardH}
	v.land = ui.Rect{X: v.box.X + cbPad, Y: v.box.Y + cbLandY, W: cbW - 2*cbPad, H: cbLandH}
	v.Buy = h.buyButtons(v, f)
	if h.live != nil {
		v.Steps, v.Buy = nil, nil
	}
	return v, true
}

func (h *HUD) sliderRange(v BoardView) int {
	if v.Band {
		return bandTop(v.Chain, h.Crew.Horn)
	}
	if h.Board.Drag == DragKnob && h.Board.Crew == v.Chain.ID() && h.Board.Top > 0 {
		return h.Board.Top
	}
	base := max(v.Chain.Desired, len(v.Pods))
	return SliderTop(base, v.Wall()-base)
}

func (v BoardView) Wall() int {
	if !v.Facts.Bounded() {
		return OpenLand
	}
	return v.seatedBase() + v.Facts.Seats
}

func (v BoardView) fedWall() int {
	if !v.Facts.Bounded() {
		return OpenLand
	}
	return v.seatedBase() + min(v.Facts.Fed, v.Facts.Seats)
}

func (v BoardView) seatedBase() int {
	if v.Facts.Legacy {
		return placedCount(v.Pods)
	}
	return v.Chain.Desired - v.Facts.Short
}

func (h *HUD) boardTarget(v BoardView) int {
	c := v.Chain
	switch {
	case v.Band:
		return c.Desired
	case h.Board.Drag == DragKnob && h.Board.Crew == c.ID():
		return h.Board.Pick
	case h.Hovering && (h.Hover.ID == StepUpButton || h.Hover.ID == ScaleUpButton):
		return min(c.Desired+1, MaxCrewSize)
	case h.Hovering && (h.Hover.ID == StepDownButton || h.Hover.ID == ScaleDownButton):
		return max(c.Desired-1, 0)
	}
	return c.Desired
}

func foremanChipText(fm model.Foreman) string {
	return ui.Bridge("SHIFT ", "ReplicaSet ") + fm.Name + fmt.Sprintf(" %d/%d", fm.Ready, fm.Desired)
}

func (h *HUD) foremanChips(v BoardView, f Frame) []Button {
	x, y, right := v.box.X+cbPad, v.box.Y+cbHeadY, v.box.X+v.box.W-cbCloseW
	out := []Button{}
	standing := v.Chain.Standing()
	for _, fm := range standing[:min(len(standing), maxStanding)] {
		w := min(h.Font.Width(foremanChipText(fm), 1)+cbFlagW+4, right-x)
		if w <= cbFlagW+4 {
			break
		}
		out = append(out, Button{ID: CrewPrefix + ":pick:rs:" + fm.Name, Rect: ui.Rect{X: x, Y: y, W: w, H: cbChipH},
			Tip: "Select this shift: the ReplicaSet " + fm.Name + " that keeps its huts.", Selected: f.Selection.Kind == SelectForeman && f.Selection.ID == fm.ID()})
		x += w + crewGap
	}
	history := v.Chain.History()
	for _, fm := range history[:min(len(history), cbMaxHist)] {
		if x+cbHistW > right {
			break
		}
		out = append(out, Button{ID: CrewPrefix + ":pick:hist:" + fm.Name, Rect: ui.Rect{X: x, Y: y, W: cbHistW, H: cbChipH},
			Tip: fmt.Sprintf("Revision %d, kept for ROLLOUT UNDO (revisionHistoryLimit).", fm.Revision)})
		x += cbHistW + 1
	}
	return out
}

func (h *HUD) sliderRow(v BoardView, f Frame) ([]Button, ui.Rect) {
	x0, right, y := v.box.X+cbPad, v.box.X+v.box.W-cbPad, v.box.Y+cbSliderY
	if v.Band {
		b := hornOrders(v.Chain, h.Crew.Horn)[0]
		b.Label = "RETUNE"
		b.Rect = ui.Rect{X: right - h.Font.Width(b.Label, 1) - crewTextPad, Y: y, W: h.Font.Width(b.Label, 1) + crewTextPad, H: cbSliderH}
		return []Button{b}, ui.Rect{X: x0 + cbTrackGap, Y: y, W: b.Rect.X - x0 - 2*cbTrackGap, H: cbSliderH}
	}
	pair := ScaleButtons(v.Chain, f)
	less, more := pair[0], pair[1]
	less.ID, more.ID = StepDownButton, StepUpButton
	less.Selected, more.Selected = false, false
	less.Label, more.Label = stepLabel(v.Chain, f, -1), stepLabel(v.Chain, f, 1)
	lw := max(h.Font.Width(less.Label, 1), h.Font.Width(more.Label, 1), stepTemplateW()) + crewTextPad
	mw := lw
	less.Rect = ui.Rect{X: x0, Y: y, W: lw, H: cbSliderH}
	more.Rect = ui.Rect{X: right - mw, Y: y, W: mw, H: cbSliderH}
	track := ui.Rect{X: less.Rect.X + lw + cbTrackGap, Y: y, W: more.Rect.X - less.Rect.X - lw - 2*cbTrackGap, H: cbSliderH}
	return []Button{less, more}, track
}

func stepTemplateW() int {
	return max(Literal.Width(stepTemplate, 1)+stepIconCells*ui.Advance, Literal.Width(stepTemplateK8s, 1))
}

func stepLabel(c model.CrewChain, f Frame, step int) string {
	label := fmt.Sprintf("%+d ", step)
	if facts := CrewFactsOf(f, c); step > 0 && facts.Seats <= 0 {
		return label + noRoomStep
	} else if step > 0 && facts.Fits <= 0 {
		return label + shortCost(facts.Limit)
	}
	if pay := payoffOf(c, f, step); pay != "" {
		return label + pay
	}
	return label + ui.Bridge("HUT", "REPLICA")
}

func payoffOf(c model.CrewChain, f Frame, step int) string {
	per, ok := ReplicaPayoff(f, c.Name)
	if !ok || math.Round(per.Income*float64(step)*100) == 0 {
		return ""
	}
	return ScalePayoff(c, f, step)
}

func (v BoardView) Waiting() int {
	return CountSlots(v.Slots, SlotWaits)
}

func (h *HUD) buyButtons(v BoardView, f Frame) []Button {
	if v.Waiting() == 0 || !v.Facts.Offered {
		return nil
	}
	o := v.Facts.Offer
	label := ui.Bridge("BUY KEEP ", "ADD NODE ") + Gold(o.Cost) + ", " + Money(Rate(o.UpkeepPerSec)) + "/S"
	w := h.Font.Width(label, 1) + crewTextPad
	b := Button{ID: BuyKeepButton, Label: label, Rect: ui.Rect{X: v.land.X + v.land.W - w, Y: v.land.Y, W: w, H: v.land.H},
		Tip: buyKeepTip(v), Disabled: !o.Available, Selected: f.Guide.Button == BuyKeepButton}
	if !o.Available && o.Reason != "" {
		b.Tip = o.Reason
	}
	return []Button{b}
}

func buyKeepTip(v BoardView) string {
	o := v.Facts.Offer
	act := ui.Bridge("Raise a %s for the %s: the waiting huts move in.", "Add a Node (%s) for %s: the Pending Pods get scheduled.")
	if v.Facts.tooBig() {
		act = ui.Bridge("Raise a %s for the %s: each worker pledges more than it holds.", "Add a Node (%s) for %s: each Pod requests more than it has.")
	}
	lines := []string{KeepOfferLine(o.KeepOffer), fmt.Sprintf(act, o.Name, v.Chain.Name)}
	if pay := offerPayoff(o, v.Chain.Name, v.Facts.Pending, v.Facts.requests()); pay != "" {
		lines = append([]string{pay}, lines...)
	}
	return strings.Join(lines, "\n")
}

type plotMode int

const (
	plotSprites plotMode = iota
	plotIcons
	plotBars
)

func (v BoardView) plots() ([]ui.Rect, plotMode) {
	if v.Top <= 0 {
		return nil, plotSprites
	}
	out := make([]ui.Rect, v.Top)
	for i := range out {
		x0, x1 := SliderX(v.Track, v.Top, float64(i)), SliderX(v.Track, v.Top, float64(i+1))
		out[i] = ui.Rect{X: x0, Y: v.yard.Y, W: x1 - x0, H: v.yard.H}
	}
	switch pw := v.Track.W / v.Top; {
	case pw >= cbPlotFull:
		return out, plotSprites
	case pw >= cbPlotMini:
		return out, plotIcons
	}
	return out, plotBars
}

func (v BoardView) plotKind(i int) string {
	switch {
	case i < len(v.Slots):
		return v.Slots[i].Kind
	case i < v.Wall():
		return PlotOpen
	}
	return PlotNoLand
}

func (h *HUD) BoardButtons(v BoardView) []Button {
	out := []Button{closeButtonAs(v.box, CrewPrefix+":close", "Deselect [Esc or right click]")}
	out = append(out, v.chips...)
	out = append(out, v.slotButtons()...)
	out = append(out, Button{ID: SliderButton, Rect: ui.Rect{X: v.Track.X - cbGrab, Y: v.Track.Y - cbGrab, W: v.Track.W + 2*cbGrab, H: v.Track.H + 2*cbGrab},
		Tip: sliderTip(v)})
	out = append(out, v.Steps...)
	return append(out, v.Buy...)
}

func sliderTip(v BoardView) string {
	if v.Band {
		return ui.Bridge("Drag the Floor or the Ceiling of the War Horn's band, then RETUNE.", "Drag minReplicas or maxReplicas, then PATCH HPA.")
	}
	tip := ui.Bridge("Drag to choose the count the crew keeps; let go to order it.", "Drag to set the replicas; release to run kubectl scale.")
	if wall := v.Wall(); wall < v.Top {
		tip += ui.Bridge(fmt.Sprintf("\nPast the red mark (%d), new huts wait Pending: no Keep has room for their Pledges.", wall),
			fmt.Sprintf("\nPast the red mark (%d), new Pods stay Pending: no Node has room for their requests.", wall))
	}
	return tip + "\n$ kubectl scale deployment " + v.Chain.Name + " --replicas=N"
}

func (v BoardView) slotButtons() []Button {
	rects, _ := v.plots()
	out := []Button{}
	for i, r := range rects {
		if i >= len(v.Slots) || !v.Slots[i].Real {
			continue
		}
		s := v.Slots[i]
		out = append(out, Button{ID: CrewPrefix + ":pick:pod:" + s.Pod.Name, Rect: r, Tip: "Select the hut " + s.Pod.Name + " (Pod)"})
	}
	return out
}
