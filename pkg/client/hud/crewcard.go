package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	SelectCrew    = "crew"
	SelectForeman = "foreman"
	SelectHearth  = "hearth"
	IntentChoose  = "choose"

	CrewPrefix     = "crew"
	CrewModeChain  = ""
	CrewModeImage  = "image"
	CrewModeRes    = "resources"
	CrewModeHorn   = "horn"
	CrewModeRoll   = "rollout"
	serviceKind    = "Service"
	ownerRSPrefix  = "ReplicaSet/"
	crewRevealID   = "crew"
	vaultRevealID  = "soulvault"
	HornUnlock     = "war-horns"
	rolloutRevived = 2
)

type crewCardState struct {
	id        string
	Mode      string          `json:"-"`
	Res       model.Resources `json:"-"`
	Horn      HornPick        `json:"-"`
	orderRoom *view.RolloutRoomView
}

func init() {
	prefixHandlers[CrewPrefix] = (*HUD).pressCrew
}

func ChainOf(f Frame) (model.CrewChain, bool) {
	if !f.cardSelection() {
		return model.CrewChain{}, false
	}
	for _, c := range f.Crews {
		if ChainHolds(c, f.Selection) {
			return c, true
		}
	}
	return model.CrewChain{}, false
}

func ChainHolds(c model.CrewChain, sel Selection) bool {
	switch sel.Kind {
	case SelectCrew:
		return crewKept(c) && c.ID() == sel.ID
	case SelectForeman:
		return foremanOf(c, sel.ID) >= 0
	case SelectHearth, "walker":
		return foremanOf(c, PodForeman(sel.Pod)) >= 0
	case "structure":
		return crewKept(c) && c.Depot != "" && sel.Structure.Object.Kind == serviceKind && sel.Structure.Object.Name == c.Depot
	}
	return false
}

func PodForeman(p model.PodView) string {
	name, ok := strings.CutPrefix(p.Owner, ownerRSPrefix)
	if !ok {
		return ""
	}
	return p.Namespace + "/" + name
}

func foremanOf(c model.CrewChain, id string) int {
	for i, f := range c.Foremen {
		if f.ID() == id {
			return i
		}
	}
	return -1
}

func (h *HUD) CrewMode(c model.CrewChain) string {
	if h.Crew.id != c.ID() {
		return CrewModeChain
	}
	return h.Crew.Mode
}

func (h *HUD) SetCrewMode(c model.CrewChain, mode string, f Frame) {
	h.Crew = crewCardState{id: c.ID(), Mode: mode, Res: c.Resources, Horn: roomyHorn(FirstHorn(c), c, CrewFactsOf(f, c))}
	if mode == CrewModeRes {
		h.Crew.Res = snapResources(c.Resources, f)
	}
}

func (h *HUD) pressCrew(rest string, f Frame) []Intent {
	c, ok := ChainOf(f)
	if !ok {
		return nil
	}
	click := Intent{Kind: IntentSound, Command: "click"}
	verb, arg, _ := strings.Cut(rest, ":")
	switch verb {
	case "close":
		return h.closeChain(f)
	case "pick":
		return []Intent{pickIntent(c, arg), click}
	case "mode":
		h.SetCrewMode(c, arg, f)
		return []Intent{click}
	}
	return h.crewOrder(c, verb, arg, f)
}

func pickIntent(c model.CrewChain, arg string) Intent {
	level, name, _ := strings.Cut(arg, ":")
	switch level {
	case "rs", "hist":
		return Intent{Kind: IntentChoose, Target: SelectForeman, Command: c.Namespace + "/" + name}
	case "pod":
		return Intent{Kind: IntentChoose, Target: SelectHearth, Command: c.Namespace + "/" + name}
	}
	return Intent{Kind: IntentChoose, Target: SelectCrew, Command: c.ID()}
}

func (h *HUD) crewOrder(c model.CrewChain, verb, arg string, f Frame) []Intent {
	switch verb {
	case "describe", "yaml", "history":
		return h.crewInspect(c, verb)
	case "res":
		return h.pressResources(c, arg, f)
	case "horn":
		return h.pressHorn(c, arg)
	case CrewModeProbes:
		return h.pressProbes(c, arg, f)
	case "buykeep":
		return buyKeepFor(c, f)
	case "board":
		return []Intent{{Kind: IntentChoose, Target: SelectCrew, Command: c.ID()}, {Kind: IntentSound, Command: "click"}}
	case "delete":
		return h.askDeleteForeman(c, f)
	case "deploy":
		return deployForeman(c)
	case "makeroom":
		return h.makeRoom(c, f)
	}
	a, ok := crewAction(c, verb, arg)
	if !ok {
		return h.bareOrder(c, verb, arg)
	}
	h.Crew.Mode = CrewModeChain
	return []Intent{ActFrom(a, FromCrewCard)}
}

func crewAction(c model.CrewChain, verb, arg string) (view.Action, bool) {
	base := view.Action{Name: c.Name, Crew: view.CrewOrder{Container: c.Container, Keeper: chainKeeper(c)}}
	switch verb {
	case "scale", "step":
		base.Action, base.Replicas = view.ActScaleCrew, max(c.Desired+scaleStep(arg), 0)
	case "roll":
		base.Action, base.ID = view.ActRolloutCrew, arg
	case "image":
		base.Action, base.Image = view.ActCrewImage, arg
	default:
		return view.Action{}, false
	}
	return base, crewKept(c)
}

func scaleStep(dir string) int {
	if dir == "down" {
		return -1
	}
	return 1
}

func CrewTarget(c model.CrewChain) (string, string) {
	if c.Bare && len(c.Foremen) > 0 {
		return "replicaset", c.Foremen[0].Name
	}
	return "deployment", c.Name
}

func CrewInspectCommand(c model.CrewChain, verb string) string {
	kind, name := CrewTarget(c)
	switch verb {
	case "yaml":
		return "kubectl get " + kind + " " + name + " -o yaml"
	case "history":
		return "kubectl rollout history " + kind + "/" + name
	}
	return "kubectl describe " + kind + " " + name
}

var inspectTabs = map[string]string{"describe": DetailDescribe, "yaml": DetailYAML, "history": detailOutput}

func (h *HUD) crewInspect(c model.CrewChain, verb string) []Intent {
	cmd := CrewInspectCommand(c, verb)
	res, intents := h.RunFrom(cmd, FromCrewCard)
	h.showOutput(CrewSelection(c), inspectTabs[verb], BareCommand(cmd), res)
	return append(intents, Intent{Kind: IntentSound, Command: "click"})
}

func CrewSelection(c model.CrewChain) Selection {
	if c.Bare && len(c.Foremen) > 0 {
		return Selection{Kind: SelectForeman, ID: c.Foremen[0].ID()}
	}
	return Selection{Kind: SelectCrew, ID: c.ID()}
}
