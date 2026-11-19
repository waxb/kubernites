package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestOnlyTheFirstTroubleOfAKindAsksForAPauseAndOnlyWhenItsCardOpens(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	first, again := noDepot, oom
	first.First = true
	f := calamityFrame(t, first, again)
	h.Update(Input{MX: -5, MY: -5}, f)
	if asks := h.TakePauseAsks(); len(asks) != 0 {
		t.Fatalf("a trouble asked for a pause before its card opened: %+v", asks)
	}
	f.Clock += CalamitySettle + 1
	h.Update(Input{MX: -5, MY: -5}, f)
	asks := h.TakePauseAsks()
	if len(asks) != 1 || asks[0].Kind != AskTrouble || asks[0].Key != "trouble:"+first.ID {
		t.Fatalf("asks %+v: want exactly the first-of-its-kind trouble", asks)
	}
	f.Clock += CalamityGap
	h.Update(Input{MX: -5, MY: -5}, f)
	if again := h.TakePauseAsks(); len(again) != 0 {
		t.Fatalf("the ask came twice: %+v", again)
	}
}

func TestEveryModeNamesItsKinds(t *testing.T) {
	for mode, want := range map[string][2]bool{PauseAll: {true, true}, PauseTroubles: {true, false}, PauseChoices: {false, true}, PauseOff: {false, false}, "": {true, true}} {
		if PauseCovers(mode, AskTrouble) != want[0] || PauseCovers(mode, AskChoice) != want[1] {
			t.Errorf("%q covers wrong kinds", mode)
		}
	}
	seen := map[string]bool{}
	m := PauseAll
	for range 4 {
		if seen[m] {
			t.Fatal("the pause setting cycle repeats before it comes round")
		}
		seen[m] = true
		m = NextPauseMode(m)
	}
	if m != PauseAll {
		t.Fatalf("the cycle ends at %q", m)
	}
}

func firstTroubleAsks(t *testing.T, v view.CalamityView, gold, wisdom float64) []PauseAsk {
	t.Helper()
	h := New(nil, (&fakeRunner{}).backend())
	v.First = true
	f := calamityFrame(t, v)
	f.State.Gold, f.State.Wisdom = gold, wisdom
	h.Update(Input{MX: -5, MY: -5}, f)
	f.Clock += CalamitySettle + 1
	h.Update(Input{MX: -5, MY: -5}, f)
	return h.TakePauseAsks()
}

func TestOnlyACardWithAFixThePlayerCanUseNowPausesTheRealm(t *testing.T) {
	unready := view.CalamityView{ID: "unready:clerks", Kind: "unready", Object: "pod/default/clerks-1",
		Cause: "Boxes are sent to clerks-1 while its worker still unpacks its tools", Remedies: []view.RemedyView{}}
	if asks := firstTroubleAsks(t, unready, 100, 5); len(asks) != 0 {
		t.Fatalf("a card with only OK paused the realm: %+v", asks)
	}
	priced := unready
	priced.Invite = &view.Invitation{Label: "Shutters keep boxes away", Page: "watch", Rank: "The Shutters", Gold: 3330, Wisdom: 25}
	if asks := firstTroubleAsks(t, priced, 100, 5); len(asks) != 0 {
		t.Fatalf("a card whose only fix is a 3.33K page the player cannot buy paused the realm: %+v", asks)
	}
	if asks := firstTroubleAsks(t, priced, 4000, 30); len(asks) != 1 {
		t.Fatalf("a card whose page the player can buy now did not pause: %+v", asks)
	}
	locked := unready
	locked.Remedies = []view.RemedyView{{Label: "Learn the Shutters", Action: &view.Action{Action: "unlock", ID: "watch"}, Blocked: "needs 3,230 more MiB"}}
	if asks := firstTroubleAsks(t, locked, 100, 5); len(asks) != 0 {
		t.Fatalf("a card whose only fix is locked paused the realm: %+v", asks)
	}
	if asks := firstTroubleAsks(t, noDepot, 0, 0); len(asks) != 1 {
		t.Fatalf("a card with a fix the player can press now did not pause: %+v", asks)
	}
}
