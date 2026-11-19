package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestFormatting(t *testing.T) {
	cases := map[float64]string{0: "0", 7: "7", 7.25: "7.2", 150.4: "150", 1260: "1.26K", 2e6: "2M", 3.5e9: "3.5B", -1500: "-1.5K"}
	for v, want := range cases {
		if got := Compact(v); got != want {
			t.Errorf("Compact(%v) = %q, want %q", v, got, want)
		}
	}
	if got := Clock(0, .5); got != "DAY 1 12:00" {
		t.Errorf("Clock = %q", got)
	}
	if got := Clock(view.DefaultConfig().DayPeriod, .25); got != "DAY 2 06:00" {
		t.Errorf("Clock day 2 = %q", got)
	}
	if Nines(.999) != "99.9%" {
		t.Errorf("Nines = %q", Nines(.999))
	}
	for image, want := range map[string]string{"porter:1.1": "1.1", "porter": "latest", "reg:5000/porter": "latest"} {
		if got := ImageTag(image); got != want {
			t.Errorf("ImageTag(%q) = %q", image, got)
		}
	}
	if costText(5, 0) != ui.IconGold+"5" || costText(0, 3) != ui.IconWisdom+"3" || costText(5, 3) != ui.IconGold+"5 "+ui.IconWisdom+"3" {
		t.Error("costText mismatch")
	}
	ui.SetNamer(names.Kubernetes(64))
	defer ui.SetNamer(nil)
	if got := costText(5, 3); got != "$5 3 KN" {
		t.Errorf("costText in KUBERNETES mode = %q, want the plain amounts", got)
	}
	if romanOf(1) != "II" || romanOf(9) != "10" {
		t.Error("romanOf mismatch")
	}
}

func TestDetailCommands(t *testing.T) {
	pod := Selection{Kind: "hearth", ID: "default/a", Pod: model.PodView{Namespace: "default", Name: "a", Restarts: 1}}
	if got := DetailCommands(pod, DetailLogs); len(got) != 2 || !strings.HasSuffix(got[1], "--previous") {
		t.Errorf("logs with restarts = %v", got)
	}
	if got := DetailCommands(pod, DetailYAML); got[0] != "get pod a -n default -o yaml" {
		t.Errorf("yaml = %v", got)
	}
	node := Selection{Kind: "structure", Structure: view.Structure{Label: "Lone Tower", Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}}}
	if got := DetailCommands(node, DetailDescribe); got[0] != "describe node lone-tower" {
		t.Errorf("describe node = %v", got)
	}
	if DetailCommands(node, DetailLogs) != nil || HasLogs(node) {
		t.Error("nodes have no logs")
	}
	gate := Selection{Kind: "structure", Structure: view.Structure{Kind: view.StructGate}}
	if DetailCommands(gate, DetailDescribe) != nil || DetailCommands(Selection{}, DetailDescribe) != nil {
		t.Error("gate and empty selections have no kubectl target")
	}
	if SelectionTitle(node) != "Lone Tower" || SelectionTitle(Selection{}) != "" || !strings.Contains(SelectionTitle(pod), "a") {
		t.Error("SelectionTitle mismatch")
	}
	if got := SplitOutput("a\tb\n\n"); got[0] != "a    b" {
		t.Errorf("SplitOutput = %q", got)
	}
	if got := SplitOutput(""); got[0] != "(no output)" {
		t.Errorf("empty output = %q", got)
	}
}

func TestOpenDetailsRunsKubectl(t *testing.T) {
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	f := fullFrame(t)
	sel := Selection{Kind: "walker", ID: "default/porter-a", Pod: f.Pods[0]}
	h.OpenDetails(sel, DetailLogs)
	if h.Overlay() != OverlayDetails || len(runner.commands) != 2 {
		t.Fatalf("overlay %q, commands %v", h.Overlay(), runner.commands)
	}
	clickOn(t, h, f, "det:describe")
	if h.Details().Tab != DetailDescribe || !strings.HasPrefix(runner.commands[2], "describe pod porter-a") {
		t.Errorf("describe tab: %v", runner.commands)
	}
	failing := Selection{Kind: "structure", Structure: view.Structure{Object: view.ObjectRef{Kind: "Node", Name: "fail"}}}
	h.OpenDetails(failing, DetailDescribe)
	if h.Details().OK || !strings.Contains(strings.Join(h.Details().Lines, "\n"), "Chancery") {
		t.Errorf("refused output should explain the Chancery: %v", h.Details().Lines)
	}
	h.OpenDetails(Selection{Kind: "structure", Structure: view.Structure{Kind: view.StructGate}}, DetailDescribe)
	if !strings.Contains(strings.Join(h.Details().Lines, " "), "boxes") {
		t.Errorf("gate lore missing: %v", h.Details().Lines)
	}
	h.Update(Input{Wheel: -2}, f)
	clickOn(t, h, f, "overlay:close")
	if h.Overlay() != OverlayNone {
		t.Error("close button should close the overlay")
	}
	quiet := New(nil, Backend{})
	quiet.OpenDetails(sel, DetailDescribe)
	if quiet.Details().OK {
		t.Error("no runner should not be OK")
	}
}

