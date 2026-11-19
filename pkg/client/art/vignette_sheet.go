package art

import (
	"image"
	"image/color"
	"math"
	"strings"
)

type vgSprite struct {
	img    *image.NRGBA
	ax, ay int
}

type Sheet struct {
	sprites map[string]vgSprite
}

var vgFamilies = []string{
	"walker/", "hearth/", "ship/", "fx/", "box/", "boxpile/", "midden/", "codexbloat/", "haven/",
	"decor/", "ambient/", "border/", "icon/rank-", "icon/sealed", "icon/priority",
	"building/keep-stone/ready", "building/keep-tower/ready", "building/codex/ready", "building/warhorn/ready",
	"building/stall-", "building/chancery/ready", "building/treasury/ready",
}

func vgWanted(name string) bool {
	for _, f := range vgFamilies {
		if strings.HasPrefix(name, f) {
			return true
		}
	}
	return false
}

func NewSheet(set Set) *Sheet {
	s := &Sheet{sprites: map[string]vgSprite{}}
	for _, a := range []Atlas{set.Terrain, set.Buildings, set.Walkers, set.Effects, set.Icons, set.Portraits, set.UI} {
		if a.Image == nil {
			continue
		}
		for name, r := range a.Regions {
			if vgWanted(name) {
				s.sprites[name] = vgCut(a.Image, r)
			}
		}
	}
	return s
}

func vgCut(src *image.NRGBA, r Region) vgSprite {
	img := image.NewNRGBA(image.Rect(0, 0, r.W, r.H))
	for y := 0; y < r.H; y++ {
		copy(img.Pix[y*img.Stride:y*img.Stride+r.W*4], src.Pix[src.PixOffset(r.X, r.Y+y):src.PixOffset(r.X, r.Y+y)+r.W*4])
	}
	return vgSprite{img: img, ax: r.AnchorX, ay: r.AnchorY}
}

func (s *Sheet) Has(name string) bool {
	if s == nil {
		return false
	}
	_, ok := s.sprites[name]
	return ok
}

func (s *Sheet) Len() int {
	if s == nil {
		return 0
	}
	return len(s.sprites)
}

type vgPut struct {
	angle float64
	scale int
	tone  func(color.NRGBA) color.NRGBA
	alpha float64
}

func (p *vgPaint) put(name string, x, y int) {
	p.putWith(name, x, y, vgPut{})
}

func (p *vgPaint) putTone(name string, x, y int, tone func(color.NRGBA) color.NRGBA) {
	p.putWith(name, x, y, vgPut{tone: tone})
}

func (p *vgPaint) big(name string, x, y int) {
	p.putWith(name, x, y, vgPut{scale: 2})
}

func (p *vgPaint) stamp(w, h, ax, ay int, draw func(c *canvas)) vgSprite {
	c := newCanvas(w, h)
	draw(c)
	return vgSprite{img: c.img, ax: ax, ay: ay}
}

func (p *vgPaint) putWith(name string, x, y int, o vgPut) {
	s, ok := p.sh.sprites[name]
	if !ok {
		p.missing = append(p.missing, name)
		return
	}
	p.putSprite(s, x, y, o)
}

func (p *vgPaint) putSprite(s vgSprite, x, y int, o vgPut) {
	if o.angle == 0 {
		p.blitStraight(s, x, y, o)
		return
	}
	p.blitTurned(s, x, y, o)
}

func (p *vgPaint) blitStraight(s vgSprite, x, y int, o vgPut) {
	k := max(o.scale, 1)
	b := s.img.Bounds()
	for sy := 0; sy < b.Dy(); sy++ {
		for sx := 0; sx < b.Dx(); sx++ {
			col := s.img.NRGBAAt(sx, sy)
			if col.A == 0 {
				continue
			}
			for dy := 0; dy < k; dy++ {
				for dx := 0; dx < k; dx++ {
					p.ink(x+(sx-s.ax)*k+dx, y+(sy-s.ay)*k+dy, col, o)
				}
			}
		}
	}
}

func (p *vgPaint) blitTurned(s vgSprite, x, y int, o vgPut) {
	b := s.img.Bounds()
	k := float64(max(o.scale, 1))
	reach := int(math.Ceil(k*math.Hypot(float64(max(s.ax, b.Dx()-s.ax)), float64(max(s.ay, b.Dy()-s.ay))))) + 1
	sin, cos := math.Sincos(o.angle)
	for dy := -reach; dy <= reach; dy++ {
		for dx := -reach; dx <= reach; dx++ {
			fx, fy := (float64(dx)+0.5)/k-0.5, (float64(dy)+0.5)/k-0.5
			sx := int(math.Floor(cos*fx+sin*fy+0.5)) + s.ax
			sy := int(math.Floor(-sin*fx+cos*fy+0.5)) + s.ay
			if (image.Point{X: sx, Y: sy}).In(b) {
				p.ink(x+dx, y+dy, s.img.NRGBAAt(sx, sy), o)
			}
		}
	}
}

func (p *vgPaint) ink(x, y int, col color.NRGBA, o vgPut) {
	if col.A == 0 {
		return
	}
	if o.tone != nil {
		col = o.tone(col)
	}
	if o.alpha > 0 && o.alpha < 1 {
		col.A = uint8(float64(col.A) * o.alpha)
	}
	p.c.blend(x, y, col)
}
