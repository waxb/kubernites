package world

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	riderLegs    = 4
	riderDepth   = .001
	houndLegs    = 2
	ferryWake    = .55
	landingAside = .38
	landingAhead = .3
	laneAgeKey   = "ferry/"
)

type boatKey struct{ lane, boat int }

func (h *Harbor) seat(f Figure) {
	if h.riders == nil {
		h.riders = map[boatKey][]Figure{}
	}
	k := boatKey{f.Ride.Hop.Lane, f.Ride.Sail.Boat}
	h.riders[k] = append(h.riders[k], f)
}

func (r *Renderer) laneAge(sc Scene, l FerryLane) int {
	clock := r.agesOf()
	clock.track(sc.Age, sc.Time)
	age, _ := clock.phase(laneAgeKey + l.Isle)
	return age
}

func (r *Renderer) ferryItems(cam render.Camera, sc Scene) []Drawable {
	h := r.HarborOf()
	riders := h.riders
	h.riders = nil
	out := []Drawable{}
	for li, l := range h.Lanes {
		age := r.laneAge(sc, l)
		out = r.jettyItems(r.jettyItems(out, cam, sc, l.Home, age), cam, sc, l.Away, age)
		for k := 0; k < FerryBoats; k++ {
			out = r.ferryBoatItems(out, cam, sc, l.Boat(k, sc.Time), riders[boatKey{li, k}], age)
		}
	}
	return out
}

func (r *Renderer) jettyItems(out []Drawable, cam render.Camera, sc Scene, j Jetty, age int) []Drawable {
	zoom := cam.Scale()
	kind := pierKind(j.Out)
	for i, v := range j.Planks {
		if sx, sy := ScreenPointOf(cam, sc.Site, v); onScreen(sx, sy, zoom) {
			piece := HavenProp{Kind: kind, Variant: i % pieceKinds}
			names := r.HarborOf().pieceNames(piece, flatLevel, age)
			out = append(out, Drawable{Depth: flatDepth(v), Draw: func(dst *ebiten.Image) { r.drawPortPiece(dst, names, piece, sx, sy, zoom) }})
		}
	}
	at := j.Landing()
	if sx, sy := ScreenPointOf(cam, sc.Site, at); onScreen(sx, sy, zoom) {
		names := r.HarborOf().pieceNames(HavenProp{Kind: landingKind}, flatLevel, age)
		out = append(out, Drawable{Depth: render.DepthKey(at.X, at.Y), Draw: func(dst *ebiten.Image) {
			r.Lib.DrawFirst(dst, names, sx, sy, zoom, render.SpriteOptions{})
		}})
	}
	return out
}

func (r *Renderer) ferryBoatItems(out []Drawable, cam render.Camera, sc Scene, pose FerryPose, riders []Figure, age int) []Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
	if !shipOnScreen(sx, sy, zoom) {
		return out
	}
	frame := pose.Frame
	if sc.Reduce {
		frame = 0
	}
	look := art.FerryLook{Age: age, Dir: pose.Dir, Frame: frame}
	sort.SliceStable(riders, func(i, j int) bool {
		a, b := riders[i].Ride.Sail.Board, riders[j].Ride.Sail.Board
		return a < b || a == b && PersonKey(riders[i].Pod) < PersonKey(riders[j].Pod)
	})
	dogs := r.deckDogs(sc, riders, zoom)
	seats := art.FerrySlots(look, min(len(riders)+len(dogs), art.FerryCapacity()))
	depth := render.DepthKey(pose.Pos.X, pose.Pos.Y)
	out = append(out, Drawable{Depth: depth, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, look.Name(), sx, sy, zoom, render.SpriteOptions{})
		for i, s := range seats {
			if i >= len(riders) {
				r.drawDeckHound(dst, dogs[i-len(riders)], sx+s.DX*zoom, sy+s.DY*zoom, zoom)
			}
		}
	}})
	for i, s := range seats[:min(len(riders), len(seats))] {
		out = append(out, r.riderItem(riders[i], sx+s.DX*zoom, sy+s.DY*zoom, zoom, depth+riderDepth*float64(i+1)))
	}
	if pose.Moving && !sc.Reduce {
		out = append(out, r.ferryWake(cam, sc, pose))
	}
	return out
}