func TestCommandCardPages(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	clickOn(t, h, f, "cmd:build")
	if h.Page() != PageBuild {
		t.Fatalf("page = %q", h.Page())
	}
	if it, ok := hasIntent(clickOn(t, h, f, "build:keep"), IntentPlace); !ok || it.Target != "keep" {
		t.Errorf("build:keep intent = %+v", it)
	}
	clickOn(t, h, f, "build:codex")
	if len(h.Toasts()) == 0 || !h.Toasts()[0].Bad {
		t.Error("disabled button should toast its reason")
	}
	h.Update(Input{RightClick: true}, f)
	if h.Page() != PageMain {
		t.Error("right click backs out to main")
	}
	clickOn(t, h, f, "cmd:summon")
	if it, ok := hasIntent(clickOn(t, h, f, "summon:porter:1.0"), IntentAction); !ok || it.Action.Soulstone != "porter:1.0" {
		t.Errorf("summon intent = %+v", it)
	}
	clickOn(t, h, f, "cmd:back")
	if _, ok := HitID(h.Buttons(f), "cmd:serve"); ok {
		t.Error("SERVE is retired: the boxes on the quay are clicked instead")
	}
}

func TestTopBarAndMenu(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	if it, _ := hasIntent(clickOn(t, h, f, "speed:5"), IntentAction); it.Action.Speed != 5 {
		t.Errorf("speed = %+v", it)
	}
	if _, ok := hasIntent(clickOn(t, h, f, "top:pause"), IntentPause); !ok {
		t.Error("pause intent missing")
	}
	clickOn(t, h, f, "top:menu")
	if h.Overlay() != OverlayMenu {
		t.Fatal("menu should open")
	}
	for id, kind := range map[string]string{"menu:save": IntentSave, "menu:mute": IntentMute, "menu:title": IntentTitle} {
		h.OverlayID = OverlayMenu
		if _, ok := hasIntent(clickOn(t, h, f, id), kind); !ok {
			t.Errorf("%s should emit %s", id, kind)
		}
	}
	if pressSpeed(h, "fast", f) != nil {
		t.Error("bad speed should be ignored")
	}
}

func TestKeysAndOverlays(t *testing.T) {
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	f := fullFrame(t)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyB}}, f)
	if h.Page() != PageBuild {
		t.Error("B opens the build page")
	}
	if _, ok := hasIntent(h.Update(Input{Keys: []ebiten.Key{ebiten.KeySpace}}, f), IntentPause); !ok {
		t.Error("space pauses")
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f)
	if !h.TextFocus() {
		t.Fatal("enter opens the Speaking Stone")
	}
	h.Update(Input{Chars: []rune("get pods")}, f)
	out := h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f)
	if _, ok := hasIntent(out, IntentKubectl); !ok || runner.commands[0] != "get pods" {
		t.Errorf("stone run: %v %v", out, runner.commands)
	}
	h.Update(Input{Chars: []rune("fail")}, f)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyPageUp, ebiten.KeyPageDown}}, f)
	h.Update(Input{Wheel: 1}, f)
	h.Stone.Input, h.Stone.Cursor = nil, 0
	h.Update(Input{Chars: []rune("clear")}, f)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f)
	if len(h.Stone.Output) != 0 {
		t.Error("clear empties the stone")
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
	if h.TextFocus() {
		t.Error("escape leaves the stone")
	}
	h.PageID = PageMain
	for _, id := range []string{"cmd:grimoire", "cmd:soulvault", "cmd:chronicles", "cmd:age"} {
		clickOn(t, h, f, id)
		if h.Overlay() == OverlayNone {
			t.Errorf("%s should open an overlay", id)
		}
		h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
	}
}

