package world

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestWalkersSouthOrEastOfAFootprintStandInFront(t *testing.T) {
	lo, hi := Point{10, 10}, Point{12, 12}
	cases := []struct {
		foot Vec
		want bool
	}{
		{Vec{13, 11}, true}, {Vec{11, 13}, true}, {Vec{13, 7}, true}, {Vec{8, 13}, true},
		{Vec{9, 11}, false}, {Vec{11, 9}, false}, {Vec{12.4, 12.4}, false},
	}
	for _, c := range cases {
		if got := InFront(c.foot, lo, hi); got != c.want {
			t.Errorf("InFront(%v) = %v, want %v", c.foot, got, c.want)
		}
	}
}

func TestLiftWalkersDrawsThemOverTheBuildingTheyFront(t *testing.T) {
	building := Drawable{Depth: 25000, box: footprint{on: true, lo: Point{10, 10}, hi: Point{12, 12}, rect: ui.Rect{X: 100, Y: 100, W: 80, H: 60}}}
	east := Drawable{Depth: 20000, stand: stand(Vec{13, 7}, 150, 150)}
	behind := Drawable{Depth: 19000, stand: stand(Vec{9, 11}, 120, 150)}
	far := Drawable{Depth: 18000, stand: stand(Vec{14, 3}, 400, 20)}
	items := []Drawable{building, east, behind, far}
	liftWalkers(items, 1)
	if items[1].Depth <= building.Depth {
		t.Error("a walker east of the footprint must draw over it")
	}
	if items[2].Depth != 19000 {
		t.Error("a walker west of the footprint stays behind it")
	}
	if items[3].Depth != 18000 {
		t.Error("a walker nowhere near the sprite keeps its depth")
	}
}

func TestCalloutsFadeAndStepAroundTheHUD(t *testing.T) {
	if CalloutAlpha(0, false) != 1 || CalloutAlpha(CalloutLife, false) != 0 || CalloutAlpha(-1, false) != 0 {
		t.Error("callouts live for CalloutLife")
	}
	if a := CalloutAlpha(CalloutLife*.9, false); a <= 0 || a >= 1 {
		t.Errorf("callouts fade at the end: %v", a)
	}
	if CalloutAlpha(CalloutLife*.9, true) != 1 {
		t.Error("reduced effects do not fade")
	}
	c := Callout{Text: "relit porter-a", Count: 3}
	if CalloutText(c) != "relit porter-a x3" {
		t.Errorf("text = %q", CalloutText(c))
	}
	r := CalloutRect(c, 300, 300, 1, true)
	if r.Y+r.H > 300 || r.W < calloutFace {
		t.Errorf("callout rect %+v must sit above its object", r)
	}
	avoid := []ui.Rect{{X: r.X - 10, Y: r.Y - 5, W: r.W + 20, H: r.H + 10}}
	moved, ok := clearOf(r, avoid)
	if !ok || Overlaps(moved, avoid[0]) {
		t.Errorf("callout %+v still covers %+v", moved, avoid[0])
	}
	for _, sx := range []float64{0, 5, ViewWidth - 5, ViewWidth} {
		if r := CalloutRect(c, sx, 300, 1, true); r.X < calloutMargin || r.X+r.W > ViewWidth-calloutMargin {
			t.Errorf("a callout at x=%.0f leaves the screen: %+v", sx, r)
		}
	}
	if _, ok := clearOf(ui.Rect{X: 0, Y: 500, W: 10, H: 30}, []ui.Rect{{X: 0, Y: 480, W: 20, H: 50}}); ok {
		t.Error("a callout with nowhere to go is skipped")
	}
}

func TestClarityDrawsWithAndWithoutArt(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	sc.Ghost = nil
	ks := Keeps(sc.Site)
	sc.Selected = ks[0].ID
	goal := Anchor(ks[0])
	sc.Goal = &goal
	sc.Callouts = []Callout{{Portrait: "steward", Text: "relit porter-a", At: Anchor(ks[0]), Age: .5, Count: 2}, {Portrait: "reeve", Text: "gone", At: Anchor(ks[0]), Age: 99}}
	sc.Avoid = []ui.Rect{{X: 0, Y: 0, W: 10, H: 10}}
	sc.Pods = append(sc.Pods, model.PodView{Namespace: "default", Name: "porter-sel", Node: ks[0].Object.Name, Tier: 1, State: model.HearthReady})
	dst := ebiten.NewImage(960, 540)
	for _, lib := range []*gfx.Library{nil, gfx.Load(art.Build())} {
		r := NewRenderer(lib)
		cam = cam.CenterOn(float64(ks[0].X), float64(ks[0].Y), 960, 540)
		r.Draw(dst, cam, sc)
		sc.Selected = "default/porter-sel"
		r.Draw(dst, cam, sc)
		r.DrawCallouts(dst, cam, sc)
		sc.Selected = ks[0].ID
	}
	if !selectedPod("porter-a", "default/porter-a") || selectedPod("porter-a", "keep-1") {
		t.Error("pod selection ids")
	}
}

func TestCalloutsNeverCoverEachOther(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	at := Anchor(Keeps(sc.Site)[0])
	sc.Callouts = []Callout{
		{Portrait: "steward", Text: "The controllers took porter-d2hqw off the Node", At: at, Age: .5, Count: 2},
		{Portrait: "reeve", Text: "The kubelet relit porter-4vxnn", At: at, Age: .5},
		{Portrait: "reeve", Text: "seated a Pod", At: at, Age: .5},
	}
	placed := placeCallouts(cam, sc)
	if len(placed) != len(sc.Callouts) {
		t.Fatalf("placed %d of %d callouts", len(placed), len(sc.Callouts))
	}
	for i, a := range placed {
		for _, b := range placed[i+1:] {
			if Overlaps(a.box, b.box) {
				t.Errorf("callout %+v covers %+v", a.box, b.box)
			}
		}
	}
}