func (r *Renderer) riderItem(f Figure, sx, sy, zoom, depth float64) Drawable {
	variant, tier := LookOf(f.Pod)
	names := r.LookNames(NameKey{Role: f.role, Action: "idle", Dir: f.Pose.Dir, variant: variant, Tier: tier})
	hit := &Hit{Kind: HitWalker, ID: f.Pod.Namespace + "/" + f.Pod.Name, Pod: f.Pod, Rect: SpriteRect(r.Lib, r.ShownName(names), sx, sy, zoom, 12, 20)}
	return Drawable{person: personWorker, Depth: depth, Hit: hit, Draw: func(dst *ebiten.Image) { r.drawRider(dst, f, sx, sy, zoom) }}
}

func (r *Renderer) drawRider(dst *ebiten.Image, f Figure, sx, sy, zoom float64) {
	variant, tier := LookOf(f.Pod)
	action := "idle"
	if f.Load != nil {
		action = gfx.HoldAction
	}
	names := r.LookNames(NameKey{Role: f.role, Action: action, Dir: f.Pose.Dir, variant: variant, Tier: tier})
	for _, n := range names {
		sheet, reg, ok := r.Lib.Lookup(n)
		if !ok {
			continue
		}
		reg.H = max(reg.AnchorY-riderLegs, 1)
		gfx.DrawRegionAir(dst, sheet, reg, gfx.SpriteGeo(reg, sx, sy, zoom, render.SpriteOptions{}), [3]float32{}, 1, AirOf(f.Pod, f.role))
		r.drawLoad(dst, f.Load, f.Pose, sx, sy, zoom)
		return
	}
}

func (r *Renderer) deckDogs(sc Scene, riders []Figure, zoom float64) []string {
	out := []string{}
	if zoom < familiarZoom {
		return out
	}
	for _, f := range riders {
		for range BoatPets(r.PetPoses(sc, f)) {
			out = append(out, f.Pose.Dir)
		}
	}
	return out
}

func (r *Renderer) drawDeckHound(dst *ebiten.Image, dir string, sx, sy, zoom float64) {
	for _, n := range []string{gfx.WalkerName(model.FamiliarHound, "idle", dir, 0), gfx.WalkerName(model.FamiliarHound, "walk", dir, 0)} {
		if sheet, reg, ok := r.Lib.Lookup(n); ok {
			reg.H = max(reg.AnchorY-houndLegs, 1)
			render.DrawAt(dst, sheet, reg, sx, sy, zoom, render.SpriteOptions{})
			return
		}
	}
}

func (r *Renderer) ferryWake(cam render.Camera, sc Scene, pose FerryPose) Drawable {
	at := offset(pose.Pos, HeadingVec(pose.Dir), -ferryWake)
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	zoom := cam.Scale()
	name := gfx.EffectName("wake", int(sc.Time*6))
	return Drawable{Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y) - shipAbove, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Alpha: .7})
	}}
}

func (r *Renderer) drawSkiff(dst *ebiten.Image, skiff art.FerryLook, names []string, sx, sy, zoom float64) {
	r.Lib.Draw(dst, skiff.Name(), sx, sy, zoom, render.SpriteOptions{})
	seat := art.FerrySlots(skiff, 1)[0]
	for _, n := range names {
		if sheet, reg, ok := r.Lib.Lookup(n); ok {
			reg.H = max(reg.AnchorY-riderLegs, 1)
			render.DrawAt(dst, sheet, reg, sx+seat.DX*zoom, sy+seat.DY*zoom, zoom, render.SpriteOptions{})
			return
		}
	}
}
