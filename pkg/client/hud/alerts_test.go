package hud

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func squareView() [4][2]float64 {
	return [4][2]float64{{10, 10}, {40, 10}, {40, 30}, {10, 30}}
}

func alertFrame(t *testing.T, alerts ...Alert) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Objectives = nil
	f.View = squareView()
	f.Alerts = alerts
	return f
}

func TestViewPointProjectsThroughTheViewCorners(t *testing.T) {
	v := squareView()
	cases := []struct{ gx, gy, sx, sy float64 }{
		{10, 10, 0, TopH}, {40, 10, Width, TopH}, {10, 30, 0, ConsoleY}, {25, 20, Width / 2, (TopH + ConsoleY) / 2.0},
	}
	for _, c := range cases {
		sx, sy, ok := ViewPoint(v, c.gx, c.gy)
		if !ok || math.Abs(sx-c.sx) > 1e-6 || math.Abs(sy-c.sy) > 1e-6 {
			t.Errorf("ViewPoint(%v,%v) = %v,%v,%v want %v,%v", c.gx, c.gy, sx, sy, ok, c.sx, c.sy)
		}
	}
	if _, _, ok := ViewPoint([4][2]float64{}, 1, 1); ok {
		t.Error("a degenerate view projects nothing")
	}
}

func TestOnScreenAlertsGetNoArrow(t *testing.T) {
	f := alertFrame(t, Alert{Kind: AlertFire, At: [2]float64{25, 20}, Text: "on screen"})
	if arrows := AlertArrows(f, nil); len(arrows) != 0 {
		t.Errorf("arrows = %+v", arrows)
	}
}

func TestOffScreenAlertsPointFromTheEdge(t *testing.T) {
	f := alertFrame(t,
		Alert{Kind: AlertPending, At: [2]float64{80, 20}, Text: "east"},
		Alert{Kind: AlertFire, At: [2]float64{-40, 20}, Text: "west"},
	)
	arrows := AlertArrows(f, nil)
	if len(arrows) != 2 {
		t.Fatalf("arrows = %+v", arrows)
	}
	west, east := arrows[0], arrows[1]
	if west.Target.Text != "west" || west.Level != LevelDanger {
		t.Errorf("the most severe arrow comes first: %+v", west)
	}
	if west.X > arrowMargin+2 || west.DX > -0.9 {
		t.Errorf("west arrow = %+v", west)
	}
	if east.X < Width-arrowMargin-2 || east.DX < 0.9 {
		t.Errorf("east arrow = %+v", east)
	}
	for _, a := range arrows {
		if a.Y < TopH+arrowMargin || a.Y > ConsoleY-arrowMargin {
			t.Errorf("arrow leaves the world view: %+v", a)
		}
	}
}

func TestNearbyAlertsClusterWithACount(t *testing.T) {
	f := alertFrame(t,
		Alert{Kind: AlertPending, At: [2]float64{80, 20}, Text: "one"},
		Alert{Kind: AlertFire, At: [2]float64{81, 20.5}, Text: "two"},
		Alert{Kind: AlertKeep, At: [2]float64{80, 21}, Text: "three"},
	)
	arrows := AlertArrows(f, nil)
	if len(arrows) != 1 || arrows[0].Count != 3 || arrows[0].Target.Text != "two" || arrows[0].Level != LevelDanger {
		t.Fatalf("clustered = %+v", arrows)
	}
	b := alertButtons(f, arrows)[0]
	if !strings.Contains(b.Tip, "two") || !strings.Contains(b.Tip, "+2 more") {
		t.Errorf("tip = %q", b.Tip)
	}
}

func TestArrowsStepAroundHUDPanels(t *testing.T) {
	f := alertFrame(t, Alert{Kind: AlertFire, At: [2]float64{-5, 11}, Text: "north-west"})
	free := AlertArrows(f, nil)[0]
	block := ui.Rect{X: free.X - 30, Y: free.Y - 30, W: 60, H: 60}
	moved := AlertArrows(f, []ui.Rect{block})[0]
	if block.Inset(-arrowHalf).Contains(moved.X, moved.Y) {
		t.Errorf("arrow %d,%d still under the panel %+v", moved.X, moved.Y, block)
	}
	track := arrowTrack()
	if !track.Contains(moved.X, moved.Y) {
		t.Errorf("arrow %d,%d left the track", moved.X, moved.Y)
	}
}

func TestArrowsAreCapped(t *testing.T) {
	alerts := []Alert{}
	for i := range 40 {
		a := float64(i) / 40 * 2 * math.Pi
		alerts = append(alerts, Alert{Kind: AlertPending, At: [2]float64{25 + 200*math.Cos(a), 20 + 200*math.Sin(a)}, Text: "x"})
	}
	if n := len(AlertArrows(alertFrame(t, alerts...), nil)); n > maxArrows || n == 0 {
		t.Errorf("%d arrows", n)
	}
}

