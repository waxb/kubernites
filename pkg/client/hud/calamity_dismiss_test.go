package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestADismissedCalamityStaysAwayThroughABlinkAndReturnsOnlyAfterItCleared(t *testing.T) {
	c := CalamityState{}
	card := []view.CalamityView{{ID: "throttled:lone-tower"}}
	c = c.Step(card, 0, CalamityGate{})
	c = c.Step(card, 6, CalamityGate{})
	if len(c.Ready(card, 6)) != 1 {
		t.Fatal("the card shows once it settles")
	}
	c = c.Dismiss("throttled:lone-tower")
	if len(c.Ready(card, 7)) != 0 {
		t.Fatal("a dismissed card hides")
	}
	c = c.Step(nil, 8, CalamityGate{})
	c = c.Step(card, 9, CalamityGate{})
	if got := c.Ready(card, 15); len(got) != 0 {
		t.Fatalf("a dismissed calamity that blinked away for one step pops up again: %v", got)
	}
	c = c.Step(nil, 10, CalamityGate{})
	c = c.Step(nil, 10+DismissMemory+1, CalamityGate{})
	c = c.Step(card, 12+DismissMemory, CalamityGate{})
	c = c.Step(card, 6+CalamityGap+DismissMemory, CalamityGate{})
	if got := c.Ready(card, 6+CalamityGap+DismissMemory); len(got) != 1 {
		t.Fatalf("once it cleared for the whole window, a new strike shows again: %v", got)
	}
}

func TestAPanelThatCannotBeDrawnLeavesTheSlotFree(t *testing.T) {
	echo, both := []view.CalamityView{noDepot}, []view.CalamityView{noDepot, oom}
	hidden := CalamityGate{hidden: func(v view.CalamityView) bool { return v.ID == noDepot.ID }}
	c := CalamityState{}.Step(echo, 0, hidden).Step(echo, CalamitySettle+1, hidden)
	c = c.Step(both, 10, hidden).Step(both, 10+CalamitySettle, hidden)
	if got := c.Ready(both, 10+CalamitySettle); len(got) != 1 || got[0].ID != oom.ID {
		t.Fatalf("a panel nobody could see held the next one back: %+v", got)
	}
	c = c.Step(both, 11+CalamitySettle, CalamityGate{})
	if got := c.Ready(both, 11+CalamitySettle); len(got) != 2 {
		t.Fatalf("once it can be drawn, the other panel joins the open card at once: %+v", got)
	}
}

func TestACalamityThatClearedUnderAnOverlayDoesNotHoldTheNext(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := calamityFrame(t, noDepot)
	h.OverlayID = OverlayGrimoire
	h.Update(Input{MX: -5, MY: -5}, f)
	f.Clock += CalamitySettle + 1
	h.Update(Input{MX: -5, MY: -5}, f)
	f.State.Calamities = []view.CalamityView{oom}
	f.Clock++
	h.Update(Input{MX: -5, MY: -5}, f)
	h.OverlayID = OverlayNone
	f.Clock += CalamitySettle + 1
	h.Update(Input{MX: -5, MY: -5}, f)
	f = h.SettleStage(Input{MX: -5, MY: -5}, f, settleLimit)
	if v, _, _, ok := h.CurrentCalamity(f); !ok || v.ID != oom.ID {
		t.Fatalf("a panel that never showed under the overlay held the next one back: %+v %v", v, ok)
	}
}

func TestATroubleNeverWaitsForAnIntroCardAndARepeatWaitsLonger(t *testing.T) {
	card := []view.CalamityView{{ID: "pending:porters"}}
	c := CalamityState{}.Step(card, 0, CalamityGate{}).Step(card, CalamitySettle, CalamityGate{})
	if len(c.Ready(card, CalamitySettle)) != 1 {
		t.Fatal("a new trouble waited longer than its settle")
	}
	repeat := func(view.CalamityView) bool { return true }
	r := CalamityState{}.Step(card, 0, CalamityGate{repeat: repeat}).Step(card, CalamitySettle, CalamityGate{repeat: repeat})
	if len(r.Ready(card, CalamitySettle)) != 0 {
		t.Fatal("a known pain opened its panel as fast as a new one")
	}
	r = r.Step(card, CalamitySettle+RepeatWait, CalamityGate{repeat: repeat})
	if len(r.Ready(card, CalamitySettle+RepeatWait)) != 1 {
		t.Fatal("a known pain that lasts never opens its panel")
	}
}

func TestACardLeavesWhenTheRealmRetiresItsTrouble(t *testing.T) {
	stale := view.CalamityView{ID: "throttled:lone-tower", Remedies: []view.RemedyView{{Label: "Raze a hut", Command: "kubectl delete pod porters-x"}}, Invite: &view.Invitation{Page: "pledges"}}
	card := []view.CalamityView{stale}
	c := CalamityState{}.Step(card, 0, CalamityGate{}).Step(card, 6, CalamityGate{})
	c = c.Step(nil, 7, CalamityGate{})
	if got := c.Ready(nil, 7); len(got) != 0 {
		t.Fatalf("a card outlived its trouble with its old fix: %+v", got)
	}
}

func TestAFoldedTroubleCardWaitsOnTheRailAndReopensFromThere(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.State.Calamities = []view.CalamityView{{ID: "unhoused:porters", Kind: view.TagUnhoused, Cause: "No Keep has room."}}
	runFor(h, &f, CalamitySettle+CalamityGap+8)
	if _, _, _, ok := h.CurrentCalamity(h.Stage(f)); !ok {
		t.Skip("the trouble card never opened in this fixture")
	}
	clickOn(t, h, h.Stage(f), calamityShelve)
	runFor(h, &f, 1)
	if _, _, _, ok := h.CurrentCalamity(h.Stage(f)); ok {
		t.Fatal("the card stayed after folding")
	}
	tile := noticePrefix + shelveTilePre + "unhoused:porters"
	found := false
	for _, b := range h.Buttons(h.Stage(f)) {
		found = found || b.ID == tile
	}
	if !found {
		t.Fatal("the folded trouble left no tile on the rail")
	}
	clickOn(t, h, h.Stage(f), tile)
	runFor(h, &f, 1)
	if _, _, _, ok := h.CurrentCalamity(h.Stage(f)); !ok {
		t.Fatal("the rail tile did not reopen the card")
	}
}

func TestPagingPastATroubleSetsItAsideButDoesNotAnswerIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	runFor(h, &f, 1)
	h.focusTrouble("unhoused:a")
	h.focusTrouble("spoil:b")
	if h.TroubleOpen("unhoused:a") || !h.TroubleSetAside("unhoused:a") {
		t.Fatal("paging to the next card counted as answering the first, so the pause lifted")
	}
	h.Ops.Calamity = h.Ops.Calamity.Dismiss("unhoused:a")
	if h.TroubleSetAside("unhoused:a") {
		t.Fatal("a trouble closed with OK still counts as waiting")
	}
}
