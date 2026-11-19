package world

import (
	"math"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestCrewsFollowTheKeepsUse(t *testing.T) {
	cases := []struct {
		share float64
		slots int
		want  int
	}{
		{0, 3, 0}, {laborIdle, 3, 0}, {.05, 3, 1}, {.5, 3, 2}, {1, 3, 3}, {1.4, 3, 3}, {.5, 0, 0}, {.9, 5, 5}, {.3, 5, 2},
	}
	for _, c := range cases {
		if got := CrewWanted(c.share, c.slots); got != c.want {
			t.Errorf("share %.2f of %d slots = %d, want %d", c.share, c.slots, got, c.want)
		}
	}
	cpu, memory := NodeShares(view.NodeMetrics{CPUMilli: 500, CPURequestedMilli: 1500, CPUAllocatableMilli: 3000, MemoryBytes: 4 << 30, MemoryRequestedBytes: 1 << 30, MemoryAllocatableBytes: 8 << 30})
	if cpu != .5 || memory != .5 {
		t.Fatalf("shares %v %v", cpu, memory)
	}
	if cpu, memory := NodeShares(view.NodeMetrics{CPUMilli: 900}); cpu != 0 || memory != 0 {
		t.Fatalf("a keep with no capacity works at %v %v", cpu, memory)
	}
}

func TestShiftsGrowAndShrinkOneWorkerAtATime(t *testing.T) {
	var s shift
	s = s.step(100, 2, 50)
	if s.active != 2 || !s.works(1, 60) || s.works(2, 100) || s.works(0, 40) {
		t.Fatalf("first sight = %+v", s)
	}
	s = s.step(101, 4, 50)
	if s.active != 2 {
		t.Fatal("grew before the step")
	}
	s = s.step(102.5, 4, 50)
	if s.active != 3 || !s.works(2, 103) || s.works(2, 101) {
		t.Fatalf("grew to %+v", s)
	}
	s = s.step(105, 1, 50)
	if s.active != 2 || !s.works(2, 104) || s.works(2, 106) {
		t.Fatalf("shrank to %+v", s)
	}
	s = s.step(108, 3, 50)
	if s.active != 3 || !s.works(2, 104) || !s.works(2, 109) || s.works(2, 106) {
		t.Fatalf("a returning worker forgot his old felling: %+v", s)
	}
	s = s.step(20, 1, 50)
	if s.active != 1 || s.works(1, 25) || !s.works(0, -20) {
		t.Fatalf("time ran back and the crew kept its old shape: %+v", s)
	}
	if s.works(-1, 0) || s.works(maxLaborSlots, 0) {
		t.Fatal("a slot outside the crew works")
	}
}

func TestAWoodcutterWalksChopsAndCarriesHome(t *testing.T) {
	door, tree := Vec{10, 10}, Vec{16.5, 10.5}
	tr := cutterTrade
	jb := job{start: 100, target: 0}
	walk := tr.walk(door, tree)
	phases := []struct {
		at     float64
		action string
		shown  bool
	}{
		{99, "", false}, {100 + walk/2, "walk", true}, {100 + walk + 1, "work", true}, {100 + walk + tr.work + walk/2, "carry", true}, {100 + 2*walk + tr.work + .1, "", false},
	}
	for _, p := range phases {
		pose, ok := tr.pose(p.at, jb, door, tree)
		if ok != p.shown || pose.Action != p.action {
			t.Errorf("at %.1f: %q %v, want %q %v", p.at, pose.Action, ok, p.action, p.shown)
		}
	}
	work, _ := tr.pose(100+walk+1, jb, door, tree)
	if d := Dist(work.Pos, tree); math.Abs(d-laborStand) > 1e-9 || work.Dir != Facing(work.Pos, tree) {
		t.Errorf("the woodcutter stands %.2f from the tree facing %s", d, work.Dir)
	}
	if got := tr.doneAt(jb, door, tree); math.Abs(got-(100+walk+tr.work)) > 1e-9 {
		t.Errorf("the tree falls at %.2f", got)
	}
	if 2*Dist(door, Vec{door.X + 12, door.Y + 4})/laborSpeed+masonTrade.work >= laborCycle {
		t.Error("a far quarry takes longer than one cycle")
	}
}

func TestFelledTreesRegrowAndCutStoneMends(t *testing.T) {
	for _, tr := range []trade{cutterTrade, masonTrade} {
		if tr.stage(10, 20) != ScarNone || tr.stage(20, 20) != ScarFresh || tr.stage(20+tr.fresh, 20) != ScarMending || tr.stage(20+tr.fresh+tr.mending, 20) != ScarNone {
			t.Errorf("%s scars do not heal in order", tr.role)
		}
		if tr.window() <= tr.fresh+tr.mending {
			t.Errorf("%s looks back too briefly", tr.role)
		}
	}
	got := youngestScars([]scar{{tile: 5, at: 3}, {tile: 2, at: 9}, {tile: 5, at: 7}, {tile: 2, at: 1}})
	if len(got) != 2 || got[0].tile != 2 || got[0].at != 9 || got[1].tile != 5 || got[1].at != 7 {
		t.Fatalf("youngest scars = %+v", got)
	}
}

func TestJobsComeFromTheSeedAlone(t *testing.T) {
	tr := cutterTrade
	a, b := tr.jobAt(7, 1, 12, 40), tr.jobAt(7, 1, 12, 40)
	if a != b || a.target < 0 || a.target >= 40 {
		t.Fatalf("jobs %+v %+v", a, b)
	}
	if tr.jobAt(7, 1, 12, 0).target != -1 {
		t.Fatal("a job found a tree in a treeless land")
	}
	for _, at := range []float64{0, 17.3, 500} {
		j := tr.jobIndex(at, 7, 1)
		if start := tr.jobAt(7, 1, j, 40).start; start > at || at-start >= laborCycle {
			t.Errorf("job %d starts at %.1f for time %.1f", j, start, at)
		}
	}
	targets := map[int]bool{}
	for j := 0; j < 30; j++ {
		targets[tr.jobAt(7, 0, j, 40).target] = true
	}
	if len(targets) < 15 {
		t.Errorf("thirty jobs felled only %d distinct trees", len(targets))
	}
}
