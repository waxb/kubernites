package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
)

const (
	stoneTip    = "The Speaking Stone: click to speak to the Chancery.\nIt stands among the townsfolk, not on the Forum: it asks the realm, it is not in its record."
	stoneTipK8s = "kubectl: click to open it.\nA client of the API server, not a cluster object: it stands on the town square, not on the Forum where etcd's objects live."
)

func StoneTip() string { return ui.Bridge(stoneTip, stoneTipK8s) }

func (s *Session) stoneStands() bool {
	return hud.Frame{State: s.State, Guide: s.Guide}.StoneStands()
}

func (s *Session) clickStone() []sound.SFX {
	sfx := []sound.SFX{sound.SFXClick}
	for _, it := range s.Hud.OpenStoneFromWorld(s.Frame(true)) {
		_, fx := s.Handle(it)
		sfx = append(sfx, fx...)
	}
	return sfx
}

func (s *Session) clickDormantStone() []sound.SFX {
	sfx := []sound.SFX{}
	for _, it := range s.Hud.StoneAsleep() {
		if it.Kind == hud.IntentTutorial {
			s.notes = append(s.notes, it)
			continue
		}
		_, fx := s.Handle(it)
		sfx = append(sfx, fx...)
	}
	return sfx
}

func (s *Session) stoneBeacon() (world.Vec, bool) {
	in := s.shownGuide().Intro
	guided := in != nil && in.Target == hud.SurfaceStone || s.GuideGoal() == hud.SurfaceStone
	if !guided || !s.stoneStands() || s.Hud.Overlay() == hud.OverlayStone {
		return world.Vec{}, false
	}
	return world.StoneSpot(s.Site)
}
