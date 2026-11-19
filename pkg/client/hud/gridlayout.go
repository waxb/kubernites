package hud

import (
	"fmt"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var (
	keepCells        = map[string]int{KeyDescribe: 0, KeyTop: 1, KeyLabel: 2, KeyTaint: 3, KeyCordon: 4, KeyUncordon: 4, KeyDrain: 5, keySummon: 6, KeyWarden: 7}
	hutCells         = map[string]int{KeyDescribe: 0, KeyYAML: 1, KeyPrevious: 2, KeyLabel: 3, KeyLogs: 4, keyReseat: 5, keyRollAdd: 6, KeyDescribeDeed: 7, KeyDelete: 8}
	depotCells       = map[string]int{KeyDescribe: 0, keyEndpoints: 1, keyLookFor: 2, keyRollStrike: 5, keyTearUp: 6, KeyDelete: 8}
	codexCells       = map[string]int{KeyStatus: 1, KeyAlarms: 2, KeyCompact: 3, KeyDefrag: 4, KeyDisarm: 5, KeyEtcdVer: 6}
	chanceryCells    = map[string]int{KeyGet: 1, keyEvents: 2, keyVersion: 3, keyGetPods: 4, keyCluster: 5, keyResources: 6}
	observatoryCells = map[string]int{keyTopNodes: 1, keyTopPods: 4}
)

var actionIcons = map[string]string{
	KeyDescribe: "icon/describe", KeyTop: "icon/probe", keyTopPods: "icon/probe", keyTopNodes: "icon/probe", KeyLabel: "icon/decree",
	KeyTaint: "icon/rating-resilience", KeyCordon: "icon/pause", KeyUncordon: "icon/speed", KeyDrain: "icon/overlay-unhoused", keySummon: "icon/summon",
	KeyYAML: "icon/yaml", KeyPrevious: "icon/chronicle", KeyLogs: "icon/logs", keyReseat: "icon/roll", KeyDelete: "icon/close",
	keyEndpoints: "icon/service", KeyStatus: "icon/codex", KeyAlarms: "icon/alert", KeyCompact: "icon/stone", KeyDefrag: "icon/thread",
	KeyDisarm: "icon/key-gate", KeyEtcdVer: "icon/tech", KeyGet: "icon/describe", keyGetPods: "icon/pod", keyEvents: "icon/crier",
	keyVersion: "icon/tech", keyCluster: "icon/node", keyResources: "icon/archive",
	KeyWarden: "icon/summon", KeyDescribeDeed: "icon/describe", KeyGetDeed: "icon/describe",
	keyRollAdd: "icon/service", keyRollStrike: "icon/alert", keyLookFor: "icon/decree", keyTearUp: "icon/close",
}

func actionCell(a CardAction, f Frame) Button {
	id := ActionPrefix + ":" + a.Key
	return Button{ID: id, Label: a.Label, Icon: actionIcons[a.Key], Tip: ActionTip(a), Disabled: a.Disabled,
		Selected: f.Guide.Button == id || incidentLit(f, id), Menu: len(a.Choices) > 0}
}

func keyed(acts []CardAction, cells map[string]int, f Frame) []gridSlot {
	out := []gridSlot{}
	for _, a := range acts {
		if cell, ok := cells[a.Key]; ok {
			out = append(out, gridSlot{cell: cell, button: actionCell(a, f)})
		}
	}
	return out
}

func occupied(slots []gridSlot, cell int) bool {
	for _, s := range slots {
		if s.cell == cell && s.button.ID != "" {
			return true
		}
	}
	return false
}

func withSlot(slots []gridSlot, cell int, b Button, ok bool) []gridSlot {
	if !ok || occupied(slots, cell) {
		return slots
	}
	return append(append([]gridSlot(nil), slots...), gridSlot{cell: cell, button: b})
}

func detailsCell(f Frame) (Button, bool) {
	_, ok := targetOf(f.Selection)
	return Button{ID: detailsButton, Label: detailsLabel, Icon: "icon/describe", Tip: "kubectl describe: open its details [double click]", Selected: incidentLit(f, detailsButton)}, ok
}

func logsCell(f Frame) (Button, bool) {
	return Button{ID: logsButton, Label: "LOGS", Icon: "icon/logs", Tip: "kubectl logs: what the worker inside says", Selected: incidentLit(f, logsButton)}, HasLogs(f.Selection)
}

func landCell(f Frame, shown bool) (Button, bool) {
	return Button{ID: LandButton, Label: "LAND", Icon: "icon/node", Tip: ui.Bridge(landTip, landTipK8s), Selected: shown},
		f.Land && f.Selection.Kind == "structure"
}

func (h *HUD) hutSlots(f Frame) []gridSlot {
	slots := keyed(SelectionActions(f), hutCells, f)
	details, ok := detailsCell(f)
	slots = withSlot(slots, 0, details, ok)
	logs, ok := logsCell(f)
	slots = withSlot(slots, CentreCell, logs, ok)
	chain, ok := h.chainCell(f)
	return withSlot(slots, 6, chain, ok)
}

func (h *HUD) chainCell(f Frame) (Button, bool) {
	c, ok := ChainOf(f)
	if !ok {
		return Button{}, false
	}
	label, icon, tip := ui.Bridge("CREW", "DEPLOYMENT"), "icon/roll", ui.Bridge(chainTip, chainTipK8s)
	switch {
	case c.Bare:
		label, tip = ui.Bridge("SHIFT", "REPLICASET"), ui.Bridge(bareTip, bareTipK8s)
	case f.Selection.Kind == "structure":
		label, icon, tip = ui.Bridge("GUILD ROLL", "ENDPOINTS"), "icon/service", ui.Bridge("Show the depot's own board: its box line fanned out to the huts it feeds. Press again for the crew board.", "Show the Service's endpoints: the ready Pods it sends requests to. Press again for the Deployment board.")
	}
	return Button{ID: ChainButton, Label: label, Icon: icon, Tip: tip, Selected: h.chainSelected(f, c)}, true
}

func (h *HUD) structureSlots(f Frame) []gridSlot {
	s := f.Selection.Structure
	acts := SelectionActions(f)
	slots := keyed(acts, structureCells(s), f)
	if !HasDescribe(acts) {
		details, ok := detailsCell(f)
		slots = withSlot(slots, 0, details, ok)
	}
	if retire, ok := h.retireButton(f, ui.Rect{}); ok {
		retire.Icon = "icon/rank-low"
		slots = withSlot(slots, BackCell, retire, true)
	}
	if _, ok := depotOfStructure(s); ok {
		slots = h.depotSlots(slots, f)
	}
	for i, cell := range lineCells(s) {
		b, ok := lineCell(f, i)
		slots = withSlot(slots, cell, b, ok)
	}
	land, ok := landCell(f, h.LandShown(f))
	return withSlot(slots, 7, land, ok)
}

func structureCells(s view.Structure) map[string]int {
	switch s.Kind {
	case view.StructKeep:
		return keepCells
	case view.StructCodex:
		return codexCells
	case view.StructChancery:
		return chanceryCells
	case view.StructObservatory:
		return observatoryCells
	case view.StructVault:
		return vaultCells
	}
	return depotCells
}

func lineCells(s view.Structure) []int {
	if s.Kind == view.StructChancery {
		return []int{7}
	}
	return []int{2, 5}
}

func (h *HUD) depotSlots(slots []gridSlot, f Frame) []gridSlot {
	c, ok := ChainOf(f)
	crew := Button{ID: CrewBoardButton, Label: ui.Bridge("CREW", "DEPLOYMENT"), Icon: "icon/roll", Selected: f.Guide.Button == CrewBoardButton,
		Tip: ui.Bridge("Select the "+c.Name+" crew: its board, its slider and its land.", "Select the Deployment "+c.Name+": replicas, requests and Nodes.")}
	slots = withSlot(slots, 3, crew, ok && !c.Bare)
	chain, ok := h.chainCell(f)
	return withSlot(slots, CentreCell, chain, ok)
}

func lineCell(f Frame, i int) (Button, bool) {
	lines := CardLines(f)
	if i >= len(lines) || lines[i].Action == nil {
		return Button{}, false
	}
	l := lines[i]
	o := view.CommandOffer{Label: l.Label, Command: l.Command, Action: *l.Action}
	tip := l.Label + SpeakTip(l.Command)
	return Button{ID: fmt.Sprintf("%s:card:%d:%s", linePrefix, i, DoOp), Label: ShortFixLabel(o), Icon: offerIcon(l.Action.Action), Tip: tip}, true
}

func offerIcon(action string) string {
	if action == "openDepot" {
		return "icon/service"
	}
	return "icon/roll"
}
