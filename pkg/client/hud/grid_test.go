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

func gridCell(t *testing.T, h *HUD, f Frame, id string) int {
	t.Helper()
	for _, b := range h.CommandButtons(f) {
		if b.ID != id {
			continue
		}
		for i := range GridCells {
			if CellRect(i) == b.Rect {
				return i
			}
		}
		t.Fatalf("%s sits at %+v, outside the grid", id, b.Rect)
	}
	return -1
}

func gridIDs(h *HUD, f Frame) []string {
	out := []string{}
	for _, b := range h.CommandButtons(f) {
		out = append(out, b.ID)
	}
	return out
}

func wantCells(t *testing.T, h *HUD, f Frame, want map[string]int) {
	t.Helper()
	for id, cell := range want {
		if got := gridCell(t, h, f, id); got != cell {
			t.Errorf("%s in cell %d, want %d (grid %v)", id, got, cell, gridIDs(h, f))
		}
	}
}

func guided(f Frame, target, button string) Frame {
	g := f
	g.State.Objectives = []view.ObjectiveView{{ID: "o", Text: "Do it", Target: target, Button: button}}
	g.Guide.Target, g.Guide.Button = target, button
	return g
}

func TestAKeepSelectionFillsTheGridWithItsCommands(t *testing.T) {
	f := selectStructure(staged(navFrame(t)), stoneKeep())
	h := New(nil, (&fakeRunner{}).backend())
	wantCells(t, h, f, map[string]int{"act:describe": 0, "act:top": 1, "act:label": 2, "act:taint": 3, "act:cordon": 4, "act:drain": 5, "keep:retire": 8})
	for _, id := range []string{"cmd:build", "cmd:grimoire", CmdCrew} {
		if gridCell(t, h, f, id) >= 0 {
			t.Errorf("the home grid's %s stays hidden while a Keep is selected", id)
		}
	}
	if r := h.CommandFrame(f); r != Command {
		t.Errorf("an object grid fills the whole frame: %+v", r)
	}
}

func TestTheCardKeepsOnlyInformation(t *testing.T) {
	for _, c := range cardCases(t) {
		h := New(nil, (&fakeRunner{}).backend())
		for _, b := range h.SelectionButtons(c.f) {
			if b.ID != "sel:close" && !strings.HasPrefix(b.ID, CrewPrefix+":") {
				t.Errorf("%s card still carries %s", c.name, b.ID)
			}
		}
	}
}

func TestEveryCardActionHasAGridCell(t *testing.T) {
	for _, c := range cardCases(t) {
		h := New(nil, (&fakeRunner{}).backend())
		for _, a := range SelectionActions(c.f) {
			if gridCell(t, h, c.f, ActionPrefix+":"+a.Key) < 0 {
				t.Errorf("%s: %s has no cell (grid %v)", c.name, a.Key, gridIDs(h, c.f))
			}
		}
	}
}

