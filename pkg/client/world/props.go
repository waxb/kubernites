package world

import (
	"image"
	"image/color"
	"maps"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
)

type prop struct {
	img    *ebiten.Image
	region render.Region
}

type propSet map[string]prop

type propArt struct {
	rows   []string
	ax, ay int
}

var propPalette = map[byte]color.RGBA{
	'k': inkColor, 'b': logBark, 'r': logRing, 'l': legShade, 's': sweatBlue, 'g': dizzyGold,
	'w': pagePaper, 'd': deskWood, 't': deskTop, 'p': poleWood, 'f': {255, 255, 255, 255},
	'm': markGrey, 'o': flameOrange,
}

var propArts = map[string]propArt{
	"log":    {rows: []string{".kkkkkkk.", "krbbbbbbk", "krbbbbbbk", ".kkkkkkk."}, ax: 4, ay: 3},
	"legs":   {rows: []string{"kkkkkk", "klllkk", "kkkkkk"}, ax: 0, ay: 2},
	"z":      {rows: []string{"kkkkkk", "kwwwwk", "kkkwkk", "kkwkk.", "kwwwwk", "kkkkkk"}, ax: 0, ay: 5},
	"drop":   {rows: []string{".k.", "ksk", "ksk", ".k."}, ax: 1, ay: 3},
	"star":   {rows: []string{".kkk.", "kkgkk", "kgggk", "kkgkk", ".kkk."}, ax: 2, ay: 2},
	"page":   {rows: []string{"kkkk", "kwwk", "kkkk"}, ax: 2, ay: 2},
	"desk":   {rows: []string{"kkkkkkkkk", "kttwwwttk", "kkkkkkkkk", "...kdk...", "...kdk...", "...kdk...", "...kdk...", "..kkkkk.."}, ax: 4, ay: 7},
	"pole":   {rows: poleRows(bannerPole), ax: 1, ay: bannerPole - 1},
	"flag":   {rows: []string{"kkkkkkkkkk", "kffffffffk", "kffffffffk", "kfffffkkk.", "kffffk....", "kkkkk....."}, ax: 0, ay: 0},
	"flag2":  {rows: []string{"kkkkkkkkk.", "kfffffffkk", "kffffffffk", "kffffffkk.", "kfffkk....", "kkkk......"}, ax: 0, ay: 0},
	"bubble": {rows: []string{".kkkkkkk.", "kwwwkwwwk", "kwwkgkwwk", "kwkgogkwk", "kwkgggkwk", "kwwkkkwwk", "kwwwwwwwk", ".kkkkwkk.", "....kk...", "....k...."}, ax: 4, ay: 9},
	"cross":  {rows: []string{"kk...kk", "kmk.kmk", ".kmkmk.", "..kmk..", ".kmkmk.", "kmk.kmk", "kk...kk"}, ax: 3, ay: 6},
}

func poleRows(h int) []string {
	rows := []string{"kgk"}
	for len(rows) < h-1 {
		rows = append(rows, "kpk")
	}
	return append(rows, "kkk")
}

func buildProps() propSet {
	out := propSet{}
	for _, name := range slices.Sorted(maps.Keys(propArts)) {
		art := propArts[name]
		img := paintProp(art.rows)
		b := img.Bounds()
		out[name] = prop{img: newImage(img), region: render.Region{W: b.Dx(), H: b.Dy(), AnchorX: art.ax, AnchorY: art.ay}}
	}
	return out
}

func paintProp(rows []string) *image.NRGBA {
	w := 0
	for _, row := range rows {
		w = max(w, len(row))
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, len(rows)))
	for y, row := range rows {
		for x := 0; x < len(row); x++ {
			if c, ok := propPalette[row[x]]; ok {
				img.SetNRGBA(x, y, color.NRGBA{c.R, c.G, c.B, c.A})
			}
		}
	}
	return img
}

func (r *Renderer) props() propSet {
	t := r.townOf()
	if t.props == nil {
		t.props = buildProps()
	}
	return t.props
}

func (ps propSet) draw(dst *ebiten.Image, name string, sx, sy, zoom float64, o render.SpriteOptions) {
	if p, ok := ps[name]; ok {
		render.DrawAt(dst, p.img, p.region, sx, sy, zoom, o)
	}
}

type NameKey struct {
	Role, fallback, Action, Dir string `json:"-"`
	variant, Tier, frame        int    `json:"-"`
}

func (r *Renderer) namesFor(role, fallback, action, dir string, frame int) []string {
	return r.LookNames(NameKey{Role: role, fallback: fallback, Action: action, Dir: dir, frame: frame, Tier: 1})
}

func (r *Renderer) LookNames(key NameKey) []string {
	t := r.townOf()
	if names, ok := t.names[key]; ok {
		return names
	}
	if t.names == nil {
		t.names = map[NameKey][]string{}
	}
	names := figureNames(key)
	t.names[key] = names
	return names
}
