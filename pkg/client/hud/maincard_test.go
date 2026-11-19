package hud

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

var everyMainCommand = []string{"cmd:build", "cmd:summon", "cmd:grimoire", "cmd:soulvault",
	"cmd:chronicles", "cmd:age", "cmd:siege", "cmd:door", CmdCrew}

func crowdedFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Age = view.AgeFeudal
	f.State.Reveals = append(slices.Clone(allReveals), "siege", "door", "depot", "crew", "keeps")
	f.State.Commands = []view.CommandOffer{raiseClerks, openParcels}
	f.State.Freight.Kinds = []view.KindView{{Kind: "parcel", Open: true, Depot: "parcels"}, {Kind: "ledger", Depot: "ledgers"}}
	f.State.Unlocks = map[string]bool{"war-horns": true}
	f.Pods = nil
	f.Guide = Guide{Staged: true, Introduced: slices.Clone(everyMainCommand)}
	return f
}

func rushTideIntro() *Intro {
	return &Intro{ID: "siege", Title: "The Traffic Spike", Try: "Press it to see what it asks.", Target: "cmd:siege", Via: ViaOpen}
}

func mainAndMore(t *testing.T, h *HUD, f Frame) ([]Button, []Button) {
	t.Helper()
	h.PageID = PageMain
	main := h.CommandButtons(f)
	if _, ok := HitID(main, CmdMore); !ok {
		return main, nil
	}
	h.PageID = PageMore
	more := h.CommandButtons(f)
	h.PageID = PageMain
	return main, more
}

func TestTheRushTideIntroFindsItsButtonOnAFullCard(t *testing.T) {
	f := crowdedFrame(t)
	f.Guide.Intro = rushTideIntro()
	h := New(nil, (&fakeRunner{}).backend())
	s, ok := h.Step(f)
	if !ok || s.ID != "cmd:siege" {
		t.Fatalf("the spike intro has no step on a full card: %+v %v", s, ok)
	}
	if !inside(s.Rect, Command) {
		t.Fatalf("the step %+v is not on the command card %+v", s.Rect, Command)
	}
	hole, ok := h.IntroHole(f)
	if !ok || hole != s.Rect {
		t.Fatalf("the dim has no hole over the button: %+v %v", hole, ok)
	}
	b, ok := HitButton(h.Buttons(f), s.Rect.X+s.Rect.W/2, s.Rect.Y+s.Rect.H/2)
	if !ok || b.ID != "cmd:siege" || b.Disabled {
		t.Fatalf("clicking the hole hits %+v", b)
	}
	intents := clickOn(t, h, f, "cmd:siege")
	if h.Overlay() != OverlaySiege {
		t.Fatal("pressing it opens the spike card")
	}
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != AckAction || it.Action.ID != "siege" {
		t.Fatalf("opening the card completes the intro: %+v", intents)
	}
}

func TestTheRushTideButtonFollowsTheSameRevealAsItsIntro(t *testing.T) {
	f := crowdedFrame(t)
	f.State.Unlocks = map[string]bool{}
	f.Guide.Intro = rushTideIntro()
	h := New(nil, (&fakeRunner{}).backend())
	if _, ok := HitID(h.CommandButtons(f), "cmd:siege"); !ok {
		t.Fatal("the intro speaks of the spike but its button is not drawn")
	}
	f.State.Reveals = slices.DeleteFunc(slices.Clone(f.State.Reveals), func(r string) bool { return r == "siege" })
	f.State.Unlocks = map[string]bool{"war-horns": true}
	f.Guide.Intro = nil
	if main, more := mainAndMore(t, h, f); slices.ContainsFunc(append(main, more...), func(b Button) bool { return b.ID == "cmd:siege" }) {
		t.Fatal("the button shows before the realm reveals the spike")
	}
}

func crowdedAfterCrews(t *testing.T) Frame {
	t.Helper()
	f := crowdedFrame(t)
	f.State.Objectives = append(slices.Clone(f.State.Objectives), view.ObjectiveView{ID: CrewServesObjective, Done: true})
	return f
}

