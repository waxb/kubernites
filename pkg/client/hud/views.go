package hud

import (
	"strconv"

	"github.com/waxb/kubernites/pkg/client/model"
)

const (
	LandButton  = "sel:land"
	ChainButton = "sel:chain"
	keyLand     = "land"
	keyChain    = "chain"
	landTip     = "Tint the ground it stands on: its Keep's land, or the common ground no Keep owns."
	landTipK8s  = "Tint its Node's land, or the ground that belongs to no Node."
	chainTip    = "Show the crew's chain: the crew, its shifts and huts, with the crew's commands."
	chainTipK8s = "Show the Deployment, its ReplicaSets and Pods, with its commands."
	bareTip     = "Show the shift's chain: the shift and its huts, with its commands."
	bareTipK8s  = "Show the ReplicaSet and its Pods, with its commands."
)

type viewToggles struct {
	sel   string
	land  string
	Chain string `json:"-"`
}

func (h *HUD) syncViews(f Frame) {
	if f.Selection.ID == h.View.sel {
		return
	}
	h.View.sel, h.View.land = f.Selection.ID, ""
	if c, ok := ChainOf(f); !ok || c.ID() != h.View.Chain {
		h.View.Chain = ""
	}
}

func (h *HUD) LandShown(f Frame) bool {
	return f.Land && f.cardSelection() && h.View.land != "" && h.View.land == f.Selection.ID
}

func (h *HUD) ChainShown(f Frame) bool {
	c, ok := ChainOf(f)
	return ok && (f.Selection.Kind == SelectCrew || h.View.Chain != "" && h.View.Chain == c.ID() || stallBoard(f, c))
}

func stallBoard(f Frame, c model.CrewChain) bool {
	behind := stallCrews(f)
	return len(behind) == 1 && !c.Bare && behind[0].ID() == c.ID()
}

func (h *HUD) chainSelected(f Frame, c model.CrewChain) bool {
	if depotSelected(f) {
		return h.depotBoardOn(f, c)
	}
	return h.ChainShown(f)
}

func (h *HUD) depotBoardOn(f Frame, c model.CrewChain) bool {
	return depotSelected(f) && h.View.Chain == c.ID()
}

func (h *HUD) pressView(rest string, f Frame) ([]Intent, bool) {
	click := []Intent{{Kind: IntentSound, Command: "click"}}
	switch rest {
	case keyLand:
		if f.Land {
			h.View.land = toggled(h.View.land, f.Selection.ID)
		}
		return click, true
	case keyChain:
		if c, ok := ChainOf(f); ok {
			h.View.Chain = toggled(h.View.Chain, c.ID())
			h.SetCrewMode(c, CrewModeChain, f)
		}
		return click, true
	}
	return nil, false
}

func toggled(current, id string) string {
	if current == id {
		return ""
	}
	return id
}

func (h *HUD) closeChain(f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	if depotSelected(f) {
		h.View.Chain = ""
	}
	if f.Selection.Kind == SelectCrew || depotSelected(f) {
		return append(h.leaveObject(f), click)
	}
	h.View.Chain = ""
	h.Crew = crewCardState{}
	return []Intent{click}
}

func Counted(text string, n int) string {
	if n <= 1 {
		return text
	}
	return text + " (x" + strconv.Itoa(n) + ")"
}

func ToastLabel(t Toast) string {
	return Counted(Label(t.Text), t.Count)
}
