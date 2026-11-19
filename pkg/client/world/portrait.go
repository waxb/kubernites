package world

import (
	"image"
	"image/color"
	"math"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	portraitFoot   = 24
	portraitTop    = 6
	portraitWalkHz = 6.0
	portraitStroll = 6.0
	portraitPuffs  = 3
	portraitBob    = 1.5
	portraitBobHz  = .7
)

type sitter struct{ kind, variant string }

var portraitStructures = map[string]sitter{
	"keep": {kind: view.StructKeep}, "gate": {kind: view.StructGate}, "treasury": {kind: view.StructTreasury},
	"quartermaster": {kind: view.StructQuartermaster}, "owl": {kind: view.StructCodex}, "codex": {kind: view.StructCodex},
	"depot": {kind: view.DepotStructure("")}, "chancery": {kind: view.StructChancery}, "stewards": {kind: view.StructStewards},
	"herald": {kind: view.StructHerald}, "wayfinder": {kind: view.StructWayfinder}, "observatory": {kind: view.StructObservatory},
	"archive": {kind: view.StructArchive}, "warhorn": {kind: view.StructWarHorn}, "vault": {kind: view.StructVault},
	"greatkeep": {kind: view.StructKeep, variant: "great"}, "tower": {kind: view.StructKeep, variant: "tower"},
	"tidings": {kind: view.StructHerald}, "map": {kind: view.StructWayfinder}, "council": {kind: view.StructStewards},
}

var portraitScenes = map[string]func(p portrait) ([]piece, bool){
	"hut": portrait.hut, "harbor": portrait.harbor, "fields": portrait.fields, "land": portrait.land,
	"goldship": portrait.harbor, "berth": portrait.harbor, "box": portrait.harbor, "isle": portrait.harbor, "camp": portrait.hut,
}

var (
	skyBands      = []color.RGBA{{150, 186, 204, 255}, {170, 200, 212, 255}, {190, 212, 214, 255}}
	grassBands    = []color.RGBA{{104, 146, 72, 255}, {96, 138, 66, 255}, {88, 128, 60, 255}}
	waterBands    = []color.RGBA{{58, 104, 140, 255}, {52, 96, 132, 255}, {46, 88, 124, 255}}
	ripple        = color.RGBA{120, 164, 190, 255}
	tuft          = color.RGBA{72, 110, 52, 255}
	portraitCloud = color.RGBA{236, 240, 236, 255}
	portraitShade = color.RGBA{206, 216, 220, 255}
)

const (
	cloudDrift = 4.0
	cloudCount = 2
)

type portrait struct {
	lib    *gfx.Library
	box    ui.Rect
	ground float64
	site   view.Site
	age    int
	t      float64
}

type piece struct {
	Name  string  `json:"-"`
	X, Y  float64 `json:"-"`
	Zoom  float64 `json:"-"`
	alpha float32
	Flag  *art.FlagMark `json:"-"`
	frame int
	Img   *ebiten.Image `json:"-"`
}

func PortraitSprites() []string {
	out := make([]string, 0, len(portraitStructures)+len(portraitScenes)+len(portraitWalkers)+len(portraitVignettes)+len(portraitEmblems))
	for k := range portraitStructures {
		out = append(out, k)
	}
	for k := range portraitScenes {
		out = append(out, k)
	}
	for k := range portraitWalkers {
		out = append(out, k)
	}
	for k := range portraitVignettes {
		out = append(out, k)
	}
	for k := range portraitEmblems {
		out = append(out, k)
	}
	return out
}

func DrawPortrait(lib *gfx.Library, dst *ebiten.Image, box ui.Rect, sprite string, site view.Site, age int, t float64) bool {
	pieces, ok := PortraitPieces(lib, box, sprite, site, age, t)
	if !ok {
		return false
	}
	clip := dst.SubImage(image.Rect(box.X, box.Y, box.X+box.W, box.Y+box.H)).(*ebiten.Image)
	paintBackdrop(clip, box, sprite == "harbor", t)
	for _, pc := range pieces {
		if pc.Img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(pc.X, pc.Y)
			clip.DrawImage(pc.Img, op)
			continue
		}
		o := render.SpriteOptions{Alpha: pc.alpha}
		if pc.Flag != nil {
			gfx.DrawFlag(clip, *pc.Flag, pc.frame, pc.X, pc.Y, pc.Zoom, o)
			continue
		}
		lib.Draw(clip, pc.Name, pc.X, pc.Y, pc.Zoom, o)
	}
	return true
}

