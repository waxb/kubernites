package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func stagedFrame(t *testing.T, introduced ...string) Frame {
	t.Helper()
	f := fullFrame(t)
	f.Guide = Guide{Staged: true, Introduced: introduced}
	return f
}

func buttonIDs(buttons []Button) map[string]bool {
	out := map[string]bool{}
	for _, b := range buttons {
		out[b.ID] = true
	}
	return out
}

func TestAnUnstagedRealmShowsEverythingRevealed(t *testing.T) {
	f := fullFrame(t)
	for _, id := range []string{MeterGold, SurfaceMinimap, SurfaceStone, "cmd:build", "anything"} {
		if !f.Shown(id) {
			t.Errorf("%s hidden without an intro queue (old save)", id)
		}
	}
}

func TestTheStagedHUDShowsOnlyWhatWasIntroduced(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := stagedFrame(t, MeterGold, view.ParcelPile, "cmd:grimoire")
	got := buttonIDs(h.Buttons(f))
	for _, hidden := range []string{"cmd:build", "tab:EVT", "tab:KUBE", "info:" + Compact(f.State.Wisdom)} {
		if got[hidden] {
			t.Errorf("%s shows before its intro", hidden)
		}
	}
	if !got["cmd:grimoire"] || got["cmd:serve"] {
		t.Error("an introduced command shows, and the retired SERVE never does")
	}
	if got[CmdSummonID] || f.Shown(CmdSummonID) {
		t.Error("SUMMON shows before its card teaches it")
	}
	if stats := h.TopStats(f); len(stats) != 1 || stats[0].Meter != MeterGold {
		t.Errorf("top bar = %+v", stats)
	}
	if intents := h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f); h.Overlay() == OverlayStone || len(intents) != 0 {
		t.Error("Enter opens a Stone the player has not met")
	}
	if h.MinimapJump(Minimap.X+40, Minimap.Y+40, f) != nil {
		t.Error("a hidden minimap takes clicks")
	}
	f.Notices = []Notice{{ID: "news:1", Kind: NoticeNews, Title: "x"}}
	if h.RailRect(f).W != 0 {
		t.Error("the rail shows before its intro")
	}
	f.Guide.Intro = &Intro{ID: "build", Title: "Build", Target: "cmd:build"}
	if !buttonIDs(h.Buttons(f))["cmd:build"] || !f.Teaching("cmd:build") {
		t.Error("the surface being taught shows")
	}
}

func TestIntroMomentPointsAtItsTarget(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := stagedFrame(t, MeterGold, "cmd:serve")
	f.Guide.Intro = &Intro{ID: "build", Title: "The BUILD page", Purpose: "Raise buildings that work for you.", Try: "Press BUILD.", Target: "cmd:build"}
	target, ok := h.TargetRect("cmd:build", f)
	if !ok {
		t.Fatal("no rect for the build button")
	}
	card := h.IntroCard(f)
	if Overlaps(card, target) || card.Y+card.H > ConsoleY || card.Y < TopH {
		t.Errorf("card %+v covers the target %+v or leaves the map", card, target)
	}
	if !h.ObjectiveShown(f) || len(h.ObjectiveLines(f)) == 0 {
		t.Error("the task card never vanishes while an intro speaks")
	}
	if Overlaps(card, h.ObjectiveRectOf(f)) {
		t.Errorf("the intro card %+v covers the task card %+v", card, h.ObjectiveRectOf(f))
	}
	if got := buttonIDs(h.Buttons(f)); !got[IntroAck] {
		t.Errorf("a command card offers GOT IT too, so no card can strand the viewer: %v", got)
	}
	bare := f
	bare.Guide.Intro = &Intro{ID: "build", Title: "The BUILD page", Try: "Press BUILD.", Target: "cmd:build"}
	if fresh := New(nil, Backend{}); fresh.IntroCard(f).H <= fresh.IntroCard(bare).H {
		t.Error("the card says what the thing is for without a click")
	}
	h.Update(Input{MX: -10, MY: -10}, f)
	if !h.Captures(card.X+2, card.Y+2, f) {
		t.Error("the card takes its clicks")
	}
	f.Guide.Intro = &Intro{ID: "tidings", Title: "Tidings", Purpose: "What the realm reports.", Try: "Read one.", Target: SurfaceTidings}
	h.Update(Input{MX: -10, MY: -10}, f)
	if h.IntroShown(h.Stage(f)) {
		t.Error("the next intro waits a beat after the last one")
	}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	intents := clickOn(t, h, f, IntroAck)
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != "ack" || it.Action.ID != "tidings" {
		t.Fatalf("GOT IT = %+v", intents)
	}
	if h.PressIntro("ack", fullFrame(t)) != nil {
		t.Error("no intro, nothing to acknowledge")
	}
}

