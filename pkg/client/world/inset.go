package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	InsetW    = 48
	InsetH    = 56
	InsetZoom = 2
	insetFeet = 3.0 / 4
	insetFPS  = 8
)

func InsetCamera(site view.Site, at Vec, w, h int) render.Camera {
	cam := render.NewCamera(w/2, int(float64(h)*insetFeet))
	cam.Zoom = InsetZoom
	cam.X = (at.X - at.Y) * render.TileWidth / 2
	cam.Y = (at.X+at.Y)*render.TileHeight/2 - elevationAt(site, at)*render.StepHeight
	return cam
}

type InsetKey struct {
	Person string
	X, Y   int
	Tick   int
	Night  int
}

func (r *Renderer) InsetKey(sc Scene) (InsetKey, bool) {
	mark, ok := r.Focus()
	if !ok {
		return InsetKey{}, false
	}
	return InsetKey{Person: mark.Person, X: int(mark.Pos.X * render.TileWidth * InsetZoom), Y: int(mark.Pos.Y * render.TileHeight * InsetZoom),
		Tick: int(sc.Time * insetFPS), Night: int(Darkness(sc.DayPhase) * 20)}, true
}

func (r *Renderer) DrawInset(dst *ebiten.Image, sc Scene) bool {
	mark, ok := r.Focus()
	if !ok {
		return false
	}
	b := dst.Bounds()
	cullW, cullH, r.inset = float64(b.Dx()), float64(b.Dy()), true
	defer func() { cullW, cullH, r.inset = ViewWidth, ViewHeight, false }()
	cam := InsetCamera(sc.Site, mark.Pos, b.Dx(), b.Dy())
	cam.Y -= r.HarborOf().Lift(mark.Pos)
	dst.Fill(tileColors[view.TileWater])
	r.drawTerrain(dst, cam, sc, b)
	r.drawForumGround(dst, cam, sc)
	r.drawKeepLand(dst, cam, sc)
	r.drawLivingGround(dst, cam, sc)
	r.drawDecks(dst, cam, sc)
	r.drawRollRoads(dst, cam, sc)
	restore := r.clipTo(b)
	items := r.sceneItems(cam, sc)
	restore()
	for _, it := range items {
		if it.Draw != nil {
			it.Draw(dst)
		}
	}
	drawNight(dst, sc.DayPhase)
	r.drawLifeGlow(dst, cam, sc)
	r.drawLivingGlow(dst, cam, sc)
	return true
}
