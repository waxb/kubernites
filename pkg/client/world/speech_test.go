package world

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestBubbleTypesAndGrows(t *testing.T) {
	sp := Speech{Text: "Burning bright! That's the good kind of burning.", Start: 10, CPS: 30}
	if Typed(sp, 10, false) != 0 || Typed(sp, 10.5, false) != 15 || Typed(sp, 100, false) != len(sp.Text) {
		t.Errorf("typed %d %d %d", Typed(sp, 10, false), Typed(sp, 10.5, false), Typed(sp, 100, false))
	}
	if Typed(sp, 10, true) != len(sp.Text) {
		t.Error("with Reduce effects the whole line shows at once")
	}
	small := BubbleRect(BubbleText(sp, 3), 400, 300)
	half := BubbleRect(BubbleText(sp, 15), 400, 300)
	full := BubbleRect(BubbleText(sp, len(sp.Text)), 400, 300)
	if !(small.W < half.W && half.W <= full.W) || small.H > full.H {
		t.Errorf("the bubble starts small and grows: %v %v %v", small, half, full)
	}
	if full.W > bubbleWidth+2*bubblePad {
		t.Errorf("the bubble's width is capped: %v", full)
	}
	lines := BubbleText(sp, len(sp.Text))
	if len(lines) < 2 || strings.Join(lines, " ") != sp.Text {
		t.Errorf("a long line wraps, word by word: %q", lines)
	}
	part := BubbleText(sp, 20)
	if strings.Join(part, " ") != sp.Text[:20] {
		t.Errorf("typed so far %q", part)
	}
}

func TestBubbleStaysThenFades(t *testing.T) {
	sp := Speech{Text: "Box in, box out.", Start: 0, CPS: 32}
	done := sp.Done()
	if BubbleAlpha(sp, done+BubbleHold-0.1, false) != 1 {
		t.Error("the bubble stays a few seconds after the line ends")
	}
	if a := BubbleAlpha(sp, done+BubbleHold+BubbleFade/2, false); a <= 0 || a >= 1 {
		t.Errorf("then fades: %v", a)
	}
	if BubbleAlpha(sp, done+BubbleHold+BubbleFade+0.01, false) != 0 {
		t.Error("then is gone")
	}
	if BubbleAlpha(sp, done+BubbleHold+BubbleFade/2, true) != 0 {
		t.Error("with Reduce effects it leaves without fading")
	}
}

func TestBubbleNeverCoversTheHUD(t *testing.T) {
	box := BubbleRect([]string{"Hello there"}, 400, 300)
	card := ui.Rect{X: box.X - 10, Y: box.Y - 10, W: box.W + 20, H: box.H + 12}
	moved, above, ok := bubbleSpot(box, 400, 300, 330, []ui.Rect{card})
	if !ok || Overlaps(moved, card) {
		t.Fatalf("the bubble moves clear of a card: %v %v", moved, card)
	}
	if above {
		t.Error("moved below the feet, the tail points up")
	}
	everywhere := ui.Rect{X: 0, Y: 0, W: ViewWidth, H: ViewHeight}
	if _, _, ok := bubbleSpot(box, 400, 300, 330, []ui.Rect{everywhere}); ok {
		t.Error("with no clear spot the bubble is not drawn")
	}
}

func TestSpeechFollowsTheWalkerAndReadsAsItsOwnRegion(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	var porter model.PodView
	for _, p := range sc.Pods {
		if p.State == model.HearthReady && !p.System {
			porter = p.Person(0)
			break
		}
	}
	if porter.Name == "" {
		t.Skip("no burning porter")
	}
	sc.Focus = PersonKey(porter)
	sc.Real = 50
	sc.Speech = &Speech{Person: sc.Focus, Text: "Box in, box out.", Start: 49, CPS: 30}
	r := NewRenderer(gfx.Load(art.Build()))
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	mark, ok := r.Focus()
	if !ok || !mark.Seen {
		t.Fatalf("the renderer marks the selected walker: %+v", mark)
	}
	var runs []ui.TextRun
	stop := ui.RecordText(func(run ui.TextRun) { runs = append(runs, run) })
	r.DrawSpeech(dst, cam, sc)
	stop()
	text := []string{}
	for _, run := range runs {
		if run.Region == BubbleRegion {
			text = append(text, run.Text)
		}
	}
	if strings.Join(text, " ") != "Box in, box out." {
		t.Fatalf("bubble text = %q (runs %+v)", text, runs)
	}
	sc.Time += 3
	r.Draw(dst, cam, sc)
	if next, _ := r.Focus(); next.Pos == mark.Pos && next.Activity != "resting" && next.Activity != "waiting" && next.Activity != "serving" {
		t.Errorf("the mark follows the walker as it moves: %+v then %+v", mark, next)
	}
}

func TestBubbleTypesEveryLetterOfAWordTheFontWouldDecorate(t *testing.T) {
	sp := Speech{Text: "Night work. Same boxes, by lantern light.", CPS: 30}
	lines := BubbleText(sp, len(sp.Text))
	if got := strings.Join(lines, " "); got != sp.Text {
		t.Errorf("fully typed bubble reads %q", got)
	}
}

