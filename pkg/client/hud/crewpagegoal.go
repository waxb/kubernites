package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
)

func (h *HUD) crewPageGoal(goal string, f Frame) string {
	mode := strings.TrimPrefix(goal, CrewModeOpen)
	c, ok := ChainOf(f)
	if !ok || mode == "" || h.CrewMode(c) != mode {
		return goal
	}
	if _, ok := makeRoomButton(c, h.modeRoom(c, f), f); ok && mode != CrewModeHorn {
		return MakeRoomButton
	}
	switch mode {
	case CrewModeImage:
		return classGoal(c, f, goal)
	case CrewModeRes:
		return h.applyButton(crewView{chain: c, mode: mode}, f).ID
	case CrewModeHorn:
		return hornOrders(c, h.Crew.Horn)[0].ID
	}
	return goal
}

func classGoal(c model.CrewChain, f Frame, fallback string) string {
	buttons := classButtons(c, f)
	if len(buttons) == 0 {
		return fallback
	}
	want := ""
	if o, ok := f.current(); ok {
		want = o.Command
	}
	for _, b := range buttons {
		if tag := strings.TrimPrefix(b.ID, CrewPrefix+":image:"); want != "" && strings.HasSuffix(strings.TrimSpace(want), tag) {
			return b.ID
		}
	}
	return buttons[0].ID
}