func TestEveryMainCommandHasACellOrAPlaceOnMore(t *testing.T) {
	f := crowdedAfterCrews(t)
	h := New(nil, (&fakeRunner{}).backend())
	main, more := mainAndMore(t, h, f)
	if len(more) == 0 {
		t.Fatal("once the crews carry, SUMMON waits on MORE")
	}
	if len(main) != 9 {
		t.Fatalf("a full card uses all nine cells, got %d", len(main))
	}
	seen := map[string]bool{}
	for _, b := range append(slices.Clone(main), more...) {
		seen[b.ID] = true
	}
	for _, id := range everyMainCommand {
		if !seen[id] {
			t.Errorf("%s has neither a cell nor a place on MORE", id)
		}
	}
	for _, page := range [][]Button{main, more} {
		for i, a := range page {
			for _, b := range page[i+1:] {
				if Overlaps(a.Rect, b.Rect) {
					t.Errorf("%s overlaps %s", a.ID, b.ID)
				}
			}
		}
	}
	if b, ok := HitID(more, "cmd:back"); !ok || b.Rect != CellRect(8) {
		t.Errorf("MORE ends with BACK in the usual corner: %+v", b)
	}
}

func TestMoreOpensItsPageAndBackReturns(t *testing.T) {
	f := crowdedAfterCrews(t)
	h := New(nil, (&fakeRunner{}).backend())
	clickOn(t, h, f, CmdMore)
	if h.Page() != PageMore {
		t.Fatalf("MORE opens its page, got %q", h.Page())
	}
	clickOn(t, h, f, "cmd:back")
	if h.Page() != PageMain {
		t.Fatalf("BACK returns to the main card, got %q", h.Page())
	}
	if got := NavPath("cmd:build", NavState{Page: PageMore}); !slices.Equal(got, []string{"cmd:back", "cmd:build"}) {
		t.Errorf("from MORE the path to BUILD = %v", got)
	}
}

func TestAGuidedCommandIsNeverSentToMore(t *testing.T) {
	for _, id := range everyMainCommand {
		f := crowdedFrame(t)
		f.Guide.Intro = &Intro{ID: "x", Title: "x", Try: "x", Target: id, Via: ViaOpen}
		h := New(nil, (&fakeRunner{}).backend())
		main, _ := mainAndMore(t, h, f)
		if _, ok := HitID(main, id); !ok {
			t.Errorf("the intro for %s points at MORE's page", id)
		}
		f.Guide.Intro = nil
		f.State.Objectives = []view.ObjectiveView{{ID: "o", Text: "Do it", Target: id}}
		f.Guide.Target = id
		main, _ = mainAndMore(t, h, f)
		if _, ok := HitID(main, id); !ok {
			t.Errorf("the task's %s is not on the main card", id)
		}
	}
}

func TestACardWithoutChroniclesStillReservesTheGuidedCell(t *testing.T) {
	f := crowdedFrame(t)
	f.State.Reveals = slices.DeleteFunc(slices.Clone(f.State.Reveals), func(r string) bool { return r == "chronicles" })
	f.Guide.Intro = rushTideIntro()
	h := New(nil, (&fakeRunner{}).backend())
	main, more := mainAndMore(t, h, f)
	if _, ok := HitID(main, "cmd:siege"); !ok {
		t.Fatal("the spike lost its cell")
	}
	if len(main)+len(more) < len(everyMainCommand)-1 {
		t.Errorf("commands went missing: %d on main, %d on MORE", len(main), len(more))
	}
}

func TestTheRushTideCellOpensItsCardEvenWhenTheCallCannotBeAnswered(t *testing.T) {
	f := crowdedFrame(t)
	f.Guide.Intro = rushTideIntro()
	f.State.Siege = SiegeRetry()
	h := New(nil, (&fakeRunner{}).backend())
	b, ok := HitID(h.CommandButtons(f), "cmd:siege")
	if !ok || b.Disabled || b.Note == "" {
		t.Fatalf("while the tide rests the cell still opens its card and shows the wait: %+v", b)
	}
	clickOn(t, h, f, "cmd:siege")
	if h.Overlay() != OverlaySiege {
		t.Fatal("the card opens")
	}
	if answer := h.SiegeButtons(f)[1]; !answer.Disabled {
		t.Error("ANSWER THE CALL waits for the next tide")
	}
}