func TestEachKindKeepsOneLayout(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	pod := fullFrame(t)
	pod.Selection = Selection{Kind: "hearth", ID: "default/porter-a", Pod: pod.Pods[0]}
	wantCells(t, h, pod, map[string]int{"act:describe": 0, "act:yaml": 1, "act:previous": 2, "act:label": 3, "act:logs": 4, "act:delete": 8})
	codex := selectStructure(fullFrame(t), view.Structure{ID: "codex-1", Kind: view.StructCodex, State: view.StateReady,
		Object: view.ObjectRef{Kind: "Pod", Namespace: "kube-system", Name: "etcd-lone-tower"}})
	wantCells(t, h, codex, map[string]int{"sel:details": 0, "act:status": 1, "act:alarms": 2, "act:compact": 3, "act:defrag": 4, "act:disarm": 5, "act:etcd-version": 6, GridHome: 8})
	chancery := selectStructure(fullFrame(t), view.Structure{ID: "chancery-1", Kind: view.StructChancery, State: view.StateReady})
	wantCells(t, h, chancery, map[string]int{"act:get": 1, "act:events": 2, "act:version": 3, "act:get-pods": 4, "act:cluster-info": 5, "act:api-resources": 6, GridHome: 8})
	observatory := selectStructure(fullFrame(t), view.Structure{ID: "observatory-1", Kind: view.StructObservatory, State: view.StateReady})
	wantCells(t, h, observatory, map[string]int{"act:top-nodes": 1, "act:top-pods": 4, GridHome: 8})
	depot := selectStructure(staged(crewFrame(Selection{})), view.Structure{ID: "guildhall-1", Kind: view.StructGuildhall, State: view.StateReady,
		Object: view.ObjectRef{Kind: "Service", Name: "parcels"}})
	wantCells(t, h, depot, map[string]int{"act:describe": 0, "act:endpoints": 1, CrewBoardButton: 3, ChainButton: 4, "act:delete": 8})
	herald := selectStructure(fullFrame(t), view.Structure{ID: "herald-1", Kind: view.StructHerald, State: view.StateReady,
		Object: view.ObjectRef{Kind: "Pod", Namespace: "kube-system", Name: "coredns"}})
	wantCells(t, h, herald, map[string]int{"sel:details": 0, GridHome: 8})
}

func TestTheTowerOffersSummonAndNoRetire(t *testing.T) {
	f := towerCard(t)
	h := New(nil, (&fakeRunner{}).backend())
	wantCells(t, h, f, map[string]int{"act:summon": 6, GridHome: 8})
	if gridCell(t, h, f, "keep:retire") >= 0 {
		t.Error("the Lone Tower cannot be retired")
	}
}

func TestAGridCellRunsItsCommandWithTheCommandInItsTip(t *testing.T) {
	f := selectStructure(staged(fullFrame(t)), stoneKeep())
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	b, ok := HitID(h.CommandButtons(f), "act:cordon")
	if !ok || !strings.Contains(b.Tip, "$ kubectl cordon stone-keep-2") {
		t.Fatalf("cordon cell = %+v", b)
	}
	clickOn(t, h, f, "act:cordon")
	if runner.commands[len(runner.commands)-1] != "cordon stone-keep-2" {
		t.Fatalf("ran %v", runner.commands)
	}
}

func TestTheBackCellReturnsToTheHomeGrid(t *testing.T) {
	f := selectStructure(fullFrame(t), view.Structure{ID: "chancery-1", Kind: view.StructChancery, State: view.StateReady})
	h := New(nil, (&fakeRunner{}).backend())
	h.Update(Input{}, f)
	if _, ok := hasIntent(clickOn(t, h, f, GridHome), IntentSelect); !ok {
		t.Fatal("BACK deselects")
	}
	if gridCell(t, h, f, "cmd:build") < 0 {
		t.Errorf("after BACK the home grid shows at once: %v", gridIDs(h, f))
	}
	g := f
	g.Selection = Selection{}
	h.Update(Input{}, g)
	h.Update(Input{}, f)
	if gridCell(t, h, f, "act:get-pods") != 4 {
		t.Errorf("selecting it again shows its grid: %v", gridIDs(h, f))
	}
	if _, ok := hasIntent(h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f), IntentSelect); !ok {
		t.Error("Esc deselects too")
	}
}

func TestANewSelectionLeavesTheBuildPage(t *testing.T) {
	f := fullFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.PageID = PageBuild
	h.Update(Input{}, f)
	g := selectStructure(f, stoneKeep())
	h.Update(Input{}, g)
	if h.PageID != PageMain {
		t.Errorf("page = %q", h.PageID)
	}
}

func TestHotkeysPressTheCellsOfAnObjectGrid(t *testing.T) {
	f := selectStructure(staged(fullFrame(t)), stoneKeep())
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	if !h.GridKeysTaken(f) {
		t.Fatal("an object grid takes the grid digits")
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.Key5}}, f)
	if len(runner.commands) == 0 || runner.commands[len(runner.commands)-1] != "cordon stone-keep-2" {
		t.Fatalf("5 presses the centre cell: %v", runner.commands)
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.Key1}}, f)
	if h.Overlay() != OverlayDetails {
		t.Errorf("1 presses DESCRIBE: overlay %q", h.Overlay())
	}
	home := fullFrame(t)
	if h2 := New(nil, runner.backend()); h2.GridKeysTaken(home) {
		t.Error("the home grid leaves the digits alone")
	}
}

