package hud

import (
	"fmt"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	MakeRoomButton = CrewPrefix + ":makeroom"
	makeRoomLabel  = "MAKE ROOM FIRST"
	makeRoomCell   = 7
)

func hutsWord(n int) string {
	if n == 1 {
		return ui.Bridge("1 hut", "1 Pod")
	}
	return fmt.Sprintf(ui.Bridge("%d huts", "%d Pods"), n)
}

func RoomNeed(ro *view.RolloutRoomView) string {
	if ro == nil || ro.Huts <= 0 {
		return ""
	}
	have := ui.Bridge("you have %d", "the Nodes have room for %d")
	switch ro.Bound {
	case view.BoundEnd:
		return fmt.Sprintf(ui.Bridge("needs room for %s once it ends; ", "needs room for %s once it completes; ")+have, hutsWord(ro.Huts), ro.Fit)
	case view.BoundCrowd:
		return fmt.Sprintf(ui.Bridge("needs room for %s at what they really take; ", "needs room for %s at their real usage; ")+have, hutsWord(ro.Huts), ro.Fit)
	case view.BoundStranded:
		return fmt.Sprintf(ui.Bridge("needs room for %s off a Keep behind a Ward; ", "needs room for %s off a Node behind a taint; ")+have, hutsWord(ro.Huts), ro.Fit)
	}
	return fmt.Sprintf(ui.Bridge("needs room for %s at once; ", "needs room for %s at once; ")+have, hutsWord(ro.Huts), ro.Fit)
}

func MakeRoomPlan(ro *view.RolloutRoomView, crew string) string {
	switch {
	case ro == nil:
		return ""
	case ro.Settling:
		return ui.Bridge("The huts sent home are still leaving: their room frees in a moment.", "The removed Pods are still terminating: their room frees in a moment.")
	case ro.Rising:
		return ui.Bridge("A Keep is on its way: the roll-out fits once it stands.", "A Node is joining: the rollout fits once it is Ready.")
	case ro.Reserve:
		return ui.Bridge("Place the Keep the reserve sends: the roll-out fits once it stands.", "Add the Node the reserve sends: the rollout fits once it is Ready.")
	case ro.To <= 0:
		return ui.Bridge("Buy a Keep first: no scale-down makes room.", "Add a Node first: no scale-down makes room.")
	case ro.Horned && ro.Shrink == 0:
		return fmt.Sprintf(ui.Bridge("MAKE ROOM FIRST holds the War Horn at Floor %d, Ceiling %d, so the Horn raises no huts while it rolls; once the roll-out ends, a card offers to raise it back.",
			"MAKE ROOM FIRST sets the HPA's minReplicas to %d and maxReplicas to %d, so it adds no Pods during the rollout; once the rollout completes, a card offers to restore them."), ro.Floor, ro.Ceiling)
	case ro.Horned:
		return fmt.Sprintf(ui.Bridge("MAKE ROOM FIRST lowers the War Horn to Floor %d, Ceiling %d: the Horn owns the count and sends %s home; once the roll-out ends, a card offers to raise it back.",
			"MAKE ROOM FIRST lowers the HPA to minReplicas %d, maxReplicas %d: the HPA owns the count and removes %s; once the rollout completes, a card offers to restore them."), ro.Floor, ro.Ceiling, hutsWord(ro.Shrink))
	}
	return fmt.Sprintf(ui.Bridge("MAKE ROOM FIRST scales the %s down by %d, to %d; or buy a Keep.", "MAKE ROOM FIRST scales %s down by %d, to %d; or add a Node."), crew, ro.Shrink, ro.To)
}

