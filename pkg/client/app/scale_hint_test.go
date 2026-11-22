package app

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
)

func TestTheEnlargeHintCoversNoTitleButton(t *testing.T) {
	for _, liveOnly := range []bool{false, true} {
		g := NewGame(nil, Options{LiveOnly: liveOnly}, nil, nil)
		g.Scene = SceneTitle
		r := g.ScaleHintRect()
		if r.Y < 0 || r.Y+r.H > hud.Height {
			t.Errorf("live only %v: the hint %+v leaves the screen", liveOnly, r)
		}
		if panel := g.Title.Panel(); overlaps(r, panel.Inset(-2)) {
			t.Errorf("live only %v: the hint %+v lies on the title menu %+v", liveOnly, r, panel)
		}
	}
}
