package hud

import (
	"strings"
	"testing"
)

func TestTheLensInksOnlyWhatTheListsHaveCountedAndTheIslandRisesOnlyWhileTheRealmIsDrawn(t *testing.T) {
	p := &LivePicker{Step: LiveStepOpening, Progress: LiveProgress{List: "nodes", Total: 22}}
	p.Advance(5 * surveyFPS)
	if len(p.survey.inked) != 0 {
		t.Errorf("before any list counted a thing the lens inked %v", p.survey.inked)
	}
	p.Progress = LiveProgress{List: "pods", Count: 500, Done: 3, Total: 22, Nodes: 3, Namespaces: 9, Pods: 58}
	p.Advance(30 * surveyFPS)
	marks := surveyMarks(p.Progress)
	kinds := map[byte]int{}
	for _, m := range marks {
		kinds[m.kind]++
		if _, ok := p.survey.inked[m]; !ok {
			t.Errorf("after 30 s of reading the lens never passed over %c %d", m.kind, m.i)
		}
	}
	if kinds['k'] != 3 || kinds['b'] != surveyBanners || kinds['h'] != surveyHuts {
		t.Errorf("marks %v, want 3 keeps, %d banners, %d huts", kinds, surveyBanners, surveyHuts)
	}
	if len(p.survey.inked) != len(marks) {
		t.Errorf("the lens inked %d marks, the lists counted %d", len(p.survey.inked), len(marks))
	}
	if p.survey.rolled() != 0 || p.survey.risen() != 0 {
		t.Errorf("while reading: rolled %v, risen %v", p.survey.rolled(), p.survey.risen())
	}
	p.Progress = LiveProgress{Step: LiveDrawing, List: "events", Done: 22, Total: 22, Nodes: 3, Namespaces: 9, Pods: 58}
	rolled, risen := 0.0, 0.0
	for range surveyRollUp + surveyRise + 10 {
		p.Advance(1)
		if p.survey.rolled() < rolled || p.survey.risen() < risen {
			t.Fatalf("the parchment unrolled or the island sank: rolled %v after %v, risen %v after %v", p.survey.rolled(), rolled, p.survey.risen(), risen)
		}
		if p.survey.risen() > 0 && p.survey.rolled() < 1 {
			t.Fatalf("the island rose (%v) before the parchment rolled up (%v)", p.survey.risen(), p.survey.rolled())
		}
		rolled, risen = p.survey.rolled(), p.survey.risen()
	}
	if rolled != 1 || risen != 1 {
		t.Errorf("after the roll and the rise: rolled %v, risen %v", rolled, risen)
	}
	caption := strings.Join(LiveProgress{List: "pods", Count: 500, Done: 2, Total: 21, Nodes: 3}.Caption(), "\n")
	for _, want := range []string{"Reading pods: 500 so far", "2 of 21 lists", "3 nodes"} {
		if !strings.Contains(caption, want) {
			t.Errorf("caption %q lacks %q", caption, want)
		}
	}
}

func TestTheLoadingCaptionSaysWhatHappensAfterTheLists(t *testing.T) {
	done := LiveProgress{List: "events", Done: 41, Total: 41, Nodes: 3, Namespaces: 15, Pods: 58}
	cases := map[string]string{"": "Every list is read", LiveReadingUsage: "Every list is read: reading usage from metrics-server", LiveDrawing: "Every list is read: drawing the realm"}
	for step, want := range cases {
		p := done
		p.Step = step
		if got := p.Caption()[0]; got != want {
			t.Errorf("step %q reads %q, want %q", step, got, want)
		}
	}
}

func TestAnOpeningThatFailsWhileDrawingSaysTheRealmCouldNotBeDrawn(t *testing.T) {
	read := LiveProgress{List: "events", Done: 41, Total: 41, Nodes: 3, Namespaces: 15, Pods: 58}
	cases := map[string]string{
		"":               "THE CLUSTER DID NOT ANSWER",
		LiveReadingUsage: "THE CLUSTER DID NOT ANSWER",
		LiveDrawing:      "THE REALM COULD NOT BE DRAWN",
	}
	for step, want := range cases {
		p := read
		p.Step = step
		if got := p.FailTitle(); got != want {
			t.Errorf("failed at step %q: the title reads %q, want %q", step, got, want)
		}
	}
	if got := (LiveProgress{List: "pods", Count: 12, Done: 3, Total: 41}).FailTitle(); got != "THE CLUSTER DID NOT ANSWER" {
		t.Errorf("failed while reading pods: the title reads %q", got)
	}
}
