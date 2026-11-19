package hud

import (
	"fmt"
	"strconv"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	stepperValueW  = 44
	stepperLabelW  = 150
	MaxCrewSize    = 100
	hornOverrules  = "\nThe War Horn sets this crew's count: it will overrule you."
	scaleNoRoomTip = "\nNo Keep has room for another hut: it would wait Unhoused (Pending) until a Keep frees room or you buy one."
	noRoomStep     = "NO ROOM"
	useGap         = 6
)

func order(id, label, tip, cmd string, f Frame) Button {
	full := CrewPrefix + ":" + id
	b := Button{ID: full, Label: label, Tip: tip, Selected: f.Guide.Button == full}
	b.Tip += SpeakTip(cmd)
	return b
}

func CommandOf(a view.Action) string {
	cmd, err := view.CrewActionCommand(a)
	if err != nil {
		return ""
	}
	return cmd
}

func DeploymentButtons(c model.CrewChain, f Frame) []Button {
	if c.Bare {
		return inspectButtons(c, f)
	}
	out := []Button{}
	if f.Revealed(crewRevealID) {
		out = append(out, ScaleButtons(c, f)...)
	}
	if f.Revealed(vaultRevealID) && len(NewerClasses(f, c.Image)) > 0 {
		out = append(out, order("mode:"+CrewModeImage, "UPGRADE", "Roll a newer class out: a new shift (ReplicaSet) takes the huts over one by one.", "", f))
	}
	if len(knownFields(f)) > 0 {
		out = append(out, order("mode:"+CrewModeRes, "RESOURCES", ui.Bridge("Set what each worker pledges and may use: the Pledge and the Binding decide its standing.", "Set what each Pod requests and may use: requests and limits decide its QoS class."), "", f))
	}
	if rolloutsKnown(c, f) {
		out = append(out, rolloutButtons(c, f)...)
	}
	if f.State.Unlocks[HornUnlock] {
		out = append(out, order("mode:"+CrewModeHorn, ui.Bridge("WAR HORN", "AUTOSCALE"), "Let a War Horn (HorizontalPodAutoscaler) set the crew's count from CPU use.", "", f))
	}
	return append(out, inspectButtons(c, f)...)
}

func scaleCommand(c model.CrewChain, dir string) string {
	if a, ok := crewAction(c, "scale", dir); ok {
		return CommandOf(a)
	}
	kind, name := CrewTarget(c)
	return fmt.Sprintf("kubectl scale %s %s --replicas=%d", kind, name, max(c.Desired+scaleStep(dir), 0))
}

func ScaleButtons(c model.CrewChain, f Frame) []Button {
	less := order("scale:down", "SCALE -1", "One hut fewer: the shift sends a hut home.", scaleCommand(c, "down"), f)
	less.Disabled = c.Desired == 0
	more := order("scale:up", "SCALE +1", "One hut more: the shift raises another hut.", scaleCommand(c, "up"), f)
	more.Disabled = c.Desired >= MaxCrewSize
	facts := CrewFactsOf(f, c)
	more.Cost, more.Tip = scaleRoom(facts, more.Cost, more.Tip)
	if charterCrossed(facts.CharterAt, c.Desired, c.Desired+1) {
		more.Tip += "\n" + charterLine(facts.CharterAt)
	}
	if c.Horn != nil {
		less.Tip += hornOverrules
		more.Tip += hornOverrules
	}
	return []Button{less, more}
}

func rolloutsKnown(c model.CrewChain, f Frame) bool {
	return f.Revealed(vaultRevealID) || c.Revision >= rolloutRevived
}