func TestHotkeysWalkTheBuildPage(t *testing.T) {
	f := fullFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.PageID = PageBuild
	if !h.GridKeysTaken(f) {
		t.Fatal("a page takes the grid digits")
	}
	intents := h.Update(Input{Keys: []ebiten.Key{ebiten.Key1}}, f)
	if it, ok := hasIntent(intents, IntentPlace); !ok || it.Target != "keep" {
		t.Errorf("1 picks the first building: %+v", intents)
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.Key9}}, f)
	if h.PageID != PageMain {
		t.Errorf("9 is BACK on a page: %q", h.PageID)
	}
	placing := f
	placing.Placing = "keep"
	h.PageID = PageBuild
	if h.GridKeysTaken(placing) {
		t.Error("while placing, the digits are not grid keys")
	}
}

func TestADisabledCellRefusesItsHotkey(t *testing.T) {
	f := selectStructure(staged(fullFrame(t)), stoneKeep())
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	h.Update(Input{Keys: []ebiten.Key{ebiten.Key6}}, f)
	if len(runner.commands) != 0 || len(h.Toasts()) == 0 || !strings.Contains(h.Toasts()[0].Text, "Masonry") {
		t.Errorf("a locked DRAIN says why: %v %+v", runner.commands, h.Toasts())
	}
}

func TestTheGuideLeavesAnObjectForAHomeCommand(t *testing.T) {
	f := walkFrame(t, "cmd:grimoire")
	f = selectStructure(f, stoneKeep())
	h := New(nil, (&fakeRunner{}).backend())
	w := h.WalkGuide(f)
	if !w.OK() || len(w.Steps) != 2 || w.Steps[0] != "sel:close" || w.Steps[1] != "cmd:grimoire" {
		t.Fatalf("walk = %+v", w)
	}
	g := walkFrame(t, "build:keep")
	g = selectStructure(g, stoneKeep())
	if w := h.WalkGuide(g); !w.OK() || w.Steps[0] != "sel:close" || w.Steps[len(w.Steps)-1] != "build:keep" {
		t.Fatalf("walk to a building = %+v", w)
	}
}

func TestTheGuideLightsTheObjectsCell(t *testing.T) {
	f := fullFrame(t)
	f.Guide.Target, f.Guide.Button = "structure:chancery", "act:get-pods"
	f = selectStructure(f, view.Structure{ID: "chancery-1", Kind: view.StructChancery, State: view.StateReady})
	h := New(nil, (&fakeRunner{}).backend())
	step, ok := h.Step(f)
	if !ok || step.ID != "act:get-pods" || step.Rect != CellRect(4) {
		t.Fatalf("step = %+v %v", step, ok)
	}
	b, _ := HitID(h.CommandButtons(f), "act:get-pods")
	if !b.Selected {
		t.Error("the guided cell is marked")
	}
}

func TestACrewGridScalesFromTheCentre(t *testing.T) {
	f := crewFrame(crewSel())
	h := New(nil, (&fakeRunner{}).backend())
	wantCells(t, h, f, map[string]int{"crew:describe": 0, "crew:yaml": 1, "crew:mode:" + CrewModeRoll: 2, "crew:scale:down": 3, "crew:scale:up": 4, GridHome: 8})
	a, _ := actionOf(clickOn(t, h, f, "crew:scale:up"))
	if a.Action != view.ActScaleCrew || a.Replicas != 4 {
		t.Errorf("SCALE +1 = %+v", a)
	}
	if !h.ChainShown(f) {
		t.Error("a selected crew shows its board")
	}
}