func TestALongCalloutWrapsInsteadOfCutting(t *testing.T) {
	c := Callout{Text: "The kube-apiserver warded lone-tower with node.kubernetes.io/not-ready as it joined: its kubelet is not ready (TaintNodesByCondition admission on new nodes)"}
	text := CalloutText(c)
	if strings.Contains(text, "...") {
		t.Fatalf("the callout was cut: %q", text)
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 3 {
		t.Fatalf("a long callout keeps one line: %q", text)
	}
	for _, l := range lines {
		if len([]rune(l)) > calloutChars {
			t.Fatalf("a callout line of %d characters: %q", len([]rune(l)), l)
		}
	}
	if strings.Join(strings.Fields(text), " ") != c.Text {
		t.Fatalf("the wrapped callout lost words: %q", text)
	}
	tall, short := CalloutRect(c, 300, 300, 1, true), CalloutRect(Callout{Text: "relit porter-a"}, 300, 300, 1, true)
	if tall.H <= short.H || tall.Y+tall.H > short.Y+short.H {
		t.Fatalf("a wrapped callout %+v grows upward from where a short one %+v stands", tall, short)
	}
}

func TestACalloutNeverCoversALabelTheWorldDrew(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	at := Anchor(Keeps(sc.Site)[0])
	sc.Callouts = []Callout{{Portrait: "reeve", Text: "The kubelet relit porter-4vxnn", At: at, Age: .5}}
	natural := placeCallouts(cam, sc)
	if len(natural) != 1 {
		t.Fatal("the callout was not placed: the test proves nothing")
	}
	r := NewRenderer(nil)
	r.note(DrawnLabel, "label", "", natural[0].box.Inset(-1))
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.DrawCallouts(dst, cam, sc)
	for _, b := range r.Drawn() {
		if b.Kind == DrawnCallout && Overlaps(b.Rect, natural[0].box.Inset(-1)) {
			t.Fatalf("the callout %+v covers the label %+v the world drew before it", b.Rect, natural[0].box)
		}
	}
}

func TestAForemansCalloutKeepsClearOfTheCalloutsWhereTheyWereDrawn(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	at := Anchor(Keeps(sc.Site)[0])
	sc.Callouts = []Callout{{Portrait: "reeve", Text: "The kubelet relit porter-4vxnn", At: at, Age: .5}}
	natural := placeCallouts(cam, sc)
	if len(natural) != 1 {
		t.Fatal("the callout was not placed: the test proves nothing")
	}
	r := NewRenderer(nil)
	r.note(DrawnLabel, "label", "", natural[0].box.Inset(-1))
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.DrawCallouts(dst, cam, sc)
	obstacles := r.SayObstacles(cam, sc)
	for _, b := range r.Drawn() {
		if b.Kind != DrawnCallout {
			continue
		}
		if !slices.ContainsFunc(obstacles, func(o ui.Rect) bool { return o == b.Rect || o.Inset(-1) == b.Rect.Inset(-1) || containsBox(o, b.Rect) }) {
			t.Fatalf("a foreman's callout keeps clear of %+v, not of the callout drawn at %+v", obstacles, b.Rect)
		}
	}
}

func containsBox(outer, inner ui.Rect) bool {
	return inner.X >= outer.X && inner.Y >= outer.Y && inner.X+inner.W <= outer.X+outer.W && inner.Y+inner.H <= outer.Y+outer.H
}

func TestACalloutByTheTopEdgeStaysOnTheScreenBelowTheTopBar(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	at := Anchor(Keeps(sc.Site)[0])
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	cam = cam.Pan(sx-float64(ViewWidth)/2, sy-12)
	topBar := ui.Rect{X: 0, Y: 0, W: ViewWidth, H: 18}
	sc.Avoid = []ui.Rect{topBar}
	sc.Callouts = []Callout{{Portrait: "steward", Text: "The shift porters raised porters-69k7m to stay whole", At: at, Age: .5}}
	r := NewRenderer(nil)
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	r.DrawCallouts(dst, cam, sc)
	found := false
	for _, b := range r.Drawn() {
		if b.Kind != DrawnCallout {
			continue
		}
		found = true
		if b.Rect.Y < 0 || Overlaps(b.Rect, topBar) {
			t.Fatalf("a callout for an object by the top edge sits at %+v: above the screen or on the top bar %+v (the card-following novice at 15:39)", b.Rect, topBar)
		}
	}
	if !found {
		t.Fatal("the callout for an object on the screen was not drawn")
	}
}

func TestAForemansCalloutByTheTopEdgeStaysOnTheScreenBelowTheTopBar(t *testing.T) {
	topBar := ui.Rect{X: 0, Y: 0, W: ViewWidth, H: 18}
	box, ok := ClearAbove(ui.Rect{X: 300, Y: -30, W: 200, H: 24}, []ui.Rect{topBar})
	if !ok || box.Y < 0 || Overlaps(calloutShade(box), topBar) {
		t.Fatalf("a foreman's callout over an object by the top edge sits at %+v (placed %v): above the screen or on the top bar", box, ok)
	}
}
