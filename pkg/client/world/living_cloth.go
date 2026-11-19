package world

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	buntingBase    = 13.0
	buntingPerTier = 6.0
	buntingSag     = 3.0
	buntingBias    = .06
	pennantW       = 4
	pennantH       = 6
	lineHeight     = 15.0
	linePull       = 8.0
	lineSag        = 2.0
	lineBias       = .04
	lineNear       = 1.7
	lineFar        = 3.6
	linesPerCrew   = 3
	clothW         = 4
	clothH         = 5
	swayPixels     = .7
)

var crewHues = []color.NRGBA{
	{206, 80, 66, 255}, {222, 164, 54, 255}, {86, 148, 78, 255}, {70, 128, 178, 255},
	{150, 94, 170, 255}, {224, 120, 60, 255}, {64, 156, 150, 255}, {208, 108, 140, 255},
}

var clothHues = []color.NRGBA{{244, 242, 232, 255}, {232, 214, 170, 255}, {104, 150, 196, 255}, {196, 98, 84, 255}}

func CrewHue(name string) color.NRGBA {
	return crewHues[int(model.Hash(name))%len(crewHues)]
}

func crewHues6(names []string) []color.NRGBA {
	out := make([]color.NRGBA, 0, min(len(names), pennantsPerKeep))
	for _, n := range names {
		if len(out) == pennantsPerKeep {
			break
		}
		out = append(out, CrewHue(n))
	}
	return out
}

func shade(c color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{uint8(float64(c.R) * t), uint8(float64(c.G) * t), uint8(float64(c.B) * t), c.A}
}

func (r *Renderer) buntingItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	zoom := cam.Scale()
	for _, f := range l.folk {
		if len(f.crews) == 0 {
			continue
		}
		lo, hi := Footprint(f.s)
		rise := (buntingBase + buntingPerTier*float64(max(f.s.W, f.s.H, 1)-1)) * zoom
		elev := elevationAt(sc.Site, Vec{float64(hi.X), float64(hi.Y)})
		ax, ay := cam.GridToScreenElevated(float64(lo.X)-.5, float64(hi.Y)+.5, elev)
		bx, by := cam.GridToScreenElevated(float64(hi.X)+.5, float64(hi.Y)+.5, elev)
		if !onScreen(bx, by, zoom) {
			continue
		}
		cols, seed := crewHues6(f.crews), float64(model.Hash(f.s.ID)%100)/10
		out = append(out, Drawable{Depth: render.DepthKey(float64(hi.X)+.5, float64(hi.Y)+.5) + buntingBase/1000 + buntingBias, Draw: func(dst *ebiten.Image) {
			drawBunting(dst, ax, ay-rise, bx, by-rise, zoom, cols, sc.Time+seed, sc.Reduce)
		}})
	}
	return out
}

func pixelScale(zoom float64) int { return max(int(math.Round(zoom)), 1) }

func drawBunting(dst *ebiten.Image, ax, ay, bx, by, zoom float64, cols []color.NRGBA, t float64, still bool) {
	px := pixelScale(zoom)
	steps := int(math.Max(math.Abs(bx-ax), 1) / float64(px))
	sag := buntingSag * zoom
	cord := color.NRGBA{58, 44, 34, 255}
	for i := 0; i <= steps; i++ {
		u := float64(i) / float64(steps)
		x, y := ax+(bx-ax)*u, ay+(by-ay)*u+sag*4*u*(1-u)
		ui.Fill(dst, ui.Rect{X: int(x), Y: int(y), W: px, H: px}, cord)
	}
	for i, c := range cols {
		u := float64(i+1) / float64(len(cols)+1)
		x, y := ax+(bx-ax)*u, ay+(by-ay)*u+sag*4*u*(1-u)+float64(px)
		if !still {
			x += math.Sin(t*2.2+float64(i)*1.7) * swayPixels * zoom
		}
		drawFlag(dst, int(x), int(y), px, c)
	}
}

func drawFlag(dst *ebiten.Image, x, y, px int, c color.NRGBA) {
	for row := 0; row < pennantH; row++ {
		w := max(pennantW-(row*pennantW)/pennantH, 1)
		for k := 0; k < w; k++ {
			col := c
			if k == w-1 && w > 1 {
				col = shade(c, .72)
			}
			ui.Fill(dst, ui.Rect{X: x + (k-pennantW/2)*px, Y: y + row*px, W: px, H: px}, col)
		}
	}
}

type clothLine struct {
	a, b  Vec
	crew  string
	hang  uint32
	depth float64
}

