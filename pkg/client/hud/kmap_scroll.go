package hud

import "math"

const (
	KmapKeySpeed  = 640.0
	kmapKeyAccel  = 3600.0
	kmapFriction  = 4.5
	kmapStopSpeed = 8.0
	kmapWheelKick = 260.0
	KmapWheelStep = 48.0
	kmapGlideRate = 10.0
	kmapGlideDone = 0.5
	kmapDragSlop  = 4
	kmapMaxDt     = 0.05
	kmapDragBlend = 0.5
)

type KmapScroll struct {
	Pos, Vel float64
	Target   float64
	Glide    bool
	Drag     bool
	Moved    bool
	LastX    int
	FromX    int
}

type ScrollInput struct {
	Dt     float64 `json:"-"`
	Push   float64 `json:"-"`
	Wheel  float64 `json:"-"`
	Held   bool    `json:"-"`
	X      int     `json:"-"`
	Reduce bool    `json:"-"`
}

func (s KmapScroll) Grab(x int) KmapScroll {
	return KmapScroll{Pos: s.Pos, Drag: true, LastX: x, FromX: x}
}

func (s KmapScroll) GlideTo(pos, limit float64, reduce bool) KmapScroll {
	next := KmapScroll{Pos: s.Pos, Target: math.Min(math.Max(pos, 0), limit), Glide: true}
	if reduce {
		next.Pos, next.Glide = next.Target, false
	}
	return next
}

func (s KmapScroll) Step(in ScrollInput, limit float64) KmapScroll {
	dt := math.Min(math.Max(in.Dt, 0), kmapMaxDt)
	if in.Push != 0 || in.Wheel != 0 {
		s.Glide = false
	}
	switch {
	case s.Drag:
		s = s.dragStep(in, dt)
	case s.Glide:
		s = s.glideStep(dt)
	default:
		s = s.coast(in, dt)
	}
	return s.kick(in).clamp(limit)
}

func (s KmapScroll) dragStep(in ScrollInput, dt float64) KmapScroll {
	if !in.Held {
		s.Drag = false
		if in.Reduce || !s.Moved {
			s.Vel = 0
		}
		return s
	}
	delta := float64(s.LastX - in.X)
	s.Pos += delta
	if dt > 0 {
		s.Vel = s.Vel*(1-kmapDragBlend) + delta/dt*kmapDragBlend
	}
	s.LastX = in.X
	if in.X-s.FromX > kmapDragSlop || s.FromX-in.X > kmapDragSlop {
		s.Moved = true
	}
	return s
}

func (s KmapScroll) glideStep(dt float64) KmapScroll {
	s.Vel = 0
	s.Pos += (s.Target - s.Pos) * (1 - math.Exp(-kmapGlideRate*dt))
	if math.Abs(s.Target-s.Pos) < kmapGlideDone {
		s.Pos, s.Glide = s.Target, false
	}
	return s
}

func (s KmapScroll) coast(in ScrollInput, dt float64) KmapScroll {
	switch {
	case in.Push != 0 && in.Reduce:
		s.Vel = in.Push * KmapKeySpeed
	case in.Push != 0:
		s.Vel = approach(s.Vel, in.Push*KmapKeySpeed, kmapKeyAccel*dt)
	case in.Reduce:
		s.Vel = 0
	default:
		s.Vel *= math.Exp(-kmapFriction * dt)
		if math.Abs(s.Vel) < kmapStopSpeed {
			s.Vel = 0
		}
	}
	s.Pos += s.Vel * dt
	return s
}

func (s KmapScroll) kick(in ScrollInput) KmapScroll {
	switch {
	case in.Wheel == 0 || s.Drag:
	case in.Reduce:
		s.Pos += in.Wheel * KmapWheelStep
	default:
		s.Vel += in.Wheel * kmapWheelKick
	}
	return s
}

func (s KmapScroll) clamp(limit float64) KmapScroll {
	limit = math.Max(limit, 0)
	switch {
	case s.Pos <= 0:
		s.Pos, s.Vel = 0, math.Max(s.Vel, 0)
	case s.Pos >= limit:
		s.Pos, s.Vel = limit, math.Min(s.Vel, 0)
	}
	if s.Pos == 0 && s.Vel < 0 || s.Pos == limit && s.Vel > 0 {
		s.Vel = 0
	}
	return s
}

func approach(v, target, step float64) float64 {
	if v < target {
		return math.Min(v+step, target)
	}
	return math.Max(v-step, target)
}
