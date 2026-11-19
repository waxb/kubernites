package world

import (
	"math"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestPaceFollowsServedRateAndThrottling(t *testing.T) {
	idle := PaceOf(0, false, false)
	slow := PaceOf(.1, true, false)
	busy := PaceOf(2, true, false)
	tired := PaceOf(2, true, true)
	if idle.Mood != MoodRest || slow.Mood != MoodSteady || busy.Mood != MoodBrisk || tired.Mood != MoodStrained {
		t.Fatalf("moods = %s %s %s %s", idle.Mood, slow.Mood, busy.Mood, tired.Mood)
	}
	if !(tired.Speed < slow.Speed && slow.Speed < busy.Speed) {
		t.Fatalf("speeds tired=%v slow=%v busy=%v", tired.Speed, slow.Speed, busy.Speed)
	}
	if PaceOf(99, true, false).Speed != busy.Speed {
		t.Error("the brisk pace is capped")
	}
}

func TestWorkloadReadsThePodMetrics(t *testing.T) {
	p := model.PodView{Namespace: "default", Name: "a"}
	sc := Scene{Metrics: view.MetricsSnapshot{Pods: map[string]view.PodMetrics{"default/a": {ServedRate: .5, Throttled: true}}}}
	if rate, working, throttled := Workload(sc, p, true); rate != .5 || !working || !throttled {
		t.Fatalf("measured = %v %v %v", rate, working, throttled)
	}
	if _, working, _ := Workload(sc, p, false); working {
		t.Error("no Guildhall, no work")
	}
	sc.Metrics.Pods["default/a"] = view.PodMetrics{ServedRate: .01}
	if _, working, _ := Workload(sc, p, true); working {
		t.Error("a trickle of requests is rest")
	}
	sc.Metrics.Pods = nil
	sc.ArrivalRate = 1
	if _, working, _ := Workload(sc, p, true); !working {
		t.Error("unmeasured workers follow the arrivals")
	}
}

func TestRestingWorkersNeedARealTrickleToWake(t *testing.T) {
	cases := []struct {
		resting, working bool
		rate             float64
		want             bool
	}{
		{true, true, .05, false},
		{true, true, .2, true},
		{true, true, 0, true},
		{false, true, .05, true},
		{true, false, .5, false},
	}
	for _, c := range cases {
		if got := Wakes(c.resting, c.rate, c.working); got != c.want {
			t.Errorf("Wakes(%v, %v, %v) = %v", c.resting, c.rate, c.working, got)
		}
	}
}

func TestStrideFinishesTheLoopBeforeResting(t *testing.T) {
	cycle := 10.0
	s := newStride(0, 0, false, false)
	s.clock = 7
	s.walk(s.tick(.25), 1, cycle, true)
	if s.resting || s.clock != 7.25 {
		t.Fatalf("still walking home: %+v", s)
	}
	for now := .5; now < 4; now += .25 {
		s.walk(s.tick(now), 1, cycle, true)
	}
	if !s.resting || math.Mod(s.clock, cycle) != 0 {
		t.Fatalf("rests at home at the loop start: %+v", s)
	}
	rested := s.clock
	s.walk(s.tick(4.25), 1, cycle, true)
	if s.clock != rested {
		t.Fatal("resting walkers keep still")
	}
	s.walk(s.tick(4.5), 2, cycle, false)
	if s.resting || s.clock != rested+.5 {
		t.Fatalf("wakes and leaves home: %+v", s)
	}
	if s.tick(99) != maxStep {
		t.Error("long gaps are clamped")
	}
}

func TestStrideWakesAtHomeAndSmoothsStrain(t *testing.T) {
	s := newStride(3, 7, false, true)
	if !s.resting {
		t.Fatal("idle on first sight rests at once")
	}
	s.walk(s.tick(3.25), 1, 4, false)
	if got := math.Mod(s.clock, 4); got < 0 || got > .26 {
		t.Fatalf("woke mid-route: phase %v", got)
	}
	s.feel(s.tick(3.5), true)
	if s.strain <= 0 || s.strain >= strainOn {
		t.Fatalf("strain eases in: %v", s.strain)
	}
	for now := 3.75; now < 8; now += .25 {
		s.feel(s.tick(now), true)
	}
	if s.strain < strainOn {
		t.Fatalf("strain builds up: %v", s.strain)
	}
	if s.tick(1) != 0 || s.at != 1 {
		t.Fatal("time running backwards re-anchors the stride")
	}
	if newStride(0, 0, true, false).strain != 1 {
		t.Error("a throttled worker is strained from first sight")
	}
}

func TestTownForgetsVanishedStrides(t *testing.T) {
	r := NewRenderer(nil)
	tw := r.townOf()
	tw.strideFor("a", 0, 1, false, false)
	tw.strideFor("b", 0, 2, false, false)
	tw.strideFor("b", 9, 2, false, false).tick(9)
	tw.prune(9)
	if _, ok := tw.strides["a"]; ok {
		t.Error("stale stride kept")
	}
	if _, ok := tw.strides["b"]; !ok {
		t.Error("live stride dropped")
	}
	if r.townOf() != tw {
		t.Error("town state is kept")
	}
}

func TestLoopStartAndCycleEnds(t *testing.T) {
	if loopStart(3.7, 0) != 3.7 || loopStart(7.5, 2) != 6 {
		t.Error("loop starts snap to whole cycles")
	}
	home, work, bank := Vec{0, 0}, Vec{3, 0}, Vec{3, 3}
	total := LoopLength(home, work, bank)
	if p := CyclePose(total-.01, home, work, bank); p.Pos != home || p.Action != "idle" {
		t.Errorf("the loop ends resting at home: %+v", p)
	}
	if p := CyclePose(0, home, work, bank); p.Action != "walk" || p.Pos != home {
		t.Errorf("the loop starts by leaving home: %+v", p)
	}
	if DeliveryAgeAt(0, home, work, bank) < 0 {
		t.Error("delivery timing")
	}
}

func TestWalkersFinishTheirRoundBeforeResting(t *testing.T) {
	sc, _, r := moodScene(t)
	var p model.PodView
	for _, q := range sc.Pods {
		if q.State == model.HearthReady && sc.Metrics.Pods[PodKey(q)].ServedRate >= 2 {
			p = q
			break
		}
	}
	rt := RouteOf(sc.Site)
	for i := 0; i < 8; i++ {
		sc.Time += .25
		r.FigureFor(sc, p, rt)
	}
	sc.Metrics.Pods[PodKey(p)] = view.PodMetrics{}
	sc.Time += .25
	f, _ := r.FigureFor(sc, p, rt)
	if f.Mood != MoodSteady {
		t.Fatalf("an idle worker first walks home: %+v", f)
	}
}