func TestEveryWorkerWearsItsOwnAir(t *testing.T) {
	base := model.PodView{Namespace: "default", Name: "porters-7d9f-x2kq", Sprite: "porter", Profession: "Porter",
		People: []model.Kubernite{{Name: "porter", Sprite: "porter"}, {Name: "helper", Sprite: "porter"}}}
	a, b := AirOf(base.Person(0), "porter"), AirOf(base.Person(0), "porter")
	if a != b || !a.On {
		t.Errorf("the same worker always looks the same: %+v %+v", a, b)
	}
	differ := 0
	for i := range 40 {
		p := base
		p.Name = "porters-7d9f-" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		x, y := AirOf(p.Person(0), "porter"), AirOf(p.Person(1), "porter")
		if x != y {
			differ++
		}
		if x.Skin < 0 || x.Skin > 3 || x.Hair < 0 || x.Hair > 5 || x.Warm < -1 || x.Warm > 1 || x.Light < -1 || x.Light > 1 {
			t.Fatalf("air out of range: %+v", x)
		}
	}
	if differ < 38 {
		t.Errorf("two workers of one hut rarely look alike: only %d of 40 pairs differ", differ)
	}
	sys := base
	sys.System = true
	if AirOf(sys.Person(0), "porter").On || AirOf(base.Person(0), "steward").On {
		t.Error("only the trades' workers vary at draw time")
	}
}

func shapePixels(sh bubbleShape) map[[2]int]bool {
	out := map[[2]int]bool{}
	for _, rs := range [][]ui.Rect{sh.fill, sh.ink} {
		for _, r := range rs {
			for y := r.Y; y < r.Y+r.H; y++ {
				for x := r.X; x < r.X+r.W; x++ {
					out[[2]int{x, y}] = true
				}
			}
		}
	}
	return out
}

func TestTheSpeechBubbleHasRoundedCornersAndATailToTheSpeaker(t *testing.T) {
	box := ui.Rect{X: 100, Y: 50, W: 60, H: 24}
	for _, c := range []struct {
		sx    float64
		above bool
	}{{130, true}, {94, true}, {165, false}, {240, false}} {
		px := shapePixels(shapeBubble(box, c.sx, c.above))
		for _, corner := range [][2]int{{box.X, box.Y}, {box.X + box.W - 1, box.Y}, {box.X, box.Y + box.H - 1}, {box.X + box.W - 1, box.Y + box.H - 1}} {
			if px[corner] {
				t.Errorf("the bubble fills its corner %v: its corners are rounded", corner)
			}
		}
		tip := box.Y + box.H + bubbleTail - 1
		if !c.above {
			tip = box.Y - bubbleTail
		}
		toward := min(max(int(c.sx), box.X-bubbleReach), box.X+box.W-1+bubbleReach)
		near := false
		for dx := -1; dx <= 1; dx++ {
			near = near || px[[2]int{toward + dx, tip}]
		}
		if !near {
			t.Errorf("speaker at x %v (above %v): the tail's tip at row %d does not point at the speaker (x %d)", c.sx, c.above, tip, toward)
		}
	}
}

func speakingPorter(t *testing.T) (Scene, render.Camera, *Renderer) {
	t.Helper()
	sc, cam := sceneFor(t, .5)
	for _, p := range sc.Pods {
		if p.State == model.HearthReady && !p.System {
			sc.Focus = PersonKey(p.Person(0))
			break
		}
	}
	if sc.Focus == "" {
		t.Skip("no burning porter")
	}
	sc.Real = 50
	sc.Speech = &Speech{Person: sc.Focus, Text: "Box in, box out, and the next box in.", Start: 48.5, CPS: 30}
	return sc, cam, NewRenderer(gfx.Load(art.Build()))
}

func TestTheSpeechBubbleRegistersItsWholeShapeTailIncludedAndKeepsItClearOfTheHUD(t *testing.T) {
	sc, cam, r := speakingPorter(t)
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	mark, ok := r.Focus()
	if !ok || !mark.Seen {
		t.Fatalf("no walker in view: %+v", mark)
	}
	sx, sy := ScreenPointOf(cam, sc.Site, mark.Pos)
	sy -= r.HarborOf().Lift(mark.Pos) * cam.Scale()
	head := sy - bubbleHead*cam.Scale()
	lines := BubbleText(*sc.Speech, Typed(*sc.Speech, sc.Real, sc.Reduce))
	home := BubbleRect(lines, sx, head)
	layouts := [][]ui.Rect{nil}
	for _, side := range []int{-1, 1} {
		cover := ui.Rect{X: home.X - 4, Y: 0, W: home.W + 8, H: ViewHeight}
		near := ui.Rect{X: home.X + home.W/2 + home.W/2*side, Y: 0, W: 2, H: ViewHeight}
		if side < 0 {
			near.X = home.X - home.W/2 - 2
		}
		layouts = append(layouts, []ui.Rect{cover}, []ui.Rect{cover, near})
	}
	for i, avoid := range layouts {
		sc.Avoid = avoid
		r.Draw(dst, cam, sc)
		r.DrawSpeech(dst, cam, sc)
		var reg ui.Rect
		for _, b := range r.Drawn() {
			if b.Kind == DrawnSpeech {
				reg = b.Rect
			}
		}
		box, above, ok := bubbleSpot(home, sx, head, sy, avoid)
		if !ok {
			if reg.W != 0 {
				t.Errorf("layout %d: no spot for the bubble, yet it registered %+v", i, reg)
			}
			continue
		}
		if reg.W == 0 {
			t.Fatalf("layout %d: the bubble drew at %+v and registered nothing", i, box)
		}
		for px := range shapePixels(shapeBubble(box, sx, above)) {
			if !reg.Contains(px[0], px[1]) {
				t.Fatalf("layout %d: the bubble paints %v outside its registered rect %+v (box %+v, speaker x %.0f)", i, px, reg, box, sx)
			}
		}
		for _, a := range avoid {
			if Overlaps(reg, a) {
				t.Errorf("layout %d: the bubble's shape %+v covers the HUD's %+v", i, reg, a)
			}
		}
	}
}
