package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestChipSizesToItsText(t *testing.T) {
	r := ChipRect(Objective, "kubectl describe node lone-tower")
	label := ChipLabel("kubectl describe node lone-tower")
	if label != "kubectl describe node lone-tower" || ui.MaxChars(r.W-20, 1) < len(label) {
		t.Errorf("chip %+v cannot hold %q", r, label)
	}
	if r.X+r.W != Objective.X+Objective.W-6 {
		t.Error("the chip is right-aligned in the card")
	}
	if long := ChipRect(Objective, strings.Repeat("x", 80)); long.W != chipMaxW {
		t.Errorf("long chip = %d", long.W)
	}
	if ChipLabel("get nodes") != "kubectl get nodes" {
		t.Error("chip label adds kubectl once")
	}
}

func TestSpeakItChip(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.State.Objectives[0].Command = "get nodes"
	if r := h.ObjectiveRectOf(f); ChipRect(r, "get nodes").Y+ChipH > r.Y+r.H {
		t.Errorf("the chip falls out of the card: %+v", r)
	}
	clickOn(t, h, f, "obj:speak")
	if !h.TextFocus() || string(h.Stone.Input) != "get nodes" {
		t.Fatalf("chip opens the Stone pre-filled: %q", string(h.Stone.Input))
	}
	h.OverlayID = OverlayNone
	h.Stone.SetInput("")
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f)
	if string(h.Stone.Input) != "get nodes" {
		t.Errorf("enter pre-fills too: %q", string(h.Stone.Input))
	}
	if len(h.Stone.History) != 0 {
		t.Error("the command never runs by itself")
	}
	full := h.ObjectiveRectOf(f).H
	quiet := f
	quiet.State.Objectives = slices.Clone(f.State.Objectives)
	quiet.State.Objectives[0].Command = ""
	quiet.Clock += 2 * CounterTick
	h.Update(Input{MX: -10, MY: -10}, quiet)
	if len(h.ObjectiveButtons(quiet)) != 0 {
		t.Error("no command: no chip a second later")
	}
	quiet.State.Objectives[0].ID = "another-step"
	if h.ObjectiveRectOf(quiet).H >= full {
		t.Error("a step with no command: a shorter card")
	}
}

func TestScriptoriumApply(t *testing.T) {
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	f := fullFrame(t)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}, Shift: true}, f)
	if h.Overlay() != OverlayScriptorium || !h.TextFocus() {
		t.Fatalf("shift+enter opens the Scriptorium, got %q", h.Overlay())
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}, Ctrl: true}, f)
	if !h.Script.Pending || len(runner.applied) != 0 {
		t.Fatal("ctrl+enter marks the apply for the next frame")
	}
	out := h.Update(Input{}, f)
	if _, ok := hasIntent(out, IntentKubectl); !ok || !h.Script.OK || len(runner.applied) != 1 {
		t.Fatalf("apply result %+v", h.Script)
	}
	h.Script.Editor.InsertText("broken")
	clickOn(t, h, f, "scr:apply")
	h.Update(Input{}, f)
	if h.Script.OK || h.Script.ErrLine != 3 {
		t.Errorf("error line = %d ok %v", h.Script.ErrLine, h.Script.OK)
	}
}

func TestScriptoriumDrafts(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	h.Open(OverlayScriptorium)
	h.Script.Editor.InsertText("draft")
	clickOn(t, h, f, "scr:tpl:SCROLL")
	if h.Script.Confirm != TemplateScroll {
		t.Fatal("a dirty draft asks before replacing")
	}
	h.Update(Input{Chars: []rune("n")}, f)
	if h.Script.Confirm != "" || h.Script.Template != TemplateCrew {
		t.Error("N keeps the draft")
	}
	clickOn(t, h, f, "scr:tpl:HEARTH")
	clickOn(t, h, f, "scr:yes")
	if h.Script.Template != TemplateHearth || !strings.Contains(h.Script.Editor.Text(), "kind: Pod") {
		t.Error("YES loads the template")
	}
	h.Script.Editor.Insert('x')
	clickOn(t, h, f, "scr:tpl:BLANK")
	h.Update(Input{Chars: []rune("y")}, f)
	if !h.Script.Editor.Empty() {
		t.Error("Y loads the blank template")
	}
	clickOn(t, h, f, "scr:apply")
	h.Update(Input{}, f)
	if h.Script.OK {
		t.Error("an empty script is refused")
	}
	h.Script.Editor.Insert('x')
	clickOn(t, h, f, "scr:tpl:CREW")
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyTab, ebiten.KeyEnter, ebiten.KeyArrowUp, ebiten.KeyEscape}, Chars: []rune("a")}, f)
	if h.Overlay() != OverlayNone {
		t.Error("escape closes the Scriptorium")
	}
	if ErrorLine("no line here") != 0 {
		t.Error("ErrorLine without a line")
	}
}

