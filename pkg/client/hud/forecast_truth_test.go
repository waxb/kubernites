package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func bookedFrame(seated, fed, short int) Frame {
	f := boardFrame()
	f.State.Crews[0].Room = view.RoomFit{Seated: seated, Fed: fed, Short: short, Limit: view.FlowCPU}
	f.State.Crews[0].Fits = max(0, min(seated, fed))
	f.State.Crews[0].Takes = view.ResourcePair{CPU: 500, Memory: 64 << 20}
	f.State.Crews[0].Use = view.ResourcePair{CPU: 500, Memory: 64 << 20}
	f.State.Room = view.RoomBook{Keeps: []view.KeepRoom{{Name: "lone-tower", Free: view.ResourcePair{CPU: 1250, Memory: 4<<30 - 192<<20},
		Spare: view.ResourcePair{CPU: 500, Memory: 3 << 30}, Seated: map[string]int{"porters": 3}}}}
	return f
}

func TestThePlusButtonCountsTheFoodEachWorkerReallyEats(t *testing.T) {
	f := bookedFrame(5, 1, 0)
	more := ScaleButtons(f.Crews[0], f)[1]
	if !strings.Contains(more.Tip, "room for 1 more") || !strings.Contains(more.Tip, "would have room for 5 more") || !strings.Contains(more.Tip, "go hungry") {
		t.Fatalf("the + tooltip with 5 seats but food for 1 reads %q", more.Tip)
	}
	hungry := bookedFrame(5, 0, 0)
	if b := ScaleButtons(hungry.Crews[0], hungry)[1]; b.Cost != shortCost(view.FlowCPU) || !strings.Contains(b.Tip, "go hungry") {
		t.Fatalf("with seats but no food the + reads %q / %q", b.Cost, b.Tip)
	}
	full := bookedFrame(0, 0, 0)
	if b := ScaleButtons(full.Crews[0], full)[1]; b.Cost != summonNoRoom {
		t.Fatalf("with no seat at all the + reads %q", b.Cost)
	}
}

func TestTheBoardPreviewSaysAllFitOnlyWhenEveryNewWorkerIsSeatedAndFed(t *testing.T) {
	cases := []struct {
		seated, fed, short, to int
		want                   string
	}{
		{5, 5, 0, 5, "ALL FIT"},
		{5, 1, 0, 5, "ALL PLACED, 1 GO HUNGRY"},
		{1, 1, 0, 5, "1 OF 2 FIT"},
		{3, 3, 2, 5, "1 OF 2 FIT"},
	}
	for _, c := range cases {
		f := bookedFrame(c.seated, c.fed, c.short)
		v := BoardView{Chain: f.Crews[0], Facts: CrewFactsOf(f, f.Crews[0]), Pods: crewPods(f.Crews[0], f), Target: c.to}
		got := []string{}
		for _, p := range previewUsePieces(v, BoardDeltaOf(v)) {
			got = append(got, p.Text)
		}
		if line := strings.Join(got, " "); !strings.Contains(line, c.want) {
			t.Errorf("seated %d fed %d short %d, 3 -> %d reads %q, want %q", c.seated, c.fed, c.short, c.to, line, c.want)
		}
	}
}

func TestKeepsFreeCountsTheRoomOwedToWorkersAlreadyOrdered(t *testing.T) {
	f := bookedFrame(5, 5, 0)
	f.State.Room.Owed = []view.OwedRoom{{Crew: "clerks", Count: 2, Asks: view.ResourcePair{CPU: 500, Memory: 64 << 20}, Takes: view.ResourcePair{CPU: 500, Memory: 64 << 20}}}
	v := BoardView{Chain: f.Crews[0], Facts: CrewFactsOf(f, f.Crews[0])}
	if cpu, _ := landFree(v); cpu != 250 {
		t.Fatalf("with two clerks owed 1,000 of the 1,250 free rations, KEEPS FREE reads %d", cpu)
	}
}