func TestIntroTargetsResolveEverySurface(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.State.Reveals = append(append([]string(nil), f.State.Reveals...), "stone")
	cases := map[string]ui.Rect{SurfaceTidings: InfoWell, SurfaceMinimap: Minimap, SurfaceGate: CardBox, SurfaceRail: RailSlot(0)}
	for target, want := range cases {
		if got, ok := h.TargetRect(target, f); !ok || got != want {
			t.Errorf("%s = %+v %v, want %+v", target, got, ok, want)
		}
	}
	for _, target := range []string{SurfaceStone, SurfaceLedger, MeterGold, "cmd:build"} {
		if r, ok := h.TargetRect(target, f); !ok || r.W == 0 {
			t.Errorf("%s has no rect", target)
		}
	}
	if _, ok := h.TargetRect("surface:nowhere", f); ok {
		t.Error("an unknown target has no rect")
	}
	for _, target := range []ui.Rect{InfoWell, TopBar, RailSlot(0), {}} {
		card := PlaceCard(target, target.W > 0, IntroW, 90)
		if card.X < 0 || card.X+card.W > Width || card.Y < TopH || card.Y+card.H > ConsoleY {
			t.Errorf("card %+v for %+v leaves the map", card, target)
		}
		if target.W > 0 && Overlaps(card, target) {
			t.Errorf("card %+v covers %+v", card, target)
		}
	}
	if !Passive(Intro{Target: SurfaceRail}) || Passive(Intro{Target: "cmd:build"}) {
		t.Error("surfaces are passive, commands are tried")
	}
	if x, y := ClipToRect(10, 10, 100, 10, ui.Rect{X: 0, Y: 0, W: 20, H: 20}); x != 20 || y != 10 {
		t.Errorf("clip = %v,%v", x, y)
	}
}

func TestObjectiveSpeaksWithItsPayoffAndTarget(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.Guide = Guide{Why: "so that requests are served while you build", Target: "build:codex"}
	f.State.Objectives[0].Earn = "serve requests: +1 request each"
	lines := h.ObjectiveLines(f)
	texts := []string{}
	for _, l := range lines {
		texts = append(texts, l.Text)
	}
	joined := strings.Join(texts, "|")
	if !strings.Contains(joined, "so that requests") || strings.Contains(joined, "Press serve") || !strings.Contains(joined, ui.Decorate("+1 request each")) {
		t.Errorf("task lines = %v", texts)
	}
	if BeaconButton(f, PageMain) != "cmd:build" || BeaconButton(f, PageBuild) != "build:codex" {
		t.Error("the beacon walks to the target's page")
	}
	f.Guide.Target = "summon:porter:1.0"
	if BeaconButton(f, PageSummon) != "summon:porter:1.0" || BeaconButton(f, PageMain) != "cmd:summon" {
		t.Error("the summon target goes through the Summon page")
	}
	f.Guide.Target = "pod:default/porter-b"
	if BeaconButton(f, PageMain) != "" || !MapTarget(f.Guide.Target) {
		t.Error("a map target lights no button")
	}
	at := [2]float64{12, 30}
	f.GoalAt = &at
	intents := clickOn(t, h, f, "obj:goal")
	if it, ok := hasIntent(intents, IntentFocus); !ok || it.X != 12 || it.Y != 30 {
		t.Errorf("SHOW ME = %+v", intents)
	}
}

func TestLessonsOpenFromTheirTidingsLine(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.Events = []Tiding{{Kind: "lesson", Text: "What just happened: Scale to 3", Title: "Scale to 3", Detail: "You scaled the crew."}, {Kind: "event", Text: "Started"}}
	clickOn(t, h, f, "tiding:0")
	if h.LessonTitle() != "Scale to 3" {
		t.Fatalf("hint = %q", h.LessonTitle())
	}
	if _, ok := findButton(h.Buttons(f), "tiding:1"); ok {
		t.Error("plain tidings are not buttons")
	}
	if h.PressTiding("9", f) != nil || h.PressTiding("x", f) != nil {
		t.Error("bad tiding index")
	}
	if AutonomyLine(1) != "Your servants did 1 thing for you today" || !strings.Contains(AutonomyLine(12), "12 things") {
		t.Error("autonomy counter text")
	}
	f.AutonomyToday = 3
	if h.TidingTop(f, WellBody()) <= WellBody().Y+3 {
		t.Error("the counter takes the first line")
	}
}

