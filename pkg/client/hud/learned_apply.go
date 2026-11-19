package hud

import (
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	applyPrefix = "apply"
	applyW      = 96
)

var applyModes = map[string]string{"watch": CrewModeProbes, "bell": CrewModeProbes, "pledges": CrewModeRes, "bindings": CrewModeRes, "war-horns": CrewModeHorn}

type pendingMode struct {
	crew string
	mode string
}

func applyButton(p KMapPlaque, r ui.Rect) (Button, bool) {
	mode, ok := applyModes[p.Page.ID]
	if !ok || p.Fog || p.Page.Rank < 1 {
		return Button{}, false
	}
	tip := ui.Bridge("Close the Grimoire and open a crew's ", "Close the Grimoire and open a Deployment's ") + modeWord(mode) +
		ui.Bridge(": a learned page does nothing until a crew wears it.", ": the field does nothing until it is set on a workload.")
	return Button{ID: applyPrefix + ":" + p.Page.ID, Rect: r, Label: ui.Bridge("ON A CREW", "SET IT"), Tip: tip}, true
}

func modeWord(mode string) string {
	switch mode {
	case CrewModeProbes:
		return ui.Bridge("WATCH", "PROBES")
	case CrewModeHorn:
		return ui.Bridge("WAR HORN", "AUTOSCALE")
	}
	return "RESOURCES"
}

func (h *HUD) pressApply(page string, f Frame) []Intent {
	mode, ok := applyModes[page]
	if !ok {
		return nil
	}
	c, found := applyCrew(f, mode)
	if !found {
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	h.OverlayID, h.pending = OverlayNone, pendingMode{crew: c.ID(), mode: mode}
	sel := CrewSelection(c)
	return []Intent{{Kind: IntentChoose, Target: sel.Kind, Command: sel.ID}, {Kind: IntentSound, Command: "click"}}
}

func applyCrew(f Frame, mode string) (model.CrewChain, bool) {
	want := troubledCrew(f, mode)
	var first model.CrewChain
	found := false
	for _, c := range f.Crews {
		if !crewKept(c) || c.Bare {
			continue
		}
		if c.Name == want {
			return c, true
		}
		if !found {
			first, found = c, true
		}
	}
	return first, found
}

func troubledCrew(f Frame, mode string) string {
	if mode != CrewModeProbes {
		return ""
	}
	for _, v := range f.State.Calamities {
		if v.Mended != "" {
			continue
		}
		_, id, _ := cutKind(v.ID)
		switch v.Kind {
		case view.CalamityWarm:
			return id
		case view.CalamityFrozen:
			return crewOfHut(f, id)
		}
	}
	return ""
}

func crewOfHut(f Frame, hut string) string {
	for _, p := range f.Pods {
		if p.Name == hut {
			return p.Crew
		}
	}
	return ""
}

func cutKind(id string) (string, string, bool) {
	for i := range len(id) {
		if id[i] == ':' {
			return id[:i], id[i+1:], true
		}
	}
	return id, "", false
}

func (h *HUD) applyPending(f Frame) {
	if h.pending.crew == "" {
		return
	}
	c, ok := ChainOf(f)
	if !ok || c.ID() != h.pending.crew {
		return
	}
	h.SetCrewMode(c, h.pending.mode, f)
	h.pending = pendingMode{}
}

func init() { prefixHandlers[applyPrefix] = (*HUD).pressApply }
