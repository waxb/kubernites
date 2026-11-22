package app

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

func TestALiveClusterOpensAtTheClosestZoomWhereTheArtShowsBest(t *testing.T) {
	rig := connectAt(t, hud.LevelReadOnly, livetest.Options{})
	s := rig.g.Sess
	if s == nil || !s.Live {
		t.Fatal("not connected")
	}
	closest := render.ZoomLevels[len(render.ZoomLevels)-1]
	if got := s.Cam.Scale(); got != closest {
		t.Fatalf("a live cluster opens at zoom %v, want the closest, %v", got, closest)
	}
}
