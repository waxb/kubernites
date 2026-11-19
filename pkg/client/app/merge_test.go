package app

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/view"
)

func TestIdenticalTidingsMergeWithACount(t *testing.T) {
	burst := "The gleaming box bursts open: every box delivered is worth x3 for 20 seconds more."
	procs := []view.Proclamation{
		{T: 1, Kind: "info", Text: burst}, {T: 2, Kind: "info", Text: burst}, {T: 3, Kind: "info", Text: burst},
		{T: 4, Kind: "info", Text: "Something else"}, {T: 5, Kind: "info", Text: burst},
	}
	got := Tidings(nil, procs, false, 0)
	if len(got) != 3 || got[0].Text != burst || got[1].Text != "Something else" || got[2].Text != burst+" (x3)" {
		t.Fatalf("tidings %+v", got)
	}
}

func TestTheHintLineGivesTheHint(t *testing.T) {
	procs := []view.Proclamation{{T: 1, Kind: "lesson", Title: "Cordon Keep node-a", Text: "A Keep is a Node. It runs your huts."}}
	got := Tidings(nil, procs, false, 0)
	if len(got) != 1 || got[0].Text != "What just happened: A Keep is a Node." || got[0].Title != "Cordon Keep node-a" {
		t.Fatalf("hint line %+v", got)
	}
	if LessonLine("Cordon Keep node-a", "") != lessonLead+"Cordon Keep node-a" || LessonLine("", "") != lessonLead+"a lesson" {
		t.Fatal("a line without text")
	}
}

func TestIdenticalRailNewsMergeWithACount(t *testing.T) {
	spoiled := "A request expired at the harbor: no hut answered within its deadline (it timed out)."
	procs := []view.Proclamation{
		{T: 1, Kind: "info", Text: "The War Horn sounds: porters to 5"},
		{T: 2, Kind: "info", Text: spoiled},
		{T: 3, Kind: "info", Text: "The War Horn sounds: clerks to 3"},
		{T: 4, Kind: "info", Text: spoiled},
	}
	news := MergeNews(NewsNotices(procs, 0, 1, view.Site{}))
	titles := map[string]hud.Notice{}
	for _, n := range news {
		if _, dup := titles[n.Title]; dup {
			t.Fatalf("two log rows read %q: %+v", n.Title, news)
		}
		titles[n.Title] = n
	}
	horn := titles["The War Horn sounds"]
	if horn.Count != 2 || !strings.Contains(horn.Detail, "porters to 5") || !strings.Contains(horn.Detail, "clerks to 3") {
		t.Fatalf("merged horn notice %+v", horn)
	}
	if lost := titles["A request expired at the harbor"]; lost.Count != 2 || strings.Count(lost.Detail, "expired") != 1 {
		t.Fatalf("merged spoil notice %+v", lost)
	}
	more := append(procs, view.Proclamation{T: 5, Kind: "info", Text: "The War Horn sounds: archers to 2"})
	for _, n := range MergeNews(NewsNotices(more, 0, 2, view.Site{})) {
		if n.Title == "The War Horn sounds" && n.Count != 3 {
			t.Fatalf("a later horn did not join the row: %+v", n)
		}
	}
}

func TestAStepsHintFoldsIntoItsOneCompletionLine(t *testing.T) {
	procs := []view.Proclamation{
		{T: 1, Kind: "info", Text: fulfilledLead + "Cordon Keep node-a"},
		{T: 1, Kind: "lesson", Title: "Cordon Keep node-a", Text: "A Keep is a Node. It runs your huts."},
		{T: 2, Kind: "lesson", Title: "The shift rebuilds what you delete", Text: "A shift raises another hut when one is deleted."},
	}
	got := Tidings(nil, procs, false, 0)
	if len(got) != 2 {
		t.Fatalf("a step's completion took %d lines: %+v", len(got), got)
	}
	if got[1].Text != fulfilledLead+"Cordon Keep node-a. A Keep is a Node." || got[1].Kind != "lesson" || got[1].Title != "Cordon Keep node-a" {
		t.Fatalf("the completion line %+v", got[1])
	}
	if got[0].Text != lessonLead+"A shift raises another hut when one is deleted." {
		t.Fatalf("a line about what the realm did on its own %+v", got[0])
	}
}
