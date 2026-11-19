package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func invitedCard() view.CalamityView {
	card := oom
	card.Invite = &view.Invitation{Label: "Right-size the haulers: ask 50m of CPU, not 490m", Page: "pledges", Rank: "Labor Pledge", Gold: 150, Wisdom: 25}
	return card
}

func grimoireState(f Frame) Frame {
	next := f
	next.State.Age = view.AgeFeudal
	next.State.Grimoire = []view.GrimoirePage{
		{ID: "speaking-stone", Name: "The Speaking Stone", Era: view.AgeDark, State: "unlocked"},
		{ID: "pledges", Name: "Pledges", Era: view.AgeFeudal, State: "available", MaxRank: 2,
			Ranks: []view.RankView{{Title: "Labor Pledge", Cost: view.Cost{Gold: 150, Wisdom: 25}}, {Title: "Lore Pledge", Cost: view.Cost{Gold: 250, Wisdom: 25}}}},
	}
	return next
}

func TestALockedRemedyShowsItsPriceOnTheCard(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	plain := settled(h, calamityFrame(t, oom))
	without, _ := h.CalamityRect(plain)
	h = New(nil, (&fakeRunner{}).backend())
	f := settled(h, calamityFrame(t, invitedCard()))
	r, _ := h.CalamityRect(f)
	if r.H <= without.H {
		t.Fatalf("the card did not grow for its invitation: %d, without %d", r.H, without.H)
	}
	b, ok := HitID(h.CalamityButtons(f), InviteButton("pledges"))
	if !ok || !contains(r, b.Rect) {
		t.Fatalf("no invitation inside the card %+v: %+v %v", r, b, ok)
	}
	for _, part := range []string{"LABOR PLEDGE", Gold(150), Wisdom(25)} {
		if !strings.Contains(b.Label, part) {
			t.Errorf("the invitation %q lacks %q", b.Label, part)
		}
	}
	if !strings.Contains(b.Tip, "Right-size the haulers") {
		t.Errorf("the tip does not name the locked answer: %q", b.Tip)
	}
	for _, other := range h.CalamityButtons(f) {
		if other.ID != b.ID && Overlaps(other.Rect, b.Rect) {
			t.Errorf("the invitation overlaps %s", other.ID)
		}
	}
}

func TestAnInvitationOpensTheGrimoireOnItsPage(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := grimoireState(settled(h, calamityFrame(t, invitedCard())))
	intents := clickOn(t, h, f, InviteButton("pledges"))
	if h.Overlay() != OverlayGrimoire || h.Grimoire != "pledges" {
		t.Fatalf("the invitation opened %q on %q (%+v)", h.Overlay(), h.Grimoire, intents)
	}
	h.Update(Input{MX: -5, MY: -5}, f)
	if p, seen := h.focusPlaque(h.Knowledge(f)); !seen || p.Page.ID != "pledges" {
		t.Fatalf("the Pledges plaque is not in view: %+v %v", p, seen)
	}
}

func TestAnInvitedPageHoldsUntilThePointerMoves(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := grimoireState(settled(h, calamityFrame(t, invitedCard())))
	clickOn(t, h, f, InviteButton("pledges"))
	other, ok := h.Knowledge(f).Find("speaking-stone")
	if !ok {
		t.Fatal("no Speaking Stone plaque")
	}
	r := h.ScreenOf(other.Rect())
	x, y := r.X+r.W/2, r.Y+r.H/2
	for range 2 {
		h.Update(Input{MX: x, MY: y}, f)
	}
	if got := h.focusedPage(h.Knowledge(f)); got != "pledges" {
		t.Fatalf("the pointer left over another plaque took the invited page: %q", got)
	}
	h.Update(Input{MX: x + 1, MY: y}, f)
	if got := h.focusedPage(h.Knowledge(f)); got != "speaking-stone" {
		t.Fatalf("hover no longer follows the pointer once it moves: %q", got)
	}
}
