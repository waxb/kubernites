package hud

import (
	"slices"
	"sort"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const minCloseHit = CloseFace + 2*ClosePad

func overlayIDs() []string {
	out := make([]string, 0, len(Overlays))
	for id := range Overlays {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func closeAffordance(buttons []Button) (Button, bool) {
	for _, b := range buttons {
		if IsClose(b) {
			return b, true
		}
	}
	return Button{}, false
}

func checkClose(t *testing.T, name string, window ui.Rect, own []Button) Button {
	t.Helper()
	x, ok := closeAffordance(own)
	switch {
	case !ok:
		t.Errorf("%s has no close button", name)
	case x.Rect.W < minCloseHit || x.Rect.H < minCloseHit:
		t.Errorf("%s close hit %dx%d, want at least %dx%d", name, x.Rect.W, x.Rect.H, minCloseHit, minCloseHit)
	case x.Face.W < CloseFace || x.Face.H < CloseFace || !contains(x.Rect, x.Face):
		t.Errorf("%s close face %+v inside hit %+v", name, x.Face, x.Rect)
	case !contains(window, x.Rect):
		t.Errorf("%s close %+v sticks out of the window %+v", name, x.Rect, window)
	}
	for _, b := range own {
		if b.ID != x.ID && b.ID != "overlay:body" && Overlaps(b.Rect, x.Rect) {
			t.Errorf("%s: %s %+v overlaps the close button %+v", name, b.ID, b.Rect, x.Rect)
		}
	}
	return x
}

func contains(outer, inner ui.Rect) bool {
	return inner.X >= outer.X && inner.Y >= outer.Y && inner.X+inner.W <= outer.X+outer.W && inner.Y+inner.H <= outer.Y+outer.H
}

func openedHUD(t *testing.T, overlay string) (*HUD, Frame) {
	t.Helper()
	f := fullFrame(t)
	f.Selection = Selection{Kind: "hearth", ID: "default/porter-a", Pod: fixturePods()[0]}
	h := New(nil, (&fakeRunner{}).backend())
	h.OpenDetails(f.Selection, DetailDescribe)
	h.Open(overlay)
	return h, f
}

func TestEveryOverlayHasALargeCloseButton(t *testing.T) {
	for _, id := range overlayIDs() {
		h, f := openedHUD(t, id)
		checkClose(t, id, h.OverlayRect(), h.OverlayButtons(f))
	}
}

func TestEveryCloseButtonCloses(t *testing.T) {
	for _, id := range overlayIDs() {
		h, f := openedHUD(t, id)
		x, _ := closeAffordance(h.OverlayButtons(f))
		h.Update(Input{MX: x.Rect.X + 1, MY: x.Rect.Y + x.Rect.H - 2, Click: true}, f)
		if h.Overlay() == id {
			t.Errorf("clicking the edge of %s's close button left it open", id)
		}
	}
}

func outsidePoint(t *testing.T, h *HUD, f Frame) (int, int) {
	t.Helper()
	for _, p := range [][2]int{{480, ConsoleY - 3}, {3, 200}, {Width - 3, 200}, {3, ConsoleY - 3}, {60, TopH + 4}} {
		if h.OverlayRect().Contains(p[0], p[1]) {
			continue
		}
		if _, hit := HitButton(h.Buttons(f), p[0], p[1]); !hit {
			return p[0], p[1]
		}
	}
	t.Fatalf("no empty spot outside %s", h.Overlay())
	return 0, 0
}

func TestClickingOutsideClosesNonModalWindows(t *testing.T) {
	for _, id := range overlayIDs() {
		h, f := openedHUD(t, id)
		x, y := outsidePoint(t, h, f)
		h.Update(Input{MX: x, MY: y, Click: true}, f)
		modal := slices.Contains(ModalOverlays, id)
		switch {
		case modal && h.Overlay() != id:
			t.Errorf("a click outside closed the modal %s", id)
		case !modal && h.Overlay() == id:
			t.Errorf("a click outside left %s open", id)
		case !modal && !h.Consumed():
			t.Errorf("the click that closed %s fell through to the world", id)
		}
	}
}

func TestClickingInsideKeepsTheWindowOpen(t *testing.T) {
	for _, id := range overlayIDs() {
		h, f := openedHUD(t, id)
		r := h.OverlayRect()
		for _, p := range [][2]int{{r.X + 3, r.Y + r.H - 3}, {r.X + r.W/2, r.Y + r.H - 3}} {
			if _, hit := HitButton(h.OverlayButtons(f), p[0], p[1]); hit {
				continue
			}
			h.Update(Input{MX: p[0], MY: p[1], Click: true}, f)
			if h.Overlay() == "" {
				t.Errorf("a click inside %s at %v closed it", id, p)
			}
		}
	}
}

func lessonHUD(t *testing.T) (*HUD, Frame) {
	t.Helper()
	f := fullFrame(t)
	h := New(nil, Backend{})
	h.Update(Input{MX: -5, MY: -5}, f)
	h.Lesson("A Keep", "A Keep runs huts for you: a Node.")
	return h, f
}

func TestTheLessonCardClosesByItsButtonsAndNotByAClickElsewhere(t *testing.T) {
	h, f := lessonHUD(t)
	checkClose(t, "info card", h.LessonRect(f), h.LessonButtons(f))
	x, _ := closeAffordance(h.LessonButtons(f))
	h.Update(Input{MX: x.Rect.X + x.Rect.W - 2, MY: x.Rect.Y + 1, Click: true}, f)
	if h.LessonTitle() != "" {
		t.Fatal("the info card ignored its close button")
	}
	h, f = lessonHUD(t)
	if !h.Captures(h.LessonRect(f).X+5, h.LessonRect(f).Y+5, f) {
		t.Fatal("clicks on the info card fall through to the world")
	}
	h.Update(Input{MX: 600, MY: 250, Click: true}, f)
	if h.LessonTitle() == "" {
		t.Fatal("a click elsewhere closed the info card before the player read it")
	}
	if h.Consumed() {
		t.Fatal("the info card swallowed a click meant for the world")
	}
	clickOn(t, h, f, LessonAck)
	if h.LessonTitle() != "" {
		t.Fatal("GOT IT left the info card open")
	}
}

func TestTheSelectionCardHasACloseButton(t *testing.T) {
	f := fullFrame(t)
	f.Selection = Selection{Kind: "hearth", ID: "default/porter-a", Pod: fixturePods()[0]}
	h := New(nil, Backend{})
	own := h.SelectionButtons(f)
	checkClose(t, "selection card", CardBox, own)
	if _, ok := hasIntent(clickOn(t, h, f, "sel:close"), IntentSelect); !ok {
		t.Fatal("the selection card's X does not clear the selection")
	}
	f.Selection = Selection{Kind: "structure", ID: "keep-1", Pod: model.PodView{}}
	checkClose(t, "structure card", CardBox, h.SelectionButtons(f))
}

func introFrame(t *testing.T, in Intro) Frame {
	t.Helper()
	f := stagedFrame(t, "cmd:serve", "cmd:build", MeterGold)
	f.Guide.Intro = &in
	return f
}

func TestIntroButtonsSitOnTheirFaces(t *testing.T) {
	h := New(nil, Backend{})
	f := introFrame(t, Intro{ID: "gold", Title: "Gold", Purpose: "Your balance.", Try: "Look at it.", Target: MeterGold, Via: ViaAck})
	card := h.IntroCard(f)
	for _, b := range h.IntroButtons(f) {
		if b.Rect != b.Face || !contains(card, b.Rect) {
			t.Fatalf("%s hit %+v face %+v card %+v", b.ID, b.Rect, b.Face, card)
		}
		if b.Tip != "" || b.Page != "" {
			t.Errorf("%s raises a window over the card: %q", b.ID, b.Tip)
		}
	}
}

func TestIntroCardsStayUntilTried(t *testing.T) {
	h := New(nil, Backend{})
	f := introFrame(t, Intro{ID: "gold", Title: "Gold", Purpose: "Your balance.", Try: "Look at it.", Target: MeterGold, Via: ViaAck})
	card := h.IntroCard(f)
	h.Update(Input{MX: card.X + card.W + 40, MY: card.Y + card.H + 40, Click: true}, f)
	if _, closable := closeAffordance(h.IntroButtons(f)); closable {
		t.Fatal("an intro card must be tried or acknowledged, not closed")
	}
	if !h.IntroShown(f) {
		t.Fatal("a click elsewhere dismissed the intro")
	}
}

func offscreenIntro(t *testing.T) (*HUD, Frame) {
	t.Helper()
	f := introFrame(t, Intro{ID: "clerks", Title: "A new trade: clerks", Purpose: "Ledgers count twice.", Try: "Give the clerks a depot.", Target: "stallspot:ledger", Via: ViaAck})
	f.GoalAt = &[2]float64{40, 8}
	f.Beacons = []ui.Rect{{X: 1100, Y: -60, W: 30, H: 50}}
	return New(nil, Backend{}), f
}

func TestAnIntroPointingOffscreenPointsAtAnEdgeBelowTheTopBarAndCanShowIt(t *testing.T) {
	h, f := offscreenIntro(t)
	hole, ok := h.IntroHole(f)
	view := ui.Rect{X: 0, Y: TopH, W: Width, H: ConsoleY - TopH}
	if !ok || hole.X < view.X || hole.Y < view.Y || hole.X+hole.W > view.X+view.W || hole.Y+hole.H > view.Y+view.H {
		t.Fatalf("the pointer %+v (%v) leaves the visible map %+v", hole, ok, view)
	}
	if card := h.IntroCard(f); Overlaps(card, TopBar) {
		t.Errorf("the intro card %+v covers the top bar", card)
	}
	show, ok := HitID(h.IntroButtons(f), IntroShow)
	if !ok || show.Label != "SHOW" {
		t.Fatalf("an offscreen target offers SHOW: %+v", h.IntroButtons(f))
	}
	intents := h.PressIntro("show", f)
	if it, found := hasIntent(intents, IntentFocus); !found || it.X != 40 || it.Y != 8 {
		t.Errorf("SHOW glides to the target: %+v", intents)
	}
	f.Beacons = []ui.Rect{{X: 400, Y: 200, W: 30, H: 50}}
	if _, ok := HitID(h.IntroButtons(f), IntroShow); ok {
		t.Error("a visible target needs no SHOW")
	}
}
