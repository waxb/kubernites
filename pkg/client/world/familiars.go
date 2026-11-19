package world

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	RavenRadius  = .55
	ravenSpin    = 1.7
	ravenLift    = 12.0
	houndBehind  = .32
	houndSide    = .22
	familiarZoom = .75
	maxFamiliars = 2
	houndHeel    = .12
)

var houndTint = [3]float32{.55, .38, .26}

func settled(mood string) bool {
	return mood == MoodRest || mood == MoodSick
}

func RavenPose(t float64, seed uint32, owner Pose, mood string) Pose {
	local := t + seedOffset(seed)
	if settled(mood) {
		spot := Vec{owner.Pos.X + .45, owner.Pos.Y - .15}
		action := "idle"
		if int(local/2.5)%3 == 0 {
			action = "work"
		}
		return Pose{Pos: spot, Action: action, Dir: "sw", Frame: Frame(local, 3)}
	}
	a := local * ravenSpin
	pos := Vec{owner.Pos.X + RavenRadius*math.Cos(a), owner.Pos.Y + RavenRadius*math.Sin(a)}
	ahead := Vec{owner.Pos.X + RavenRadius*math.Cos(a+.2), owner.Pos.Y + RavenRadius*math.Sin(a+.2)}
	action := "walk"
	if owner.Carry {
		action = "carry"
	}
	return Pose{Pos: pos, Action: action, Dir: Facing(pos, ahead), Frame: Frame(local, 8)}
}

func HoundPose(t float64, seed uint32, owner Pose, mood string) Pose {
	back := Vec{-houndBehind, houndSide}
	if owner.Dir == "sw" {
		back = Vec{houndSide, -houndBehind}
	}
	pos := Vec{owner.Pos.X + back.X, owner.Pos.Y + back.Y}
	switch {
	case settled(mood) || owner.Action == "idle":
		return Pose{Pos: pos, Action: "idle", Dir: owner.Dir, Frame: Frame(t+seedOffset(seed), 2)}
	case owner.Action == "work":
		return Pose{Pos: pos, Action: "work", Dir: owner.Dir, Frame: Frame(t+seedOffset(seed), 3)}
	case mood == MoodBrisk:
		return Pose{Pos: pos, Action: "carry", Dir: owner.Dir, Frame: owner.Frame}
	}
	return Pose{Pos: pos, Action: "walk", Dir: owner.Dir, Frame: owner.Frame}
}

type petPose struct {
	Kind   string `json:"-"`
	name   string
	Pose   Pose `json:"-"`
	Aboard bool `json:"-"`
}

func (r *Renderer) PetPoses(sc Scene, f Figure) []petPose {
	pets, names := PetsOf(f.Pod, f.Person), PetNamesOf(f.Pod, f.Person)
	seed := f.seedOf()
	out := make([]petPose, 0, min(len(pets), maxFamiliars))
	for i, kind := range pets {
		if i >= maxFamiliars {
			break
		}
		pet := r.houndPose(sc, f, seed+uint32(i))
		if kind != model.FamiliarHound {
			pet = petPose{Kind: kind, Pose: RavenPose(sc.Time, seed+uint32(i), f.Pose, f.Mood)}
		}
		if i < len(names) {
			pet.name = names[i]
		}
		out = append(out, pet)
	}
	return out
}

func (r *Renderer) houndPose(sc Scene, f Figure, seed uint32) petPose {
	pose := HoundPose(sc.Time, seed, f.Pose, f.Mood)
	switch {
	case f.Ride.Aboard(sc.Time):
		return petPose{Kind: model.FamiliarHound, Pose: Pose{Pos: f.Pose.Pos, Action: "idle", Dir: f.Pose.Dir}, Aboard: true}
	case f.Ride != nil:
		pose.Pos = f.Ride.besideWait()
	case WetAt(sc.Site, pose.Pos) && r.HarborOf().Lift(pose.Pos) == 0:
		pose.Pos = Vec{f.Pose.Pos.X - houndHeel, f.Pose.Pos.Y - houndHeel}
	}
	return petPose{Kind: model.FamiliarHound, Pose: pose}
}

func BoatPets(pets []petPose) []string {
	out := []string{}
	for _, p := range pets {
		if p.Aboard {
			out = append(out, p.Kind)
		}
	}
	return out
}

func (r *Renderer) FamiliarItems(cam render.Camera, sc Scene, f Figure) []Drawable {
	if cam.Scale() < familiarZoom {
		return nil
	}
	var out []Drawable
	for _, pet := range r.PetPoses(sc, f) {
		var d Drawable
		switch {
		case pet.Aboard:
			continue
		case pet.Kind == model.FamiliarHound:
			d = r.houndDrawable(cam, sc, pet.Pose)
		default:
			d = r.ravenDrawable(cam, sc, pet.Pose)
		}
		if f.Pod.Name != "" && pet.name != "" {
			sx, sy := ScreenPointOf(cam, sc.Site, pet.Pose.Pos)
			d.Hit = &Hit{Kind: HitFamiliar, ID: PodKey(f.Pod), Pod: f.Pod, Pet: pet.name, Rect: SpriteRect(r.Lib, gfx.WalkerName(pet.Kind, pet.Pose.Action, pet.Pose.Dir, pet.Pose.Frame), sx, sy, cam.Scale(), 10, 10)}
		}
		out = append(out, d)
	}
	return out
}

func (r *Renderer) ravenDrawable(cam render.Camera, sc Scene, pose Pose) Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
	name := gfx.WalkerName(model.FamiliarRaven, pose.Action, pose.Dir, pose.Frame)
	flying := pose.Action == "walk" || pose.Action == "carry"
	return Drawable{Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y) + .06, Draw: func(dst *ebiten.Image) {
		if r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{}) {
			return
		}
		lift := 2.0
		if flying {
			lift = ravenLift
		}
		r.Lib.Draw(dst, gfx.EffectName("raven", pose.Frame), sx, sy-lift*zoom, zoom, render.SpriteOptions{})
	}}
}

func (r *Renderer) houndDrawable(cam render.Camera, sc Scene, pose Pose) Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
	sy -= r.HarborOf().Lift(pose.Pos) * zoom
	names := []string{gfx.WalkerName(model.FamiliarHound, pose.Action, pose.Dir, pose.Frame), gfx.WalkerName(model.FamiliarHound, "walk", pose.Dir, pose.Frame)}
	stand := []string{gfx.WalkerName("sheep", pose.Action, pose.Dir, pose.Frame), gfx.WalkerName("sheep", "walk", pose.Dir, pose.Frame)}
	return Drawable{Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y) + .05, Draw: func(dst *ebiten.Image) {
		if _, ok := r.Lib.DrawFirst(dst, names, sx, sy, zoom, render.SpriteOptions{}); ok {
			return
		}
		r.Lib.DrawFirst(dst, stand, sx, sy, zoom, render.SpriteOptions{Tint: houndTint, Scale: .8})
	}}
}