func rolloutButtons(c model.CrewChain, f Frame) []Button {
	roll := func(verb, label, tip string) Button {
		a, _ := crewAction(c, "roll", verb)
		return order("roll:"+verb, label, tip, CommandOf(a), f)
	}
	out := []Button{roll(view.RolloutRestart, "RESTART", "Raise every hut anew, one by one, with the same class.")}
	if len(c.History()) > 0 {
		out = append(out, roll(view.RolloutUndo, "UNDO", "Return to the previous revision: the old shift is called back."))
	}
	if c.Paused {
		out = append(out, roll(view.RolloutResume, "RESUME", "Let the paused rollout move again."))
	} else {
		out = append(out, roll(view.RolloutPause, "PAUSE", "Hold the rollout where it stands: template changes wait."))
	}
	return append(out, order("history", "HISTORY", "The revisions this crew remembers.", CrewInspectCommand(c, "history"), f))
}

func inspectButtons(c model.CrewChain, f Frame) []Button {
	kind, _ := CrewTarget(c)
	return []Button{
		order("describe", "DESCRIBE", "What the Chancery knows of this "+kind+": strategy, ReplicaSets and events.", CrewInspectCommand(c, "describe"), f),
		order("yaml", "YAML", "The "+kind+" as it is declared.", CrewInspectCommand(c, "yaml"), f),
	}
}

func classButtons(c model.CrewChain, f Frame) []Button {
	out := []Button{}
	for _, image := range NewerClasses(f, c.Image) {
		a, _ := crewAction(c, "image", image)
		out = append(out, order("image:"+image, image, withPayoffLine("Roll "+image+" out to the crew.", rollOutPayoff(f.State, image)), CommandOf(a), f))
	}
	return out
}

func (h *HUD) applyButton(v crewView, f Frame) Button {
	a, ok := ResourcesAction(v.chain, h.Crew.Res, f)
	b := order("res:apply", "APPLY", "Write the new requests and limits into the crew's template: a rollout follows.", CommandOf(a), f)
	switch {
	case PledgeOverBinding(h.Crew.Res):
		b.Disabled, b.Tip = true, "A request may not exceed its limit: raise the limit or lower the request."
	case !ok:
		b.Disabled, b.Tip = true, "Nothing changed yet: step a request or a limit first."
	default:
		b.Tip += roomTipLine(h.modeRoom(v.chain, f))
	}
	return b
}

func hornOrders(c model.CrewChain, p HornPick) []Button {
	if c.Horn == nil {
		return []Button{HornOrder(c, p)}
	}
	retune := view.Action{Action: view.ActRetuneHorn, Name: c.Name, Crew: view.CrewOrder{Min: p.Min, Max: p.Max, CPU: p.CPU}}
	b := order("horn:retune", "RETUNE", "Patch the War Horn with this Floor, Ceiling and Pitch (kubectl patch hpa).", CommandOf(retune), Frame{})
	b.Disabled = p == FirstHorn(c)
	if b.Disabled {
		b.Tip = "Step the Floor, Ceiling or Pitch first: RETUNE patches the War Horn with them."
	}
	return []Button{b, HornOrder(c, p)}
}

func HornOrder(c model.CrewChain, p HornPick) Button {
	if c.Horn != nil {
		a := view.Action{Action: view.ActSilenceHorn, Name: c.Name}
		return order("horn:silence", "SILENCE", "Take the War Horn down: the crew keeps its last count.", CommandOf(a), Frame{})
	}
	a := view.Action{Action: view.ActAutoscaleCrew, Name: c.Name, Crew: view.CrewOrder{Min: p.Min, Max: p.Max, CPU: p.CPU}}
	return order("horn:sound", "SOUND", "Raise the War Horn with this Floor, Ceiling and Pitch.", CommandOf(a), Frame{})
}

