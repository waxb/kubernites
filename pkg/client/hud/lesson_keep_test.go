package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAButtonPressNeverClosesALessonCard(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := fullFrame(t)
	f.Guide = Guide{Staged: true, Intro: &Intro{ID: "rail", Title: "Matters at hand", Try: "Glance at the rail.", Target: SurfaceRail, Via: ViaAck}}
	h.Result("A GILDED SHIP DOCKED", "Claim its boxes before they fade.")
	h.Update(Input{MX: -5, MY: -5}, f)
	if !h.LessonShown(f) {
		t.Fatal("the gilded ship's card is not up")
	}
	clickOn(t, h, f, IntroAck)
	if !h.LessonShown(f) {
		t.Fatal("GOT IT on an intro card closed the gilded ship's card too")
	}
	h.Update(Input{MX: Width / 2, MY: (TopH + ConsoleY) / 2, Click: true}, f)
	if !h.LessonShown(f) {
		t.Fatal("a click on the world set the hint card aside before the player read it")
	}
	clickOn(t, h, f, LessonAck)
	if h.LessonShown(f) {
		t.Fatal("GOT IT did not close the hint card")
	}
}

func TestAnIntroCardStepsAsideWhileATroubleCardIsUp(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := calamityFrame(t, noDepot)
	f.Guide = Guide{Intro: &Intro{ID: "rail", Title: "Matters at hand", Try: "Glance at the rail.", Target: SurfaceRail, Via: ViaAck}}
	h.Update(Input{MX: -5, MY: -5}, f)
	if !h.IntroShown(h.Stage(f)) {
		t.Fatal("the intro card is not up")
	}
	from := h.Now
	trouble := func(s Frame) bool { _, up := h.ShownCalamity(s); return up }
	at := waitUntil(t, h, &f, paceLimit, "the trouble card", trouble)
	if at < from+ReadTime("Matters at hand Glance at the rail.") {
		t.Fatalf("the trouble took the intro card's place %.1f s after it came, before its reading time", at-from)
	}
	if h.IntroShown(h.Stage(f)) {
		t.Fatal("an intro card landed beside a trouble card")
	}
	if !h.Aside(actIntro + "rail") {
		t.Fatal("the intro card was not set aside to come back")
	}
	h.Update(Input{MX: -5, MY: -5}, f)
	if text := troubleText(h, f); !strings.Contains(text, AsideLine(1)) {
		t.Fatalf("the trouble card does not say a card waits in its tray: %q", text)
	}
	f.Clock += TroubleRead
	h.Update(Input{MX: -5, MY: -5}, f)
	f.State.Calamities = nil
	waitUntil(t, h, &f, paceLimit, "the intro card back", func(s Frame) bool { return h.IntroShown(s) })
	noAttentionFaults(t, h)
}

func TestAReadyNextHasItsButtonAndItsClickPicksTheTile(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.Guide = Guide{Staged: true}
	f.State.Next = &view.NextView{Kind: view.NextBerth, ID: "haven-0", Label: "Berth at Gate Harbor", Ready: true, Target: "harbor:berth:haven-0"}
	if !f.Shown(SurfaceBerths) {
		t.Fatal("the board says a berth is READY, yet the berth controls stay hidden")
	}
	f.State.Next = &view.NextView{Label: "Stone Keep", Ready: true, Build: "keep"}
	placed := false
	for _, in := range h.PressNext(f) {
		placed = placed || in.Kind == IntentPlace && in.Target == "keep"
	}
	if !placed {
		t.Fatal("NEXT: Stone Keep READY opened BUILD without picking the Keep")
	}
}

func TestAPractisedStepHidesItsHowToUntilAsked(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	o := view.ObjectiveView{ID: view.ReserveKeepObjective, Text: "Place the Keep the reserve sends.", Why: "so that", Hint: "BUILD a Keep elsewhere on the island.",
		Progress: "ready: raise a Keep", Target: "cmd:build", Build: "keep", Practiced: true}
	f.State.Objectives = []view.ObjectiveView{o}
	f.Guide = Guide{Why: o.Why, Target: "build:keep"}
	if hasLine(h.ObjectiveLines(f), "BUILD a Keep elsewhere") {
		t.Fatal("a practised step still spells out its how-to")
	}
	if !h.guideFaded(f) {
		t.Fatal("a practised step still lights its button")
	}
	clickOn(t, h, f, objectiveHint)
	if !hasLine(h.ObjectiveLines(f), "BUILD a Keep elsewhere") {
		t.Fatal("HOW? did not bring the how-to back")
	}
}

func hasLine(lines []Line, part string) bool {
	for _, l := range lines {
		if strings.Contains(l.Text, part) {
			return true
		}
	}
	return false
}

func troubleText(h *HUD, f Frame) string {
	for _, s := range h.Surfaces(f) {
		if strings.HasPrefix(s.Key, ActTrouble) {
			return s.Text
		}
	}
	return ""
}
