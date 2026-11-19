package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func railNotices() []Notice {
	at := [2]float64{12, 30}
	return []Notice{
		{ID: "news:1", Kind: NoticeNews, Icon: "icon/chronicle", Title: "Lone Tower is ready", Detail: "Lone Tower is ready.", Since: 5},
		{ID: "decision:cmd:chronicles", Kind: NoticeDecision, Icon: "icon/chronicle", Title: "An incident report awaits", Detail: "Why?", Count: 2, Intent: "cmd:chronicles", Since: 3},
		{ID: "danger:fire", Kind: NoticeDanger, Icon: "icon/alert", Title: "Huts guttering", Detail: "porter-a is guttering", Count: 3, At: &at, Since: 1},
		{ID: "news:2", Kind: NoticeNews, Icon: "icon/age", Title: "A trade route opens", Since: 9},
		{ID: "decision:cmd:age", Kind: NoticeDecision, Icon: "icon/age", Title: "Open next panel", Intent: "cmd:age", Since: 8},
		{ID: "danger:storm", Kind: NoticeDanger, Icon: "icon/node", Title: "A storm", At: &at, Since: 7},
	}
}

func TestNoticesSortDangerDecisionNewsNewestFirst(t *testing.T) {
	got := SortNotices(railNotices())
	want := []string{"danger:storm", "danger:fire", "decision:cmd:age", "decision:cmd:chronicles", "news:2", "news:1"}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("order = %v, want %v", ids(got), want)
		}
	}
	if railNotices()[0].ID != "news:1" {
		t.Error("sorting mutated its input")
	}
}

func ids(ns []Notice) []string {
	out := []string{}
	for _, n := range ns {
		out = append(out, n.ID)
	}
	return out
}

func railFrame(t *testing.T, n int) Frame {
	f := fullFrame(t)
	all := railNotices()
	for i := len(all); i < n; i++ {
		all = append(all, Notice{ID: "news:x" + string(rune('a'+i)), Kind: NoticeNews, Icon: "icon/chronicle", Title: "news", Since: float64(i)})
	}
	f.Notices = SortNotices(all[:n])
	return f
}

func TestRailShowsSixAndAnOverflow(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := railFrame(t, 9)
	buttons := h.RailButtons(f)
	if len(buttons) != RailVisible+1 {
		t.Fatalf("rail buttons = %d", len(buttons))
	}
	last := buttons[len(buttons)-1]
	if last.ID != "notice:more" || last.Label != "+3" {
		t.Errorf("overflow = %+v", last)
	}
	for i, b := range buttons {
		if b.Rect.X+b.Rect.W > Width || b.Rect.X < Width-60 || b.Rect.Y < TopH || b.Rect.Y+b.Rect.H > ConsoleY {
			t.Errorf("button %d at %+v leaves the right edge of the world view", i, b.Rect)
		}
		for j, c := range buttons[:i] {
			if Overlaps(b.Rect, c.Rect) {
				t.Errorf("buttons %d and %d overlap", i, j)
			}
		}
	}
	if !strings.Contains(buttons[0].Tip, "Click") || !strings.Contains(buttons[0].Tip, "dismiss") {
		t.Errorf("tip = %q", buttons[0].Tip)
	}
	if got := h.RailButtons(railFrame(t, 4)); len(got) != 4 {
		t.Errorf("four notices, %d buttons", len(got))
	}
	h.Open(OverlayGrimoire)
	if len(h.RailButtons(f)) != 0 || h.RailRect(f) != (ui.Rect{}) {
		t.Error("the rail hides behind open overlays")
	}
}

