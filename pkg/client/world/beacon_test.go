package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func TestPlotArrowPointsAtThePlotsGroundNotTheSpritesTop(t *testing.T) {
	r := NewRenderer(testLib())
	site := view.Site{}
	plot := Ghost{Kind: view.StructKeep, X: 14, Y: 12, W: 3, H: 3, Valid: true}
	sc := Scene{Site: site, Plot: &plot}
	cam := render.NewCamera(480, 60).CenterOn(15, 13, 960, 540)
	spot, ok := r.plotSpot(cam, sc)
	if !ok {
		t.Fatal("no spot for the plot")
	}
	gx, gy := ScreenPointOf(cam, site, Vec{15.5, 13.5})
	if spot.sx != gx || spot.sy != gy {
		t.Fatalf("the arrow stands at %.1f,%.1f, the plot's ground centre is %.1f,%.1f", spot.sx, spot.sy, gx, gy)
	}
}
