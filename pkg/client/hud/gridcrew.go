package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
)

const (
	CrewModeOpen = CrewPrefix + ":mode:"
	crewBack     = CrewModeOpen
)

var (
	crewChainCells = map[string]int{"describe": 0, "yaml": 1, "mode:" + CrewModeRoll: 2, "scale:down": 3, "scale:up": CentreCell,
		"mode:" + CrewModeImage: 5, "mode:" + CrewModeRes: 6, "mode:" + CrewModeHorn: 7, CrewModeProbes: 1}
	rolloutCells = map[string]int{"roll:restart": 0, "roll:undo": 1, "roll:pause": 2, "roll:resume": 2, "history": 3}
	hornCells    = map[string]int{"horn:sound": CentreCell, "horn:retune": CentreCell, "horn:silence": 5}
	modeIcons    = map[string]string{CrewModeRoll: "icon/roll", CrewModeImage: "icon/academy", CrewModeRes: "icon/wood", CrewModeHorn: "icon/warhorn"}
	orderIcons   = map[string]string{"describe": "icon/describe", "yaml": "icon/yaml", "scale:up": "icon/scale-up", "scale:down": "icon/scale-down",
		"roll:restart": "icon/roll", "roll:undo": "icon/back", "roll:pause": "icon/pause", "roll:resume": "icon/speed", "history": "icon/chronicle",
		"res:apply": "icon/apply", "horn:sound": "icon/warhorn", "horn:retune": "icon/warhorn", "horn:silence": "icon/close",
		CrewModeProbes: "icon/probe", "probes:readiness": "icon/probe", "probes:liveness": "icon/alert"}
)

func (h *HUD) crewGrid(f Frame) bool {
	c, ok := ChainOf(f)
	switch {
	case !ok:
		return false
	case hearthSelected(f) || f.Selection.Kind == SelectForeman && !c.Bare:
		return h.ChainShown(f)
	case c.Bare:
		return f.Selection.Kind == SelectForeman || f.Selection.Kind == SelectCrew
	}
	return f.Selection.Kind == SelectCrew
}

func orderKey(b Button) string {
	return strings.TrimPrefix(b.ID, CrewPrefix+":")
}

func gridOrder(b Button, cell int) gridSlot {
	next := b
	key := orderKey(b)
	if next.Icon == "" {
		next.Icon = orderIcons[key]
	}
	if mode, ok := strings.CutPrefix(key, "mode:"); ok {
		next.Icon, next.Menu = modeIcons[mode], true
	}
	return gridSlot{cell: cell, button: next}
}

func ordersIn(buttons []Button, cells map[string]int) []gridSlot {
	out := []gridSlot{}
	for _, b := range buttons {
		if cell, ok := cells[orderKey(b)]; ok {
			out = append(out, gridOrder(b, cell))
		}
	}
	return out
}

func (h *HUD) crewSlots(f Frame) []gridSlot {
	c, _ := ChainOf(f)
	if c.Bare {
		return h.bareSlots(c, f)
	}
	ro := h.modeRoom(c, f)
	switch mode := h.CrewMode(c); mode {
	case CrewModeRoll:
		return withCrewBack(h.withMakeRoom(ordersIn(withRoom(rolloutButtons(c, f), ro), rolloutCells), c, f))
	case CrewModeImage:
		return withCrewBack(h.withMakeRoom(classSlots(withRoom(classButtons(c, f), ro)), c, f))
	case CrewModeRes:
		return withCrewBack(h.withMakeRoom([]gridSlot{gridOrder(h.applyButton(crewView{chain: c, mode: mode}, f), CentreCell)}, c, f))
	case CrewModeHorn:
		return withCrewBack(ordersIn(hornOrders(c, h.Crew.Horn), hornCells))
	case CrewModeProbes:
		return withCrewBack(h.withMakeRoom(ordersIn(append(withRoom(probeButtons(c, f), ro), yamlOrder(c, f)...), probeCells), c, f))
	}
	slots := ordersIn(gridOrders(c, f), crewChainCells)
	return append(slots, gridSlot{cell: BackCell, button: h.crewHome(f)})
}

func gridOrders(c model.CrewChain, f Frame) []Button {
	out := []Button{}
	for _, b := range DeploymentButtons(c, f) {
		if strings.HasPrefix(b.ID, CrewPrefix+":roll:") || b.ID == CrewPrefix+":history" || b.ID == crewYAML && watchKnown(f) {
			continue
		}
		out = append(out, b)
	}
	if watchKnown(f) {
		out = append(out, watchOrder(f))
	}
	if rolloutsKnown(c, f) {
		out = append(out, order("mode:"+CrewModeRoll, "ROLLOUT", "Restart, undo or pause the crew's rollout.", "", f))
	}
	return out
}

func classSlots(buttons []Button) []gridSlot {
	out := []gridSlot{}
	for i, b := range buttons[:min(len(buttons), BackCell)] {
		next := b
		next.Icon = "icon/academy"
		out = append(out, gridSlot{cell: i, button: next})
	}
	return out
}

var crewBackTip = "Back to the crew's commands [Esc or " + backKey + "]"

func withCrewBack(slots []gridSlot) []gridSlot {
	back := Button{ID: crewBack, Label: "BACK", Icon: "icon/back", Tip: crewBackTip}
	return append(slots, gridSlot{cell: BackCell, button: back})
}

func (h *HUD) crewHome(f Frame) Button {
	if f.Selection.Kind == SelectCrew {
		return homeButton()
	}
	return Button{ID: ChainButton, Label: "BACK", Icon: "icon/back", Tip: homeTipCrew}
}

func (h *HUD) foremanSlots(f Frame) []gridSlot {
	slots := []gridSlot{}
	if details, ok := detailsCell(f); ok {
		slots = append(slots, gridSlot{cell: 0, button: details})
	}
	chain, ok := h.chainCell(f)
	return withSlot(slots, 7, chain, ok)
}

func (h *HUD) crewPage(f Frame) bool {
	c, ok := ChainOf(f)
	return ok && h.crewGrid(f) && h.CrewMode(c) != CrewModeChain
}
