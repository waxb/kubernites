package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheAgeGoalReadsItsAsksByKindNotFromProse(t *testing.T) {
	f := fullFrame(t)
	f.State.AgeUp = view.AgeUpView{NextAge: view.AgeFeudal, Requirements: []view.Requirement{
		{Kind: view.RequireNine, Met: true, Latched: true, Text: "Held 90% ready over a full window: done, and it stays done (now 73.9% fresh)"},
		{Kind: view.RequirePort, Text: "Needs a second Gateway: the harbor has one"},
		{Kind: view.RequireGold, Gold: 5000, Text: "Hold 4 minutes of your steady throughput (19.6 requests a second, without bursts or your own carrying), at least 5,000: now 5,000, set once every other line is met."},
		{Kind: view.RequireWisdom, Met: true, Text: "Hold 65 points"},
		{Kind: view.RequireChronicle, Text: "Write at least one incident report"},
	}}
	got := []string{}
	for _, a := range ageAsks(f) {
		got = append(got, a.label)
	}
	want := []string{"One Nine (held)", "a caravel port", "4 minutes of your income (now 5,000 gold)", "wisdom", "a Chronicle"}
	if !slices.Equal(got, want) {
		t.Fatalf("the goal reads %q, want %q (a latched ask says it was held; a live size says it is live)", got, want)
	}
}

func ageGoalCase(nine, port, gold, wisdom, chronicle, set bool, price float64) view.AgeUpView {
	return view.AgeUpView{NextAge: view.AgeFeudal, Requirements: []view.Requirement{
		{Kind: view.RequireNine, Met: nine, Latched: nine},
		{Kind: view.RequirePort, Met: port},
		{Kind: view.RequireGold, Met: gold, Gold: price, Set: set},
		{Kind: view.RequireWisdom, Met: wisdom},
		{Kind: view.RequireChronicle, Met: chronicle},
	}}
}

func TestTheAgeGoalSaysItsPriceIsLiveThenSetInTwoRowsUncut(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	defer ui.SetNamer(nil)
	cases := []struct {
		name string
		view view.AgeUpView
		want string
	}{
		{"every line to do", ageGoalCase(false, false, false, false, false, false, 116545), "4 minutes of your income (now 116,545 gold)"},
		{"gold met while live", ageGoalCase(true, true, true, true, false, false, 116545), "4 minutes of your income (now 116,545 gold)"},
		{"set and to do", ageGoalCase(true, true, false, true, true, true, 116545), "116,545 gold (set)"},
		{"every line done", ageGoalCase(true, true, true, true, true, true, 116545), "116,545 gold (set)"},
	}
	for _, kube := range []bool{false, true} {
		if kube {
			ui.SetNamer(names.Kubernetes(64))
		} else {
			ui.SetNamer(nil)
		}
		for _, c := range cases {
			f.State.AgeUp = c.view
			labels := []string{}
			for _, a := range ageAsks(f) {
				labels = append(labels, a.label)
			}
			if !kube && !slices.Contains(labels, c.want) {
				t.Errorf("%s: the goal reads %q, want %q", c.name, labels, c.want)
			}
			lines := h.ageGoalLines(f, Objective.W-16)
			for _, l := range lines {
				if strings.HasSuffix(l.Text, "...") {
					t.Errorf("names %v, %s: the goal is cut: %q", kube, c.name, l.Text)
				}
			}
			if len(lines) > 1+2*ageGoalRows {
				t.Errorf("names %v, %s: the goal takes %d rows", kube, c.name, len(lines))
			}
		}
	}
}