func TestOverlayActions(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	clickOn(t, h, f, "cmd:grimoire")
	clickOn(t, h, f, "grim:pledges")
	clickOn(t, h, f, "grim:muster")
	if it, _ := hasIntent(clickOn(t, h, f, "learn:muster"), IntentAction); it.Action.Action != "unlock" || it.Action.ID != "muster" {
		t.Errorf("learn = %+v", it)
	}
	h.OverlayID = OverlaySoulvault
	if it, _ := hasIntent(clickOn(t, h, f, "vault:porter:1.1"), IntentAction); it.Action.Image != "porter:1.1" {
		t.Errorf("vault = %+v", it)
	}
	if got := clickOn(t, h, f, "vault:owned"); len(got) != 0 {
		t.Errorf("owned class should do nothing: %v", got)
	}
	h.Update(Input{Wheel: -1, MX: Overlay.X + 5, MY: Overlay.Y + 40}, f)
	h.OverlayID = OverlayChronicles
	if it, _ := hasIntent(clickOn(t, h, f, "chron:c1:a"), IntentAction); it.Action.Answer != "a" {
		t.Errorf("chronicle = %+v", it)
	}
	h.OverlayID = OverlayNone
	harbor := f
	harbor.Selection = Selection{Kind: SelHaven, ID: "haven-0"}
	if it, _ := hasIntent(clickOn(t, h, harbor, BerthButton), IntentAction); it.Action.Action != view.ActOpenBerth {
		t.Errorf("open berth = %+v", it)
	}
	h.OverlayID = OverlayAge
	dark := f
	dark.State.Reforge.Available = false
	clickOn(t, h, dark, "age:advance")
	if len(h.Toasts()) == 0 {
		t.Error("unavailable age should toast")
	}
	h.Update(Input{MX: Overlay.X + 3, MY: Overlay.Y + Overlay.H - 3, Click: true}, f)
	if h.Overlay() != OverlayAge {
		t.Error("clicking the overlay body must not close it")
	}
}

func TestWellTabsSelectionAndMinimap(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	clickOn(t, h, f, "tab:TECH")
	if h.Tab() != TabTech {
		t.Error("tab switch")
	}
	clickOn(t, h, f, "well:tech")
	if h.Overlay() != OverlayGrimoire {
		t.Error("tech well opens the manual")
	}
	h.OverlayID = OverlayNone
	clickOn(t, h, f, "tab:KUBE")
	clickOn(t, h, f, "well:stone")
	if !h.TextFocus() {
		t.Error("kube well opens the stone")
	}
	h.OverlayID = OverlayNone
	clickOn(t, h, f, "tab:EVT")
	h.Update(Input{Wheel: -1, MX: InfoWell.X + 10, MY: InfoWell.Y + 40}, f)
	f.Selection = Selection{Kind: "hearth", ID: "default/porter-a", Pod: f.Pods[0]}
	clickOn(t, h, f, "act:logs")
	if h.Details().Tab != DetailLogs || !strings.HasPrefix(h.Details().Lines[0], "$ kubectl logs porter-a") {
		t.Errorf("the LOGS action opens logs: %+v", h.Details())
	}
	h.OverlayID = OverlayNone
	if _, ok := hasIntent(h.Update(Input{RightClick: true}, f), IntentSelect); !ok {
		t.Error("right click clears the selection")
	}
	f.Placing = "keep"
	if _, ok := hasIntent(h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f), IntentCancel); !ok {
		t.Error("escape cancels placement")
	}
	cx, cy := GridToMinimap(f.Site, 20, 22)
	it, ok := hasIntent(h.Update(Input{MX: int(cx), MY: int(cy), Click: true}, f), IntentJump)
	if !ok || it.X < 18 || it.X > 22 || it.Y < 20 || it.Y > 24 {
		t.Errorf("minimap jump = %+v", it)
	}
	if got := h.MinimapJump(Minimap.X+1, Minimap.Y+1, f); got != nil {
		t.Errorf("outside the island diamond: %v", got)
	}
	if !h.Captures(5, 5, f) || !h.Captures(Objective.X+2, Objective.Y+2, f) || h.Captures(Width/2, 250, f) {
		t.Error("Captures mismatch")
	}
}

func TestMinimapRoundTrip(t *testing.T) {
	site := testSite(t)
	for _, p := range [][2]float64{{0, 0}, {47, 0}, {10, 30}, {47, 47}} {
		x, y := GridToMinimap(site, p[0], p[1])
		gx, gy := MinimapToGrid(site, int(x+.5), int(y+.5))
		if abs(gx-p[0]) > 1.5 || abs(gy-p[1]) > 1.5 {
			t.Errorf("round trip %v -> %.1f,%.1f", p, gx, gy)
		}
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
