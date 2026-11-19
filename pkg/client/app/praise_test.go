package app

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAFirstStepRingsTheVictoryBellAndARepeatTheSmallOne(t *testing.T) {
	if fx, _ := ProclamationSound(view.Proclamation{Kind: "lesson", Major: true}); fx != sound.SFXVictory {
		t.Fatalf("a first step rings %v", fx)
	}
	if fx, ok := ProclamationSound(view.Proclamation{Kind: "lesson"}); !ok || fx != sound.SFXUnlock {
		t.Fatalf("a repeat step rings %v %v", fx, ok)
	}
}

func TestEveryStepEndReadsItsTitleAndItsWisdom(t *testing.T) {
	end, ok := stepEndOf(view.Proclamation{Kind: "info", Text: fulfilledLead + "Cordon Keep node-a (+2 wisdom)", Major: true})
	if !ok || end.title != "Cordon Keep node-a" || end.wisdom != 2 || !end.major || end.skipped {
		t.Fatalf("%+v %v", end, ok)
	}
	end, ok = stepEndOf(view.Proclamation{Kind: "info", Text: "Already done: Open the depot"})
	if !ok || end.title != "Open the depot" || !end.skipped {
		t.Fatalf("%+v %v", end, ok)
	}
	if _, ok := stepEndOf(view.Proclamation{Kind: "info", Text: "The War Horn sounds"}); ok {
		t.Fatal("a horn is not a step")
	}
	if got := TickText([]stepEnd{{title: "Open the depot", skipped: true}}); got != "Already done: Open the depot" {
		t.Fatalf("tick %q", got)
	}
}
