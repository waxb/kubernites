package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func shortOfRoom(plan view.RoomPlan) *view.RolloutRoomView {
	return &view.RolloutRoomView{Crew: "porters", Surge: 4, Room: 1, Huts: 4, Fit: 1, Bound: view.BoundSurge, Short: 3, Waits: 3,
		Line: "the roll-out needs room for 4 more huts at once; the Keeps have room for 1", RoomPlan: plan}
}

func gridButton(h *HUD, f Frame, id string) (Button, bool) {
	for _, s := range h.crewSlots(f) {
		if s.button.ID == id {
			return s.button, true
		}
	}
	return Button{}, false
}

func TestEveryRollOutButtonShowsItsRoomAndOffersMakeRoomFirst(t *testing.T) {
	f := crewSelFrame(crewSel())
	f.State.Crews = []view.CrewView{{Name: "porters", Image: "porter:1.0", Replicas: 16, Rollout: shortOfRoom(view.RoomPlan{Shrink: 3, To: 13})}}
	h := New(nil, Backend{})
	for _, mode := range []string{CrewModeImage, CrewModeRoll} {
		h.SetCrewMode(f.Crews[0], mode, f)
		slots := h.crewSlots(f)
		pressed := 0
		for _, s := range slots {
			id := s.button.ID
			if id == crewBack || id == MakeRoomButton || strings.HasSuffix(id, ":history") || s.button.Disabled {
				continue
			}
			pressed++
			if !strings.Contains(s.button.Tip, "Needs room for 4 huts at once; you have 1.") {
				t.Errorf("%s: %s's tooltip does not say the room before the press: %q", mode, id, s.button.Tip)
			}
		}
		if pressed == 0 {
			t.Fatalf("%s: no roll-out button to check", mode)
		}
		b, ok := gridButton(h, f, MakeRoomButton)
		if !ok || b.Label != makeRoomLabel {
			t.Fatalf("%s: a roll-out short of room offers no MAKE ROOM FIRST", mode)
		}
		intents := h.pressCrew("makeroom", f)
		if len(intents) == 0 || intents[0].Action.Action != view.ActScaleCrew || intents[0].Action.Replicas != 13 {
			t.Fatalf("%s: MAKE ROOM FIRST does %+v, want the porters scaled to 13 (down by the plan's 3)", mode, intents)
		}
		if got := h.CrewMode(f.Crews[0]); got != mode {
			t.Fatalf("MAKE ROOM FIRST leaves the %s page for %q: the roll-out button is gone", mode, got)
		}
		if goal := h.crewPageGoal(CrewModeOpen+mode, f); mode == CrewModeImage && goal != MakeRoomButton {
			t.Fatalf("the guide inside UPGRADE lights %q while the roll-out has no room", goal)
		}
	}
	f.State.Crews[0].Rollout = shortOfRoom(view.RoomPlan{To: 16, Ceiling: 16, Floor: 4, Horned: true})
	h.SetCrewMode(f.Crews[0], CrewModeImage, f)
	intents := h.pressCrew("makeroom", f)
	if len(intents) == 0 || intents[0].Action.Action != view.ActRetuneHorn || intents[0].Action.Crew.Max != 16 {
		t.Fatalf("a War Horn owns the count, yet MAKE ROOM FIRST does %+v instead of holding its Ceiling at 16", intents)
	}
	if text := rolloutRoomText(f, f.Crews[0]); !strings.Contains(text, "War Horn at Floor 4, Ceiling 16") || !strings.Contains(text, "a card offers to raise it back") {
		t.Fatalf("the horned plan reads %q", text)
	}
	f.State.Crews[0].Rollout = shortOfRoom(view.RoomPlan{})
	if _, ok := gridButton(h, f, MakeRoomButton); ok {
		t.Fatal("MAKE ROOM FIRST shows with no scale-down that makes room")
	}
	if text := rolloutRoomText(f, f.Crews[0]); !strings.Contains(text, "Buy a Keep first") {
		t.Fatalf("a roll-out no scale-down can help reads %q", text)
	}
}

func TestAScaleDownUnderWaySaysSoInsteadOfBuyAKeep(t *testing.T) {
	f := crewSelFrame(crewSel())
	f.State.Crews = []view.CrewView{{Name: "porters", Image: "porter:1.0", Replicas: 7, Rollout: shortOfRoom(view.RoomPlan{Settling: true})}}
	if text := rolloutRoomText(f, f.Crews[0]); strings.Contains(text, "Buy a Keep") || !strings.Contains(text, "still leaving") {
		t.Fatalf("right after MAKE ROOM FIRST, while the huts sent home leave, the card reads %q", text)
	}
}

func TestTheGuideReachesMakeRoomFirstInTheModeThatShowsIt(t *testing.T) {
	for _, mode := range []string{CrewModeImage, CrewModeProbes, CrewModeRoll, CrewModeRes} {
		path := NavPath(MakeRoomButton, NavState{Object: true, Crew: true, Chain: true, Mode: mode})
		if len(path) != 1 || path[0] != MakeRoomButton {
			t.Errorf("inside %s the guide walks %v to MAKE ROOM FIRST, which stands right there", mode, path)
		}
	}
}

func TestAKeepOnItsWayAsksForNoScaleDown(t *testing.T) {
	f := crewSelFrame(crewSel())
	for _, c := range []struct {
		plan view.RoomPlan
		want string
	}{
		{view.RoomPlan{Rising: true}, "A Keep is on its way: the roll-out fits once it stands"},
		{view.RoomPlan{Reserve: true}, "Place the Keep the reserve sends: the roll-out fits once it stands"},
	} {
		f.State.Crews = []view.CrewView{{Name: "porters", Image: "porter:1.0", Replicas: 15, Rollout: shortOfRoom(c.plan)}}
		h := New(nil, Backend{})
		h.SetCrewMode(f.Crews[0], CrewModeImage, f)
		if _, ok := gridButton(h, f, MakeRoomButton); ok {
			t.Errorf("%+v: MAKE ROOM FIRST shows while a Keep on its way gives the roll-out room (engaged bot seed 11 at 15:07)", c.plan)
		}
		if text := rolloutRoomText(f, f.Crews[0]); !strings.Contains(text, c.want) || strings.Contains(text, "Buy a Keep first") {
			t.Errorf("%+v: the card reads %q, not %q", c.plan, text, c.want)
		}
	}
}
