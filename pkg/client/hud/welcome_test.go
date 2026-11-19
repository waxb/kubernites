package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/view"
)

func startIntents(t *testing.T, id string) []Intent {
	t.Helper()
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	h.Open(OverlayWelcome)
	out := clickOn(t, h, f, id)
	if h.Overlay() != OverlayNone {
		t.Fatalf("%s did not close the welcome", id)
	}
	return out
}

func guidanceSet(intents []Intent) string {
	for _, it := range intents {
		if it.Kind == IntentAction && it.Action.Action == "setGuidance" {
			return it.Action.ID
		}
	}
	return ""
}

func TestButtonOneStartsTheNoviceGuideWithoutChangingTheRealm(t *testing.T) {
	intents := startIntents(t, "welcome:begin")
	if got := guidanceSet(intents); got != "" {
		t.Fatalf("button 1 sent setGuidance %q: today's guide must stay byte for byte", got)
	}
	if _, ok := hasIntent(intents, IntentNames); ok {
		t.Fatal("button 1 changed the names")
	}
}

func TestButtonTwoSetsTheStrategistAndLeavesTheNames(t *testing.T) {
	intents := startIntents(t, "welcome:strategy")
	if got := guidanceSet(intents); got != view.GuidanceStrategist {
		t.Fatalf("button 2 set %q", got)
	}
	if _, ok := hasIntent(intents, IntentNames); ok {
		t.Fatal("button 2 changed the names")
	}
}

func TestButtonThreeSetsKubernetesAndTurnsTheNamesOn(t *testing.T) {
	intents := startIntents(t, "welcome:kube")
	if got := guidanceSet(intents); got != view.GuidanceKubernetes {
		t.Fatalf("button 3 set %q", got)
	}
	if _, ok := hasIntent(intents, IntentNames); !ok {
		t.Fatal("button 3 left the story names on")
	}
}

func TestButtonThreeKeepsNamesThatAreAlreadyOn(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.TrueNames = true
	h.Open(OverlayWelcome)
	intents := clickOn(t, h, f, "welcome:kube")
	if _, ok := hasIntent(intents, IntentNames); ok {
		t.Fatal("button 3 toggled names that were already on")
	}
}

func TestTheNumberKeysPressTheButtons(t *testing.T) {
	for key, want := range map[ebiten.Key]string{ebiten.Key1: "", ebiten.Key2: view.GuidanceStrategist, ebiten.Key3: view.GuidanceKubernetes} {
		h := New(nil, (&fakeRunner{}).backend())
		f := fullFrame(t)
		h.Open(OverlayWelcome)
		intents := h.Update(Input{Keys: []ebiten.Key{key}}, f)
		if h.Overlay() != OverlayNone || guidanceSet(intents) != want {
			t.Errorf("key %v: overlay %q, guidance %q, want %q", key, h.Overlay(), guidanceSet(intents), want)
		}
	}
}

func TestInformationAndTheRoadAreSubPagesThatComeBack(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	for _, id := range []string{"welcome:info", "welcome:road"} {
		h.Open(OverlayWelcome)
		clickOn(t, h, f, id)
		if h.Overlay() != OverlayWelcome || h.welcomePage == "" {
			t.Fatalf("%s left the welcome (overlay %q, page %q)", id, h.Overlay(), h.welcomePage)
		}
		h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
		if h.Overlay() != OverlayWelcome || h.welcomePage != "" {
			t.Fatalf("Esc on %s did not come back to the choice (overlay %q, page %q)", id, h.Overlay(), h.welcomePage)
		}
	}
}

func TestTheMenuCyclesTheExperienceAndTogglesTheCalmRealm(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	order := []string{view.GuidanceStrategist, view.GuidanceKubernetes, view.GuidanceNovice}
	for _, want := range order {
		h.Open(OverlayMenu)
		intents := clickOn(t, h, f, "menu:experience")
		got := guidanceSet(intents)
		if want == view.GuidanceNovice && got != "" || want != view.GuidanceNovice && got != want {
			t.Fatalf("after %q the menu set %q, want %q", f.Experience(), got, want)
		}
		f.State.Guidance = want
		if want == view.GuidanceNovice {
			f.State.Guidance = ""
		}
	}
	h.Open(OverlayMenu)
	on := clickOn(t, h, f, "menu:calm")
	if len(on) < 2 || on[1].Action.Action != "setCalm" || on[1].Action.ID != "on" {
		t.Fatalf("calm on = %+v", on)
	}
	f.State.CalmRealm = true
	h.Open(OverlayMenu)
	off := clickOn(t, h, f, "menu:calm")
	if len(off) < 2 || off[1].Action.ID != "off" {
		t.Fatalf("calm off = %+v", off)
	}
}

func TestTheWelcomeSubPagesFit(t *testing.T) {
	f := fullFrame(t)
	for _, page := range []string{WelcomeInfo, WelcomeRoad} {
		h := New(nil, Backend{})
		h.OpenWelcome(page)
		assertTextFits(t, h, f, overlayPanel(OverlayWelcome, Overlays[OverlayWelcome]))
	}
}
