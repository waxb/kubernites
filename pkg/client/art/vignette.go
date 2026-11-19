package art

import (
	"hash/fnv"
	"image"
	"math"
	"sort"
	"strings"
)

const (
	VignetteH      = 84
	VignetteMinW   = 220
	VignetteMaxW   = 400
	VignetteFrames = 12
	VignetteFPS    = 8
	ThumbW         = 92
	ThumbH         = 44
)

const (
	SceneOOMKilled     = "oomkilled"
	SceneCrashLoop     = "crashloop"
	SceneImagePull     = "imagepull"
	SceneEviction      = "eviction"
	SceneStorm         = "storm"
	SceneTremor        = "tremor"
	SceneUnschedulable = "unschedulable"
	SceneDrought       = "drought"
	SceneAdoption      = "adoption"
	SceneDiskPressure  = "diskpressure"
	SceneFloorLimit    = "floorlimit"
	SceneNoSpace       = "nospace"
	ScenePreempted     = "preempted"
	SceneNoDepot       = "no-depot"
	SceneNoCrew        = "no-crew"
	SceneSpoiling      = "spoiling"
	SceneQuayFull      = "quay-full"
	SceneRushTide      = "rush-tide"
	SceneContract      = "contract"
	SceneAgeUp         = "age-up"
	SceneReforge       = "reforge"
	SceneIncident      = "incident"
	SceneEagerWatchman = "eagerwatchman"
)

type vgScene struct {
	paint func(p *vgPaint)
	focus int
}

var vgScenes = map[string]vgScene{}

func vgRegister(id string, focus int, paint func(p *vgPaint)) {
	vgScenes[id] = vgScene{paint: paint, focus: focus}
}

type vgPaint struct {
	c       *canvas
	sh      *Sheet
	frame   int
	state   uint64
	bg      uint64
	w, h    int
	cx      int
	arg     string
	missing []string
}

func VignetteScenes() []string {
	out := make([]string, 0, len(vgScenes))
	for id := range vgScenes {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func SceneBase(scene string) string {
	base, _, _ := strings.Cut(scene, ":")
	return base
}

func HasVignette(scene string) bool {
	_, ok := vgScenes[SceneBase(scene)]
	return ok
}

func VignetteWidth(w int) int {
	return min(max(w, VignetteMinW), VignetteMaxW)
}

func ThumbRect(scene string, w int) image.Rectangle {
	w = VignetteWidth(w)
	s, ok := vgScenes[SceneBase(scene)]
	if !ok {
		s = vgScenes[SceneIncident]
	}
	x := min(max(w/2+s.focus-ThumbW/2, 0), w-ThumbW)
	y := VignetteH - ThumbH - 6
	return image.Rect(x, y, x+ThumbW, y+ThumbH)
}

func (s *Sheet) Vignette(scene string, seed uint64, w, frame int) *image.NRGBA {
	img, _ := s.paintVignette(scene, seed, w, frame)
	return img
}

func (s *Sheet) VignetteMissing(scene string, seed uint64, w, frame int) []string {
	_, missing := s.paintVignette(scene, seed, w, frame)
	return missing
}

func (s *Sheet) paintVignette(scene string, seed uint64, w, frame int) (*image.NRGBA, []string) {
	w = VignetteWidth(w)
	base, arg, _ := strings.Cut(scene, ":")
	sc, ok := vgScenes[base]
	if !ok {
		base, sc = SceneIncident, vgScenes[SceneIncident]
	}
	if s == nil {
		s = &Sheet{sprites: map[string]vgSprite{}}
	}
	p := &vgPaint{c: newCanvas(w, VignetteH), sh: s, frame: ((frame % VignetteFrames) + VignetteFrames) % VignetteFrames,
		state: seed ^ vgHash(base), bg: vgHash(base), w: w, h: VignetteH, cx: w / 2, arg: arg}
	sc.paint(p)
	p.edges()
	return p.c.img, p.missing
}

func vgHash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func vgMix(state *uint64) uint64 {
	*state += 0x9e3779b97f4a7c15
	z := *state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func (p *vgPaint) next() uint64 {
	return vgMix(&p.state)
}

func (p *vgPaint) intn(n int) int {
	if n <= 1 {
		return 0
	}
	return int(p.next() % uint64(n))
}

func (p *vgPaint) bgIntn(n int) int {
	if n <= 1 {
		return 0
	}
	return int(vgMix(&p.bg) % uint64(n))
}

func (p *vgPaint) jit(n int) int {
	return p.intn(2*n+1) - n
}

func (p *vgPaint) pick(list []string) string {
	return list[p.intn(len(list))]
}

func (p *vgPaint) cyc(n int) int {
	return p.frame % max(n, 1)
}

func (p *vgPaint) phase() float64 {
	return 2 * math.Pi * float64(p.frame) / VignetteFrames
}

func (p *vgPaint) bob(amp float64, offset float64) int {
	return int(math.Round(amp * math.Sin(p.phase()+offset)))
}

func (p *vgPaint) sides(fill func(x int, left bool)) {
	for x := p.cx - 118; x > 6; x -= 34 {
		fill(x, true)
	}
	for x := p.cx + 118; x < p.w-6; x += 34 {
		fill(x, false)
	}
}