func TestRailNeverOverlapsTheHUD(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := railFrame(t, 9)
	f.State.Siege.Phase = "active"
	h.Lesson("Serve 5 requests by hand.", strings.Repeat("A long detail line that wraps and wraps. ", 6))
	rail := h.RailRect(f)
	if rail.W == 0 {
		t.Fatal("no rail")
	}
	widest := Objective
	widest.H = ConsoleY - Objective.Y
	for name, r := range map[string]ui.Rect{
		"objective": h.ObjectiveRectOf(f), "lesson": h.LessonRect(f), "siege": SiegeStrip, "minimap": Minimap, "command": Command,
		"tidings": InfoWell, "gate card": CardBox, "top bar": TopBar, "console": ConsoleBox, "task column": widest,
	} {
		if Overlaps(rail, r) {
			t.Errorf("rail %+v overlaps the %s %+v", rail, name, r)
		}
	}
	found := false
	for _, b := range h.BlockedRects(f) {
		found = found || b == rail
	}
	if !found {
		t.Error("alert arrows do not step around the rail")
	}
	f.View = [4][2]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	f.Alerts = []Alert{{Kind: AlertFire, At: [2]float64{30, 1}, Text: "far east"}, {Kind: AlertStorm, At: [2]float64{30, 3}, Text: "east"}}
	for _, a := range AlertArrows(f, h.BlockedRects(f)) {
		if Overlaps(ArrowRect(a), rail) {
			t.Errorf("arrow %+v sits on the rail", ArrowRect(a))
		}
	}
	if !h.Captures(rail.X+2, rail.Y+2, f) {
		t.Error("clicks on the rail fall through to the map")
	}
}

func TestClickingNoticesGoesThereOrOpens(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := railFrame(t, 6)
	intents := clickOn(t, h, f, "notice:danger:fire")
	if it, ok := hasIntent(intents, IntentFocus); !ok || it.X != 12 || it.Y != 30 {
		t.Errorf("danger click = %+v", intents)
	}
	if it, ok := hasIntent(intents, IntentNotice); !ok || it.Target != "danger:fire" || it.Command != NoticeOpen {
		t.Errorf("danger click does not report the notice: %+v", intents)
	}
	clickOn(t, h, f, "notice:decision:cmd:chronicles")
	if h.Overlay() != OverlayChronicles {
		t.Fatalf("decision click opened %q", h.Overlay())
	}
	h.Open(OverlayNone)
	h.TabID = TabTech
	intents = clickOn(t, h, f, "notice:news:1")
	if h.TabID != TabEvents {
		t.Error("news without a place opens the tidings")
	}
	if it, ok := hasIntent(intents, IntentNotice); !ok || it.Target != "news:1" {
		t.Errorf("news click = %+v", intents)
	}
	f = railFrame(t, 9)
	h.TabID = TabTech
	clickOn(t, h, f, "notice:more")
	if h.TabID != TabEvents {
		t.Error("the overflow opens the tidings")
	}
	if got := h.PressNotice("missing", f); got != nil {
		t.Errorf("a vanished notice does something: %+v", got)
	}
}

func TestRightClickDismissesANotice(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := railFrame(t, 6)
	f.Selection = Selection{Kind: "structure", ID: "x"}
	var target Button
	for _, b := range h.Buttons(f) {
		if b.ID == "notice:decision:cmd:age" {
			target = b
		}
	}
	intents := h.Update(Input{MX: target.Rect.X + 2, MY: target.Rect.Y + 2, RightClick: true}, f)
	if it, ok := hasIntent(intents, IntentNotice); !ok || it.Command != NoticeDismiss || it.Target != "decision:cmd:age" {
		t.Fatalf("right click = %+v", intents)
	}
	if _, ok := hasIntent(intents, IntentSelect); ok {
		t.Error("dismissing a notice also dropped the selection")
	}
}

func TestCountBadgesSitInsideTheTile(t *testing.T) {
	tile := RailSlot(2)
	for _, digits := range []string{"3", "12", "99"} {
		w, _ := ui.Font{}.Measure(digits, 1)
		b := BadgeRect(tile, w)
		if b.X < tile.X || b.Y < tile.Y || b.X+b.W > tile.X+tile.W || b.Y+b.H > tile.Y+tile.H {
			t.Errorf("badge %q at %+v leaves the tile %+v", digits, b, tile)
		}
		if b.X+b.W != tile.X+tile.W-1 || b.Y+b.H != tile.Y+tile.H-1 {
			t.Errorf("badge %q at %+v is not in the bottom-right corner of %+v", digits, b, tile)
		}
		if b.H < ui.CapHeight+2 || b.W < w+2 {
			t.Errorf("badge %q at %+v squeezes its digits", digits, b)
		}
	}
}