func roomCheck(ro *view.RolloutRoomView, crew string) string {
	need := RoomNeed(ro)
	switch {
	case need == "" && (ro == nil || ro.Fits || ro.Text == ""):
		return ""
	case need == "":
		return ui.Bridge("A roll-out would wait: ", "A rollout would stall: ") + ro.Text + "."
	case !ro.Lacks():
		text := ui.Bridge("Before you press: the roll-out ", "Before you apply: the rollout ") + need + "."
		if ro.Dip {
			text += ui.Bridge(" While it rolls, fewer huts carry and the Nines dip.", " During the rollout fewer Pods serve and the SLI dips.")
		}
		return text
	}
	text := ui.Bridge("A roll-out would wait: it ", "A rollout would stall: it ") + need + ". " + MakeRoomPlan(ro, crew)
	if ro.Surgeless != "" {
		text += ui.Bridge(" Or set its EXTRA HUTS to 0: it then sends old huts home before it raises new ones.",
			" Or set maxSurge to 0: the rollout then removes old Pods before it creates new ones.")
	}
	return text
}

func roomTipLine(ro *view.RolloutRoomView) string {
	if need := RoomNeed(ro); need != "" {
		return "\n" + upperFirst(need) + "."
	}
	return ""
}

func makeRoomButton(c model.CrewChain, ro *view.RolloutRoomView, f Frame) (Button, bool) {
	if ro == nil || !ro.Lacks() || ro.To <= 0 {
		return Button{}, false
	}
	a, ok := ro.MakeRoom(c.Name)
	if !ok {
		return Button{}, false
	}
	tip := MakeRoomPlan(ro, c.Name)
	if ro.Surgeless != "" {
		tip += ui.Bridge("\nOr let the roll-out send old huts home first, with no extra huts:", "\nOr roll out with maxSurge 0:") + SpeakTip(ro.Surgeless)
	}
	b := order("makeroom", makeRoomLabel, upperFirst(RoomNeed(ro))+".\n"+tip, CommandOf(a), f)
	b.Icon, b.Page = "icon/scale-down", "unhoused"
	return b, true
}

func crewRoom(f Frame, c model.CrewChain) *view.RolloutRoomView {
	cv, ok := crewByName(f, c.Name)
	if !ok {
		return nil
	}
	return cv.Rollout
}

func (h *HUD) modeRoom(c model.CrewChain, f Frame) *view.RolloutRoomView {
	if h.CrewMode(c) == CrewModeRes {
		if h.Crew.orderRoom != nil && h.Crew.orderRoom.Crew == c.Name {
			return h.Crew.orderRoom
		}
		return nil
	}
	return crewRoom(f, c)
}

func withRoom(buttons []Button, ro *view.RolloutRoomView) []Button {
	line := roomTipLine(ro)
	if line == "" {
		return buttons
	}
	out := make([]Button, len(buttons))
	for i, b := range buttons {
		out[i] = b
		if !b.Disabled {
			out[i].Tip += line
		}
	}
	return out
}

func (h *HUD) withMakeRoom(slots []gridSlot, c model.CrewChain, f Frame) []gridSlot {
	b, ok := makeRoomButton(c, h.modeRoom(c, f), f)
	if !ok {
		return slots
	}
	out := []gridSlot{}
	for _, s := range slots {
		if s.cell != makeRoomCell {
			out = append(out, s)
		}
	}
	return append(out, gridSlot{cell: makeRoomCell, button: b})
}

func (h *HUD) makeRoom(c model.CrewChain, f Frame) []Intent {
	ro := h.modeRoom(c, f)
	if ro == nil {
		return nil
	}
	a, ok := ro.MakeRoom(c.Name)
	if !ok {
		return nil
	}
	return []Intent{ActFrom(a, FromCrewCard)}
}

func (h *HUD) PendingOrder(f Frame) (view.Action, bool) {
	c, ok := ChainOf(f)
	if !ok || h.CrewMode(c) != CrewModeRes {
		return view.Action{}, false
	}
	return ResourcesAction(c, h.Crew.Res, f)
}

func (h *HUD) SetOrderRoom(ro *view.RolloutRoomView) {
	h.Crew.orderRoom = ro
}