func PortraitPieces(lib *gfx.Library, box ui.Rect, sprite string, site view.Site, age int, t float64) ([]piece, bool) {
	if lib == nil {
		return nil, false
	}
	p := portrait{lib: lib, box: box, ground: float64(box.Y + box.H - portraitFoot), site: site, age: age, t: t}
	if s, ok := portraitStructures[sprite]; ok {
		return p.building(portraitStructure(site, s))
	}
	if scene, ok := portraitScenes[sprite]; ok {
		return scene(p)
	}
	if role, ok := portraitWalkers[sprite]; ok {
		return p.folk(role)
	}
	if scene, ok := portraitVignettes[sprite]; ok {
		return p.vignette(scene)
	}
	if name, ok := portraitEmblems[sprite]; ok {
		return p.emblem(name)
	}
	if strings.HasPrefix(sprite, art.PageIconName("")) {
		return p.emblem(sprite)
	}
	return nil, false
}

func portraitStructure(site view.Site, want sitter) view.Structure {
	kind := want.kind
	depot := strings.HasSuffix(kind, "-")
	for _, s := range site.Structures {
		if s.State == view.StateReady && (s.Kind == kind || depot && strings.HasPrefix(s.Kind, kind)) && (want.variant == "" || s.Variant == want.variant) {
			return s
		}
	}
	if depot {
		kind = view.DepotStructure("parcel")
	}
	return view.Structure{ID: "portrait-" + kind, Kind: kind, Variant: want.variant, W: 2, H: 2, State: view.StateReady}
}

func paintBackdrop(dst *ebiten.Image, b ui.Rect, water bool, t float64) {
	horizon := b.Y + b.H*3/5
	band := (horizon - b.Y + len(skyBands) - 1) / len(skyBands)
	for i, c := range skyBands {
		ui.Fill(dst, ui.Rect{X: b.X, Y: b.Y + i*band, W: b.W, H: band}, c)
	}
	low, mark, shift := grassBands, tuft, 0
	if water {
		low, mark, shift = waterBands, ripple, int(t*4)
	}
	rest := b.Y + b.H - horizon
	for i, c := range low {
		ui.Fill(dst, ui.Rect{X: b.X, Y: horizon + i*rest/len(low), W: b.W, H: rest/len(low) + 1}, c)
	}
	for i := range 14 {
		x := b.X + (i*37+shift+i*i*5)%b.W
		y := horizon + 4 + (i*23)%max(rest-6, 1)
		ui.Fill(dst, ui.Rect{X: x, Y: y, W: 3, H: 1}, mark)
	}
	paintClouds(dst, b, t)
}

func paintClouds(dst *ebiten.Image, b ui.Rect, t float64) {
	for i := range cloudCount {
		w := 22 + 8*i
		x := b.X - w + int(t*cloudDrift*float64(i+1)+float64(i*61))%(b.W+w)
		y := b.Y + 10 + i*18
		ui.Fill(dst, ui.Rect{X: x + 4, Y: y, W: w - 10, H: 3}, portraitCloud)
		ui.Fill(dst, ui.Rect{X: x, Y: y + 3, W: w, H: 4}, portraitCloud)
		ui.Fill(dst, ui.Rect{X: x + 2, Y: y + 7, W: w - 4, H: 1}, portraitShade)
	}
}

func (p portrait) fit(reg render.Region) float64 {
	return gfx.PortraitScale(reg.W, reg.H, p.box.W-8, int(p.ground)-p.box.Y-portraitTop)
}

func (p portrait) stand(reg render.Region, zoom float64) (float64, float64) {
	sx := float64(p.box.X) + float64(p.box.W)/2 - (float64(reg.W)/2-float64(reg.AnchorX))*zoom
	return sx, p.ground - float64(reg.H-reg.AnchorY)*zoom
}

func (p portrait) first(names []string) (string, render.Region, bool) {
	for _, n := range names {
		if _, reg, ok := p.lib.Lookup(n); ok {
			return n, reg, true
		}
	}
	return "", render.Region{}, false
}

func (p portrait) building(s view.Structure) ([]piece, bool) {
	names := gfx.StructureLookNames(s, p.age)
	switch s.Kind {
	case view.StructKeep:
		names = gfx.KeepNames(s, p.age)
	case view.StructVault:
		names = []string{art.VaultName("available", 0)}
	}
	name, reg, ok := p.first(names)
	if !ok {
		return nil, false
	}
	zoom := p.fit(reg)
	sx, sy := p.stand(reg, zoom)
	out := []piece{{Name: name, X: sx, Y: sy, Zoom: zoom}}
	for i, m := range p.lib.Flags(name) {
		out = append(out, piece{Flag: &m, frame: FlagFrame(p.t, s, i), X: sx, Y: sy, Zoom: zoom})
	}
	if s.Kind == view.StructKeep {
		dx, dy := art.KeepVent(gfx.KeepLookOf(s, p.age))
		out = append(out, p.puffs(vent{dx: dx, dy: dy}, sx, sy, zoom)...)
	}
	return append(out, p.stroll("porter", max(zoom, 1), 0)...), true
}

