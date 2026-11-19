package world

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	ErrandSchedule = "schedule"
	ErrandRaise    = "raise"
	ErrandLight    = "light"
	ErrandDouse    = "douse"
	ErrandScribble = "scribble"
	ErrandFallen   = "fallen"

	PhaseOut  = "out"
	PhaseAct  = "act"
	PhaseBack = "back"
	PhaseStay = "stay"

	ErrandSpeed  = 1.7
	errandFade   = .25
	ScribbleLife = .9
	FallLife     = 3.2
	fallTopple   = .35
	fallFade     = 1.2
	MaxErrands   = 32
	MaxLeg       = 5.0
)

var ErrandActs = map[string]float64{ErrandSchedule: .9, ErrandRaise: 1.1, ErrandLight: 1.0, ErrandDouse: 1.0}

var podBound = map[string]bool{ErrandRaise: true, ErrandLight: true, ErrandDouse: true}

var errandFallback = map[string]string{"quartermaster": "builder", "steward": "builder", "reeve": "builder", "clerk": "codex-keeper"}

type ErrandStep struct {
	Pose     Pose
	Phase    string
	Progress float64
	Alpha    float64
	Skiff    string
}

func ErrandLife(kind string, from, to Vec) float64 {
	switch kind {
	case ErrandScribble:
		return ScribbleLife
	case ErrandFallen:
		return FallLife
	}
	return 2*min(Dist(from, to)/ErrandSpeed, MaxLeg) + ErrandActs[kind]
}

func ErrandAt(e Errand, to Vec, t float64) (ErrandStep, bool) {
	age := t - e.Start
	if e.Life <= 0 || age < 0 || age > e.Life {
		return ErrandStep{}, false
	}
	act := min(ErrandActs[e.Kind], e.Life)
	walk := (e.Life - act) / 2
	step := ErrandStep{Alpha: fadeAlpha(age, e.Life)}
	switch {
	case act == 0:
		step.Phase, step.Progress = PhaseStay, age/e.Life
		step.Pose = Pose{Pos: to, Action: "work", Dir: "se", Frame: Frame(age, 8)}
	case age < walk:
		step.Phase, step.Progress = PhaseOut, age/walk
		step.Pose = Pose{Pos: Lerp(e.From, to, age/walk), Action: "walk", Dir: Facing(e.From, to), Frame: Frame(age, 7)}
	case age < walk+act:
		step.Phase, step.Progress = PhaseAct, (age-walk)/act
		step.Pose = Pose{Pos: to, Action: "work", Dir: Facing(e.From, to), Frame: Frame(age, 8)}
	default:
		p := min((age-walk-act)/max(walk, .001), 1)
		step.Phase, step.Progress = PhaseBack, p
		step.Pose = Pose{Pos: Lerp(to, e.From, p), Action: "walk", Dir: Facing(to, e.From), Frame: Frame(age, 7)}
	}
	return step, true
}

func fadeAlpha(age, life float64) float64 {
	return min(min(age/errandFade, (life-age)/errandFade), 1)
}

func (r *Renderer) ErrandTarget(e Errand) Vec {
	if home, ok := r.Homes[e.Pod]; ok && podBound[e.Kind] {
		return Vec{home.X + .55, home.Y + .95}
	}
	return e.To
}

func (r *Renderer) ErrandItems(cam render.Camera, sc Scene) []Drawable {
	out := []Drawable{}
	drawn := 0
	for _, e := range sc.Errands {
		if drawn >= MaxErrands {
			break
		}
		var items []Drawable
		switch e.Kind {
		case ErrandScribble:
			continue
		case ErrandFallen:
			items = r.fallenItems(cam, sc, e)
		default:
			items = r.WalkingErrand(cam, sc, e)
		}
		if len(items) > 0 {
			drawn++
			out = append(out, items...)
		}
	}
	return out
}