func TestLedgerDrawsOnlyWhatWasIntroduced(t *testing.T) {
	f := stagedFrame(t, MeterGold, "cmd:serve")
	nodes := LedgerNodes(f)
	have := map[string]bool{}
	for _, n := range nodes {
		have[n.ID] = true
	}
	if !have[NodeShips] || !have[NodeGold] || have[NodeWisdom] || have[NodeHands] || have[NodeTier] {
		t.Errorf("staged ledger = %v", have)
	}
	full := fullFrame(t)
	full.State.Unlocks["pledges"] = true
	full.State.Reveals = append(slices.Clone(full.State.Reveals), "hands")
	all := LedgerNodes(full)
	edges := LedgerEdges(all)
	if len(all) != 12 || len(edges) != len(LedgerEdgesOf) {
		t.Errorf("full ledger: %d nodes, %d edges", len(all), len(edges))
	}
	for i, n := range all {
		r := LedgerNodeRect(n)
		if !inside(r, Overlay) {
			t.Errorf("%s %+v leaves the ledger", n.ID, r)
		}
		for _, m := range all[:i] {
			if Overlaps(r, LedgerNodeRect(m)) {
				t.Errorf("%s overlaps %s", n.ID, m.ID)
			}
		}
		for _, l := range n.Lines {
			if w, _ := (ui.Font{}).Measure(l, 1); w > LedgerNodeW-12 {
				t.Errorf("%s line %q is wider than its box", n.ID, l)
			}
		}
	}
	h := New(nil, (&fakeRunner{}).backend())
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyF1}}, full)
	if h.Overlay() != OverlayLedger {
		t.Fatal("F1 opens the ledger")
	}
	clickOn(t, h, full, LedgerButton)
	if h.Overlay() != OverlayNone {
		t.Error("the ? button toggles it")
	}
}

func TestEdgesLeaveTheirBoxes(t *testing.T) {
	a, b := ui.Rect{X: 0, Y: 0, W: 10, H: 10}, ui.Rect{X: 30, Y: 0, W: 10, H: 10}
	if x0, _, x1, _ := EdgePoints(a, b); x0 != 10 || x1 != 30 {
		t.Error("left to right")
	}
	if x0, _, x1, _ := EdgePoints(b, a); x0 != 30 || x1 != 10 {
		t.Error("right to left")
	}
	c := ui.Rect{X: 0, Y: 30, W: 10, H: 10}
	if _, y0, _, y1 := EdgePoints(a, c); y0 != 10 || y1 != 30 {
		t.Error("down")
	}
	if _, y0, _, y1 := EdgePoints(c, a); y0 != 30 || y1 != 10 {
		t.Error("up")
	}
	d := ui.Rect{X: 30, Y: 30, W: 10, H: 10}
	if x0, y0, x1, y1 := EdgePoints(d, a); x0 != 30 || y0 != 30 || x1 != 10 || y1 != 10 {
		t.Error("diagonal up-left")
	}
}

func TestTopBarTipsNameSourcesAndUses(t *testing.T) {
	f := caesarFrame(t)
	for name, tip := range map[string]string{"gold": GoldTip(f), "wisdom": WisdomTip(f), "renown": RenownTip(f), "nines": NinesTip(f), "hands": HandsTip(f), "lore": LoreTip(f)} {
		if !strings.Contains(tip, "From:") && !strings.Contains(tip, "Used:") && !strings.Contains(tip, "Grows") {
			t.Errorf("%s tip lacks sources or uses: %q", name, tip)
		}
	}
	if !strings.Contains(GoldTip(f), "Upkeep now: -0.35") || !strings.Contains(WisdomTip(f), "+3") {
		t.Errorf("tips must carry the real numbers: %q / %q", GoldTip(f), WisdomTip(f))
	}
	f.State.SLO.Samples = 5
	if SLOFactor(f) != 1 {
		t.Error("an unproven Nines is neutral")
	}
	f.State.SLO.Samples, f.State.SLO.Current = 100, .5
	if SLOFactor(f) != .8 {
		t.Errorf("slo factor = %v", SLOFactor(f))
	}
}