func TestClickingAnArrowFocusesTheCamera(t *testing.T) {
	h := New(nil, Backend{})
	f := alertFrame(t, Alert{Kind: AlertFire, At: [2]float64{80, 20}, Text: "porter-a is guttering"})
	out := clickOn(t, h, f, "alert:0")
	it, ok := hasIntent(out, IntentFocus)
	if !ok || it.X != 80 || it.Y != 20 {
		t.Fatalf("focus = %+v %v", it, ok)
	}
	if !h.ping.on || h.ping.at != [2]float64{80, 20} {
		t.Error("the focused spot is pinged")
	}
	h.Open(OverlayGrimoire)
	for _, b := range h.Buttons(f) {
		if strings.HasPrefix(b.ID, "alert:") {
			t.Error("arrows hide behind overlays")
		}
	}
}

func TestMinimapBlipsFocusTheirAlert(t *testing.T) {
	h := New(nil, Backend{})
	f := alertFrame(t, Alert{Kind: AlertKeep, At: [2]float64{12, 14}, Text: "Stone Keep is NotReady"})
	var blip Button
	for _, b := range h.Buttons(f) {
		if b.ID == "blip:0" {
			blip = b
		}
	}
	if blip.ID == "" || !Minimap.Contains(blip.Rect.X+blip.Rect.W/2, blip.Rect.Y+blip.Rect.H/2) || blip.Tip == "" {
		t.Fatalf("blip = %+v", blip)
	}
	it, ok := hasIntent(clickOn(t, h, f, "blip:0"), IntentFocus)
	if !ok || it.X != 12 || it.Y != 14 {
		t.Errorf("blip focus = %+v", it)
	}
	if h.pressAlert("9", f) != nil || h.pressBlip("x", f) != nil {
		t.Error("stale ids do nothing")
	}
}

func TestTabCyclesTheAlertsBySeverity(t *testing.T) {
	h := New(nil, Backend{})
	f := alertFrame(t,
		Alert{Kind: AlertPending, At: [2]float64{1, 1}, Text: "low"},
		Alert{Kind: AlertFire, At: [2]float64{2, 2}, Text: "high"},
	)
	tab := Input{Keys: []ebiten.Key{ebiten.KeyTab}}
	first, _ := hasIntent(h.Update(tab, f), IntentFocus)
	second, _ := hasIntent(h.Update(tab, f), IntentFocus)
	third, _ := hasIntent(h.Update(tab, f), IntentFocus)
	if first.X != 2 || second.X != 1 || third.X != 2 {
		t.Errorf("cycle = %v %v %v", first.X, second.X, third.X)
	}
	quiet := alertFrame(t)
	if _, ok := hasIntent(h.Update(tab, quiet), IntentFocus); ok {
		t.Error("tab does nothing without alerts")
	}
}

func TestHomeKeyFocusesTheFirstKeep(t *testing.T) {
	h := New(nil, Backend{})
	f := alertFrame(t)
	it, ok := hasIntent(h.Update(Input{Keys: []ebiten.Key{ebiten.KeyH}}, f), IntentFocus)
	if !ok || it.X != float64(f.Site.Gate.X) || it.Y != float64(f.Site.Gate.Y) {
		t.Errorf("home without a keep = %+v", it)
	}
}

func TestQuayKeyReturnsTheCameraToTheQuay(t *testing.T) {
	h := New(nil, Backend{})
	f := alertFrame(t)
	it, ok := hasIntent(h.Update(Input{Keys: []ebiten.Key{ebiten.KeyP}}, f), IntentGo)
	if !ok || it.Target != view.ParcelPile {
		t.Errorf("the quay key = %+v", it)
	}
}

