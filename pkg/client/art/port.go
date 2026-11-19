package art

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

const (
	PortLevels = 4
	PortPrefix = "port/"
	portStone  = 3
)

var PortKinds = []string{"deck", "pier-x", "pier-y", "house", "sign", "bollard", "crane", "warehouse", "lighthouse", "landing", "stake"}

var portVariantCount = map[string]int{"deck": 3, "pier-x": 3, "pier-y": 3, "warehouse": 2}

type PortLook struct {
	Kind    string
	Level   int
	Age     int
	Variant int
}

type portPiece struct {
	size  func(l PortLook) size
	paint func(c *canvas, l PortLook)
}

func PortVariants(kind string) int { return max(portVariantCount[kind], 1) }

func ClampPortLevel(level int) int { return min(max(level, 1), PortLevels) }

func (l PortLook) clamped() PortLook {
	n := PortVariants(l.Kind)
	return PortLook{Kind: l.Kind, Level: ClampPortLevel(l.Level), Age: clampAge(l.Age), Variant: ((l.Variant % n) + n) % n}
}

func (l PortLook) Name() string {
	k := l.clamped()
	return fmt.Sprintf("%s%s/%d/%d/%d", PortPrefix, k.Kind, k.Level, k.Age, k.Variant)
}

func (l PortLook) stone() bool { return l.Level >= portStone }

func ParsePortName(name string) (PortLook, bool) {
	rest, ok := strings.CutPrefix(name, PortPrefix)
	parts := strings.Split(rest, "/")
	if !ok || len(parts) != 4 {
		return PortLook{}, false
	}
	nums := [3]int{}
	for i, p := range parts[1:] {
		v, err := strconv.Atoi(p)
		if err != nil {
			return PortLook{}, false
		}
		nums[i] = v
	}
	l := PortLook{Kind: parts[0], Level: nums[0], Age: nums[1], Variant: nums[2]}
	if _, known := portPieces()[l.Kind]; !known || l.clamped() != l {
		return PortLook{}, false
	}
	return l, true
}

func portPieces() map[string]portPiece {
	tile := func(PortLook) size { return havenTileSize }
	return map[string]portPiece{
		"deck":       {tile, paintPortDeck},
		"pier-x":     {tile, func(c *canvas, l PortLook) { paintPortPier(c, l, true) }},
		"pier-y":     {tile, func(c *canvas, l PortLook) { paintPortPier(c, l, false) }},
		"house":      {houseSize, paintPortHouse},
		"sign":       {func(PortLook) size { return size{20, 30, 6, 27} }, paintPortSign},
		"bollard":    {func(PortLook) size { return size{11, 12, 4, 10} }, paintPortBollard},
		"crane":      {func(PortLook) size { return craneSize }, paintPortCrane},
		"warehouse":  {func(PortLook) size { return buildingLikeSize(2, warehouseHeight) }, paintPortWarehouse},
		"lighthouse": {func(PortLook) size { return buildingLikeSize(1, lighthouseHeight) }, paintPortLighthouse},
		"landing":    {func(PortLook) size { return landingSize }, paintPortLanding},
		"stake":      {func(PortLook) size { return stakeSize }, paintPortStake},
	}
}

func PortSprite(l PortLook) (*image.NRGBA, Region) {
	pc, ok := portPieces()[l.Kind]
	if !ok {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1)), Region{W: 1, H: 1}
	}
	k := l.clamped()
	sz := pc.size(k)
	c := newCanvas(sz.w, sz.h)
	pc.paint(c, k)
	return c.img, Region{W: sz.w, H: sz.h, AnchorX: sz.ax, AnchorY: sz.ay}
}

func PortSpriteByName(name string) (*image.NRGBA, Region, bool) {
	if l, ok := ParsePortName(name); ok {
		img, r := PortSprite(l)
		return img, r, true
	}
	if f, ok := ParseFerryName(name); ok {
		img, r := FerrySprite(f)
		return img, r, true
	}
	return nil, Region{}, false
}
