package app

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAStepStaysOnTheCardTenSecondsBeforeItsAdviceChanges(t *testing.T) {
	climb := view.ObjectiveView{ID: view.ClerkClimbObjective, Text: "Pull clerk class 3.0 from the registry.", Progress: "Pull clerk:3.0 from the registry: 6925/7100 MiB"}
	lever := climb
	lever.Progress, lever.Button = "Scale the clerks to 14 to serve faster: about 2 min", "crew:scale:up"
	h := stepHold{}
	shown, changes := "", 0
	for at := 0.0; at < 20; at += 2 {
		v := climb
		if int(at/2)%2 == 1 {
			v = lever
		}
		got := h.steady([]view.ObjectiveView{v}, at)[0].Progress
		if got != shown {
			shown, changes = got, changes+1
		}
	}
	if changes > 3 {
		t.Fatalf("the step's line changed %d times in 20 s (it once changed five times in 20 s)", changes)
	}
	h.release()
	if got := h.steady([]view.ObjectiveView{lever}, 21)[0].Progress; got != lever.Progress {
		t.Fatalf("after the player acts the card still shows %q", got)
	}
	next := view.ObjectiveView{ID: "decree", Text: "Write a crew in the Scriptorium."}
	if got := h.steady([]view.ObjectiveView{next}, 22)[0]; got.ID != "decree" {
		t.Fatalf("a new step waits behind the old one: %+v", got)
	}
}

func TestAStepThatMovesOnToANewLineShowsItAtOnce(t *testing.T) {
	learn := view.ObjectiveView{ID: "depot-badge", Text: "Let the Stewards keep the roll.", Progress: "ready: read about Badges in the manual", Target: "cmd:grimoire"}
	label := learn
	label.Progress, label.Target = "ready: mark the parcels depot with app=porters", "cmd:stone"
	h := stepHold{}
	h.steady([]view.ObjectiveView{learn}, 0)
	if got := h.steady([]view.ObjectiveView{label}, 2)[0].Progress; got != label.Progress {
		t.Fatalf("the step moved on, yet the card still says %q (the old line 'read about Badges in the manual' stayed for 4 s after the step moved on)", got)
	}
}
