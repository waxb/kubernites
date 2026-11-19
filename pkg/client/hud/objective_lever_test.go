package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAWaitsLeverAndEstimateShowWholeOnTheObjectiveCard(t *testing.T) {
	waits := []view.ObjectiveView{
		{ID: "two-nines", Text: "Grow the harbor", Purpose: "More ships and busier crews: the next step waits for the work they do",
			Progress: "your crews carry 30 boxes a second for a minute (now 22.9); else it opens in about 5 min",
			Earn:     "Faster: Open Sunward Wharf (2,400 MiB, +19% ships): more boxes to carry.", Hint: "Carry that many boxes each second for a minute and the step opens."},
		{ID: view.PorterClassObjective, Text: "Train the porters to class 2.0", Purpose: "Twice the requests per porter, and the big ships come",
			Progress: "Berth at Frostwater Quay (20 MiB, +8% ships) to add capacity faster: about 6 min",
			Earn:     "Pull porter:2.0 from the registry: 1031/3000 MiB. Memory: the realm adds 8.2 a second"},
		{ID: "one-nine", Text: "Hold 99% ready: deliver 9 in 10 boxes fresh", Purpose: "Nine in ten boxes fresh: the second tier asks no less (an SLO)",
			Progress: "Open Frostwater Quay (150 MiB, +51% ships) to add capacity faster; 81.6% delivered fresh (need 90%)"},
		{ID: "crates", Text: "Grow the harbor", Purpose: "More ships and busier crews: the next step waits for the work they do",
			Progress: "your crews carry 42 boxes a second for a minute (now 28.6); else it opens in about 4 min",
			Earn:     "Meanwhile, ready the next step: pull clerk:2.0 at the registry: 6904/27000 MiB."},
	}
	h := New(nil, Backend{})
	for _, o := range waits {
		f := fullFrame(t)
		f.State.Objectives = []view.ObjectiveView{o}
		for _, l := range h.ObjectiveLines(f) {
			if strings.HasSuffix(l.Text, "...") {
				t.Errorf("%s: the card cuts a line: %q", o.ID, l.Text)
			}
		}
	}
}