func (r *Renderer) WalkingErrand(cam render.Camera, sc Scene, e Errand) []Drawable {
	to := r.ErrandTarget(e)
	step, ok := ErrandAt(e, to, sc.Time)
	if !ok {
		return nil
	}
	step = r.RoutedStep(sc, e, to, step)
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, step.Pose.Pos)
	if !onScreen(sx, sy, zoom) {
		return nil
	}
	names := r.namesFor(e.Role, errandFallback[e.Role], step.Pose.Action, step.Pose.Dir, step.Pose.Frame)
	alpha := float32(step.Alpha)
	skiff := art.FerryLook{Skiff: true, Age: sc.Age, Dir: step.Skiff, Frame: Frame(sc.Time, ferryPoleHz)}
	person := Drawable{person: e.Role, Depth: render.DepthKey(step.Pose.Pos.X, step.Pose.Pos.Y) + .05, stand: stand(step.Pose.Pos, sx, sy), Draw: func(dst *ebiten.Image) {
		if skiff.Dir != "" {
			r.drawSkiff(dst, skiff, names, sx, sy, zoom)
			return
		}
		if _, ok := r.Lib.DrawFirst(dst, names, sx, sy, zoom, render.SpriteOptions{Alpha: alpha}); !ok {
			fallbackFigure(dst, sx, sy, zoom, 10, look{alpha: alpha})
		}
	}}
	return append([]Drawable{person}, r.errandProps(cam, sc, e, step, to)...)
}

func (r *Renderer) errandProps(cam render.Camera, sc Scene, e Errand, step ErrandStep, to Vec) []Drawable {
	zoom := cam.Scale()
	switch {
	case e.Kind == ErrandSchedule && step.Phase != PhaseBack:
		sx, sy := ScreenPointOf(cam, sc.Site, step.Pose.Pos)
		hue := BannerHue(e.Pod)
		return []Drawable{{Depth: render.DepthKey(step.Pose.Pos.X, step.Pose.Pos.Y) + .06, Draw: func(dst *ebiten.Image) {
			r.drawBanner(dst, sx+4*zoom, sy, zoom, hue, sc.Time)
		}}}
	case step.Phase != PhaseAct:
		return nil
	}
	fx, ok := actEffects[e.Kind]
	if !ok || (sc.Reduce && e.Kind == ErrandRaise) {
		return nil
	}
	sx, sy := ScreenPointOf(cam, sc.Site, Vec{to.X - .15, to.Y - .35})
	frame := min(int(step.Progress*6), 5)
	opts := render.SpriteOptions{Scale: actScales[e.Kind]}
	return []Drawable{{Depth: render.DepthKey(to.X, to.Y) + .08, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, gfx.EffectName(fx, frame), sx, sy, zoom, opts)
	}}}
}

var actScales = map[string]float64{ErrandLight: .5, ErrandDouse: .6}

var actEffects = map[string]string{ErrandRaise: "sparkle", ErrandLight: "fire", ErrandDouse: "smoke"}

func (r *Renderer) fallenItems(cam render.Camera, sc Scene, e Errand) []Drawable {
	age := sc.Time - e.Start
	if age < 0 || age > e.Life {
		return nil
	}
	zoom := cam.Scale()
	at := e.To
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	angle := -math.Pi / 2 * min(age/fallTopple, 1)
	alpha := float32(min((e.Life-age)/fallFade, 1))
	names := r.LookNames(NameKey{Role: e.Role, Action: "idle", Dir: "se", variant: FaceOf(e.Role, e.Pod), Tier: max(e.Tier, 1)})
	items := []Drawable{{person: fallenPerson, Depth: render.DepthKey(at.X, at.Y) + .04, Draw: func(dst *ebiten.Image) {
		drawLying(dst, r.Lib, names, sx, sy, zoom, angle, look{tint: fallTint, alpha: alpha})
	}}}
	if dust := age - fallTopple; dust >= 0 && dust < .5 {
		name := gfx.EffectName("dust", int(dust/.5*6))
		items = append(items, Drawable{Depth: render.DepthKey(at.X, at.Y) + .09, Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx-8*zoom, sy, zoom, render.SpriteOptions{Alpha: .8})
		}})
	}
	return items
}
