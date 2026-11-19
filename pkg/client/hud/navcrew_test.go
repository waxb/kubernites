package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func crewSelFrame(sel Selection) Frame {
	f := crewFrame(sel, allReveals...)
	f.State.Unlocks[HornUnlock] = true
	return f
}

func crewNavFrames(t *testing.T) map[string]Frame {
	t.Helper()
	stall := depotBoardFrame()
	stall.State.Reveals = append(stall.State.Reveals, allReveals...)
	stall.State.Unlocks = map[string]bool{HornUnlock: true}
	keep := selectStructure(crewSelFrame(Selection{}), view.Structure{ID: "keep-1", Kind: view.StructKeep, State: view.StateReady})
	return map[string]Frame{
		"nothing": crewSelFrame(Selection{}), "keep": keep, "stall": stall, "crew": crewSelFrame(crewSel()),
		"hut":     crewSelFrame(Selection{Kind: "hearth", ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")}),
		"foreman": crewSelFrame(Selection{Kind: SelectForeman, ID: "default/porters-7d9f"}),
	}
}

var (
	gridCrewTargets  = []string{ScaleUpButton, ScaleDownButton, CrewModeOpen + CrewModeRoll, "crew:roll:restart", CrewModeOpen + CrewModeImage, CrewModeOpen + CrewModeHorn}
	boardCrewTargets = []string{StepUpButton, StepDownButton, SliderButton}
)

func crewNavCases(t *testing.T) map[string][]string {
	t.Helper()
	cases := map[string][]string{}
	for name, f := range crewNavFrames(t) {
		targets := []string{CmdCrew}
		for _, r := range stallRows(f) {
			targets = append(targets, StallTarget(r.Kind))
		}
		switch name {
		case "hut", "foreman":
			targets = append(targets, ChainButton)
			targets = append(targets, gridCrewTargets...)
			targets = append(targets, boardCrewTargets...)
		case "crew":
			targets = append(targets, gridCrewTargets...)
			targets = append(targets, boardCrewTargets...)
		case "stall":
			targets = append(targets, ChainButton)
			targets = append(targets, boardCrewTargets...)
		}
		cases[name] = targets
	}
	return cases
}

func TestCrewTargetsAreReachableFromEverySelection(t *testing.T) {
	frames := crewNavFrames(t)
	for name, targets := range crewNavCases(t) {
		f := frames[name]
		for _, target := range targets {
			for _, page := range []string{PageMain, PageBuild, PageSummon} {
				h := New(nil, (&fakeRunner{}).backend())
				h.Update(Input{}, f)
				h.PageID = page
				if err := walkTo(t, h, f, target); err != nil {
					t.Errorf("%s from a %s selection on page %s: %v", target, name, page, err)
				}
			}
		}
	}
}