func TestAlertLevelsAndDraw(t *testing.T) {
	for _, k := range []string{AlertFire, AlertDark, AlertStorm, AlertKeep, AlertGhost, AlertFallen, AlertImp, AlertSiege, AlertPending, "unknown"} {
		if AlertLevel(k) < LevelNotice || AlertLevel(k) > LevelDanger || alertIcon(k) == "" {
			t.Errorf("kind %s has no level or icon", k)
		}
	}
	if AlertLevel(AlertFire) <= AlertLevel(AlertPending) {
		t.Error("fire outranks a pending hearth")
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	f := alertFrame(t,
		Alert{Kind: AlertFire, At: [2]float64{80, 20}, Text: "a"}, Alert{Kind: AlertFire, At: [2]float64{80, 21}, Text: "b"},
		Alert{Kind: AlertPending, At: [2]float64{-9, 50}, Text: "c"}, Alert{Kind: AlertSiege, At: [2]float64{20, -40}, Text: "d"})
	for _, reduce := range []bool{false, true} {
		h := New(nil, Backend{})
		h.Reduce = reduce
		h.Update(Input{MX: -1, MY: -1}, f)
		clickOn(t, h, f, "alert:0")
		h.Now = 0.3
		h.Draw(dst, f)
		h.Now = 5
		h.Draw(dst, f)
	}
}

func TestMenuTogglesEdgeScroll(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	h.Open(OverlayMenu)
	var edge Button
	for _, b := range h.Buttons(f) {
		if b.ID == "menu:edge" {
			edge = b
		}
		if strings.HasPrefix(b.ID, "menu:") && !MenuBox.Contains(b.Rect.X+b.Rect.W, b.Rect.Y+b.Rect.H) {
			t.Errorf("%s leaves the menu box", b.ID)
		}
	}
	if edge.Label != "EDGE SCROLL: OFF" {
		t.Errorf("edge scroll is off by default: %+v", edge)
	}
	if _, ok := hasIntent(clickOn(t, h, f, "menu:edge"), IntentEdge); !ok {
		t.Error("the menu toggles edge scrolling")
	}
	h.SetEdgeScroll(true)
	h.Open(OverlayMenu)
	for _, b := range h.MenuButtons(f) {
		if b.ID == "menu:edge" && b.Label != "EDGE SCROLL: ON" {
			t.Errorf("label = %q", b.Label)
		}
	}
}

func TestArrowsCaptureTheMouse(t *testing.T) {
	h := New(nil, Backend{})
	f := alertFrame(t, Alert{Kind: AlertFire, At: [2]float64{80, 20}, Text: "east"})
	a := AlertArrows(f, nil)[0]
	if !h.Captures(a.X, a.Y, f) {
		t.Error("clicks on an arrow never reach the world")
	}
	if h.Captures(Width/2, (TopH+ConsoleY)/2, f) {
		t.Error("the open world is not captured")
	}
}

func TestOneArrowPerCauseWhateverTheDistance(t *testing.T) {
	alerts := []Alert{}
	for i := range 6 {
		a := float64(i) / 6 * 2 * math.Pi
		alerts = append(alerts, Alert{Kind: AlertPending, At: [2]float64{25 + 400*math.Cos(a), 20 + 400*math.Sin(a)}, Text: "hut waits Unhoused"})
	}
	alerts = append(alerts, Alert{Kind: AlertFire, At: [2]float64{25 + 400*math.Cos(0.5), 20 + 400*math.Sin(0.5)}, Text: "hut is guttering"})
	arrows := AlertArrows(alertFrame(t, alerts...), nil)
	byKind := map[string]int{}
	for _, a := range arrows {
		byKind[a.Kind]++
	}
	if byKind[AlertPending] != 1 || byKind[AlertFire] != 1 {
		t.Fatalf("arrows per cause = %v", byKind)
	}
	for _, a := range arrows {
		if a.Kind == AlertPending && a.Count != 6 {
			t.Errorf("the Unhoused arrow counts %d huts, want 6", a.Count)
		}
	}
}

func TestAWideToastStepsAroundAnAlertArrowAtTheEdge(t *testing.T) {
	h := New(nil, Backend{})
	for y := 11.0; y <= 29; y += .5 {
		f := alertFrame(t, Alert{Kind: AlertFire, At: [2]float64{-40, y}, Text: "west"}, Alert{Kind: AlertFire, At: [2]float64{-40.5, y}, Text: "west two"},
			Alert{Kind: AlertFire, At: [2]float64{-41, y}, Text: "west three"}, Alert{Kind: AlertFire, At: [2]float64{-41.5, y}, Text: "west four"})
		f.State.Objectives = []view.ObjectiveView{{ID: "war-horn", Text: "Scale the porters with an autoscaler so that they grow and shrink with the load", Purpose: "Read about autoscalers in the manual: a scaler measures the porters' CPU use and calls more", Progress: "Read about autoscalers in the manual: 1579/5669 MiB, 30/30 points"}}
		arrows := h.Arrows(f)
		if len(arrows) == 0 {
			continue
		}
		h.ToastList = nil
		for i := range 8 {
			h.Toast(fmt.Sprintf("A War Horn keeps the %d porters now: it calls more when they eat past 70%% of their food pledge and sends them home when the load eases.", 15+i), false)
		}
		for _, r := range h.ToastRects(f) {
			for _, a := range arrows {
				if Overlaps(r, h.ArrowSpan(a)) {
					t.Fatalf("a toast at %+v covers the alert arrow at %+v", r, h.ArrowSpan(a))
				}
			}
		}
	}
}