func TestDeltasFloatFromTheirMeter(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	before, after := view.GameState{}, view.GameState{}
	before.Gold, after.Gold, after.Wisdom = 100, 88, 3
	for _, d := range MeterDeltas(before, after) {
		h.Delta(d.Meter, d.Value)
	}
	if got := h.LiveDeltas(); len(got) != 2 || got[0].Text != "-12" || got[0].Good || got[1].Text != "+3" {
		t.Fatalf("deltas = %+v", got)
	}
	if DeltaText(.001) != "" || DeltaText(1.5) != "+1.5" {
		t.Error("delta text")
	}
	if rise, a := DeltaOffset(0, false); rise != 0 || a != 255 {
		t.Error("a fresh delta is opaque")
	}
	if rise, _ := DeltaOffset(DeltaLife/2, true); rise != 0 {
		t.Error("reduced effects do not float")
	}
	dst := ebiten.NewImage(Width, Height)
	h.Draw(dst, f)
	h.Update(Input{}, Frame{Clock: 100})
	if len(h.LiveDeltas()) != 0 {
		t.Error("deltas expire")
	}
}

func TestKeepCardsCordonDrainAndTaint(t *testing.T) {
	f := staged(fullFrame(t))
	keep := view.Structure{ID: "keep-2", Kind: view.StructKeep, Variant: "stone", State: view.StateReady, Label: "Stone Keep 2", Object: view.ObjectRef{Kind: "Node", Name: "stone-keep-2"}}
	f.Selection = Selection{Kind: "structure", ID: keep.ID, Structure: keep}
	acts := SelectionActions(f)
	if len(acts) != 6 || acts[1].Command != "cordon stone-keep-2" || acts[2].Command != "drain stone-keep-2 --ignore-daemonsets" || !acts[2].Disabled ||
		acts[3].Choices[0].Command != "taint node stone-keep-2 ward=true:NoSchedule" {
		t.Fatalf("keep actions = %+v", acts)
	}
	f.Nodes = map[string]model.NodeView{"stone-keep-2": {Unschedulable: true, Taints: []string{"ward"}, TaintSpecs: []string{"ward=true:NoSchedule"}}}
	f.State.Grimoire = append(f.State.Grimoire, view.GrimoirePage{ID: "masonry", Ranks: []view.RankView{{Learned: true, Runes: []string{"drain"}}}})
	acts = SelectionActions(f)
	if acts[1].Label != "UNCORDON" || acts[2].Disabled || acts[3].Choices[3].Command != "taint node stone-keep-2 ward:NoSchedule-" {
		t.Errorf("cordoned, warded keep = %+v", acts)
	}
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	intents := clickOn(t, h, f, "act:uncordon")
	echo, ok := hasIntent(intents, IntentEcho)
	if !ok || echo.Command != "uncordon stone-keep-2" || echo.Tab != "ok" || runner.commands[len(runner.commands)-1] != "uncordon stone-keep-2" {
		t.Fatalf("uncordon = %+v, ran %v", intents, runner.commands)
	}
	if _, ok := hasIntent(intents, IntentKubectl); !ok {
		t.Error("the realm is polled after a command")
	}
	for i, b := range h.CommandButtons(f) {
		if !inside(b.Rect, Command) || b.Rect.W != CellW {
			t.Errorf("action %d %+v does not fill a grid cell", i, b.Rect)
		}
	}
	f.State.Reveals = []string{"gold"}
	if len(SelectionActions(f)) != 0 {
		t.Error("actions wait for the Speaking Stone")
	}
}

func TestHearthCardsDescribeLogAndDelete(t *testing.T) {
	f := fullFrame(t)
	f.Selection = Selection{Kind: "hearth", ID: "default/porter-a", Pod: f.Pods[0]}
	acts := SelectionActions(f)
	if len(acts) != 6 || acts[0].Command != "describe pod porter-a" || acts[2].Command != "delete pod porter-a" {
		t.Fatalf("hearth actions = %+v", acts)
	}
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	clickOn(t, h, f, "act:describe")
	if h.Overlay() != OverlayDetails || h.Details().Lines[0] != "$ kubectl describe pod porter-a" {
		t.Errorf("describe = %+v", h.Details())
	}
	h.Open(OverlayNone)
	f.Selection.Pod = model.PodView{Namespace: "kube-system", Name: "etcd-lone-tower", System: true}
	acts = SelectionActions(f)
	if !acts[2].Disabled || acts[0].Command != "describe pod etcd-lone-tower -n kube-system" {
		t.Errorf("citadel hearth = %+v", acts)
	}
	f.Selection.Pod = model.PodView{Namespace: "default", Name: "fail-me"}
	intents := clickOn(t, h, f, "act:delete")
	if echo, ok := hasIntent(intents, IntentEcho); !ok || echo.Tab != "error" {
		t.Errorf("a refused command echoes as an error: %+v", intents)
	}
	if h.PressAction("7", f) != nil || h.PressAction(KeyPrevious, f) != nil {
		t.Error("an out-of-range action does nothing")
	}
	if FirstLine("\n  node/x cordoned\nmore") != "node/x cordoned" {
		t.Error("first line")
	}
}