func TestRailDrawsNewNoticesWithAndWithoutMotion(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := railFrame(t, 9)
	for i := range f.Notices {
		f.Notices[i].Since = f.Clock - .1
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	if RailSlide(.1, false) <= 0 || RailSlide(1, false) != 0 || RailSlide(.1, true) != 0 {
		t.Error("slide-in")
	}
	if RailPulse(1, false) <= 0 || RailPulse(RailPulseFor+1, false) != 0 || RailPulse(1, true) != 0 {
		t.Error("pulse")
	}
	for _, reduce := range []bool{false, true} {
		f.Reduce = reduce
		h.Update(Input{MX: Width - 15, MY: TopH + 20}, f)
		h.Draw(dst, f)
	}
	if !strings.HasPrefix(h.Hover.ID, "notice:") {
		t.Fatalf("hovering the rail hovers %q", h.Hover.ID)
	}
	r := h.TooltipRect([]string{"a tooltip line"}, f)
	if r.X+r.W > h.Hover.Rect.X {
		t.Errorf("the rail tooltip %+v covers the rail %+v", r, h.Hover.Rect)
	}
}

func TestTheRailShowsItsFirstMatterWhileItsOwnCardPointsAtIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := railFrame(t, 1)
	f.Guide.Intro = &Intro{ID: "rail", Title: "Matters at hand", Target: SurfaceRail, Via: ViaAck}
	r := h.RailRect(f)
	if r.W == 0 {
		t.Fatal("the rail's own card points at an empty rail")
	}
	if hole, ok := h.IntroHole(f); !ok || !Overlaps(hole, r) {
		t.Fatalf("the rail's card points at %+v, not the rail %+v", hole, r)
	}
	f.Guide.Intro = &Intro{ID: "speed", Title: "Speed", Target: SurfaceSpeed, Via: ViaAck}
	if h.RailRect(f).W != 0 {
		t.Fatal("another card shows the rail beside it")
	}
}

func TestAFreshNoticesPulsePaintsInsideTheRailsRegisteredRect(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := railFrame(t, 2)
	h.Update(Input{MX: -10, MY: -10}, f)
	var rail ui.Rect
	for _, s := range h.Surfaces(f) {
		if s.Key == "window:"+PanelNotices {
			rail = s.Rect
		}
	}
	if rail.W == 0 {
		t.Fatal("the rail is not registered")
	}
	for _, b := range h.RailButtons(f) {
		if ring := b.Rect.Inset(-1); !containsRect(rail, ring) {
			t.Fatalf("a fresh notice's pulse ring %+v paints outside the rail's registered rect %+v", ring, rail)
		}
	}
}

func TestTheWorldsCalloutsKeepClearOfTheRailsWholeRegisteredRect(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := railFrame(t, 2)
	h.Update(Input{MX: -10, MY: -10}, f)
	var rail ui.Rect
	for _, s := range h.Surfaces(f) {
		if s.Key == "window:"+PanelNotices {
			rail = s.Rect
		}
	}
	if rail.W == 0 {
		t.Fatal("the rail is not registered")
	}
	avoid := append(h.AvoidRects(f), h.PanelRects(f)...)
	if !slices.ContainsFunc(avoid, func(r ui.Rect) bool { return containsRect(r, rail) }) {
		t.Fatalf("the world's callouts keep clear of %v, not of the rail's registered rect %+v: a callout ending on its left edge overlaps it", avoid, rail)
	}
}

func TestANoticeOnTurnKeepsItsBoxWhenAnotherNoticeJoinsTheRail(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := railFrame(t, 1)
	h.Director.Stage.Primed = true
	h.Director.Stage.Slot = SlotBoxOf{Key: ActNotice + f.Notices[0].ID}
	box := func(f Frame) (ui.Rect, bool) {
		for _, p := range WindowSurfaces(h, f) {
			if p.Key == ActNotice+f.Notices[0].ID && p.Box {
				return p.Rect, true
			}
		}
		return ui.Rect{}, false
	}
	before, ok := box(f)
	if !ok {
		t.Fatal("the notice on turn is no box: the test proves nothing")
	}
	more := railFrame(t, 2)
	more.Notices = append([]Notice{f.Notices[0]}, more.Notices[1])
	after, ok := box(more)
	if !ok || after.W != before.W || after.H != before.H {
		t.Fatalf("a notice joining the rail resized the notice on turn from %+v to %+v", before, after)
	}
}
