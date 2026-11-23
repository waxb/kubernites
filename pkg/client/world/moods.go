package world

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	sweatZoom    = .75
	snoreZoom    = 1.0
	shiverRate   = 7.0
	voidScale    = .6
	faintBoxSize = 1
	faintBoxSide = 7.0
	faintBoxDrop = 2.0
	faintHead    = 14.0
	faintLift    = 6.0
	faintAngle   = -math.Pi / 2
)

func (r *Renderer) figureItems(cam render.Camera, sc Scene, f Figure, rt route) []Drawable {
	if f.Ride.Aboard(sc.Time) {
		r.HarborOf().seat(f)
		return r.FamiliarItems(cam, sc, f)
	}
	out := []Drawable{r.FigureDrawable(cam, sc, f)}
	if f.deliver && !f.Pod.System {
		if d, ok := r.DeliveryItem(cam, sc, f.since, rt.bank); ok {
			out = append(out, d)
		}
	}
	return append(out, r.FamiliarItems(cam, sc, f)...)
}

func (r *Renderer) FigureDrawable(cam render.Camera, sc Scene, f Figure) Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, f.Pose.Pos)
	sy -= r.HarborOf().Lift(f.Pose.Pos) * zoom
	variant, tier := LookOf(f.Pod)
	action := f.Pose.Action
	if f.Load != nil {
		action = gfx.HoldAction
	}
	names := r.LookNames(NameKey{Role: f.role, Action: action, Dir: f.Pose.Dir, frame: f.Pose.Frame, variant: variant, Tier: tier})
	air := AirOf(f.Pod, f.role)
	if air.Bob > 0 && f.Pose.Action == "walk" && f.Pose.Frame%2 == 1 && f.Mood != MoodRest && f.Mood != MoodSick {
		sy -= zoom
	}
	pod := f.Pod
	shown := r.ShownName(names)
	hit := &Hit{Kind: HitWalker, ID: PodKey(pod), Pod: pod, Rect: SpriteRect(r.Lib, shown, sx, sy, zoom, 12, 20)}
	if f.Mood != MoodRest && f.Mood != MoodSick {
		hit.Shapes = ShapeAt(shown, sx, sy, zoom)
	}
	d := Drawable{person: personWorker, Depth: render.DepthKey(f.Pose.Pos.X, f.Pose.Pos.Y) + .05, Hit: hit, stand: stand(f.Pose.Pos, sx, sy)}
	switch f.Mood {
	case MoodRest:
		d.Draw = r.restingDraw(sc, f, names, sx, sy, zoom, air)
	case MoodSick:
		d.Draw = r.sickDraw(sc, f, names, sx, sy, zoom, air)
	case MoodStrained:
		d.Draw = r.uprightDraw(sc, f, names, sx, sy, zoom, flushTint, air)
	default:
		d.Draw = r.uprightDraw(sc, f, names, sx, sy, zoom, [3]float32{}, air)
	}
	return d
}

func (r *Renderer) uprightDraw(sc Scene, f Figure, names []string, sx, sy, zoom float64, tint [3]float32, air gfx.Air) func(*ebiten.Image) {
	strained := f.Mood == MoodStrained
	load, pose, coin, mark := f.Load, f.Pose, f.Pose.Carry && f.role != ScribeRole, f.mark
	now, reduce, seed := sc.Time, sc.Reduce, f.seedOf()
	return func(dst *ebiten.Image) {
		drawn, ok := r.Lib.DrawFirstAir(dst, names, sx, sy, zoom, render.SpriteOptions{Tint: tint}, air)
		if !ok {
			fallbackFigure(dst, sx, sy, zoom, 10, look{tint: tint})
		}
		switch {
		case load != nil:
			r.drawLoad(dst, load, pose, sx, sy, zoom)
		case coin:
			r.Lib.Draw(dst, gfx.EffectName("coin", int(now*8)), sx, sy-r.headLift(drawn, zoom), zoom, render.SpriteOptions{})
		}
		if strained && mark && zoom >= sweatZoom {
			r.drawSweat(dst, sx, sy, zoom, now+seedOffset(seed), reduce)
		}
	}
}

func (r *Renderer) restingDraw(sc Scene, f Figure, names []string, sx, sy, zoom float64, air gfx.Air) func(*ebiten.Image) {
	local := sc.Time + seedOffset(f.seedOf())
	dir, snore := f.Pose.Dir, f.mark && zoom >= snoreZoom && !sc.Reduce
	return func(dst *ebiten.Image) {
		r.drawSeated(dst, names, sx, sy, zoom, logSeat, dir, look{air: air})
		if snore {
			r.drawSnore(dst, sx, sy, zoom, local)
		}
	}
}

func (r *Renderer) sickDraw(sc Scene, f Figure, names []string, sx, sy, zoom float64, air gfx.Air) func(*ebiten.Image) {
	shiver := 0.0
	if !sc.Reduce && Frame(sc.Time, shiverRate)%2 == 1 {
		shiver = zoom
	}
	oom := f.Pod.OOM()
	t := sc.Time
	if sc.Reduce {
		t = 0
	}
	role, mark, dir := f.role, f.mark, f.Pose.Dir
	return func(dst *ebiten.Image) {
		if oom {
			r.drawFaint(dst, role, names, sx, sy, zoom, t, mark, air)
			return
		}
		r.drawSeated(dst, names, sx+shiver, sy, zoom, groundSeat, dir, look{tint: sickTint, air: air})
		if mark {
			r.drawDizzy(dst, sx, sy+2*zoom, zoom, t)
		}
	}
}

func DroppedBox(role string) (string, bool) {
	kind, ok := TradeKindIndex(role)
	if !ok {
		return "", false
	}
	return gfx.BoxName(faintBoxSize, KindName(kind), 0), true
}

func (r *Renderer) drawFaint(dst *ebiten.Image, role string, names []string, sx, sy, zoom, t float64, mark bool, air gfx.Air) {
	if box, ok := DroppedBox(role); ok {
		r.Lib.Draw(dst, box, sx+faintBoxSide*zoom, sy+faintBoxDrop*zoom, zoom, render.SpriteOptions{})
	}
	drawLying(dst, r.Lib, names, sx, sy, zoom, faintAngle, look{tint: sickTint, air: air})
	if mark {
		r.Lib.Draw(dst, gfx.EffectName("void", int(t*6)), sx-faintHead*zoom, sy-faintLift*zoom, zoom, render.SpriteOptions{Scale: voidScale})
	}
}
