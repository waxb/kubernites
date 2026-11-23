package world

import (
	"hash/fnv"
	"image"
	"math"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var theScreen = ui.Rect{W: ViewWidth, H: ViewHeight}

func clippedStructures(r *Renderer, dst *ebiten.Image, cam render.Camera, sc Scene) []Drawable {
	restore := r.clipTo(dst.Bounds())
	defer restore()
	return r.structureItems(cam, sc)
}

func structureItemOf(items []Drawable, id string) (Drawable, bool) {
	for _, it := range items {
		if it.Hit != nil && it.Hit.Kind == HitStructure && it.Hit.ID == id {
			return it, true
		}
	}
	return Drawable{}, false
}

func pickedAt(r *Renderer, area ui.Rect, id string) (int, int, bool) {
	for y := area.Y; y < area.Y+area.H; y += 2 {
		for x := area.X; x < area.X+area.W; x += 2 {
			if got, ok := r.Pick(x, y); ok && got.Kind == HitStructure && got.ID == id {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func visiblePart(rect ui.Rect) ui.Rect {
	x0, y0 := max(rect.X, 0), max(rect.Y, 0)
	x1, y1 := min(rect.X+rect.W, ViewWidth), min(rect.Y+rect.H, ViewHeight)
	return ui.Rect{X: x0, Y: y0, W: max(x1-x0, 0), H: max(y1-y0, 0)}
}

func flagCloths(r *Renderer, it Placed, zoom float64) []ui.Rect {
	out := []ui.Rect{}
	for _, m := range r.Lib.Flags(r.ShownName(it.names)) {
		x, y := it.sx+float64(m.X)*zoom, it.sy+float64(m.Y)*zoom
		out = append(out, ui.Rect{X: int(x), Y: int(y), W: int(art.FlagClothW * zoom), H: int(art.FlagClothH * zoom)})
	}
	return out
}

func TestABuildingHalfOffTheScreenIsDrawnAndAnswersAClickOnItsVisibleHalf(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	sc.Selected = ""
	k, ok := FirstOfKind(sc.Site, view.StructKeep, false)
	if !ok {
		t.Fatal("the realm has no keep")
	}
	r := NewRenderer(testLib())
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	c := FootprintCentre(k)
	for _, zoom := range render.ZoomLevels {
		cam.Zoom = zoom
		home := cam.CenterOn(c.X, c.Y, ViewWidth, ViewHeight)
		r.Draw(dst, home, sc)
		at, ok := r.HitOf(HitStructure, k.ID)
		if !ok {
			t.Fatalf("zoom %v: the keep in the middle of the screen has no hit", zoom)
		}
		cx, cy := float64(at.Rect.X)+float64(at.Rect.W)/2, float64(at.Rect.Y)+float64(at.Rect.H)/2
		for _, edge := range []struct {
			name   string
			dx, dy float64
		}{{"left", cx, 0}, {"right", cx - ViewWidth, 0}, {"top", 0, cy}, {"bottom", 0, cy - ViewHeight}} {
			pan := home.Pan(edge.dx, edge.dy)
			for _, selected := range []string{"", k.ID} {
				sc.Selected = selected
				r.Draw(dst, pan, sc)
				hit, ok := r.HitOf(HitStructure, k.ID)
				if !ok {
					t.Fatalf("zoom %v, %s edge: the keep has no hit", zoom, edge.name)
				}
				part := visiblePart(hit.Rect)
				if part.W == 0 || part.H == 0 || part == hit.Rect {
					t.Fatalf("zoom %v, %s edge: the keep at %+v is not half on the screen", zoom, edge.name, hit.Rect)
				}
				item, ok := structureItemOf(clippedStructures(r, dst, pan, sc), k.ID)
				if !ok || item.Draw == nil {
					t.Errorf("zoom %v, %s edge: the keep half on the screen is not drawn", zoom, edge.name)
				}
				if _, _, ok := pickedAt(r, part, k.ID); !ok {
					t.Errorf("zoom %v, %s edge: a click on the keep's visible half at %+v does not answer", zoom, edge.name, part)
				}
				if selected != "" && !hasLabel(r.Drawn()) {
					t.Errorf("zoom %v, %s edge: the selected keep half on the screen has no label", zoom, edge.name)
				}
			}
		}
		sc.Selected = ""
	}
}

func hasLabel(boxes []DrawnBox) bool {
	for _, b := range boxes {
		if b.Kind == DrawnLabel {
			return true
		}
	}
	return false
}

func TestEveryBuildingThatReachesTheScreenIsDrawnWhileTheCameraMoves(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	r := NewRenderer(testLib())
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	culled := 0
	for _, zoom := range render.ZoomLevels {
		cam.Zoom = zoom
		start := cam.CenterOn(float64(sc.Site.Width)/2, float64(sc.Site.Height)/2, ViewWidth, ViewHeight)
		for step := -12; step <= 12; step++ {
			pan := start.Pan(float64(step)*97, float64(step)*41)
			r.Draw(dst, pan, sc)
			placed := r.placeStructures(pan, sc)
			items := clippedStructures(r, dst, pan, sc)
			for _, p := range placed {
				item, ok := structureItemOf(items, p.S.ID)
				if !ok {
					t.Fatalf("zoom %v: %s has no item", zoom, p.S.ID)
				}
				if _, ok := r.HitOf(HitStructure, p.S.ID); !ok {
					t.Fatalf("zoom %v: %s has no hit", zoom, p.S.ID)
				}
				if item.Draw != nil {
					continue
				}
				culled++
				if Overlaps(p.Rect, theScreen) {
					t.Errorf("zoom %v, step %d: %s reaches the screen at %+v but is not drawn", zoom, step, p.S.ID, p.Rect)
				}
				for _, f := range flagCloths(r, p, zoom) {
					if Overlaps(f, theScreen) {
						t.Errorf("zoom %v, step %d: the flag of %s reaches the screen at %+v but is not drawn", zoom, step, p.S.ID, f)
					}
				}
			}
		}
	}
	if culled == 0 {
		t.Fatal("no building was ever off the screen: the test moves the camera too little")
	}
}

func TestABuildingOffTheScreenKeepsItsHit(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	sc.Selected = ""
	k, _ := FirstOfKind(sc.Site, view.StructKeep, false)
	r := NewRenderer(testLib())
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	c := FootprintCentre(k)
	away := cam.CenterOn(c.X, c.Y, ViewWidth, ViewHeight).Pan(3*ViewWidth, 0)
	r.Draw(dst, away, sc)
	hit, ok := r.HitOf(HitStructure, k.ID)
	if !ok || Overlaps(hit.Rect, theScreen) {
		t.Fatalf("the keep is not off the screen: %+v %v", hit.Rect, ok)
	}
	if item, _ := structureItemOf(clippedStructures(r, dst, away, sc), k.ID); item.Draw != nil {
		t.Error("the keep off the screen is drawn")
	}
	if _, _, ok := pickedAt(r, hit.Rect, k.ID); !ok {
		t.Error("the keep off the screen no longer answers a pointer there, as one in a letterbox bar")
	}
}

func contains(outer, inner ui.Rect) bool {
	return inner.X >= outer.X && inner.Y >= outer.Y && inner.X+inner.W <= outer.X+outer.W && inner.Y+inner.H <= outer.Y+outer.H
}

func paintedBy(r *Renderer, cam render.Camera, site view.Site, p Placed) []ui.Rect {
	zoom := cam.Scale()
	out := []ui.Rect{p.Rect.Inset(-int(max(1, zoom)))}
	for _, m := range r.Lib.Flags(r.ShownName(p.names)) {
		x, y := math.Round(p.sx+float64(m.X)*zoom), math.Round(p.sy+float64(m.Y)*zoom)
		out = append(out, ui.Rect{X: int(x), Y: int(y), W: int(math.Ceil(art.FlagClothW * zoom)), H: int(math.Ceil(art.FlagClothH * zoom))})
	}
	lo, hi := Footprint(p.S)
	for y := lo.Y; y <= hi.Y; y++ {
		for x := lo.X; x <= hi.X; x++ {
			sx, sy := cam.GridToScreen(float64(x), float64(y))
			out = append(out, ui.Rect{X: int(math.Floor(sx - 16*zoom)), Y: int(math.Floor(sy - 8*zoom)), W: int(32 * zoom), H: int(16 * zoom)})
		}
	}
	ax, ay := ScreenPointOf(cam, site, Anchor(p.S))
	w, h := float64(p.S.W*12)*zoom, float64(10+6*p.S.W)*zoom
	return append(out, ui.Rect{X: int(math.Floor(ax - w/2)), Y: int(math.Floor(ay - h - 8*zoom)), W: int(w), H: int(h)})
}

func TestTheReachOfABuildingHoldsEverythingItsDrawPaints(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	r := NewRenderer(testLib())
	flags := 0
	for _, zoom := range render.ZoomLevels {
		cam.Zoom = zoom
		for _, p := range r.placeStructures(cam, sc) {
			reach := r.structureReach(cam, sc.Site, p)
			for _, painted := range paintedBy(r, cam, sc.Site, p) {
				if !contains(reach, painted) {
					t.Errorf("zoom %v: %s paints %+v outside its reach %+v", zoom, p.S.ID, painted, reach)
				}
			}
			flags += len(r.Lib.Flags(r.ShownName(p.names)))
		}
	}
	if flags == 0 {
		t.Fatal("no building in the realm flies a flag")
	}
}

func TestATallyBoardOrWhistleIsDrawnWhenAnyOfItReachesTheScreen(t *testing.T) {
	clip := image.Rect(0, 0, ViewWidth, ViewHeight)
	fm := model.Foreman{Namespace: "market", Name: "shop-7d9", Ready: 12, Desired: 120}
	var font ui.Font
	hidden := 0
	for _, sx := range []float64{-200, -30, -12, 0, 480, ViewWidth - 3, ViewWidth + 9, ViewWidth + 200} {
		for _, head := range []int{-200, -1, 5, 270, ViewHeight + 13, ViewHeight + 200} {
			board, text := tallyBoard(fm, sx, head)
			w, h := font.Measure(text, 1)
			painted := []ui.Rect{board, board.Offset(1, 1), {X: board.X + 2, Y: board.Y + 2, W: w, H: h}}
			got, want := tallyShows(clip, tallyMark{board: board, text: text}), anyMeets(clip, painted)
			if want && !got {
				t.Errorf("board %+v reaches the screen but is not shown", board)
			}
			hidden += boolInt(!got)
			whistle := tallyMark{Whistle: true, x: sx + 4, top: head + 4, zoom: 1}
			bubble := ui.Rect{X: int(whistle.x), Y: whistle.top - whistleH, W: whistleW, H: whistleH}
			painted = []ui.Rect{bubble, bubble.Offset(1, 1), {X: bubble.X + whistleW, Y: bubble.Y, W: 6, H: whistleH}}
			got, want = tallyShows(clip, whistle), anyMeets(clip, painted)
			if want && !got {
				t.Errorf("whistle %+v reaches the screen but is not shown", bubble)
			}
			hidden += boolInt(!got)
		}
	}
	if hidden == 0 {
		t.Error("no board or whistle far off the screen is hidden")
	}
}

func anyMeets(clip image.Rectangle, rects []ui.Rect) bool {
	for _, r := range rects {
		if Overlaps(r, ui.Rect{X: clip.Min.X, Y: clip.Min.Y, W: clip.Dx(), H: clip.Dy()}) {
			return true
		}
	}
	return false
}

func TestABuildingOffTheScreenStillMakesTheImagesItsDrawWouldMake(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	sc.Selected = ""
	k, _ := FirstOfKind(sc.Site, view.StructKeep, false)
	r := NewRenderer(testLib())
	r.livingOf().folk = []keepFolk{{s: k, quiet: IvyFullAfter + 1}}
	level := r.ivyFor(k)
	if level == 0 {
		t.Fatal("the quiet keep grows no ivy")
	}
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	c := FootprintCentre(k)
	away := cam.CenterOn(c.X, c.Y, ViewWidth, ViewHeight).Pan(3*ViewWidth, 0)
	var placed Placed
	for _, p := range r.placeStructures(away, sc) {
		if p.S.ID == k.ID {
			placed = p
		}
	}
	name := r.ShownName(placed.names)
	flags := r.Lib.Flags(name)
	if len(flags) == 0 {
		t.Fatalf("the keep's sprite %s flies no flag", name)
	}
	item, _ := structureItemOf(clippedStructures(r, dst, away, sc), k.ID)
	if r.ivyPainted(name, level) {
		t.Fatal("the ivy was painted before the frame drew")
	}
	if item.Draw == nil {
		t.Fatal("the keep off the screen leaves its ivy unpainted")
	}
	item.Draw(dst)
	if !r.ivyPainted(name, level) {
		t.Error("the keep off the screen did not paint its ivy in its place in the frame")
	}
	for i, m := range flags {
		if !gfx.HasFlagCloth(m, FlagFrame(sc.Time, k, i)) {
			t.Errorf("flag %d of the keep off the screen has no cloth for this frame", i)
		}
	}
	if again, _ := structureItemOf(clippedStructures(r, dst, away, sc), k.ID); again.Draw != nil {
		t.Error("the keep off the screen draws again with nothing left to make")
	}
}

func flagSeedByHasher(s view.Structure, i int) (float64, float64) {
	h := fnv.New32a()
	h.Write([]byte(strconv.Itoa(s.X) + "," + strconv.Itoa(s.Y) + "," + strconv.Itoa(i)))
	v := h.Sum32()
	phase := float64(v%1000) / 1000
	pace := 1 - flagPaceSwing + 2*flagPaceSwing*float64((v/1000)%1000)/1000
	return phase, pace
}

func TestFlagSeedIsTheFNVOfThePlaceAndTheIndex(t *testing.T) {
	for _, xy := range [][2]int{{0, 0}, {3, 7}, {-1, 12}, {120, -45}, {198, 198}, {-2147483648, 2147483647}, {1 << 40, -(1 << 41)}} {
		for i := range 4 {
			s := view.Structure{X: xy[0], Y: xy[1]}
			gp, gq := flagSeed(s, i)
			wp, wq := flagSeedByHasher(s, i)
			if gp != wp || gq != wq {
				t.Errorf("%v flag %d: seed %v %v, the FNV of the same text %v %v", xy, i, gp, gq, wp, wq)
			}
		}
	}
}
