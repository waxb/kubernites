package app

import (
	"math"

	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

func (s *Session) StallGhost(kind string, gx, gy float64) (*world.Ghost, bool) {
	if !view.StallKind(kind) {
		return nil, false
	}
	sl, ok := s.Site.StallSlotNear(gx, gy)
	if !ok {
		w, h := s.Footprint(kind)
		return &world.Ghost{Kind: kind, X: int(math.Floor(gx + .5)), Y: int(math.Floor(gy + .5)), W: w, H: h}, true
	}
	sv := s.SurveyAt(kind, sl.X, sl.Y)
	return &world.Ghost{Kind: kind, X: sl.X, Y: sl.Y, W: sl.W, H: sl.H, Valid: sv.OK}, true
}
