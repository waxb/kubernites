package hud

import (
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const CrewDelete = CrewPrefix + ":delete"

var bareCells = map[string]int{"describe": 0, "yaml": 1, "scale:down": 3, "scale:up": CentreCell}

const setClassCell = 5

func (h *HUD) bareSlots(c model.CrewChain, f Frame) []gridSlot {
	if h.CrewMode(c) == CrewModeImage && classSettable(c, f) {
		return withCrewBack(classSlots(classButtons(c, f)))
	}
	slots := withSlot(ordersIn(append(inspectButtons(c, f), ScaleButtons(c, f)...), bareCells), deployCell, deployButton(c, f), deployable(c, f))
	slots = withSlot(slots, setClassCell, setClassButton(f), classSettable(c, f))
	return append(slots, gridSlot{cell: 7, button: h.bareChain(f)}, gridSlot{cell: BackCell, button: bareDelete(c, f)})
}

func classSettable(c model.CrewChain, f Frame) bool {
	return crewKept(c) && f.Revealed(vaultRevealID) && len(NewerClasses(f, c.Image)) > 0
}

func setClassButton(f Frame) Button {
	b := order("mode:"+CrewModeImage, ui.Bridge("SET CLASS", "SET IMAGE"),
		ui.Bridge("Give the shift a newer class: only the huts it raises from now on take it, and every hut standing keeps its old class.",
			"Set the ReplicaSet's template image: a ReplicaSet never replaces Pods that already run, so only new Pods get it."), "", f)
	b.Icon, b.Menu = modeIcons[CrewModeImage], true
	return b
}

func (h *HUD) bareChain(f Frame) Button {
	switch f.Selection.Kind {
	case SelectCrew:
		return homeButton()
	case SelectHearth, "walker":
		return Button{ID: ChainButton, Label: "BACK", Icon: "icon/back", Tip: homeTipCrew}
	}
	return Button{ID: ChainButton, Label: ui.Bridge("SHIFT", "REPLICASET"), Icon: "icon/roll", Tip: ui.Bridge(bareTip, bareTipK8s), Selected: h.ChainShown(f)}
}

func bareDeleteCommand(c model.CrewChain) string {
	kind, name := CrewTarget(c)
	return "delete " + kind + " " + name
}

func bareDelete(c model.CrewChain, f Frame) Button {
	b := order("delete", "DELETE", ui.Bridge("Send this shift away: its huts go with it, and nothing raises them again.",
		"Delete the ReplicaSet: its Pods are deleted with it (cascading deletion)."), "kubectl "+bareDeleteCommand(c), f)
	b.Icon, b.Menu = "icon/close", true
	return b
}

func (h *HUD) askDeleteForeman(c model.CrewChain, f Frame) []Intent {
	if !c.Bare {
		return nil
	}
	_, name := CrewTarget(c)
	cmd := bareDeleteCommand(c)
	yes := CardAction{Key: cmd, Label: ui.Bridge("Yes, send "+name+" and its huts away", "Yes, delete the ReplicaSet "+name+" and its Pods"),
		Command: cmd, Tip: "kubectl " + cmd}
	h.openChooser(CardAction{Key: "delete", Label: "DELETE", Choices: []CardAction{yes}}, f)
	h.Ops.Chooser.from = FromCrewCard
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) bareOrder(c model.CrewChain, verb, arg string) []Intent {
	if !c.Bare || verb != "scale" && verb != "step" {
		return nil
	}
	res, intents := h.RunFrom(scaleCommand(c, arg), FromCrewCard)
	h.Toast(answerLine(res.Output), !res.OK)
	return append(intents, Intent{Kind: IntentSound, Command: soundFor(res.OK)})
}
