package art

import (
	"fmt"
	"image"
	"strconv"
	"strings"
	"sync"
)

const (
	BuildIconW      = 24
	BuildIconH      = 22
	buildIconFit    = 20
	buildIconFoot   = 1
	buildIconPrefix = "buildicon/"
	keepIconPrefix  = buildIconPrefix + "keep-"
)

func BuildIconName(kind string, age int) string { return buildIconPrefix + AgedPortrait(kind, age) }

func (k KeepLook) BuildIconName() string {
	return fmt.Sprintf("%s%s/%d/%08x", buildIconPrefix, k.Kind(), k.Age, k.Seed)
}

func ParseKeepIconName(name string) (KeepLook, bool) {
	rest, ok := strings.CutPrefix(name, keepIconPrefix)
	if !ok {
		return KeepLook{}, false
	}
	return parseKeepPath(rest)
}

func parseKeepPath(rest string) (KeepLook, bool) {
	parts := strings.Split(rest, "/")
	if len(parts) != 3 {
		return KeepLook{}, false
	}
	kind := strings.SplitN(parts[0], "-", 2)
	age, err := strconv.Atoi(parts[1])
	seed, err2 := strconv.ParseUint(parts[2], 16, 32)
	if len(kind) != 2 || err != nil || err2 != nil || age != clampAge(age) {
		return KeepLook{}, false
	}
	k := KeepLook{Size: keepOneOf(kind[0], KeepSizes, ""), Class: keepOneOf(kind[1], KeepClasses, ""), Age: age, Seed: uint32(seed)}
	return k, k.Size != "" && k.Class != ""
}

func buildIconRegion() Region {
	return Region{W: BuildIconW, H: BuildIconH, AnchorX: BuildIconW / 2, AnchorY: BuildIconH - buildIconFoot}
}

func KeepBuildIcon(k KeepLook) (*image.NRGBA, Region) {
	c := newCanvas(BuildIconW, BuildIconH)
	paintBuildIcon(c, readyKeep(k))
	return c.img, buildIconRegion()
}

func paintBuildIcon(c *canvas, sprite *canvas) {
	fit := fitSprite(sprite, buildIconFit)
	c.paste(fit, (BuildIconW-fit.w)/2, BuildIconH-buildIconFoot-fit.h, false)
}

type shotKey struct {
	kind string
	age  int
}

type shot struct {
	once sync.Once
	c    *canvas
}

type shotMemo struct {
	mu    sync.Mutex
	shots map[shotKey]*shot
}

func newShotMemo() *shotMemo { return &shotMemo{shots: map[shotKey]*shot{}} }

func (m *shotMemo) get(kind string, age int) *canvas {
	m.mu.Lock()
	k := shotKey{kind, age}
	s, ok := m.shots[k]
	if !ok {
		s = &shot{}
		m.shots[k] = s
	}
	m.mu.Unlock()
	s.once.Do(func() { s.c = agedBuilding(kind, age) })
	return s.c
}

func addBuildIcons(b *atlasBuilder, m *shotMemo) {
	r := buildIconRegion()
	for _, k := range append(append([]string{}, BuildingKinds...), StallKinds...) {
		for age := 0; age < AgeTiers; age++ {
			kind, a := k, age
			b.add(BuildIconName(kind, a), r.W, r.H, r.AnchorX, r.AnchorY, func(c *canvas) { paintBuildIcon(c, m.get(kind, a)) })
		}
	}
}
