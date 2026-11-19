package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func charterFrame() Frame {
	f := bookedFrame(5, 5, 0)
	f.State.Crews[0].CharterAt = 4
	return f
}

func TestEveryPorterScaleToFourSaysTheMerchantsCharterFollows(t *testing.T) {
	f := charterFrame()
	if more := ScaleButtons(f.Crews[0], f)[1]; !strings.Contains(more.Tip, "merchant's charter") {
		t.Errorf("SCALE +1 from three porters to four hides the charter the fourth brings: %q", more.Tip)
	}
	four := view.Action{Action: view.ActScaleCrew, Name: "porters", Replicas: 4}
	if l := withRemedyEffect(CommandLine{Label: "Scale the porters to 4", Action: &four}, f); !strings.Contains(l.Effect, "merchant's charter") || !l.EffectBad {
		t.Errorf("a card's scale to four reads %q (bad %v): it hides the charter", l.Effect, l.EffectBad)
	}
	if line := scaleCharter(f, four); !strings.Contains(line, "more boxes than 4 can carry") {
		t.Errorf("the board's scale to four tells %q", line)
	}
	v := BoardView{Chain: f.Crews[0], Facts: CrewFactsOf(f, f.Crews[0]), Pods: crewPods(f.Crews[0], f), Target: 4}
	got := []string{}
	for _, p := range previewUsePieces(v, BoardDeltaOf(v)) {
		got = append(got, p.Text)
	}
	if line := strings.Join(got, " "); !strings.Contains(line, "A CHARTER FOLLOWS") {
		t.Errorf("dragging the porters to four previews %q", line)
	}
	f.State.Crews[0].Next = 5
	if tip := NextTip(&view.NextView{Kind: view.NextMilestone, ID: "porters", Label: "porters to 5: x2 carry speed"}, f.State); !strings.Contains(tip, "merchant's charter") {
		t.Errorf("the NEXT step to five porters hides the charter the fourth brings: %q", tip)
	}
	f.State.Crews[0].CharterAt = 0
	if line := scaleCharter(f, four); line != "" {
		t.Errorf("with the charter sailed, a scale still tells %q", line)
	}
}
