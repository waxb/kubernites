package hud

import (
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	CrewModeProbes = "probes"
	readinessRune  = "readinessProbe"
	livenessRune   = "livenessProbe"
	crewYAML       = CrewPrefix + ":yaml"
)

var probeCells = map[string]int{"probes:" + view.ProbeReadiness: 3, "probes:" + view.ProbeLiveness: CentreCell, "yaml": 1}

type probeKind struct {
	kind, rune, story, k8s, tip string
}

var probeKinds = []probeKind{
	{view.ProbeReadiness, readinessRune, "SHUTTERS", "READINESS",
		ui.Bridge("The Shutters: no box is sent to a hut until it answers that it is ready.", "readinessProbe: the Pod leaves the Service's endpoints until it passes.")},
	{view.ProbeLiveness, livenessRune, "BELL", "LIVENESS",
		ui.Bridge("The Watchman's Bell: a worker that stops answering is restarted.", "livenessProbe: the kubelet restarts a container that fails it.")},
}

func watchKnown(f Frame) bool { return RuneKnown(f, readinessRune) || RuneKnown(f, livenessRune) }

func shuttersOnly(f Frame) bool { return RuneKnown(f, readinessRune) && !RuneKnown(f, livenessRune) }

func bellOnly(f Frame) bool { return RuneKnown(f, livenessRune) && !RuneKnown(f, readinessRune) }

func watchLegend(f Frame) [2]string {
	switch {
	case bellOnly(f):
		return [2]string{"WATCH: the Bell on every hut of the crew", "PROBES: liveness on the pod template"}
	case shuttersOnly(f):
		return [2]string{"WATCH: the Shutters on every hut of the crew", "PROBES: readiness on the pod template"}
	}
	return modeLegends[CrewModeProbes]
}

func crewWatch(f Frame, crew string) view.CrewWatchView {
	if f.State.Watch == nil {
		return view.CrewWatchView{}
	}
	for _, w := range f.State.Watch.Crews {
		if w.Crew == crew {
			return w
		}
	}
	return view.CrewWatchView{}
}

func probeAction(c model.CrewChain, kind string, f Frame) view.Action {
	w := crewWatch(f, c.Name)
	on := kind == view.ProbeReadiness && w.Readiness != nil || kind == view.ProbeLiveness && w.Liveness != nil
	container := w.Container
	if container == "" {
		container = c.Container
	}
	order := view.CrewOrder{Container: container, Probe: kind, Off: on}
	if !on {
		order.Period, order.Threshold, order.Timeout = view.WatchPeriod, view.WatchThreshold, max(w.Patience, view.WatchPatience)
	}
	if !on && kind == view.ProbeLiveness {
		order.Delay = w.Delay
	}
	return view.Action{Action: view.ActCrewProbes, Name: c.Name, Crew: order}
}

func probeButtons(c model.CrewChain, f Frame) []Button {
	out := []Button{}
	for _, p := range probeKinds {
		if !RuneKnown(f, p.rune) {
			continue
		}
		a := probeAction(c, p.kind, f)
		state := "OFF"
		if a.Crew.Off {
			state = "ON"
		}
		out = append(out, order("probes:"+p.kind, ui.Bridge(p.story+": "+state, p.k8s+": "+state), p.tip, CommandOf(a), f))
	}
	return out
}

func watchOrder(f Frame) Button {
	tip := ui.Bridge("Set the Watch on this crew: the Shutters and the Watchman's Bell.", "Add readiness and liveness probes to the Deployment (kubectl patch).")
	switch {
	case bellOnly(f):
		tip = ui.Bridge("Set the Watch on this crew: the Watchman's Bell.", "Add a liveness probe to the Deployment (kubectl patch).")
	case shuttersOnly(f):
		tip = ui.Bridge("Set the Watch on this crew: the Shutters.", "Add a readiness probe to the Deployment (kubectl patch).")
	}
	return order(CrewModeProbes, ui.Bridge("WATCH", "PROBES"), tip, "", f)
}

func yamlOrder(c model.CrewChain, f Frame) []Button {
	for _, b := range inspectButtons(c, f) {
		if b.ID == crewYAML {
			return []Button{b}
		}
	}
	return nil
}

func (h *HUD) watchGoal(goal string, f Frame) string {
	c, ok := ChainOf(f)
	if !ok || h.CrewMode(c) != CrewModeProbes {
		return goal
	}
	if _, ok := makeRoomButton(c, h.modeRoom(c, f), f); ok {
		return MakeRoomButton
	}
	for _, p := range probeKinds {
		if RuneKnown(f, p.rune) && !probeAction(c, p.kind, f).Crew.Off {
			return CrewPrefix + ":" + CrewModeProbes + ":" + p.kind
		}
	}
	return goal
}

func (h *HUD) pressProbes(c model.CrewChain, arg string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	if arg == "" {
		h.SetCrewMode(c, CrewModeProbes, f)
		return []Intent{click}
	}
	return []Intent{ActFrom(probeAction(c, arg, f), FromButton), click}
}
