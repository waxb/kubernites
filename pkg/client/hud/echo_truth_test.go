package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func echoFrame(command string) Frame {
	return Frame{State: view.GameState{Snapshot: view.Snapshot{Objectives: []view.ObjectiveView{{ID: "pledges", Command: command}}}}}
}

func TestATroubleCardHidesOnlyBehindAStepThatSpeaksItsOwnFix(t *testing.T) {
	step := "kubectl set resources deployment porters --requests=cpu=300m"
	unhoused := view.CalamityView{ID: "unhoused:porters", Kind: view.TagUnhoused,
		Remedies: []view.RemedyView{{Label: "Pledge less", Command: "kubectl set resources deployment porters -c=porter --requests=cpu=250m"}}}
	if EchoesObjective(unhoused, echoFrame(step)) {
		t.Fatal("an UNHOUSED card is hidden for as long as the pledge step stands because its own fix also sets resources")
	}
	same := view.CalamityView{ID: "spoiling:parcel", Kind: view.CalamitySpoiling,
		Remedies: []view.RemedyView{{Label: "Scale", Command: "kubectl scale deployment porters --replicas=13"}}}
	if EchoesObjective(same, echoFrame("kubectl scale deployment porters --replicas=6")) {
		t.Fatal("a card scaling to 13 is hidden behind a step scaling to 6")
	}
	if !EchoesObjective(same, echoFrame("kubectl scale deployment porters --replicas=13")) {
		t.Fatal("a card whose fix is the step's own command is shown twice")
	}
}