func TestAPledgeFixSaysHowManyWillWaitUnhousedBeforeItIsPressed(t *testing.T) {
	f := bookedFrame(5, 5, 0)
	a := view.Action{Action: view.ActCrewResources, Name: "porters", Crew: view.CrewOrder{Requests: "memory=2Gi"}}
	l := withRemedyEffect(CommandLine{Label: "Pledge the housing", Action: &a}, f)
	if !l.EffectBad || !strings.Contains(l.Effect, "wait Unhoused") {
		t.Fatalf("a pledge of 2Gi for three porters on one 4Gi Keep reads %q (bad %v)", l.Effect, l.EffectBad)
	}
	small := view.Action{Action: view.ActCrewResources, Name: "porters", Crew: view.CrewOrder{Requests: "cpu=100m"}}
	if l := withRemedyEffect(CommandLine{Label: "Pledge less", Action: &small}, f); l.EffectBad || !strings.Contains(l.Effect, "room for") {
		t.Fatalf("a smaller pledge reads %q (bad %v)", l.Effect, l.EffectBad)
	}
	scale := view.Action{Action: view.ActScaleCrew, Name: "porters", Replicas: 6}
	if l := withRemedyEffect(CommandLine{Label: "Scale", Action: &scale}, bookedFrame(5, 1, 0)); !l.EffectBad || !strings.Contains(l.Effect, "hungry") {
		t.Fatalf("a scale past the food reads %q", l.Effect)
	}
}

func TestAFixTheRealmWouldRefuseIsGreyedWithItsReason(t *testing.T) {
	a := view.Action{Action: "buyKeep", For: "porters"}
	b := LineButtons("line:cal:0", ui.Rect{X: 10, Y: 10, W: 300, H: 16}, CommandLine{Label: "Add a Keep", Action: &a, Blocked: "needs 2,500 more memory"})
	if len(b) != 1 || !b[0].Disabled || !strings.HasPrefix(b[0].Tip, "Needs 2,500 more memory") {
		t.Fatalf("a refused fix reads %+v", b)
	}
	l := withRemedyEffect(CommandLine{Label: "Add a Keep", Action: &a, Blocked: "needs 2,500 more memory"}, boardFrame())
	if !l.EffectBad || l.Effect != "Needs 2,500 more memory." {
		t.Fatalf("the reason line reads %q", l.Effect)
	}
}

func TestTheImpsResultCardTellsItsLatestHutAndNeverQueuesTwice(t *testing.T) {
	h := New(nil, Backend{})
	h.Result("A POD WAS KILLED", "toppled porters-a")
	h.nextResult()
	h.Result("A POD WAS KILLED", "toppled porters-b")
	if h.LessonCard.Text != "toppled porters-b" || len(h.Results) != 0 {
		t.Fatalf("the shown card reads %q with %d queued", h.LessonCard.Text, len(h.Results))
	}
}

func TestManyTroublesAtOnceNeverClaimACountOfCards(t *testing.T) {
	got, _ := MergeAnnouncements(AnnounceCalamity, []string{"Pod porters-a was killed.", "Hunger struck node-a."}, true)
	if strings.Contains(got, "calamities strike") || !strings.Contains(got, "(and 1 more)") {
		t.Fatalf("two calamity notices merge as %q", got)
	}
}

func TestACharterTellsWhyItIsGreyedAndWhatItGave(t *testing.T) {
	terms := view.CharterView{Cost: 2000, Reason: "Learn the Charters of the Isles in the Grimoire"}
	if got, _ := charterStatus(VoyageRow{Name: "Gullsreach"}, terms); got != "UNCHARTERED: learn the Charters page" {
		t.Fatalf("a closed charter reads %q", got)
	}
	if got, _ := charterStatus(VoyageRow{Name: "Gullsreach", Owned: true}, terms); !strings.Contains(got, "build Keeps") {
		t.Fatalf("an empty chartered isle reads %q", got)
	}
}

func TestReopeningACrewShowsItsScaleControlsNotTheWatch(t *testing.T) {
	f := boardFrame()
	h, _ := openBoard(t, f)
	h.SetCrewMode(f.Crews[0], CrewModeProbes, f)
	h.leaveObject(f)
	if got := h.CrewMode(f.Crews[0]); got != CrewModeChain {
		t.Fatalf("the crew window reopens in %q", got)
	}
	_ = model.CrewChain{}
}