func (h *HUD) stepperButtons(v crewView) []Button {
	keys := []string{}
	switch {
	case v.mode == CrewModeRes:
		for _, rf := range v.fields {
			keys = append(keys, "res:"+rf.key)
		}
	case v.mode == CrewModeHorn:
		keys = []string{"horn:min", "horn:max", "horn:cpu"}
	}
	out := []Button{}
	for i, key := range keys {
		y := v.body.Y + i*crewRowStep
		x := v.body.X + stepperLabelW
		less, more := "Less", "More"
		if i < len(v.fields) && v.mode == CrewModeRes {
			effect := ui.Bridge(v.fields[i].story, v.fields[i].k8s)
			less, more = "Less. "+effect, "More. "+effect
		}
		out = append(out,
			Button{ID: CrewPrefix + ":" + key + ":down", Label: "-", Rect: ui.Rect{X: x, Y: y, W: stepperW, H: crewRowH}, Tip: less},
			Button{ID: CrewPrefix + ":" + key + ":up", Label: "+", Rect: ui.Rect{X: x + stepperW + stepperValueW, Y: y, W: stepperW, H: crewRowH}, Tip: more})
		if b, ok := useButton(v, i, key, x+2*stepperW+stepperValueW+useGap, y); ok {
			out = append(out, b)
		}
	}
	return out
}

func useButton(v crewView, i int, key string, x, y int) (Button, bool) {
	if v.mode != CrewModeRes || i >= len(v.fields) {
		return Button{}, false
	}
	rf := v.fields[i]
	n := rf.measured(v.peak)
	if n <= 0 {
		return Button{}, false
	}
	label := ui.Bridge("USE "+rf.icon()+strconv.FormatInt(n, 10), "USE "+quantity(n, rf.unit))
	b := Button{ID: CrewPrefix + ":" + key + ":" + useSuffix, Label: label, Rect: ui.Rect{X: x, Y: y, W: ui.Font{}.Width(label, 1) + crewTextPad, H: crewRowH}, Tip: useTip(rf, n, v.peak)}
	if rf.key == "cpu-req" && v.peak.Waits > 0 {
		b.Disabled, b.Tip = true, fmt.Sprintf(ui.Bridge("At this pledge %s Unhoused once the crew rolls over: make room first, with a Keep or fewer huts.",
			"At these requests %s Pending after the rollout: add room first."), upToHutsWait(v.peak.Waits))
	}
	return b, true
}

func useTip(rf resField, n int64, p view.UsePeakView) string {
	count := strconv.FormatInt(n, 10)
	switch rf.key {
	case "cpu-lim":
		return ui.Bridge("Raise the Binding to the most a worker can eat: "+count+". Today the workers hit their Binding at the noon peak and slow down.",
			"Raise limits.cpu to what a Pod can use: "+quantity(n, rf.unit)+". Today the Pods hit their limit at the peak and are throttled.")
	case "mem-req":
		return ui.Bridge("Set the pledge to the most rooms a worker took in the last day: "+count+", rounded up to a step.",
			"Set the request to the most memory a Pod used in the last day: "+quantity(n, rf.unit)+", rounded up to a step.")
	}
	tip := ui.Bridge("Set the pledge to the most a worker ate in the last day, the noon ships included: "+count+", rounded up to a step.",
		"Set the request to the most CPU a Pod used in the last day, the noon peak included: "+quantity(n, rf.unit)+", rounded up to a step.")
	if p.Bound {
		tip += ui.Bridge(" They reach their Binding then: raise the Binding first.", " They reach their limit then: raise the limit first.")
	}
	return tip
}

func scaleRoom(facts CrewFacts, cost, tip string) (string, string) {
	switch room := facts.Fed; {
	case facts.Seats <= 0:
		return summonNoRoom, tip + scaleNoRoomTip
	case room <= 0:
		return shortCost(facts.Limit), tip + "\n" + shortTip(facts)
	case room < OpenLand && facts.Seats > room:
		return cost, tip + fmt.Sprintf(ui.Bridge("\nYour Keeps have room for %d more at their real use, %d by pledge.", "\n%d more fit at real use, %d by requests."), room, facts.Seats) + " " + shortTip(facts)
	case room < OpenLand:
		return cost, tip + fmt.Sprintf(ui.Bridge("\nYour Keeps have room for %d more, %d after this one.", "\nThe Nodes fit %d more, %d after this one."), room, room-1)
	}
	return cost, tip
}
