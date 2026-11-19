package world

import (
	"math"
	"testing"
)

func TestStridesKeepStepWithTheWorldClockAtX20(t *testing.T) {
	s := newStride(0, 1, false, false)
	start := s.clock
	for i := 1; i <= 60; i++ {
		s.walk(s.tick(float64(i)*20/60), 1, 0, false)
	}
	if moved := s.clock - start; math.Abs(moved-20) > 1e-6 {
		t.Fatalf("a walker covered %.2fs of its loop in one real second at x20, want 20", moved)
	}
}

func TestShipClockRunsSmoothlyOnTheWorldClockAtX20(t *testing.T) {
	var c FreightClock
	c.Step(95, -5)
	world, realmAt := 0.0, 100.0
	prev := c.Step(realmAt, world)
	for frame := 1; frame <= 120; frame++ {
		world += 20.0 / 60
		if frame%15 == 0 {
			realmAt += 5
		}
		got := c.Step(realmAt, world)
		if got < prev || got-prev > 1 {
			t.Fatalf("frame %d: the ships' clock went %.2f -> %.2f", frame, prev, got)
		}
		if got-prev < .2 {
			t.Fatalf("frame %d: the ships' clock stalls between polls (%.2f -> %.2f)", frame, prev, got)
		}
		prev = got
	}
}

func TestGuidanceBobsOnRealTime(t *testing.T) {
	if got := (Scene{Time: 40, Real: 2}).RealTime(); got != 2 {
		t.Fatalf("RealTime = %v", got)
	}
	if got := (Scene{Time: 40}).RealTime(); got != 40 {
		t.Fatalf("a scene without a real clock falls back to its time: %v", got)
	}
}