func (r *Renderer) laundryItems(out []Drawable, cam render.Camera, sc Scene, l *living) []Drawable {
	zoom := cam.Scale()
	for _, line := range r.clothLines(sc) {
		ax, ay := ScreenPointOf(cam, sc.Site, line.a)
		bx, by := ScreenPointOf(cam, sc.Site, line.b)
		if !onScreen(ax, ay, zoom) && !onScreen(bx, by, zoom) {
			continue
		}
		line := line
		out = append(out, Drawable{Depth: line.depth + lineBias, Draw: func(dst *ebiten.Image) {
			drawWashing(dst, ax, ay, bx, by, zoom, line, sc.Time, sc.Reduce)
		}})
	}
	return out
}

func (r *Renderer) clothLines(sc Scene) []clothLine {
	out := []clothLine{}
	for _, c := range sc.Crews {
		if !LaundryHangs(c) {
			continue
		}
		out = append(out, r.crewLines(c)...)
	}
	return out
}

func (r *Renderer) crewLines(c model.CrewChain) []clothLine {
	names := []string{}
	for _, f := range c.Foremen {
		for _, n := range f.Pods {
			if _, ok := r.Homes[n]; ok {
				names = append(names, n)
			}
		}
	}
	sort.Strings(names)
	linked := map[string]bool{}
	out := []clothLine{}
	for _, a := range names {
		if linked[a] || len(out) == linesPerCrew {
			continue
		}
		if b, ok := r.nearHut(a, names, linked); ok {
			linked[a], linked[b] = true, true
			out = append(out, r.lineBetween(c.Name, a, b))
		}
	}
	return out
}

func (r *Renderer) nearHut(a string, names []string, linked map[string]bool) (string, bool) {
	best, gap, found := "", math.MaxFloat64, false
	for _, b := range names {
		d := Dist(r.Homes[a], r.Homes[b])
		if b != a && !linked[b] && d >= lineNear && d <= lineFar && d < gap {
			best, gap, found = b, d, true
		}
	}
	return best, found
}

func (r *Renderer) lineBetween(crew, a, b string) clothLine {
	pa, pb := Vec{r.Homes[a].X + .5, r.Homes[a].Y + .5}, Vec{r.Homes[b].X + .5, r.Homes[b].Y + .5}
	return clothLine{a: pa, b: pb, crew: crew, hang: model.Hash(a + b), depth: max(render.DepthKey(pa.X, pa.Y), render.DepthKey(pb.X, pb.Y))}
}

func drawWashing(dst *ebiten.Image, ax, ay, bx, by, zoom float64, line clothLine, t float64, still bool) {
	px := pixelScale(zoom)
	span := math.Hypot(bx-ax, by-ay)
	if span < 1 {
		return
	}
	ux, uy := (bx-ax)/span, (by-ay)/span
	ax, ay, bx, by = ax+ux*linePull*zoom, ay+uy*linePull*zoom, bx-ux*linePull*zoom, by-uy*linePull*zoom
	pole := color.NRGBA{96, 70, 48, 255}
	ui.Fill(dst, ui.Rect{X: int(ax), Y: int(ay - lineHeight*zoom), W: px, H: int(lineHeight * zoom)}, pole)
	ui.Fill(dst, ui.Rect{X: int(bx), Y: int(by - lineHeight*zoom), W: px, H: int(lineHeight * zoom)}, pole)
	ay, by = ay-lineHeight*zoom, by-lineHeight*zoom
	steps := int(math.Max(math.Abs(bx-ax), 1) / float64(px))
	sag := lineSag * zoom
	cord := color.NRGBA{70, 56, 44, 255}
	for i := 0; i <= steps; i++ {
		u := float64(i) / float64(steps)
		ui.Fill(dst, ui.Rect{X: int(ax + (bx-ax)*u), Y: int(ay + (by-ay)*u + sag*4*u*(1-u)), W: px, H: px}, cord)
	}
	n := 3 + int(line.hang%2)
	for i := 0; i < n; i++ {
		u := float64(i+1) / float64(n+1)
		x, y := ax+(bx-ax)*u, ay+(by-ay)*u+sag*4*u*(1-u)+float64(px)
		hue := clothHues[int(line.hang>>uint(i*2))%len(clothHues)]
		if i == 1 {
			hue = CrewHue(line.crew)
		}
		wave := 0
		if !still && int(math.Floor(t*1.6+float64(i)))%2 == 0 {
			wave = px
		}
		for dy := 0; dy < clothH; dy++ {
			for dx := 0; dx < clothW; dx++ {
				c := hue
				if dx == clothW-1 || dy == clothH-1 {
					c = shade(hue, .78)
				}
				ui.Fill(dst, ui.Rect{X: int(x) + dx*px - px + wave*dy/clothH, Y: int(y) + dy*px, W: px, H: px}, c)
			}
		}
	}
}
