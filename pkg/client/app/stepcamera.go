package app

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/world"
)

const (
	stepCamQuiet = 1.0
	HutTarget    = "pod:"
)

type stepCam struct {
	goal    string
	Covered string  `json:"-"`
	MovedAt float64 `json:"-"`
}

func (s *Session) cameraBusy(p *Pointer) bool {
	return p.Dragging || s.Lens.Glide.On || s.Lens.Zooming || s.Lens.Intro || s.Placing != "" || s.Clock-s.StepCam.MovedAt < stepCamQuiet
}

func (s *Session) freshHutStep() bool {
	for _, o := range s.State.Objectives {
		if !o.Done {
			return strings.HasPrefix(o.Target, HutTarget) && o.Target != s.StepCam.goal
		}
	}
	return false
}

func (s *Session) followStepHut(p *Pointer) {
	if !s.freshHutStep() || s.cameraBusy(p) {
		return
	}
	f := s.Frame(true)
	goal, away := s.Hud.StepHut(f)
	if goal == "" || goal == s.StepCam.goal {
		return
	}
	s.StepCam.goal = goal
	if away {
		s.GlideTo(f.GoalAt[0], f.GoalAt[1])
	}
}

func (s *Session) uncoverBeacon(p *Pointer, f hud.Frame) {
	if s.cameraBusy(p) || s.Hud.Overlay() != "" {
		return
	}
	key := s.beaconKey()
	if key == s.StepCam.Covered || !s.HiddenBeacons(f) {
		return
	}
	at, ok := s.beaconAt()
	if !ok {
		return
	}
	s.StepCam.Covered = key
	s.GlideTo(at.X, at.Y)
}

func (s *Session) beaconKey() string {
	for _, o := range s.State.Objectives {
		if !o.Done {
			return o.ID + " " + s.GuideGoal()
		}
	}
	return s.GuideGoal()
}

func (s *Session) beaconAt() (world.Vec, bool) {
	sc := s.BeaconScene()
	switch {
	case sc.Goal != nil:
		return *sc.Goal, true
	case sc.Beckon:
		return world.HarborFocus(s.Site)
	}
	return s.Goal()
}
