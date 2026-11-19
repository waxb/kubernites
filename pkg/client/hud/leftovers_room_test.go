package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheMoreButtonCountsRoomTheWayTheLandLineDoes(t *testing.T) {
	_, tip := scaleRoom(CrewFacts{Fed: 8, Seats: 8}, "", "One hut more")
	if !strings.Contains(tip, "room for 8 more, 7 after this one") {
		t.Fatalf("the tip reads %q: the land line says 7 once the hut is taken", tip)
	}
}

func TestTheHungryLineNamesWhatTopShows(t *testing.T) {
	facts := CrewFacts{Fed: 6, Seats: 7, Takes: view.ResourcePair{CPU: 4}, Use: useFacts{CPU: 260, Pods: 10}}
	_, tip := scaleRoom(facts, "", "One hut more")
	if strings.Contains(tip, "about 4 rations") || !strings.Contains(tip, "about 26 rations each") || !strings.Contains(tip, "TOP HUTS") || !strings.Contains(tip, "6 more") {
		t.Errorf("'each eats about 4 rations when the ships are busiest' beside TOP HUTS' 26 rations each; now %q", tip)
	}
	facts.Fed = 0
	if cost, tip := scaleRoom(facts, "", ""); cost != "HUNGRY" || !strings.Contains(tip, "about 26 rations each") || !strings.Contains(tip, "can feed no more") {
		t.Errorf("the HUNGRY button's tip reads %q", tip)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if _, tip := scaleRoom(facts, "", ""); !strings.Contains(tip, "26m CPU each") || !strings.Contains(tip, "kubectl top") {
		t.Errorf("NAMES mode reads %q", tip)
	}
	facts.Limit, facts.Takes.Memory = view.FlowMemory, 64*mebi
	if _, tip := scaleRoom(facts, "", ""); strings.Contains(tip, "rooms") || !strings.Contains(tip, "64Mi") {
		t.Errorf("NAMES mode's memory line reads %q", tip)
	}
}
