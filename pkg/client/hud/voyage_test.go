package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func voyageFrame(t *testing.T) Frame {
	t.Helper()
	f := keepsFrame(t)
	site := f.Site
	n := site.Width * site.Height
	site.Region = make([]uint8, n)
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			i := y*site.Width + x
			switch {
			case x < site.Width/2:
				site.Region[i] = 1
			case y < 10:
				site.Region[i] = 2
			}
		}
	}
	site.Islands = []view.Island{
		{ID: "home", Name: "Hearthmoor", Zone: "isle-hearthmoor", Owned: true, Center: view.Point{X: 12, Y: 20}},
		{ID: "north", Name: "The Very Windswept and Unreasonably Long-Named Isle of Brannoch", Zone: "isle-brannoch", Center: view.Point{X: 36, Y: 4}},
		{ID: "south", Name: "Sallow", Zone: "isle-sallow", Center: view.Point{X: 40, Y: 40}},
	}
	f.Site = site
	f.State.Unlocks = map[string]bool{"charters": true}
	return f
}

func TestVoyageRowsDescribeEachIsle(t *testing.T) {
	f := voyageFrame(t)
	rows := VoyageRows(f)
	if len(rows) != 3 {
		t.Fatalf("rows = %+v", rows)
	}
	home, north := rows[0], rows[1]
	if !home.Owned || home.Keeps != 1 {
		t.Errorf("home = %+v", home)
	}
	if north.Owned || north.Keeps != 0 {
		t.Errorf("north = %+v", north)
	}
	for _, r := range rows {
		for _, l := range voyageLines(r, charterTermsOf(f)) {
			if strings.Contains(l.Text, "ley") {
				t.Errorf("the retired ley power still shows: %q", l.Text)
			}
		}
	}
	for _, r := range rows {
		for _, l := range voyageLines(r, charterTermsOf(f)) {
			if w, _ := (ui.Font{}).Measure(l.Text, 1); w > voyageTextW {
				t.Errorf("line %q is %dpx, room %d", l.Text, w, voyageTextW)
			}
		}
	}
}

func TestVoyageButtonsLookAndCharter(t *testing.T) {
	h := New(nil, Backend{})
	f := voyageFrame(t)
	h.Open(OverlayVoyage)
	ids := []string{}
	for _, b := range h.Buttons(f) {
		if strings.HasPrefix(b.ID, "voyage:") {
			ids = append(ids, b.ID)
			if b.Tip == "" {
				t.Errorf("%s has no tooltip", b.ID)
			}
			if w, _ := (ui.Font{}).Measure(b.Label, 1); w > b.Rect.W-4 {
				t.Errorf("%s label %q overflows", b.ID, b.Label)
			}
		}
	}
	joined := strings.Join(ids, " ")
	for _, want := range []string{"voyage:look:home", "voyage:look:north", "voyage:charter:north", "voyage:charter:south"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %s in %s", want, joined)
		}
	}
	if strings.Contains(joined, "voyage:charter:home") {
		t.Error("an owned isle is not chartered again")
	}
	it, ok := hasIntent(clickOn(t, h, f, "voyage:look:north"), IntentFocus)
	if !ok || it.X != 36 || it.Y != 4 || h.Overlay() != OverlayNone {
		t.Errorf("look = %+v %v overlay %q", it, ok, h.Overlay())
	}
	h.Open(OverlayVoyage)
	act, ok := hasIntent(h.pressVoyage("charter:south", f), IntentAction)
	if !ok || act.Action.Action != "charterIsland" || act.Action.ID != "south" {
		t.Errorf("charter = %+v", act)
	}
	for _, b := range voyageButtons(withCharter(f, false), 0) {
		if b.ID == "voyage:charter:south" && (!b.Disabled || !strings.Contains(b.Tip, "Learn")) {
			t.Errorf("a blocked charter explains itself: %+v", b)
		}
	}
	for _, b := range voyageButtons(withCharter(f, true), 0) {
		if b.ID == "voyage:charter:south" && (b.Disabled || b.Label != "CHARTER 2K GOLD") {
			t.Errorf("an open charter is offered with its size: %+v", b)
		}
	}

	if h.pressVoyage("look:nowhere", f) != nil || h.pressVoyage("bogus", f) != nil {
		t.Error("unknown isles do nothing")
	}
}

