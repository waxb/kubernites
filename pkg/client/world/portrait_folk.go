package world

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
)

var portraitWalkers = map[string]string{
	"porter": "porter", "hauler": "hauler", "clerk": "clerk", "sealwright": "sealwright", "courier": "courier",
	"reeve": "steward", "foreman": "steward", "crew": "porter", "watchman": "lamplighter", "scriptorium": "scribe",
	"imp": "wanderer",
}

var portraitVignettes = map[string]string{
	"storm": art.SceneStorm, "famine": art.SceneDrought, "hungry": art.SceneDrought, "spoiled": art.SceneSpoiling,
	"floor": art.SceneFloorLimit, "void": art.SceneNoSpace, "tide": art.SceneRushTide, "drain": art.SceneEviction,
	"guttering": art.SceneCrashLoop, "contract": art.SceneContract, "age": art.SceneAgeUp, "chronicle": art.SceneIncident,
}

const folkZoom = 2

func (p portrait) folk(role string) ([]piece, bool) {
	cx, cy := float64(p.box.X)+float64(p.box.W)/2, p.ground-8
	work := p.walker(role, "work", "sw", cx, cy, folkZoom)
	if len(work) == 0 {
		return nil, false
	}
	return append(work, p.stroll(role, folkZoom, .5)...), true
}

func (p portrait) vignette(scene string) ([]piece, bool) {
	img, ok := p.lib.Vignette(scene, gfx.VignetteSeed(scene), art.VignetteMinW, gfx.VignetteFrame(p.t, false))
	if !ok || img == nil {
		return nil, false
	}
	focus := art.ThumbRect(scene, art.VignetteMinW)
	left := min(max(focus.Min.X+focus.Dx()/2-p.box.W/2, 0), art.VignetteMinW-p.box.W)
	crop := img.SubImage(image.Rect(left, 0, left+p.box.W, art.VignetteH)).(*ebiten.Image)
	return []piece{{Img: crop, X: float64(p.box.X), Y: float64(p.box.Y + p.box.H - art.VignetteH)}}, true
}

var portraitEmblems = map[string]string{
	"badge": art.PageIconName("marks"), "standing": art.PageIconName("ranks"), "ward": art.PageIconName("charters"),
	"spread": art.PageIconName("territory"), "shrine": art.PageIconName("daemons"),
	"nines": "icon/alert", "grimoire": "icon/tech", "academy": "icon/academy", "overlay": "icon/probe",
	"loom": "icon/loom-builders", "banner": "icon/renown", "rail": "icon/logs", "hourglass": "icon/speed",
}

const (
	emblemBob   = 2.0
	emblemBobHz = .8
)

func (p portrait) emblem(name string) ([]piece, bool) {
	_, reg, ok := p.lib.Lookup(name)
	if !ok {
		return nil, false
	}
	zoom := gfx.PortraitScale(reg.W, reg.H, p.box.W*3/4, p.box.H/2)
	bob := emblemBob * math.Sin(2*math.Pi*p.t*emblemBobHz)
	x := float64(p.box.X) + float64(p.box.W)/2 - (float64(reg.W)/2-float64(reg.AnchorX))*zoom
	y := float64(p.box.Y) + float64(p.box.H)*.42 - (float64(reg.H)/2-float64(reg.AnchorY))*zoom + bob
	return []piece{{Name: name, X: x, Y: y, Zoom: zoom}}, true
}
