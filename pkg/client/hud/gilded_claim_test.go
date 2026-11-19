package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheGildedCardsBrightButtonClaimsTheReward(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := fullFrame(t)
	claim := Intent{Kind: IntentAction, Action: view.Action{Action: "claimGolden", ID: "golden-1"}}
	h.ResultClaim("gilded", "A GILDED SHIP DOCKED", "3 gleaming boxes wait on its quay. Claim one, and the rest fade fast.", &[2]float64{4, 4}, func() bool { return true }, claim)
	h.Update(Input{MX: -5, MY: -5}, f)
	if !h.LessonShown(f) {
		t.Fatal("the gilded ship's card is not up")
	}
	label := ""
	for _, b := range h.Buttons(f) {
		if b.ID == LessonAck {
			label = b.Label
		}
	}
	if label != "CLAIM" {
		t.Errorf("the gilded card's bright button reads %q, and GOT IT dropped the reward", label)
	}
	got, ok := hasIntent(clickOn(t, h, f, LessonAck), IntentAction)
	if !ok || got.Action.Action != "claimGolden" || got.Action.ID != "golden-1" {
		t.Errorf("CLAIM did not claim the ship's box: %+v", got)
	}
	if h.LessonShown(f) {
		t.Error("the card stayed after its claim")
	}
	h.Result("A NOTICE", "plain text")
	h.Update(Input{MX: -5, MY: -5}, f)
	for _, b := range h.Buttons(f) {
		if b.ID == LessonAck && b.Label != "GOT IT" {
			t.Errorf("an ordinary card's button reads %q", b.Label)
		}
	}
}
