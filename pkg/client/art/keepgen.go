package art

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

const (
	keepFoot   = 3
	keepPrefix = "building/keep-"
)

var (
	KeepSizes   = []string{"tower", "stone", "great", "citadel"}
	KeepClasses = []string{"balanced", "compute", "memory", "rich"}
	keepHeights = map[string]int{"tower": 98, "stone": 80, "great": 102, "citadel": 126}
	keepRoofs   = []ramp{blueR, redR, greenR, slateR, tealR, woodR}
	keepCloths  = map[string]ramp{"balanced": greenR, "compute": orangeR, "memory": tealR, "rich": goldR}
)

type KeepLook struct {
	Size  string
	Class string
	Age   int
	Seed  uint32
}

func keepOneOf(v string, set []string, fallback string) string {
	for _, s := range set {
		if s == v {
			return v
		}
	}
	return fallback
}

func KeepLookOf(size, class, node string, age int) KeepLook {
	return KeepLook{Size: keepOneOf(size, KeepSizes, "stone"), Class: keepOneOf(class, KeepClasses, "balanced"), Age: clampAge(age), Seed: NameSeed(node)}
}

func (k KeepLook) Kind() string { return "keep-" + k.Size + "-" + k.Class }

func (k KeepLook) Name(state string) string {
	return fmt.Sprintf("%s/%s/%d/%08x", "building/"+k.Kind(), state, k.Age, k.Seed)
}

func ParseKeepName(name string) (KeepLook, string, bool) {
	parts := strings.Split(strings.TrimPrefix(name, keepPrefix), "/")
	if !strings.HasPrefix(name, keepPrefix) || len(parts) != 4 {
		return KeepLook{}, "", false
	}
	kind := strings.SplitN(parts[0], "-", 2)
	age, err := strconv.Atoi(parts[2])
	seed, err2 := strconv.ParseUint(parts[3], 16, 32)
	if len(kind) != 2 || err != nil || err2 != nil || !keepState(parts[1]) {
		return KeepLook{}, "", false
	}
	k := KeepLook{Size: keepOneOf(kind[0], KeepSizes, ""), Class: keepOneOf(kind[1], KeepClasses, ""), Age: age, Seed: uint32(seed)}
	if k.Size == "" || k.Class == "" || age != clampAge(age) {
		return KeepLook{}, "", false
	}
	return k, parts[1], true
}

func keepState(state string) bool {
	for _, s := range BuildingStates {
		if s == state {
			return true
		}
	}
	return false
}

func keepSize(k KeepLook) size {
	w := keepFoot*TileWidth + 4
	h := keepFoot*TileHeight + keepHeights[k.Size] + 4
	return size{w: w, h: h, ax: w / 2, ay: h - 2}
}

func keepLook(k KeepLook) look {
	l := look{age: clampAge(k.Age), variant: int(k.Seed % BuildingVariants), seed: int(k.Seed%1000003)*7 + 29}
	l.id = keepRoofs[int(k.Seed>>7)%len(keepRoofs)]
	if k.Size == "tower" {
		l.id = purpleR
	}
	return l
}

func keepCloth(k KeepLook) ramp {
	if k.Size == "tower" {
		return purpleR
	}
	return keepCloths[k.Class]
}

func keepLayers(k KeepLook) (*canvas, *canvas, look) {
	sz := keepSize(k)
	gnd, body := newCanvas(sz.w, sz.h), newCanvas(sz.w, sz.h)
	body.enableTags()
	l := keepLook(k)
	fr := isoFrame(body, keepFoot, sz.ax, sz.ay).with(l)
	fr.gnd = gnd
	fr.paintKeep(k)
	return gnd, body, l
}

func readyKeep(k KeepLook) *canvas {
	c := newCanvas(keepSize(k).w, keepSize(k).h)
	gnd, body, l := keepLayers(k)
	finishRough(c, gnd, body, l)
	return c
}

func KeepSprite(k KeepLook, state string) (*image.NRGBA, Region) {
	sz := keepSize(k)
	c := keepCanvas(k, state)
	flags := anchored(c.flags, sz.ax, sz.ay)
	if state != "ready" {
		flags = nil
	}
	return c.img, Region{W: sz.w, H: sz.h, AnchorX: sz.ax, AnchorY: sz.ay, Flags: flags}
}

func keepCanvas(k KeepLook, state string) *canvas {
	switch {
	case state == "dark":
		c := readyKeep(k)
		c.mapPixels(dusk)
		return c
	case strings.HasPrefix(state, "construction-"):
		return keepConstruction(k, state)
	}
	return readyKeep(k)
}

func keepConstruction(k KeepLook, state string) *canvas {
	sz := keepSize(k)
	stage, _ := strconv.Atoi(strings.TrimPrefix(state, "construction-"))
	stage = min(max(stage, 0), 2)
	dst := newCanvas(sz.w, sz.h)
	gnd, _, l := keepLayers(k)
	dst.paste(gnd, 0, 0, false)
	fr := isoFrame(dst, keepFoot, sz.ax, sz.ay).with(l)
	if stage > 0 {
		ready := readyKeep(k)
		cut := sz.ay - int(float64(keepFoot*TileHeight+keepHeights[k.Size])*[]float64{0, 0.4, 0.75}[stage])
		for y := cut; y < sz.h; y++ {
			for x := 0; x < sz.w; x++ {
				if col := ready.at(x, y); col.A > 0 {
					dst.dot(x, y, col)
				}
			}
		}
		scaffold(fr, keepFoot, float64(sz.ay-cut))
	}
	stakes(fr, keepFoot)
	if stage == 0 {
		fr.crate(keepFoot*0.55, keepFoot*0.8, 0.3)
		fr.c.outline(ink)
	}
	return dst
}
