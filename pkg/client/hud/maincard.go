package hud

import (
	"slices"
	"strings"
)

const (
	PageMore            = "more"
	CmdMore             = "cmd:more"
	mainCells           = 9
	moreCell            = 8
	CrewServesObjective = "crew-serves"
	CmdSummonID         = "cmd:summon"
	MoreCell            = -1
	ActsNote            = "Click: acts at once."
	MenuNote            = "Click: opens a menu (...)."
)

var moreButton = Button{ID: CmdMore, Label: "MORE", Icon: "icon/describe", Tip: "More commands: those the guide does not need now\n" + MenuNote, Menu: true}

var MainHomes = map[string]int{
	CmdCrew: 0, "cmd:build": 1, "cmd:door": 2,
	"cmd:grimoire": 3, "cmd:soulvault": 4, "cmd:chronicles": 5, "cmd:age": 6, "cmd:siege": 7, CmdSummonID: 8,
}

var mainRank = []string{"cmd:build", CmdCrew, "cmd:siege", "cmd:grimoire", "cmd:soulvault", "cmd:age",
	"cmd:chronicles", "cmd:door", CmdSummonID}

var openingToys = []string{"cmd:door", CmdSummonID}

type mainSlot struct {
	button Button
	more   bool
}

func (h *HUD) mainSlots(f Frame) []mainSlot {
	out := []mainSlot{}
	for _, e := range mainEntries {
		if f.Revealed(e.reveal) && f.Shown(e.button.ID) {
			out = append(out, mainSlot{button: mainButton(e.button, f), more: e.button.ID == CmdSummonID && crewTookOver(f)})
		}
	}
	if siegeKnown(f) && f.Shown("cmd:siege") {
		out = append(out, mainSlot{button: menuMarked(SiegeButton(f))})
	}
	if DoorOffered(f) {
		out = append(out, mainSlot{button: WithClickNote(DoorButton)})
	}
	for _, b := range offerButtons(f) {
		out = append(out, mainSlot{button: b})
	}
	return out
}

func mainButton(base Button, f Frame) Button {
	b := base
	b.Tip = withEffect(base.Tip, f.Effect(base.ID))
	if b.ID == "cmd:summon" {
		b.Tip = summonTip(f, strings.TrimSpace(base.Tip+"\n"+f.Effect(base.ID)))
	}
	if b.ID == "cmd:grimoire" {
		b = withPageWay(b, f)
	}
	return WithClickNote(b)
}

func menuMarked(b Button) Button {
	next := b
	next.Menu = true
	return WithClickNote(next)
}

func WithClickNote(b Button) Button {
	next := b
	if next.Menu {
		next.Tip += "\n" + MenuNote
	} else {
		next.Tip += "\n" + ActsNote
	}
	return next
}

func guideWants(f Frame) []string {
	out := []string{}
	for _, goal := range []string{introTarget(f), ObjectiveGoal(f), NextGoal(f)} {
		for _, id := range NavPath(goal, NavState{Page: PageMain}) {
			if p := placeOf(id); p.page == PageMain && p.overlay == "" && !slices.Contains(out, id) {
				out = append(out, id)
			}
		}
	}
	return out
}

func introTarget(f Frame) string {
	if f.Guide.Intro == nil {
		return ""
	}
	return f.Guide.Intro.Target
}

func crewTookOver(f Frame) bool {
	for _, o := range f.State.Objectives {
		if o.ID == CrewServesObjective {
			return o.Done
		}
	}
	return false
}

func slotRank(id string, wanted []string, opening bool) int {
	if i := slices.Index(wanted, id); i >= 0 {
		return i - len(wanted) - len(openingToys)
	}
	if i := slices.Index(openingToys, id); opening && i >= 0 {
		return i - len(openingToys)
	}
	if i := slices.Index(mainRank, id); i >= 0 {
		return i
	}
	return len(mainRank)
}

func claimOrder(slots []mainSlot, wanted []string, opening bool) []mainSlot {
	out := slices.Clone(slots)
	slices.SortStableFunc(out, func(a, b mainSlot) int {
		return slotRank(a.button.ID, wanted, opening) - slotRank(b.button.ID, wanted, opening)
	})
	return out
}

func layoutMain(slots []mainSlot, wanted []string, opening bool) ([]Button, []Button) {
	taken := map[int]Button{}
	overflow := []Button{}
	for _, s := range claimOrder(slots, wanted, opening) {
		home, homed := MainHomes[s.button.ID]
		_, busy := taken[home]
		if !homed || busy || s.more && !slices.Contains(wanted, s.button.ID) {
			overflow = append(overflow, s.button)
			continue
		}
		taken[home] = placedAt(s.button, home)
	}
	if len(overflow) > 0 {
		cell := moreHome(taken, wanted, opening)
		if evicted, ok := taken[cell]; ok {
			overflow = append(overflow, evicted)
		}
		taken[cell] = placedAt(moreButton, cell)
	}
	main := make([]Button, 0, len(taken))
	for _, b := range taken {
		main = append(main, b)
	}
	return byCell(main), moreOrder(overflow)
}

func moreHome(taken map[int]Button, wanted []string, opening bool) int {
	if b, ok := taken[moreCell]; !ok || !slices.Contains(wanted, b.ID) {
		return moreCell
	}
	cell, worst := moreCell, -1<<30
	for c, b := range taken {
		if r := slotRank(b.ID, wanted, opening); r > worst && !slices.Contains(wanted, b.ID) {
			cell, worst = c, r
		}
	}
	return cell
}

func moreOrder(buttons []Button) []Button {
	out := slices.Clone(buttons)
	slices.SortStableFunc(out, func(a, b Button) int {
		return slices.Index(mainRank, b.ID) - slices.Index(mainRank, a.ID)
	})
	return out
}

func placedAt(b Button, cell int) Button {
	next := b
	next.Rect = MainCell(cell)
	return next
}

func byCell(buttons []Button) []Button {
	out := slices.Clone(buttons)
	slices.SortStableFunc(out, func(a, b Button) int { return mainIndex(a) - mainIndex(b) })
	return out
}

func mainIndex(b Button) int {
	for i := range mainCells {
		if MainCell(i) == b.Rect {
			return i
		}
	}
	return mainCells
}

func (h *HUD) MainCard(f Frame) ([]Button, []Button) {
	return layoutMain(h.mainSlots(f), guideWants(f), !crewTookOver(f))
}

func (h *HUD) MainCells(f Frame) map[string]int {
	main, more := h.MainCard(f)
	out := map[string]int{}
	for _, b := range main {
		if b.ID != CmdMore {
			out[b.ID] = mainIndex(b)
		}
	}
	for _, b := range more {
		out[b.ID] = MoreCell
	}
	return out
}