func TestVoyageOpensFromTheBuildPageAndKey(t *testing.T) {
	h := New(nil, Backend{})
	f := voyageFrame(t)
	if ids := strings.Join(buildIDs(h, f), " "); !strings.Contains(ids, "cmd:voyage") {
		t.Fatalf("build page lacks the voyage: %s", ids)
	}
	clickOn(t, h, f, "cmd:voyage")
	if h.Overlay() != OverlayVoyage {
		t.Error("the voyage cell opens the panel")
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyV}}, f)
	if h.Overlay() != OverlayNone {
		t.Error("V toggles the panel closed")
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyV}}, f)
	if h.Overlay() != OverlayVoyage {
		t.Error("V opens the panel")
	}
	lone := keepsFrame(t)
	lone.Site.Islands = lone.Site.Islands[:min(len(lone.Site.Islands), 1)]
	h = New(nil, Backend{})
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyV}}, lone)
	if h.Overlay() == OverlayVoyage || strings.Contains(strings.Join(buildIDs(h, lone), " "), "cmd:voyage") {
		t.Error("no voyage without other isles")
	}
}

func TestVoyageScrollsAndDraws(t *testing.T) {
	f := voyageFrame(t)
	for i := range 12 {
		f.Site.Islands = append(f.Site.Islands, view.Island{ID: "extra" + string(rune('a'+i)), Name: "Extra", Zone: "isle-extra"})
	}
	h := New(nil, Backend{})
	h.Open(OverlayVoyage)
	h.Update(Input{Wheel: -3, MX: Overlay.X + 20, MY: Overlay.Y + 60}, f)
	if h.voyageList == 0 {
		t.Error("the wheel scrolls the isles")
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	h.Draw(dst, withCharter(f, true))
}

func TestRetireAKeepTakesTwoClicks(t *testing.T) {
	h := New(nil, Backend{})
	f := keepsFrame(t)
	stone := view.Structure{ID: "keep-2", Kind: view.StructKeep, Variant: "great", X: 30, Y: 30, W: 3, H: 3, State: view.StateReady, Label: "Great Keep 1",
		Object: view.ObjectRef{Kind: "Node", Name: "great-keep-1"}}
	f.Selection = Selection{Kind: "structure", ID: stone.ID, Structure: stone}
	out := clickOn(t, h, f, "keep:retire")
	if _, ok := hasIntent(out, IntentAction); ok {
		t.Fatal("the first click only arms the retirement")
	}
	var armed Button
	for _, b := range h.Buttons(f) {
		if b.ID == "keep:retire" {
			armed = b
		}
	}
	if armed.Label != "SURE?" || !strings.Contains(armed.Tip, "Great Keep 1") {
		t.Errorf("armed = %+v", armed)
	}
	act, ok := hasIntent(clickOn(t, h, f, "keep:retire"), IntentAction)
	if !ok || act.Action.Action != "retireKeep" || act.Action.ID != "keep-2" {
		t.Errorf("retire = %+v", act)
	}
	clickOn(t, h, f, "keep:retire")
	f.Clock += retireWindow + 1
	if _, ok := hasIntent(clickOn(t, h, f, "keep:retire"), IntentAction); ok {
		t.Error("a stale arming expires")
	}
	tower := f
	tower.Selection.Structure.Variant = "tower"
	for _, b := range h.Buttons(tower) {
		if b.ID == "keep:retire" {
			t.Error("the Lone Tower holds the control plane and is never retired")
		}
	}
	early := f
	early.State.Reveals = allReveals
	for _, b := range h.Buttons(early) {
		if b.ID == "keep:retire" {
			t.Error("retiring waits for the keeps reveal")
		}
	}
}

func withCharter(f Frame, open bool) Frame {
	next := f
	next.State.Charter = view.CharterView{Cost: 2000, Available: open}
	if !open {
		next.State.Charter.Reason = "Learn about zones in the manual"
	}
	return next
}