func TestLoomAndReforge(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	h.Open(OverlayAge)
	buttons := h.ageButtons(f)
	if b := buttons[1]; !b.Disabled || !strings.Contains(b.Tip, "Needs 9 threads") {
		t.Errorf("unaffordable row = %+v", b)
	}
	if b := buttons[2]; !b.Disabled || !strings.Contains(b.Tip, "fully woven") {
		t.Errorf("maxed row = %+v", b)
	}
	if it, _ := hasIntent(clickOn(t, h, f, "loom:royal-treasury"), IntentAction); it.Action.Action != "weave" || it.Action.ID != "royal-treasury" {
		t.Errorf("weave = %+v", it)
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyArrowDown}}, f)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyArrowUp}}, f)
	if it, _ := hasIntent(h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f), IntentAction); it.Action.ID != "royal-treasury" {
		t.Errorf("enter weaves the selected row: %+v", it)
	}
	h.loomRow = 1
	if _, ok := hasIntent(h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f), IntentSound); !ok {
		t.Error("enter on an unaffordable row refuses")
	}
}

func TestReforgeModal(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	h.Open(OverlayAge)
	clickOn(t, h, f, "reforge:open")
	if h.Overlay() != OverlayReforge {
		t.Fatal("reset opens the confirm modal")
	}
	if got := h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f); len(got) != 0 || h.Overlay() != OverlayReforge {
		t.Error("enter never confirms a reset")
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
	if h.Overlay() != OverlayAge {
		t.Error("escape means NOT YET")
	}
	clickOn(t, h, f, "reforge:open")
	clickOn(t, h, f, "reforge:cancel")
	clickOn(t, h, f, "reforge:open")
	h.Update(Input{RightClick: true}, f)
	clickOn(t, h, f, "reforge:open")
	if _, ok := hasIntent(clickOn(t, h, f, "reforge:confirm"), IntentReforge); !ok || h.Overlay() != OverlayNone {
		t.Error("confirm emits one reset and closes the modal")
	}
	f.State.Reforge.Patterns[0].Rank = 1
	h.watch(f)
	f.State.Reforge.Patterns[0].Rank = 2
	h.watch(f)
	if h.Juice.flash["royal-treasury"] != h.Now {
		t.Error("a new rank flashes")
	}
	if HandCast(f.Pods, .5) != 1 || HandCast(nil, 0) != 0 {
		t.Errorf("hand-cast = %d", HandCast(f.Pods, .5))
	}
	if pressReforgeNoop := h.pressReforge("other", f); pressReforgeNoop != nil {
		t.Error("unknown reset button")
	}
}