func TestAHutOpensItsCrewAsASubGrid(t *testing.T) {
	sel := Selection{Kind: SelectHearth, ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")}
	f := staged(crewFrame(sel))
	h := New(nil, (&fakeRunner{}).backend())
	wantCells(t, h, f, map[string]int{ChainButton: 6, "act:delete": 8})
	clickOn(t, h, f, ChainButton)
	wantCells(t, h, f, map[string]int{"crew:scale:up": 4, ChainButton: 8})
	if !h.ChainShown(f) {
		t.Error("the crew opens with its board")
	}
	clickOn(t, h, f, ChainButton)
	if gridCell(t, h, f, "act:delete") != 8 || h.ChainShown(f) {
		t.Errorf("BACK returns to the hut: %v", gridIDs(h, f))
	}
	f = guided(f, "crew:default/porters", "crew:scale:up")
	if w := h.WalkGuide(f); !w.OK() || len(w.Steps) != 2 || w.Steps[0] != ChainButton {
		t.Errorf("the guide leads from the hut to SCALE: %+v", w)
	}
}

func TestCrewModesArePagesOfTheGrid(t *testing.T) {
	f := crewFrame(crewSel(), "soulvault")
	h := New(nil, (&fakeRunner{}).backend())
	clickOn(t, h, f, "crew:mode:"+CrewModeRoll)
	wantCells(t, h, f, map[string]int{"crew:roll:restart": 0, "crew:roll:undo": 1, "crew:roll:pause": 2, "crew:history": 3, "crew:mode:": 8})
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
	if gridCell(t, h, f, "crew:scale:up") != 4 {
		t.Errorf("Esc steps back to the crew's grid: %v", gridIDs(h, f))
	}
	f = guided(f, "crew:default/porters", "crew:roll:restart")
	if w := h.WalkGuide(f); !w.OK() || len(w.Steps) != 2 || w.Steps[0] != "crew:mode:"+CrewModeRoll {
		t.Errorf("the guide opens the rollout page first: %+v", w)
	}
}

func TestTheBoardsKeepOnlyTheirTactileParts(t *testing.T) {
	f := boardFrame()
	h := New(nil, (&fakeRunner{}).backend())
	for _, b := range h.CrewButtons(f) {
		for _, order := range []string{"crew:describe", "crew:yaml", "crew:mode:"} {
			if strings.HasPrefix(b.ID, order) {
				t.Errorf("the crew board still carries %s", b.ID)
			}
		}
	}
	if _, ok := HitID(h.CrewButtons(f), SliderButton); !ok {
		t.Error("the slider stays on the board")
	}
}

func TestAPortShowsItsBerthsAndUpgradeInTheGrid(t *testing.T) {
	f := harborFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	berth, up := PortButtonID(PortVerbBerth, "haven-0"), PortButtonID(PortVerbRaise, "haven-0")
	wantCells(t, h, f, map[string]int{berth: BerthCell, up: CentreCell, PortButtonID(PortVerbClose, "haven-0"): UnberthCell, GridHome: BackCell})
	b, _ := HitID(h.CommandButtons(f), berth)
	u, _ := HitID(h.CommandButtons(f), up)
	if b.Label != "OPEN BERTH" || b.Cost != "100" || b.Disabled || u.Label != "UPGRADE: GALLEONS" || u.Cost != "200K" || !u.Disabled {
		t.Fatalf("berth %+v upgrade %+v", b, u)
	}
	if it, ok := hasIntent(clickOn(t, h, f, berth), IntentAction); !ok || it.Action.Action != view.ActOpenBerth || it.Action.ID != "haven-0" {
		t.Fatalf("OPEN BERTH = %+v", it)
	}
	f = guided(f, BerthButton, "")
	if s, ok := h.Step(f); !ok || s.ID != berth {
		t.Fatalf("the berth target lights the selected port's berth: %+v", s)
	}
	closed := f
	closed.Selection.ID = "haven-1"
	wantCells(t, h, closed, map[string]int{PortButtonID(PortVerbOpen, "haven-1"): CentreCell})
	if g := h.GuideGoal(closed); g != GateHaven {
		t.Fatalf("from a closed port the berth target walks to the Gate Harbor: %q", g)
	}
	legacy := harborFrame(t)
	legacy.State.Ports = nil
	wantCells(t, h, legacy, map[string]int{BerthButton: BerthCell, CloseBerthButton: UnberthCell})
}

func TestASpotlightLightsACellUntilItIsPressed(t *testing.T) {
	f := crewFrame(crewSel())
	h := New(nil, (&fakeRunner{}).backend())
	h.Spotlight(GridScale)
	if h.Spotlit(f) != "crew:scale:up" {
		t.Fatalf("the spotlight finds SCALE: %q", h.Spotlit(f))
	}
	clickOn(t, h, f, "crew:scale:up")
	if h.Spotlit(f) != "" {
		t.Error("pressing the cell puts the spotlight out")
	}
	if got := NavPath(GridScale, NavState{Page: PageMain, Object: true, Crew: true}); len(got) != 1 || got[0] != "crew:scale:up" {
		t.Errorf("grid:scale is a guide target: %v", got)
	}
}

func bareCrewFrame(sel Selection) Frame {
	c := crewChain()
	c.Bare, c.Name, c.Foremen = true, "", c.Foremen[:1]
	f := crewFrame(sel)
	f.Crews = []model.CrewChain{c}
	return f
}

func TestAReplicaSetCrewScalesDescribesAndDeletesFromItsGrid(t *testing.T) {
	f := bareCrewFrame(Selection{Kind: SelectForeman, ID: "default/porters-7d9f"})
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	wantCells(t, h, f, map[string]int{"crew:describe": 0, "crew:yaml": 1, ScaleDownButton: 3, ScaleUpButton: 4, ChainButton: 7, CrewDelete: 8})
	up, _ := HitID(h.CommandButtons(f), ScaleUpButton)
	if !strings.Contains(up.Tip, "$ kubectl scale replicaset porters-7d9f --replicas=4") {
		t.Fatalf("SCALE +1 names its kubectl: %q", up.Tip)
	}
	clickOn(t, h, f, ScaleUpButton)
	if runner.commands[len(runner.commands)-1] != "scale replicaset porters-7d9f --replicas=4" {
		t.Fatalf("SCALE +1 ran %v", runner.commands)
	}
	clickOn(t, h, f, CrewDelete)
	if h.Overlay() != OverlayChooser || len(h.Ops.Chooser.Options) != 1 {
		t.Fatalf("DELETE asks first: %q %+v", h.Overlay(), h.Ops.Chooser)
	}
	clickOn(t, h, f, "pick:0")
	if runner.commands[len(runner.commands)-1] != "delete replicaset porters-7d9f" {
		t.Fatalf("DELETE ran %v", runner.commands)
	}
	hut := bareCrewFrame(Selection{Kind: SelectHearth, ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")})
	h2 := New(nil, runner.backend())
	clickOn(t, h2, hut, ChainButton)
	wantCells(t, h2, hut, map[string]int{ScaleUpButton: 4, ChainButton: 7, CrewDelete: 8})
	hut = guided(hut, "pod:porters-7d9f-a", ScaleUpButton)
	if w := New(nil, runner.backend()).WalkGuide(hut); !w.OK() || len(w.Steps) != 2 || w.Steps[0] != ChainButton {
		t.Errorf("the guide leads from a hut to its shift's SCALE: %+v", w)
	}
}

func TestTheSharedCrewVerbsKeepTheirCellsOnBothKinds(t *testing.T) {
	deploy, bare := crewFrame(crewSel()), bareCrewFrame(Selection{Kind: SelectForeman, ID: "default/porters-7d9f"})
	h := New(nil, (&fakeRunner{}).backend())
	for _, id := range []string{"crew:describe", "crew:yaml", ScaleDownButton, ScaleUpButton} {
		if a, b := gridCell(t, h, deploy, id), gridCell(t, h, bare, id); a != b || a < 0 {
			t.Errorf("%s sits in cell %d for a Deployment and %d for a ReplicaSet", id, a, b)
		}
	}
}

func TestAHomeIntroShowsTheHomeGridOverASelection(t *testing.T) {
	f := selectStructure(staged(navFrame(t)), stoneKeep())
	f.Guide = Guide{Staged: true, Intro: &Intro{ID: "grimoire", Target: "cmd:grimoire", Via: ViaOpen}, Introduced: []string{"cmd:build", SurfaceStone}}
	h := New(nil, (&fakeRunner{}).backend())
	if gridCell(t, h, f, "cmd:grimoire") < 0 {
		t.Fatalf("the intro shows its home command: %v", gridIDs(h, f))
	}
	if s, ok := h.Step(f); !ok || s.ID != "cmd:grimoire" {
		t.Errorf("the intro's hole is its command: %+v", s)
	}
	f.Guide.Intro = &Intro{ID: "tidings", Target: SurfaceTidings, Via: ViaAck}
	if gridCell(t, h, f, "act:cordon") != 4 {
		t.Errorf("a surface intro leaves the Keep's grid: %v", gridIDs(h, f))
	}
}

func TestTheForemanCardCountsReplicasInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	c := crewChain()
	card := ForemanCard(c, c.Foremen[0])
	if got := ui.Name(card.Lines[0].Text); strings.Contains(got, "NODE") || !strings.Contains(got, "REPLICAS 3") {
		t.Errorf("the shift's count reads %q", got)
	}
}

func TestAPortTargetWalksToItsPortFirst(t *testing.T) {
	up := PortButtonID(PortVerbRaise, "haven-2")
	if got := NavPath(up, NavState{}); len(got) != 2 || got[0] != "haven:haven-2" || got[1] != up {
		t.Errorf("from the realm: %v", got)
	}
	if got := NavPath(up, NavState{Harbor: true, Haven: "haven-2", Object: true}); len(got) != 1 || got[0] != up {
		t.Errorf("on its port: %v", got)
	}
	if got := NavPath(up, NavState{Harbor: true, Haven: "haven-0", Object: true}); len(got) != 2 || got[0] != "haven:haven-2" {
		t.Errorf("from another port: %v", got)
	}
}

func TestTheCardNamesItsObjectLiterallyInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	f := selectStructure(fullFrame(t), view.Structure{ID: "guildhall-1", Kind: view.StructGuildhall, State: view.StateReady,
		Object: view.ObjectRef{Kind: "Service", Name: "parcels"}})
	h := New(nil, (&fakeRunner{}).backend())
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	if !hasRun(recordRuns(func() { h.DrawSelection(dst, f) }), "Service/parcels") {
		t.Error("the card renames the Service's own name")
	}
}

func TestAForemansCrewGridScalesItsReplicaSet(t *testing.T) {
	f := foremanBoardFrame()
	h := New(nil, (&fakeRunner{}).backend())
	wantCells(t, h, f, map[string]int{"crew:describe": 0, "crew:yaml": 1, ScaleDownButton: 3, ScaleUpButton: 4, GridHome: 7, CrewDelete: 8})
	up, _ := HitID(h.CommandButtons(f), ScaleUpButton)
	if !strings.Contains(up.Tip, "$ kubectl scale rs porters --replicas=4") {
		t.Fatalf("SCALE +1 = %q", up.Tip)
	}
	a, _ := actionOf(clickOn(t, h, f, ScaleUpButton))
	if a.Action != view.ActScaleCrew || a.Name != "porters" || a.Replicas != 4 || a.Crew.Keeper != view.KeeperReplicaSet {
		t.Fatalf("SCALE +1 = %+v", a)
	}
	f = guided(f, "crew:default/porters", ScaleUpButton)
	if w := h.WalkGuide(f); !w.OK() || len(w.Steps) != 1 || w.Steps[0] != ScaleUpButton {
		t.Errorf("the guide lights SCALE +1 on a shift's crew: %+v", w)
	}
}