func TestGuildhallCardListsItsRoll(t *testing.T) {
	f := caesarFrame(t)
	f.Rolls[0].Members = []model.Member{{Name: "porter-a", Ready: true}, {Name: "porter-b"}}
	s := view.Structure{Kind: view.StructGuildhall, Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	lines := GuildhallLines(s, f)
	if len(lines) != 3 || lines[0].Text != "GUILD ROLL 2 of 3 ready" || lines[2].Text != "porter-b  not ready" {
		t.Errorf("lines = %+v", lines)
	}
	s.Object.Name = "nobody"
	if got := GuildhallLines(s, f); len(got) != 1 || !strings.Contains(got[0].Text, "empty") {
		t.Errorf("empty roll = %+v", got)
	}
}

func TestOpeningTheTargetCompletesAnOpenIntro(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := stagedFrame(t, MeterGold, "cmd:serve")
	f.Guide.Intro = &Intro{ID: "build", Title: "Build", Try: "Press BUILD.", Target: "cmd:build", Via: ViaOpen}
	intents := clickOn(t, h, f, "cmd:build")
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != "ack" || it.Action.ID != "build" {
		t.Fatalf("opening BUILD = %+v", intents)
	}
	if Passive(*f.Guide.Intro) || !Passive(Intro{Via: ViaAck, Target: "cmd:x"}) {
		t.Error("the game's via decides who acknowledges")
	}
	f.Guide.Intro = &Intro{ID: "stone", Target: SurfaceStone, Via: ViaOpen}
	f.Guide.Introduced = append(f.Guide.Introduced, SurfaceStone)
	if got := h.IntroTried(f, "tab:"+TabKube); len(got) != 1 {
		t.Errorf("the KUBE tab opens the Stone: %+v", got)
	}
	if h.IntroTried(f, "cmd:serve") != nil {
		t.Error("another button does not complete it")
	}
	f.Guide.Intro.Via = ViaAct
	if h.IntroTried(f, SurfaceStone) != nil {
		t.Error("an act intro waits for the game")
	}
}

func TestRatesReadAtASensiblePrecision(t *testing.T) {
	cases := map[float64]string{0: "0", .001: "0", .384: "0.38", -.4: "-0.40", 4.26: "4.3", 12.5: "12.5", 2500: "2.5K", -1500: "-1.5K"}
	for v, want := range cases {
		if got := Rate(v); got != want {
			t.Errorf("Rate(%v) = %q, want %q", v, got, want)
		}
	}
	if Signed(.38) != "+0.38" || Signed(0) != "0" || Signed(-2) != "-2.0" {
		t.Error("signed rates")
	}
	if UpkeepLine(0) != "upkeep: none yet" || UpkeepLine(.35) != "upkeep -0.35 each second" || UpkeepCell(0, 0) != "none yet" {
		t.Error("the cost line reads none yet at zero")
	}
}

func TestHiddenSurfacesLeaveNoFrames(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := stagedFrame(t, MeterGold, view.ParcelPile, "cmd:build")
	f.State.Reveals = slices.DeleteFunc(slices.Clone(f.State.Reveals), func(id string) bool { return id == "summon" })
	f.State.Pilgrims.GateOpen = true
	if MinimapShown(f) || CardShown(f) || h.WellShown(f) || h.StripShown(f) {
		t.Fatal("unintroduced panels are drawn")
	}
	for name, r := range map[string]ui.Rect{"minimap": Minimap, "card": CardBox, "well": InfoWell} {
		if h.Captures(r.X+r.W/2, r.Y+r.H/2, f) {
			t.Errorf("the hidden %s still swallows map clicks", name)
		}
	}
	frame := h.CommandFrame(f)
	if frame.X+frame.W != Command.X+Command.W || frame.Y+frame.H != Command.Y+Command.H || frame.W >= Command.W {
		t.Errorf("the command card %+v must hug the bottom-right corner and hold only its commands", frame)
	}
	buttons := h.CommandButtons(f)
	for _, b := range buttons {
		if !inside(b.Rect, frame) {
			t.Errorf("%s %+v sits outside its card %+v", b.ID, b.Rect, frame)
		}
	}
	build, _ := findButton(buttons, "cmd:build")
	if build.Rect != MainCell(MainHomes["cmd:build"]) {
		t.Errorf("the first command lives in its home cell: %+v", build.Rect)
	}
	g := f
	g.Guide.Introduced = append(append([]string(nil), f.Guide.Introduced...), "cmd:summon", SurfaceMinimap, SurfaceGate, SurfaceTidings)
	for _, b := range h.CommandButtons(g) {
		if old, ok := findButton(buttons, b.ID); ok && old.Rect != b.Rect {
			t.Errorf("%s moved from %+v to %+v when another command arrived", b.ID, old.Rect, b.Rect)
		}
	}
	if !h.StripShown(g) || !h.Captures(ConsoleBox.X+2, ConsoleBox.Y+2, g) {
		t.Error("once every panel is introduced the console is whole again")
	}
	h.PageID = PageBuild
	if h.CommandFrame(f) != Command {
		t.Error("a sub-page shows the whole card")
	}
}

func TestSurfacesFadeInWhenIntroduced(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := stagedFrame(t, MeterGold, "cmd:serve")
	f.Clock = 10
	h.NoteAppearances(f)
	if h.AppearProgress("cmd:serve", 10) != 1 || h.AppearProgress(KeyMinimap, 10) != 1 {
		t.Fatal("what shows when the realm loads does not fade")
	}
	f.Guide.Introduced = append(f.Guide.Introduced, SurfaceMinimap, "cmd:build")
	f.Clock = 20
	h.NoteAppearances(f)
	if p := h.AppearProgress(KeyMinimap, 20); p != 0 {
		t.Errorf("a new surface starts hidden: %v", p)
	}
	if p := h.AppearProgress("cmd:build", 20+AppearFor/2); p <= 0 || p >= 1 {
		t.Errorf("a new command fades in: %v", p)
	}
	if h.AppearProgress(KeyMinimap, 20+AppearFor) != 1 {
		t.Error("the fade ends")
	}
	dst := ebiten.NewImage(Width, Height)
	f.Clock = 20.2
	h.Draw(dst, f)
	h.Reduce = true
	if h.AppearProgress("cmd:build", 20) != 1 {
		t.Error("reduced effects skip the fade")
	}
}

func TestTheDepotSaysWhichOfItsHutsCarryOutsideTheCrew(t *testing.T) {
	f := caesarFrame(t)
	f.Crews = []model.CrewChain{{Namespace: "default", Name: "porters", Image: "porter:1.0", Desired: 6, Ready: 0}}
	f.Rolls = []model.GuildRoll{{Service: "parcels", Namespace: "default", Ready: 5, Endpoints: 5}}
	f.Pods = nil
	for _, name := range []string{"porter-a", "porter-b", "porter-c", "porter-d", "porter-lone-tower"} {
		owner := ""
		if name == "porter-lone-tower" {
			owner = "Node/lone-tower"
		}
		f.Rolls[0].Members = append(f.Rolls[0].Members, model.Member{Name: name, Ready: true})
		f.Pods = append(f.Pods, model.PodView{Namespace: "default", Name: name, Ready: true, Owner: owner, State: model.HearthReady})
	}
	f.Pods = append(f.Pods, model.PodView{Namespace: "default", Name: "porters-6c7b5dbf87-65v4p", Owner: "ReplicaSet/porters-6c7b5dbf87", State: model.HearthPending})
	s := view.Structure{Kind: view.StructGuildhall, Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	if lines := GuildhallLines(s, f); !strings.Contains(lines[0].Text, "5 outside the crew") {
		t.Errorf("the crew card reads WANTS 6 READY 0 and the roll %q (watch-world: \"GUILD ROLL 5 of 5 ready\" beside \"READY 0\")", lines[0].Text)
	}
	depot := view.Structure{Kind: view.DepotStructure("parcels"), Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	f.State.Freight.Kinds[0].Crew = 5
	joined := ""
	for _, l := range DepotLines(depot, f) {
		joined += l.Text + "|"
	}
	if !strings.Contains(joined, "5 outside the crew") {
		t.Errorf("the depot card reads %q beside a crew card with READY 0 (watch-world: \"5 porters carry\")", joined)
	}
}
