package art

import (
	"fmt"
	"image"
	"strings"
)

const (
	towerNode          = "lone-tower"
	keepPreviewSuffix  = "-keep-preview"
	keepPortraitPrefix = "portrait/keep-"
	portraitFit        = 36
	portraitFoot       = 6
)

var keepKindSizes = map[string]string{"keep-tower": "tower", "keep-stone": "stone", "keep-great": "great", "keep-citadel": "citadel"}

func KeepPreviewNode(size string) string {
	if size == "tower" {
		return towerNode
	}
	return size + keepPreviewSuffix
}

func CanonicalKeep(kind string, age int) (KeepLook, bool) {
	size, ok := keepKindSizes[kind]
	if !ok {
		return KeepLook{}, false
	}
	return KeepLookOf(size, "balanced", KeepPreviewNode(size), age), true
}

func addCanonicalKeep(b *atlasBuilder, kind string) {
	for age := 0; age < AgeTiers; age++ {
		k, _ := CanonicalKeep(kind, age)
		sz := keepSize(k)
		states := []string{"ready"}
		if age == 0 {
			states = BuildingStates
		}
		for _, st := range states {
			state := st
			b.add(BuildingLookName(kind, state, age, 0), sz.w, sz.h, sz.ax, sz.ay, func(c *canvas) { c.paste(keepCanvas(k, state), 0, 0, false) })
		}
	}
}

func (k KeepLook) PortraitName() string {
	return fmt.Sprintf("portrait/%s/%d/%08x", k.Kind(), k.Age, k.Seed)
}

func ParseKeepPortraitName(name string) (KeepLook, bool) {
	rest, ok := strings.CutPrefix(name, keepPortraitPrefix)
	if !ok {
		return KeepLook{}, false
	}
	return parseKeepPath(rest)
}

func KeepPortrait(k KeepLook) (*image.NRGBA, Region) {
	c := newCanvas(PortraitSize, PortraitSize)
	paintSpritePortrait(c, readyKeep(k))
	return c.img, Region{W: PortraitSize, H: PortraitSize, AnchorX: PortraitSize / 2, AnchorY: PortraitSize / 2}
}

func paintSpritePortrait(c *canvas, sprite *canvas) {
	portraitBackdrop(c)
	fit := fitSprite(sprite, portraitFit)
	c.paste(fit, (PortraitSize-fit.w)/2, PortraitSize-portraitFoot-fit.h, false)
	portraitFrame(c)
}
