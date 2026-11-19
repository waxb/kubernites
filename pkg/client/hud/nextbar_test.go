package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func nextFrame(t *testing.T) Frame {
	t.Helper()
	f := navFrame(t)
	f.State.Next = &view.NextView{Kind: view.NextKeep, ID: "stone", Label: "Stone Keep", Gold: 1000, Progress: .72, Target: "cmd:build", Build: "keep"}
	return f
}

func TestNextPurchaseRowReadsLikeAGoal(t *testing.T) {
	f := nextFrame(t)
	if got := NextLabel(f.State.Next); got != "NEXT: Stone Keep" {
		t.Errorf("label %q", got)
	}
	if got := NextShare(f.State.Next); got != "72%" {
		t.Errorf("share %q", got)
	}
	f.State.Next.Ready, f.State.Next.Progress = true, 1
	if got := NextShare(f.State.Next); got != "READY" {
		t.Errorf("ready share %q", got)
	}
	tip := NextTip(f.State.Next, f.State)
	if !strings.Contains(tip, "Stone Keep") || !strings.Contains(tip, "1K gold") {
		t.Errorf("tip %q", tip)
	}
	f.State.Next = &view.NextView{Kind: view.NextMilestone, ID: "porters", Label: "porters to 10: x4 carry speed", Hands: 2.5, Missing: 1, Progress: .6,
		Target: "surface:stone", Command: "kubectl scale deployment porters --replicas=10"}
	if tip := NextTip(f.State.Next, f.State); !strings.Contains(tip, "2,500 rations") || !strings.Contains(tip, "kubectl scale") {
		t.Errorf("milestone tip %q", tip)
	}
	f.State.Next = &view.NextView{Kind: view.NextIncome, ID: "income-5", Label: "Earn 5 MiB/s", Hint: "Add a Keep for more porters", Progress: .4, Target: "cmd:build"}
	if got, tip := NextLabel(f.State.Next), NextTip(f.State.Next, f.State); got != "NEXT: Earn 5 MiB/s" || !strings.Contains(tip, "How: Add a Keep for more porters.") {
		t.Errorf("rate goal %q, tip %q", got, tip)
	}
}

func TestNextRowGrowsTheBoardAndClicksAlongThePath(t *testing.T) {
	f := nextFrame(t)
	plain := f
	plain.State.Next = nil
	if FreightRect(f).H != FreightRect(plain).H+ui.LineHeight {
		t.Fatalf("the next row does not add a line: %d vs %d", FreightRect(f).H, FreightRect(plain).H)
	}
	if FreightRect(f).H > BoardMaxH {
		t.Fatalf("board %d taller than its most %d", FreightRect(f).H, BoardMaxH)
	}
	h := New(nil, (&fakeRunner{}).backend())
	h.BoardOpen = true
	clickOn(t, h, f, FreightTip+"next")
	if h.Page() != PageBuild {
		t.Fatalf("NEXT goes to the BUILD page, got %q", h.Page())
	}
	h = New(nil, (&fakeRunner{}).backend())
	h.BoardOpen = true
	f.State.Next = &view.NextView{Kind: view.NextMilestone, Label: "porters to 10", Target: "surface:stone", Command: "kubectl scale deployment porters --replicas=10", Progress: 1, Ready: true}
	clickOn(t, h, f, FreightTip+"next")
	if h.Overlay() != OverlayStone || string(h.Stone.Input) != "kubectl scale deployment porters --replicas=10" && string(h.Stone.Input) != "scale deployment porters --replicas=10" {
		t.Fatalf("a milestone opens the Stone with its command: %q %q", h.Overlay(), string(h.Stone.Input))
	}
}

func TestBoostRowCountsDown(t *testing.T) {
	f := nextFrame(t)
	base := FreightRect(f).H
	f.State.Golden.Boosts = []view.BoostView{{Kind: view.BonusFrenzy, Multiplier: 3, Ends: 500, Left: 42}}
	if FreightRect(f).H != base+ui.LineHeight || FreightRect(f).H > BoardMaxH {
		t.Fatalf("the boost row: %d from %d (max %d)", FreightRect(f).H, base, BoardMaxH)
	}
	b, ok := ActiveBoost(f)
	if !ok || BoostLabel(b) != "GILDED TIDE x3" || Duration(b.Left) != "0:42" {
		t.Fatalf("boost = %+v %v %q", b, ok, BoostLabel(b))
	}
	f.State.Golden.Boosts[0].Left = 0
	if _, ok := ActiveBoost(f); ok {
		t.Fatal("a spent boost still shows")
	}
}

func TestCrewLinesShowTheirSeasoning(t *testing.T) {
	f := fullFrame(t)
	f.State.Crews = []view.CrewView{{Name: "porters", Depot: "parcels", Ready: 12, Replicas: 12,
		MilestoneView: view.MilestoneView{Milestone: 2, Multiplier: 4, Next: 25, NextMultiplier: 8}}}
	if got := SeasonLine(f.State.Crews[0]); got != "x4 SEASONED, next x2 at 25" {
		t.Errorf("season %q", got)
	}
	if got := SeasonLine(view.CrewView{MilestoneView: view.MilestoneView{Multiplier: 1, Next: 5, NextMultiplier: 2}}); got != "next x2 at 5 ready" {
		t.Errorf("green crew %q", got)
	}
	if got := SeasonLine(view.CrewView{MilestoneView: view.MilestoneView{Milestone: 4, Multiplier: 16}}); got != "x16 SEASONED, every milestone won" {
		t.Errorf("veterans %q", got)
	}
	depot := view.Structure{Kind: view.DepotStructure("parcels"), Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	lines := DepotLines(depot, f)
	joined := ""
	for _, l := range lines {
		joined += l.Text + "|"
	}
	if !strings.Contains(joined, "x4 SEASONED") {
		t.Errorf("depot lines %q", joined)
	}
	if tip := ChipTip(f.State.Freight.Kinds[0], 1, true, f.State.Crews...); !strings.Contains(tip, "x4 SEASONED") {
		t.Errorf("chip tip %q", tip)
	}
}

func TestCarriedFitsTheDepotCard(t *testing.T) {
	box := PortraitBox()
	x, w := box.X+box.W+8, CardBox.X+CardBox.W-box.X-box.W-14
	for _, c := range [][2]float64{{.61, .72}, {12.5, 13}, {0, 0}, {99.9, 120}} {
		if s := CarriedSuffix(c[0], c[1]); len(s) > BarRoom("CARRIED", s, x, w) {
			t.Errorf("%q needs %d characters, the card holds %d", s, len(s), BarRoom("CARRIED", s, x, w))
		}
	}
}