func TestGateWatchAndBerths(t *testing.T) {
	h := New(nil, Backend{})
	h.SetStaging(false)
	f := fullFrame(t)
	crewCarries(&f, false)
	h.Update(Input{}, f)
	crewCarries(&f, true)
	h.Update(Input{}, f)
	if len(h.Toasts()) != 1 || h.Toasts()[0].Text != CrewTakesToast {
		t.Fatalf("takeover toast = %+v", h.Toasts())
	}
	if _, ok := HitID(h.CommandButtons(f), "cmd:serve"); ok {
		t.Error("SERVE is retired: the boxes stay clickable on the quay")
	}
	crewCarries(&f, false)
	h.Update(Input{}, f)
	f.Clock += NoticeGap
	f.State.RealmTime += GateSettle
	h.Update(Input{}, f)
	if !h.GateReopened() || len(h.Toasts()) != 2 || !h.Toasts()[1].Bad {
		t.Error("reopened gate pulses and warns")
	}
	harbor := f
	harbor.Selection = Selection{Kind: SelHaven, ID: "haven-0"}
	if it, _ := hasIntent(clickOn(t, h, harbor, CloseBerthButton), IntentAction); it.Action.Action != view.ActCloseBerth {
		t.Errorf("close a berth = %+v", it)
	}
	harbor.State.Berths.Open = 0
	if b := CloseBerth(harbor); !b.Disabled || b.Tip != noBerthTip {
		t.Errorf("close with no berth open = %+v", b)
	}
	h.Open(OverlayMenu)
	if _, ok := hasIntent(clickOn(t, h, f, "menu:reduce"), IntentReduce); !ok {
		t.Error("menu toggles reduced effects")
	}
	h.Open(OverlayMenu)
	if _, ok := hasIntent(clickOn(t, h, f, "menu:names"), IntentNames); !ok {
		t.Error("menu toggles Kubernetes names")
	}
	for trueNames, want := range map[bool]string{false: "NAMES: STORY", true: "NAMES: KUBERNETES"} {
		f.TrueNames = trueNames
		if b := buttonByID(h.MenuButtons(f), "menu:names"); b.Label != want {
			t.Errorf("names label = %q, want %q", b.Label, want)
		}
	}
}

func buttonByID(buttons []Button, id string) Button {
	for _, b := range buttons {
		if b.ID == id {
			return b
		}
	}
	return Button{}
}

func TestSupplyAndJuice(t *testing.T) {
	f := fullFrame(t)
	f.State.Reveals = append(slices.Clone(f.State.Reveals), "hands", "lore")
	hands, lore := SupplyOf(f.Metrics.Nodes)
	if SupplyText("HANDS", hands, "") != "HANDS 1.3/1.9" || SupplyText("LORE", lore, "GI") != "LORE 1.0/3.0GI" {
		t.Errorf("supply = %q %q", SupplyText("HANDS", hands, ""), SupplyText("LORE", lore, "GI"))
	}
	if SupplyColor(Supply{1, 4}) != ui.Palette.Cream || SupplyColor(Supply{1, 2}) != ui.Palette.Cream || SupplyColor(Supply{3, 2}) != supplyFull || SupplyColor(Supply{1, 1}) != supplyFull {
		t.Error("supply colours: white with room, dark red when full")
	}
	mid := SupplyColor(Supply{3, 4})
	if mid == ui.Palette.Cream || mid == supplyFull || mid.R < mid.G {
		t.Errorf("three quarters full is not on the way to red: %v", mid)
	}
	if !strings.HasSuffix(SupplyText("X", Supply{3, 2}, ""), "!") || (Supply{}).Ratio() != 0 {
		t.Error("overcommitted supply is marked")
	}
	for age, want := range map[float64]bool{-1: false, .1: true, 1: true, 2.9: true, 4: false} {
		if _, ok := BannerOffset(age); ok != want {
			t.Errorf("banner at %v = %v", age, ok)
		}
	}
	if off, _ := BannerOffset(1); off != 0 {
		t.Error("banner holds in place")
	}
	if VignetteAlpha(0) != vignetteAlpha || VignetteAlpha(1) != 0 || VignetteAlpha(-1) != 0 {
		t.Error("vignette timing")
	}
	if pulse(0, true) != 1 || pulse(.25, false) != 1 {
		t.Error("pulse")
	}
	if Hands(1300) != "1.30H" || !strings.HasPrefix(LaborLine(view.PodMetrics{CPUMilli: 100}), "FOOD 100") {
		t.Errorf("labor = %q", LaborLine(view.PodMetrics{CPUMilli: 100}))
	}
	h := New(nil, Backend{})
	h.Update(Input{}, f)
	for _, b := range h.StatButtons(f) {
		if b.ID == "info:"+SupplyCount(Rations(hands)) && (!strings.Contains(b.Tip, supplyTip) || !strings.Contains(b.Tip, "1,300 of 1,900 rations")) {
			t.Errorf("supply tip = %q", b.Tip)
		}
	}
	if got := clickOn(t, h, f, "info:"+SupplyCount(Rations(hands))); got != nil {
		t.Error("stats are not clickable")
	}
}
