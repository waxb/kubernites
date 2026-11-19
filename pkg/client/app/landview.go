package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/world"
)

type landCache struct {
	id  string
	sig uint64
	has bool
}

func (s *Session) selectionLand() bool {
	if s.Selection.Kind != world.HitStructure {
		return false
	}
	key := landCache{id: s.Selection.ID, sig: s.Land.borders.Sig}
	if s.lands.id == key.id && s.lands.sig == key.sig && s.lands.id != "" {
		return s.lands.has
	}
	sel := s.Selection.Structure
	key.has = len(TintsFor(s.Land.borders, s.Site, TintInput{Selected: &sel})) > 0
	s.lands = key
	return key.has
}

func (s *Session) landShown() bool {
	return s.Hud.LandShown(hud.Frame{Selection: s.Selection, Land: s.selectionLand(), Guide: s.shownGuide()})
}