func (p portrait) puffs(v vent, sx, sy, zoom float64) []piece {
	vx, vy := sx+v.dx*zoom, sy+v.dy*zoom
	top := vy
	out := []piece{}
	if v.stack {
		out = append(out, piece{Name: stackName, X: vx, Y: vy, Zoom: zoom})
		top -= stackRise * zoom
	}
	tick := int(p.t * wispRate)
	for k := range portraitPuffs {
		x := vx + (float64(k)-float64(portraitPuffs-1)/2)*wispSpread*zoom
		out = append(out, piece{Name: wispNames[(tick+k*2)%art.WispFrames], X: x, Y: top, Zoom: zoom, alpha: .6})
	}
	return out
}

func (p portrait) walker(role, action, dir string, x, y, zoom float64) []piece {
	name, _, ok := p.first(gfx.WalkerNames(role, action, dir, Frame(p.t, portraitWalkHz)))
	if !ok {
		return nil
	}
	return []piece{{Name: name, X: x, Y: y, Zoom: zoom}}
}

func (p portrait) stroll(role string, zoom, phase float64) []piece {
	u := math.Mod(p.t/portraitStroll+phase, 1)
	dir, k := "se", 2*u
	if u >= .5 {
		dir, k = "sw", 2-2*u
	}
	span := float64(p.box.W) * .55
	x := float64(p.box.X) + float64(p.box.W)*.2 + k*span
	y := p.ground + 2 + k*span/4
	return p.walker(role, "walk", dir, x, y, zoom)
}

func (p portrait) hut() ([]piece, bool) {
	name, reg, ok := p.first(HutNames(1, model.HearthReady, ""))
	if !ok {
		return nil, false
	}
	zoom := p.fit(reg)
	sx, sy := p.stand(reg, zoom)
	out := append([]piece{{Name: name, X: sx, Y: sy, Zoom: zoom}}, p.puffs(HutVent(name), sx, sy, zoom)...)
	return append(out, p.walker("porter", "work", "sw", sx+16*zoom, sy+4, zoom)...), true
}

func (p portrait) harbor() ([]piece, bool) {
	name := gfx.ShipName("cog", 0, "sw", Frame(p.t, 2))
	_, reg, ok := p.lib.Lookup(name)
	if !ok {
		return nil, false
	}
	zoom := gfx.PortraitScale(reg.W, reg.H, p.box.W/2, p.box.H/2)
	bob := portraitBob * math.Sin(2*math.Pi*p.t*portraitBobHz)
	sx, sy := float64(p.box.X)+float64(p.box.W)/2, p.ground-6+bob
	return []piece{
		{Name: gfx.EffectName("wake", int(p.t*6)), X: sx, Y: sy + 4, Zoom: zoom / 2, alpha: .7},
		{Name: name, X: sx, Y: sy, Zoom: zoom},
	}, true
}

var (
	fieldArtOnce sync.Once
	fieldArt     map[string]*ebiten.Image
)

func fieldImages() map[string]*ebiten.Image {
	fieldArtOnce.Do(func() {
		fieldArt = map[string]*ebiten.Image{
			"berries": ebiten.NewImageFromImage(PaintField("berries")), "wheat": ebiten.NewImageFromImage(PaintField("wheat")),
			"land": ebiten.NewImageFromImage(PaintLandTile()),
		}
	})
	return fieldArt
}

func (p portrait) fields() ([]piece, bool) {
	art := fieldImages()
	out := []piece{}
	cx, cy := float64(p.box.X+p.box.W/2-landTileW), p.ground-30
	for row := range 3 {
		for col := range 3 {
			crop := "wheat"
			if row == 2 {
				crop = "berries"
			}
			x := cx + float64(col-row)*landTileW/2
			y := cy + float64(col+row)*landTileH/2
			out = append(out, piece{Img: art[crop], X: x, Y: y - cropRise})
		}
	}
	return append(out, p.stroll("porter", 1, .25)...), true
}

func (p portrait) land() ([]piece, bool) {
	_, reg, ok := p.lib.Lookup(gfx.DecorName("rock", 0))
	if !ok {
		return nil, false
	}
	zoom := gfx.PortraitScale(reg.W, reg.H, p.box.W/4, p.box.H/4)
	cx, cy := float64(p.box.X)+float64(p.box.W)/2, p.ground-4
	out := []piece{
		{Name: gfx.DecorName("rock", 0), X: float64(p.box.X) + 24, Y: p.ground - 20, Zoom: zoom},
		{Name: gfx.DecorName("rock", 1), X: float64(p.box.X+p.box.W) - 26, Y: p.ground - 6, Zoom: zoom},
	}
	out = append(out, p.walker("builder", "work", "sw", cx, cy, zoom)...)
	return append(out, piece{Name: gfx.EffectName("dust", int(p.t*8)), X: cx - 10*zoom, Y: cy, Zoom: zoom, alpha: .7}), true
}
