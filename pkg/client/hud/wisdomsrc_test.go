package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

func studyFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Study = view.StudyView{Every: 23, Progress: 14, Earned: 3, Objective: 2}
	return f
}

func TestWisdomTipNamesServiceAndObjectives(t *testing.T) {
	tip := WisdomTip(studyFrame(t))
	for _, want := range []string{"+1 every 23 boxes delivered", "9 to go", "+2 for each objective met", "+3", "Chronicle"} {
		if !strings.Contains(tip, want) {
			t.Errorf("the tip %q lacks %q", tip, want)
		}
	}
}

func TestLedgerShowsServiceAsAWisdomSource(t *testing.T) {
	f := studyFrame(t)
	var study LedgerNode
	for _, n := range LedgerNodes(f) {
		if n.ID == nodeStudy {
			study = n
		}
	}
	lines := strings.Join(study.Lines, " | ")
	if study.ID == "" || !strings.Contains(lines, "/ 23 delivered") || !strings.Contains(lines, "per objective") {
		t.Fatalf("service node = %+v", study)
	}
	for _, n := range LedgerNodes(f) {
		for _, l := range append([]string{n.Title}, n.Lines...) {
			if w, _ := (ui.Font{Literal: true}).Measure(lexicon.TrueNames(l), 1); (n.ID == nodeStudy || n.ID == NodeWisdom) && w > LedgerNodeW-12 {
				t.Errorf("%s line %q is too wide with Kubernetes names", n.ID, lexicon.TrueNames(l))
			}
		}
	}
	edges := LedgerEdges(LedgerNodes(f))
	into, from := false, false
	for _, e := range edges {
		into = into || e.From == nodeCrews && e.To == nodeStudy
		from = from || e.From == nodeStudy && e.To == NodeWisdom
	}
	if !into || !from {
		t.Fatalf("service is not wired from the crews to the points meter: %+v", edges)
	}
	staged := stagedFrame(t, MeterGold, "cmd:serve")
	for _, n := range LedgerNodes(staged) {
		if n.ID == nodeStudy {
			t.Fatal("service shows before the points meter is introduced")
		}
	}
}
