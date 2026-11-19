package hud

import (
	"math"
	"testing"
)

const frame = 1.0 / 60

func run(s KmapScroll, in ScrollInput, limit float64, frames int) KmapScroll {
	for range frames {
		s = s.Step(in, limit)
	}
	return s
}

func TestScrollClampsToTheMap(t *testing.T) {
	cases := []struct {
		pos, vel, limit, want float64
	}{{-40, -100, 300, 0}, {340, 200, 300, 300}, {120, 0, 300, 120}, {50, 10, 0, 0}}
	for _, c := range cases {
		got := KmapScroll{Pos: c.pos, Vel: c.vel}.Step(ScrollInput{}, c.limit)
		if got.Pos != c.want || (got.Pos == c.limit || got.Pos == 0) && got.Vel != 0 && c.pos != c.want {
			t.Errorf("%+v -> pos %v vel %v, want %v", c, got.Pos, got.Vel, c.want)
		}
	}
}

func TestDraggingFollowsThePointer(t *testing.T) {
	s := KmapScroll{Pos: 100}.Grab(500)
	s = s.Step(ScrollInput{Dt: frame, Held: true, X: 460}, 1000)
	if s.Pos != 140 || !s.Drag || !s.Moved {
		t.Fatalf("drag 40 left = %+v", s)
	}
	s = s.Step(ScrollInput{Dt: frame, Held: true, X: 480}, 1000)
	if s.Pos != 120 {
		t.Fatalf("drag back = %+v", s)
	}
	tap := KmapScroll{Pos: 100}.Grab(500).Step(ScrollInput{Dt: frame, Held: true, X: 502}, 1000)
	if tap.Moved {
		t.Error("a nudge inside the slop is still a click")
	}
	if released := tap.Step(ScrollInput{Dt: frame, X: 502}, 1000); released.Drag || released.Vel != 0 {
		t.Errorf("a released click does not coast: %+v", released)
	}
}

func TestReleasedDragsCoastThenSettle(t *testing.T) {
	s := KmapScroll{Pos: 200}.Grab(600)
	for x := 590; x >= 540; x -= 10 {
		s = s.Step(ScrollInput{Dt: frame, Held: true, X: x}, 2000)
	}
	at := s.Pos
	s = s.Step(ScrollInput{Dt: frame, X: 540}, 2000)
	if s.Drag || s.Vel <= 0 {
		t.Fatalf("release keeps the fling: %+v", s)
	}
	s = run(s, ScrollInput{Dt: frame, X: 540}, 2000, 30)
	if s.Pos <= at+10 {
		t.Errorf("the map coasted only from %v to %v", at, s.Pos)
	}
	s = run(s, ScrollInput{Dt: frame, X: 540}, 2000, 240)
	if s.Vel != 0 {
		t.Errorf("inertia never settles: %+v", s)
	}
}

func TestReducedMotionStopsAtOnce(t *testing.T) {
	s := KmapScroll{Pos: 200}.Grab(600)
	for x := 590; x >= 540; x -= 10 {
		s = s.Step(ScrollInput{Dt: frame, Held: true, X: x, Reduce: true}, 2000)
	}
	at := s.Pos
	s = run(s, ScrollInput{Dt: frame, X: 540, Reduce: true}, 2000, 10)
	if s.Pos != at || s.Vel != 0 {
		t.Errorf("reduced motion coasted: %v -> %+v", at, s)
	}
	g := KmapScroll{Pos: 0}.GlideTo(500, 300, true)
	if g.Pos != 300 || g.Glide {
		t.Errorf("reduced glide = %+v", g)
	}
	w := KmapScroll{Pos: 100}.Step(ScrollInput{Dt: frame, Wheel: 1, Reduce: true}, 1000)
	if w.Pos != 100+KmapWheelStep || w.Vel != 0 {
		t.Errorf("reduced wheel = %+v", w)
	}
}

func TestKeysAccelerateAndCoast(t *testing.T) {
	s := run(KmapScroll{}, ScrollInput{Dt: frame, Push: 1}, 5000, 30)
	if math.Abs(s.Vel-KmapKeySpeed) > 1 || s.Pos <= 0 {
		t.Fatalf("held D = %+v", s)
	}
	s = run(s, ScrollInput{Dt: frame}, 5000, 5)
	if s.Vel <= 0 || s.Vel >= KmapKeySpeed {
		t.Errorf("released key coasts and slows: %+v", s)
	}
	left := run(KmapScroll{Pos: 400}, ScrollInput{Dt: frame, Push: -1, Reduce: true}, 5000, 6)
	if left.Pos >= 400 || left.Vel != -KmapKeySpeed {
		t.Errorf("held A with reduced motion = %+v", left)
	}
}

func TestWheelKicksTheMap(t *testing.T) {
	s := KmapScroll{Pos: 100}.Step(ScrollInput{Dt: frame, Wheel: 1}, 1000)
	if s.Vel <= 0 {
		t.Fatalf("wheel forward = %+v", s)
	}
	back := KmapScroll{Pos: 100}.Step(ScrollInput{Dt: frame, Wheel: -1}, 1000)
	if back.Vel >= 0 {
		t.Fatalf("wheel back = %+v", back)
	}
}

func TestGlidesReachTheirTarget(t *testing.T) {
	s := KmapScroll{Pos: 0}.GlideTo(320, 1000, false)
	s = run(s, ScrollInput{Dt: frame}, 1000, 120)
	if s.Pos != 320 || s.Glide {
		t.Fatalf("glide = %+v", s)
	}
	s = KmapScroll{Pos: 0}.GlideTo(320, 1000, false).Step(ScrollInput{Dt: frame, Push: 1}, 1000)
	if s.Glide {
		t.Error("a key cancels a glide")
	}
	if g := (KmapScroll{}).GlideTo(-50, 1000, false); g.Target != 0 {
		t.Errorf("glide targets clamp: %+v", g)
	}
}
