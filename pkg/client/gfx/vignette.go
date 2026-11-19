package gfx

import (
	"hash/fnv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

const VignetteKeep = 96

var calamityScenes = map[string]string{
	view.TagOOMKilled: art.SceneOOMKilled, view.TagCrashLoop: art.SceneCrashLoop, view.TagImagePull: art.SceneImagePull,
	view.TagEvicted: art.SceneEviction, view.TagThrottled: art.SceneDrought, view.TagUnhoused: art.SceneUnschedulable,
	view.CalamityNoDepot: art.SceneNoDepot, view.CalamityNoCrew: art.SceneNoCrew, view.CalamitySpoiling: art.SceneSpoiling,
	art.SceneQuayFull: art.SceneQuayFull, view.CalamityStalled: art.SceneUnschedulable,
}

var chronicleScenes = map[string]string{
	"tremor": art.SceneTremor, "crashloop": art.SceneCrashLoop, "imagepull": art.SceneImagePull,
	"unschedulable": art.SceneUnschedulable, "drought": art.SceneDrought, "adoption": art.SceneAdoption,
	"eviction": art.SceneEviction, "storm": art.SceneStorm, "oomkilled": art.SceneOOMKilled,
	"diskpressure": art.SceneDiskPressure, "floorlimit": art.SceneFloorLimit, "nospace": art.SceneNoSpace,
	"preempted": art.ScenePreempted, "eagerwatchman": art.SceneEagerWatchman, "stalledrollout": art.SceneUnschedulable,
}

func CalamityScene(kind string) string {
	if s, ok := calamityScenes[kind]; ok {
		return s
	}
	return art.SceneIncident
}

func ChronicleScene(kind string) string {
	if s, ok := chronicleScenes[kind]; ok {
		return s
	}
	return art.SceneIncident
}

func ContractScene(box string, crowned, sealed bool) string {
	parts := []string{art.SceneContract}
	if box != "" {
		parts = append(parts, box)
	}
	if crowned {
		parts = append(parts, "crown")
	}
	if sealed {
		parts = append(parts, "sealed")
	}
	return strings.Join(parts, ":")
}

func VignetteSeed(id string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(id))
	return h.Sum64()
}

func VignetteFrame(seconds float64, still bool) int {
	if still || seconds <= 0 {
		return 0
	}
	return int(seconds*art.VignetteFPS) % art.VignetteFrames
}

type vigKey struct {
	scene string
	seed  uint64
	w     int
	frame int
}

type vignettes struct {
	sheet  *art.Sheet
	images map[vigKey]*ebiten.Image
	order  []vigKey
}

func newVignettes(sheet *art.Sheet) *vignettes {
	return &vignettes{sheet: sheet, images: map[vigKey]*ebiten.Image{}}
}

func (l *Library) Vignette(scene string, seed uint64, w, frame int) (*ebiten.Image, bool) {
	if l == nil || l.vig == nil {
		return nil, false
	}
	return l.vig.image(vigKey{scene: scene, seed: seed, w: art.VignetteWidth(w), frame: frame % art.VignetteFrames}), true
}

func (v *vignettes) image(k vigKey) *ebiten.Image {
	if img, ok := v.images[k]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(v.sheet.Vignette(k.scene, k.seed, k.w, k.frame))
	v.images[k] = img
	v.order = append(v.order, k)
	if len(v.order) > VignetteKeep {
		delete(v.images, v.order[0])
		v.order = append([]vigKey(nil), v.order[1:]...)
	}
	return img
}

func (l *Library) VignetteCached() int {
	if l == nil || l.vig == nil {
		return 0
	}
	return len(l.vig.images)
}
