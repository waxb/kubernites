package gfx

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
)

type Air struct {
	On    bool
	Skin  int
	Hair  int
	Warm  float32
	Light float32
	Bob   int
}

const airShaderSource = `//kage:unit pixels

package main

var Skins [16]vec4
var Hairs [6]vec4

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	c := imageSrc0At(srcPos)
	if c.a < 0.999 {
		return c * color
	}
	out := c.rgb
	found := false
	for i := 0; i < 16; i++ {
		if !found && distance(c.rgb, Skins[i].rgb) < 0.002 {
			found = true
			shade := mod(float(i), 4.0)
			for t := 0; t < 4; t++ {
				for k := 0; k < 4; k++ {
					if abs(float(t)-custom.x) < 0.5 && abs(float(k)-shade) < 0.5 {
						out = Skins[t*4+k].rgb
					}
				}
			}
		}
	}
	for i := 0; i < 6; i++ {
		if !found && distance(c.rgb, Hairs[i].rgb) < 0.002 {
			found = true
			for t := 0; t < 6; t++ {
				if abs(float(t)-custom.y) < 0.5 {
					out = Hairs[t].rgb
				}
			}
		}
	}
	if !found {
		out = clamp(out*(1.0+0.07*custom.w)+vec3(0.035, 0.0, -0.035)*custom.z, 0.0, 1.0)
	}
	return vec4(out, 1.0) * color
}
`

var (
	airOnce   sync.Once
	airShader *ebiten.Shader
	airErr    error
	airSkins  []float32
	airHairs  []float32
)

func AirShader() (*ebiten.Shader, error) {
	airOnce.Do(func() {
		airShader, airErr = ebiten.NewShader([]byte(airShaderSource))
		for _, r := range art.SkinRamps() {
			for _, c := range r {
				airSkins = append(airSkins, float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, 1)
			}
		}
		for _, c := range art.HairColours() {
			airHairs = append(airHairs, float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, 1)
		}
	})
	return airShader, airErr
}

func AirSkins() int { return len(art.SkinRamps()) }

func AirHairs() int { return len(art.HairColours()) }

func (l *Library) DrawFirstAir(dst *ebiten.Image, names []string, sx, sy, zoom float64, o render.SpriteOptions, air Air) (string, bool) {
	if !air.On {
		return l.DrawFirst(dst, names, sx, sy, zoom, o)
	}
	for _, n := range names {
		if l.DrawAir(dst, n, sx, sy, zoom, o, air) {
			return n, true
		}
	}
	return "", false
}

func (l *Library) DrawAir(dst *ebiten.Image, name string, sx, sy, zoom float64, o render.SpriteOptions, air Air) bool {
	sheet, r, ok := l.Lookup(name)
	if !ok {
		return false
	}
	DrawRegionAir(dst, sheet, r, SpriteGeo(r, sx, sy, zoom, o), o.Tint, o.Alpha, air)
	return true
}

func SpriteGeo(r render.Region, sx, sy, camZoom float64, o render.SpriteOptions) ebiten.GeoM {
	z := camZoom
	if o.Scale > 0 {
		z *= o.Scale
	}
	var geo ebiten.GeoM
	if o.Flip {
		geo.Scale(-z, z)
		geo.Translate(float64(r.W)*z, 0)
		geo.Translate(math.Round(sx-float64(r.W-r.AnchorX)*z), math.Round(sy-float64(r.AnchorY)*z))
		return geo
	}
	geo.Scale(z, z)
	geo.Translate(math.Round(sx-float64(r.AnchorX)*z), math.Round(sy-float64(r.AnchorY)*z))
	return geo
}

func DrawRegionAir(dst, sheet *ebiten.Image, r render.Region, geo ebiten.GeoM, tint [3]float32, alpha float32, air Air) {
	shader, err := AirShader()
	if !air.On || err != nil || shader == nil {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest, GeoM: geo}
		if tint != ([3]float32{}) {
			op.ColorScale.Scale(tint[0], tint[1], tint[2], 1)
		}
		if alpha > 0 && alpha < 1 {
			op.ColorScale.ScaleAlpha(alpha)
		}
		dst.DrawImage(sheet.SubImage(r.Rect()).(*ebiten.Image), op)
		return
	}
	cr, cg, cb, ca := float32(1), float32(1), float32(1), float32(1)
	if tint != ([3]float32{}) {
		cr, cg, cb = tint[0], tint[1], tint[2]
	}
	if alpha > 0 && alpha < 1 {
		ca = alpha
		cr, cg, cb = cr*ca, cg*ca, cb*ca
	}
	sl, st := float32(r.X), float32(r.Y)
	vert := func(lx, ly float64) ebiten.Vertex {
		x, y := geo.Apply(lx, ly)
		return ebiten.Vertex{DstX: float32(x), DstY: float32(y), SrcX: sl + float32(lx), SrcY: st + float32(ly), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			Custom0: float32(air.Skin), Custom1: float32(air.Hair), Custom2: air.Warm, Custom3: air.Light}
	}
	w, h := float64(r.W), float64(r.H)
	vs := []ebiten.Vertex{vert(0, 0), vert(w, 0), vert(0, h), vert(w, h)}
	op := &ebiten.DrawTrianglesShaderOptions{Images: [4]*ebiten.Image{sheet}, Uniforms: map[string]any{"Skins": airSkins, "Hairs": airHairs}}
	dst.DrawTrianglesShader(vs, []uint16{0, 1, 2, 1, 2, 3}, shader, op)
}