func TestAFinishedClassSaysWhatTheClassDoesAndWhyGoldWaits(t *testing.T) {
	f := fullFrame(t)
	f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: "porter:2.0", Owned: true, Visible: true, Description: "Twice the parcels of 1.0 for the same food."})
	f.State.Freight.Kinds = []view.KindView{{Kind: "parcel", CrewName: "porters", Open: true, ArrivalRate: 2, ServedRate: 2}}
	_, cause := RolloutBannerIn(f, model.RolloutDone{Crew: "porters", Image: "porter:2.0", From: "porter:1.0", Huts: 20, Gain: .04})
	if !strings.Contains(cause, "Twice the parcels") || !strings.Contains(cause, "more ships") || strings.Contains(cause, "+0.04") {
		t.Fatalf("a class bought while the porters kept up reads %q", cause)
	}
}

func TestTheUpgradePageSaysWhenARolloutWouldWaitForRoom(t *testing.T) {
	f := boardFrame()
	if got := rolloutRoomText(f, f.Crews[0]); got != "" {
		t.Fatalf("a crew with room reads %q", got)
	}
	f.State.Crews[0].Rollout = &view.RolloutRoomView{Crew: "porters", Fits: false, Text: "this roll-out needs room for one new porters worker"}
	if got := rolloutRoomText(f, f.Crews[0]); !strings.Contains(got, "needs room for one new porters worker") {
		t.Fatalf("a crew short of room for its surge reads %q", got)
	}
	f.State.Crews[0].Rollout = &view.RolloutRoomView{Crew: "porters", Fits: true, Surge: 2, Room: 5, Huts: 2, Fit: 2, Bound: view.BoundSurge, Line: "the roll-out needs room for 2 more huts at once; the Keeps have room for 5"}
	if got := rolloutRoomText(f, f.Crews[0]); !strings.Contains(got, "needs room for 2 huts at once; you have 2") || strings.Contains(got, "would wait") {
		t.Fatalf("a roll-out with room does not say what it needs before the press: %q", got)
	}
	f.State.Crews[0].Rollout = &view.RolloutRoomView{Crew: "porters", Surge: 3, Room: 1, Waits: 2, Short: 2, Huts: 3, Fit: 1, Bound: view.BoundSurge, RoomPlan: view.RoomPlan{Shrink: 2, To: 1},
		Line: "the roll-out needs room for 3 more huts at once; the Keeps have room for 1"}
	if got := rolloutRoomText(f, f.Crews[0]); !strings.Contains(got, "needs room for 3 huts at once; you have 1") || !strings.Contains(got, "MAKE ROOM FIRST scales the porters down by 2") {
		t.Fatalf("a roll-out short of room does not lead with the room fix: %q", got)
	}
}

func TestALearnedWatchPageOpensACrewsWatchControls(t *testing.T) {
	page := view.GrimoirePage{ID: "watch", State: "unlocked", Rank: 1, MaxRank: 1}
	if _, ok := applyButton(KMapPlaque{Page: page}, ui.Rect{W: 90, H: 14}); !ok {
		t.Fatal("a learned Watch page offers no way to put it on a crew")
	}
	if _, ok := applyButton(KMapPlaque{Page: view.GrimoirePage{ID: "watch", State: "available"}}, ui.Rect{W: 90, H: 14}); ok {
		t.Fatal("an unlearned page offers to put itself on a crew")
	}
	f := boardFrame()
	h := New(nil, Backend{})
	h.OverlayID = OverlayGrimoire
	if _, acted := hasIntent(h.pressApply("watch", f), IntentChoose); !acted || h.OverlayID != OverlayNone {
		t.Fatalf("ON A CREW neither closed the manual nor chose a crew (overlay %q)", h.OverlayID)
	}
	h.Update(Input{}, f)
	if got := h.CrewMode(f.Crews[0]); got != CrewModeProbes {
		t.Fatalf("the chosen crew opens in %q, not its WATCH controls", got)
	}
}
