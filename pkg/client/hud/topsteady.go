package hud

import (
	"reflect"
	"slices"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

type shownStat struct {
	value string
	stat  Stat
	at    float64
}

type steadyTop struct {
	stats map[string]shownStat
	clock string
	at    float64
	board steadyBoard
}

type steadyBoard struct {
	set     bool
	freight view.FreightView
	economy view.EconomyView
	at      float64
}

func (h *HUD) noteSteadyBoard(raw Frame) {
	b := &h.top.board
	if h.live != nil {
		*b = steadyBoard{}
		return
	}
	live, econ := raw.State.Freight, raw.State.Economy
	if raw.liveBoard != nil {
		live, econ = raw.liveBoard.freight, raw.liveBoard.economy
	}
	fv, e := heldFreight(live, b.freight), heldEconomy(econ, b.economy)
	same := reflect.DeepEqual(fv, live) && reflect.DeepEqual(e, econ)
	if !b.set || !same && h.topMayTick(b.at) {
		*b = steadyBoard{set: true, freight: live, economy: econ, at: h.Now}
	}
}

func (h *HUD) withSteadyBoard(f Frame) Frame {
	if b := h.top.board; b.set && h.live == nil {
		if f.liveBoard == nil {
			f.liveBoard = &steadyBoard{freight: f.State.Freight, economy: f.State.Economy}
		}
		f.State.Freight, f.State.Economy = heldFreight(f.State.Freight, b.freight), heldEconomy(f.State.Economy, b.economy)
	}
	return f
}

func heldEconomy(live, held view.EconomyView) view.EconomyView {
	out := live
	out.Income, out.Upkeep, out.Profit, out.Hand = held.Income, held.Upkeep, held.Profit, held.Hand
	out.Arriving, out.Carried, out.Lost = held.Arriving, held.Carried, held.Lost
	return out
}

func heldFreight(live, held view.FreightView) view.FreightView {
	out := live
	out.Fresh, out.TierProgress, out.Waiting = held.Fresh, held.TierProgress, held.Waiting
	out.Docking.Last, out.Docking.LastCargo, out.Docking.Next = held.Docking.Last, held.Docking.LastCargo, held.Docking.Next
	out.Losses = held.Losses
	out.Seals.Waiting, out.Seals.ArrivalRate, out.Seals.ServedRate, out.Seals.SpoiledRate, out.Seals.Losses =
		held.Seals.Waiting, held.Seals.ArrivalRate, held.Seals.ServedRate, held.Seals.SpoiledRate, held.Seals.Losses
	if len(live.Kinds) != len(held.Kinds) {
		return out
	}
	out.Kinds = slices.Clone(live.Kinds)
	for i, k := range held.Kinds {
		if k.Kind != out.Kinds[i].Kind {
			return live
		}
		c := &out.Kinds[i]
		c.Waiting, c.Oldest, c.ArrivalRate, c.ServedRate, c.SpoiledRate, c.Sealed, c.Freshness = k.Waiting, k.Oldest, k.ArrivalRate, k.ServedRate, k.SpoiledRate, k.Sealed, k.Freshness
	}
	return out
}

func statKey(st Stat) string { return st.Meter + " " + st.icon + " " + st.Id }

func statValue(st Stat) string {
	return st.Text + "|" + st.Rate + "|" + st.Upkeep + "|" + st.Net + "|" + st.Floor
}

func statShape(st Stat) [4]bool {
	return [4]bool{st.Rate != "", st.Upkeep != "", st.Net != "", st.Floor != ""}
}

func (h *HUD) topMayTick(at float64) bool {
	return h.Now-at >= CounterTick || h.Now-h.attn.pressed <= attnAnswerWindow
}

func (h *HUD) noteSteadyTop(f Frame) {
	if h.live != nil {
		h.top = steadyTop{}
		return
	}
	next := map[string]shownStat{}
	for _, st := range h.rawTopStats(f) {
		key := statKey(st)
		shown, ok := h.top.stats[key]
		switch {
		case ok && statValue(st) == shown.value:
		case ok && statShape(st) == statShape(shown.stat) && !h.topMayTick(shown.at):
		default:
			shown = shownStat{value: statValue(st), stat: st, at: h.Now}
		}
		next[key] = shown
	}
	h.top.stats = next
	clock, _ := h.rawClockSlot(f)
	if clock != h.top.clock && (h.top.clock == "" || h.topMayTick(h.top.at)) {
		h.top.clock, h.top.at = clock, h.Now
	}
}

func (h *HUD) TopStats(f Frame) []Stat {
	out := h.rawTopStats(f)
	for i, st := range out {
		shown, ok := h.top.stats[statKey(st)]
		if !ok || statShape(st) != statShape(shown.stat) {
			continue
		}
		s := shown.stat
		out[i].Text, out[i].clr, out[i].Rate, out[i].RateClr = s.Text, s.clr, s.Rate, s.RateClr
		out[i].Upkeep, out[i].UpkeepClr, out[i].Net, out[i].netClr, out[i].Floor = s.Upkeep, s.UpkeepClr, s.Net, s.netClr, s.Floor
	}
	return out
}

func (h *HUD) ClockSlot(f Frame) (string, ui.Rect) {
	clock, slot := h.rawClockSlot(f)
	if h.top.clock != "" && h.live == nil && h.Font.Width(h.top.clock, 1) <= slot.W {
		return h.top.clock, slot
	}
	return clock, slot
}
