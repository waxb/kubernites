package hud

import (
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	SurfaceTidings     = "surface:tidings"
	SurfaceMinimap     = "surface:minimap"
	SurfaceStone       = "surface:stone"
	SurfaceScriptorium = "surface:scriptorium"
	SurfaceRail        = "surface:rail"
	SurfaceGate        = "surface:gate"
	SurfaceLedger      = "surface:ledger"
	SurfaceSpeed       = "surface:speed"
	SurfaceDecree      = "surface:" + view.BorgIntro
	MeterGold          = "meter:gold"
	MeterUpkeep        = "meter:upkeep"
	MeterWisdom        = "meter:wisdom"
	MeterRenown        = "meter:renown"
	MeterNines         = "meter:nines"
	MeterHands         = "meter:hands"
	MeterLore          = "meter:lore"
	MeterCodex         = "meter:codex-bloat"
	SurfaceOverlays    = "surface:overlays"
	SurfaceHeraldry    = "surface:heraldry"
	SurfaceQoS         = "surface:qos"
	AckAction          = "ack"
	introPrefix        = "intro"
)

type Intro struct {
	ID      string
	Title   string
	Purpose string
	Lore    string
	Try     string
	Target  string
	Via     string
}

const (
	ViaAct  = "act"
	ViaOpen = "open"
	ViaAck  = "ack"
)

var openers = map[string][]string{SurfaceStone: {"tab:" + TabKube, "well:stone", "obj:speak"}, SurfaceScriptorium: {"tab:" + TabYAML, "well:yaml"}}

func Opens(target, id string) bool {
	if target == id || target == CmdDepot && strings.HasPrefix(id, stallPrefix+":") {
		return true
	}
	for _, o := range openers[target] {
		if o == id {
			return true
		}
	}
	return false
}

func (h *HUD) IntroTried(f Frame, id string) []Intent {
	in := f.Guide.Intro
	if in == nil || in.Via != ViaOpen || !Opens(in.Target, id) {
		return nil
	}
	return []Intent{act(Ack(in.ID))}
}

type Guide struct {
	Staged     bool
	Intro      *Intro
	Introduced []string
	Why        string
	Target     string
	Button     string
	Effects    map[string]string
	Queue      []string
	Held       string
	Last       *view.ObjectiveView
}

var quietTargets = []string{}

func (f Frame) Shown(id string) bool {
	g := f.Guide
	if !g.Staged {
		return true
	}
	return f.Teaching(id) || f.GoalCalls(id) || slices.Contains(g.Introduced, id) || slices.Contains(quietTargets, id)
}

func (f Frame) GoalCalls(id string) bool {
	alias := NavAliases[id]
	next := f.readyNext()
	if f.Guide.Held != "" {
		next = ""
	}
	for _, goal := range []string{f.Guide.Target, f.Guide.Button, next} {
		if goal != "" && (goal == id || goal == alias || opensToward(goal, id)) {
			return true
		}
		if _, _, port := PortTarget(goal); port && id == SurfaceBerths {
			return true
		}
	}
	return false
}

func (f Frame) readyNext() string {
	n := f.State.Next
	switch {
	case n == nil || !n.Ready || n.Command != "" || !f.Revealed("freight"):
		return ""
	case n.Build != "":
		return "build:" + n.Build
	}
	return n.Target
}

func opensToward(goal, id string) bool {
	if goal == CmdDepot {
		goal = StallTarget("")
	}
	path := NavPath(goal, NavState{Page: PageMain})
	return len(path) > 1 && slices.Contains(path[:len(path)-1], id)
}

func (f Frame) goalOnSelection() bool {
	if f.Guide.Target == "" {
		return false
	}
	if HavenSelected(f) {
		return f.GoalCalls(SurfaceBerths) || strings.HasPrefix(f.Guide.Target, portPrefix)
	}
	return SelectionMatches(f.Selection, f.Guide.Target)
}

func (f Frame) Teaching(id string) bool {
	if f.Guide.Intro == nil {
		return false
	}
	target := f.Guide.Intro.Target
	return target == id || TargetOnPage(target, PageMain) == id
}

func (f Frame) Introducing() bool { return f.Guide.Intro != nil }

func (f Frame) railQuiet() bool {
	return f.Introducing() && f.Guide.Intro.Target != SurfaceRail
}

func Passive(in Intro) bool {
	if in.Via != "" {
		return in.Via == ViaAck
	}
	return !strings.HasPrefix(in.Target, "cmd:")
}

func Ack(id string) view.Action { return view.Action{Action: AckAction, ID: id} }

func (f Frame) Effect(id string) string { return f.Guide.Effects[id] }

var mapPrefixes = []string{"pod:", "structure:", "keep:", "haven:", "pile:", "stallspot:", "pv:", "deed:", "sts:"}

const crewPlacePrefix = "crew:"

func CrewPlace(target string) (string, bool) {
	rest, ok := strings.CutPrefix(target, crewPlacePrefix)
	if !ok {
		return "", false
	}
	_, name, found := strings.Cut(rest, "/")
	return name, found && name != ""
}

func MapTarget(target string) bool {
	if _, crew := CrewPlace(target); crew {
		return true
	}
	for _, p := range mapPrefixes {
		if strings.HasPrefix(target, p) {
			return true
		}
	}
	return false
}

func withEffect(tip, effect string) string {
	if effect == "" {
		return tip
	}
	return effect + "\n" + tip
}
