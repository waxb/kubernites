package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheAgeCardNamesTheStagesTheme(t *testing.T) {
	var f Frame
	f.State.Road = view.RoadView{Current: view.AgeFeudal, Ages: []view.AgeStep{
		{ID: view.AgeDark, Theme: "dark theme", Status: "done"},
		{ID: view.AgeFeudal, Theme: "feudal theme", Status: "current"},
	}}
	if got := StageThemeLine(f); got != "feudal theme" {
		t.Fatalf("theme line %q", got)
	}
	if got := StageThemeLine(Frame{}); got != "" {
		t.Fatalf("an empty road still names a theme: %q", got)
	}
}

func TestTheArrivalCalmHoldsTheRealmCardsAdvice(t *testing.T) {
	f := fullFrame(t)
	f.State.Objectives = nil
	f.State.Calm = 30
	if got := NextLine(f); got != "" {
		t.Fatalf("during the calm the realm card advises %q", got)
	}
	f.State.Calm = 0
	if NextLine(f) != f.State.Advice {
		t.Fatal("after the calm the advice speaks again")
	}
}

func TestAnIntroSilencesTheRealmCardsAdvice(t *testing.T) {
	f := fullFrame(t)
	f.State.Objectives = nil
	f.Guide.Intro = &Intro{ID: "gate", Target: "surface:gate", Via: ViaAck}
	if !f.Introducing() {
		t.Fatal("the frame does not introduce the gate")
	}
	if got := NextLine(f); got != "" {
		t.Fatalf("while the gate is introduced the realm card advises %q", got)
	}
}
