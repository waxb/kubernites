package hud

import (
	"github.com/waxb/kubernites/pkg/soulstone"

	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

func crewChain() model.CrewChain {
	newest := model.Foreman{Namespace: "default", Name: "porters-7d9f", Hash: "7d9f", Revision: 2, Desired: 3, Ready: 2, Newest: true,
		Image: "porter:1.0", Pods: []string{"porters-7d9f-a", "porters-7d9f-b", "porters-7d9f-c"}}
	old := model.Foreman{Namespace: "default", Name: "porters-54cd", Hash: "54cd", Revision: 1, Image: "porter:1.0"}
	return model.CrewChain{Namespace: "default", Name: "porters", Image: "porter:1.0", Container: "porter", Desired: 3, Ready: 2,
		Updated: 3, Strategy: "RollingUpdate", MaxSurge: "25%", MaxUnavailable: "25%", Revision: 2, HistoryLimit: 10,
		Depot: "parcels", Foremen: []model.Foreman{newest, old}}
}

func crewPod(name string) model.PodView {
	return model.PodView{Namespace: "default", Name: name, Owner: "ReplicaSet/porters-7d9f", State: model.HearthReady, Crew: "porters"}
}

func runes(ids ...string) view.GrimoirePage {
	return view.GrimoirePage{ID: "runes", Ranks: []view.RankView{{Learned: true, Runes: ids}}}
}

func crewFrame(sel Selection, reveals ...string) Frame {
	st := view.GameState{}
	st.Reveals = append([]string{"stone", "crew"}, reveals...)
	st.Unlocks = map[string]bool{}
	st.Soulstones = []view.SoulstoneEntry{{Image: "porter:1.0", Owned: true}, {Image: "porter:1.1", Owned: true}, {Image: "porter:2.0"}, {Image: "clerk:1.0", Owned: true}}
	return Frame{State: st, Selection: sel, Crews: []model.CrewChain{crewChain()},
		Pods: []model.PodView{crewPod("porters-7d9f-a"), crewPod("porters-7d9f-b"), crewPod("porters-7d9f-c")}}
}

func crewSel() Selection { return Selection{Kind: SelectCrew, ID: "default/porters"} }

func TestTheChainAnswersEverySelectionOfTheCrew(t *testing.T) {
	depot := view.Structure{Kind: "guildhall", Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	sels := []Selection{crewSel(), {Kind: SelectForeman, ID: "default/porters-54cd"}, {Kind: SelectHearth, ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")},
		{Kind: "walker", Pod: crewPod("porters-7d9f-b")}, {Kind: "structure", Structure: depot}}
	for _, sel := range sels {
		if _, ok := ChainOf(crewFrame(sel)); !ok {
			t.Errorf("%s %s shows no crew card", sel.Kind, sel.ID)
		}
	}
	strangers := []Selection{{}, {Kind: SelectCrew, ID: "default/clerks"}, {Kind: SelectHearth, Pod: model.PodView{Name: "bare"}},
		{Kind: "structure", Structure: view.Structure{Object: view.ObjectRef{Kind: "Node", Name: "parcels"}}}}
	for _, sel := range strangers {
		if _, ok := ChainOf(crewFrame(sel)); ok {
			t.Errorf("%+v shows the porters' card", sel)
		}
	}
}

func crewButtonMap(buttons []Button) map[string]Button {
	out := map[string]Button{}
	for _, b := range buttons {
		out[b.ID] = b
	}
	return out
}

func TestCrewButtonsAppearWithTheirReveals(t *testing.T) {
	early := crewButtonMap(DeploymentButtons(crewChain(), crewFrame(crewSel())))
	for _, id := range []string{"crew:scale:down", "crew:scale:up", "crew:describe", "crew:yaml", "crew:roll:restart", "crew:roll:undo", "crew:history"} {
		if _, ok := early[id]; !ok {
			t.Errorf("%s missing early", id)
		}
	}
	for _, id := range []string{"crew:mode:image", "crew:mode:resources", "crew:mode:horn"} {
		if _, ok := early[id]; ok {
			t.Errorf("%s shown before its reveal", id)
		}
	}
	f := crewFrame(crewSel(), "soulvault")
	f.State.Unlocks[HornUnlock] = true
	f.State.Grimoire = []view.GrimoirePage{runes("resources.requests.cpu")}
	late := crewButtonMap(DeploymentButtons(crewChain(), f))
	for _, id := range []string{"crew:mode:image", "crew:mode:resources", "crew:mode:horn", "crew:roll:pause"} {
		if _, ok := late[id]; !ok {
			t.Errorf("%s missing once revealed", id)
		}
	}
	if tip := late["crew:scale:up"].Tip; !strings.Contains(tip, "kubectl scale deployment porters --replicas=4") {
		t.Errorf("scale tip = %q", tip)
	}
	bare := model.CrewChain{Bare: true, Namespace: "default", Foremen: []model.Foreman{{Namespace: "default", Name: "lantern", Newest: true}}}
	if ids := crewButtonMap(DeploymentButtons(bare, f)); len(ids) != 2 || !strings.Contains(ids["crew:describe"].Tip, "describe replicaset lantern") {
		t.Errorf("a bare shift offers %v", ids)
	}
}

func pressed(t *testing.T, h *HUD, f Frame, id string) []Intent {
	t.Helper()
	return h.Press(id, f)
}

func actionOf(intents []Intent) (view.Action, string) {
	for _, it := range intents {
		if it.Kind == IntentAction {
			return it.Action, it.From
		}
	}
	return view.Action{}, ""
}

func TestCrewOrdersAreRealmActionsFromTheCrewCard(t *testing.T) {
	h := New(nil, Backend{})
	f := crewFrame(crewSel(), "soulvault")
	cases := map[string]view.Action{
		"crew:scale:up":         {Action: view.ActScaleCrew, Name: "porters", Replicas: 4, Crew: view.CrewOrder{Container: "porter"}},
		"crew:scale:down":       {Action: view.ActScaleCrew, Name: "porters", Replicas: 2, Crew: view.CrewOrder{Container: "porter"}},
		"crew:roll:undo":        {Action: view.ActRolloutCrew, Name: "porters", ID: "undo", Crew: view.CrewOrder{Container: "porter"}},
		"crew:image:porter:1.1": {Action: view.ActCrewImage, Name: "porters", Image: "porter:1.1", Crew: view.CrewOrder{Container: "porter"}},
		"crew:horn:silence":     {Action: view.ActSilenceHorn, Name: "porters"},
		"crew:roll:restart":     {Action: view.ActRolloutCrew, Name: "porters", ID: "restart", Crew: view.CrewOrder{Container: "porter"}},
	}
	for id, want := range cases {
		got, from := actionOf(pressed(t, h, f, id))
		if got != want || from != FromCrewCard {
			t.Errorf("%s = %+v from %q, want %+v", id, got, from, want)
		}
	}
	if got := pressed(t, h, crewFrame(Selection{}), "crew:scale:up"); got != nil {
		t.Errorf("an order without a crew = %+v", got)
	}
}

func TestPickingALevelChoosesItsObject(t *testing.T) {
	h := New(nil, Backend{})
	f := crewFrame(crewSel())
	want := map[string][2]string{
		"crew:pick:deploy":             {SelectCrew, "default/porters"},
		"crew:pick:rs:porters-7d9f":    {SelectForeman, "default/porters-7d9f"},
		"crew:pick:hist:porters-54cd":  {SelectForeman, "default/porters-54cd"},
		"crew:pick:pod:porters-7d9f-a": {SelectHearth, "default/porters-7d9f-a"},
	}
	for id, w := range want {
		it := pressed(t, h, f, id)
		if len(it) == 0 || it[0].Kind != IntentChoose || it[0].Target != w[0] || it[0].Command != w[1] {
			t.Errorf("%s = %+v", id, it)
		}
	}
	if it := pressed(t, h, f, "crew:close"); len(it) == 0 || it[0].Kind != IntentSelect {
		t.Errorf("the X on a selected crew's board deselects it: %+v", it)
	}
	hut := crewFrame(Selection{Kind: SelectHearth, ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")})
	h.Press(ChainButton, hut)
	if it := pressed(t, h, hut, "crew:close"); len(it) == 0 || it[0].Kind == IntentSelect || h.ChainShown(hut) {
		t.Errorf("close = %+v, chain still shown %v", it, h.ChainShown(hut))
	}
}

func TestResourcesStepAndShowTheirQoS(t *testing.T) {
	h := New(nil, Backend{})
	f := crewFrame(crewSel())
	f.State.Grimoire = []view.GrimoirePage{runes("resources.requests.cpu", "resources.requests.memory", "resources.limits.cpu", "resources.limits.memory")}
	pressed(t, h, f, "crew:mode:resources")
	if h.Crew.Mode != CrewModeRes {
		t.Fatal("RESOURCES opened no stepper")
	}
	if a, ok := ResourcesAction(crewChain(), h.Crew.Res, f); ok {
		t.Fatalf("nothing changed, yet %+v", a)
	}
	for _, id := range []string{"cpu-req:up", "cpu-req:up", "mem-req:up", "cpu-lim:up", "cpu-lim:up", "mem-lim:up"} {
		pressed(t, h, f, "crew:res:"+id)
	}
	want := model.Resources{CPURequest: 100, MemRequest: 32, CPULimit: 100, MemLimit: 32}
	if h.Crew.Res != want || PickQoS(h.Crew.Res) != model.QoSGuaranteed {
		t.Fatalf("picks %+v qos %s", h.Crew.Res, PickQoS(h.Crew.Res))
	}
	a, from := actionOf(pressed(t, h, f, "crew:res:apply"))
	if a.Action != view.ActCrewResources || a.Crew.Requests != "cpu=100m,memory=32Mi" || a.Crew.Limits != "cpu=100m,memory=32Mi" || from != FromCrewCard {
		t.Fatalf("apply = %+v", a)
	}
	if PickQoS(model.Resources{}) != model.QoSBestEffort || PickQoS(model.Resources{CPURequest: 100}) != model.QoSBurstable {
		t.Fatal("qos preview")
	}
	if !PledgeOverBinding(model.Resources{CPURequest: 200, CPULimit: 100}) || PledgeOverBinding(model.Resources{CPURequest: 200}) {
		t.Fatal("a request above its limit")
	}
}

func TestHornStepsStayInBounds(t *testing.T) {
	p := FirstHorn(crewChain())
	if p != (HornPick{Min: 3, Max: 6, CPU: HornPitch}) {
		t.Fatalf("first horn = %+v", p)
	}
	p = StepHorn(StepHorn(p, "min", true), "min", true)
	if p.Min != 5 || StepHorn(HornPick{Min: 1, Max: 1, CPU: 10}, "min", false).Min != 1 || StepHorn(HornPick{Min: 2, Max: 2}, "max", false).Max != 2 {
		t.Fatalf("floor and ceiling bounds: %+v", p)
	}
	if StepHorn(HornPick{CPU: 100}, "cpu", true).CPU != 100 || StepHorn(HornPick{CPU: 10}, "cpu", false).CPU != 10 {
		t.Fatal("pitch bounds")
	}
	h := New(nil, Backend{})
	f := crewFrame(crewSel())
	pressed(t, h, f, "crew:mode:horn")
	pressed(t, h, f, "crew:horn:max:up")
	a, _ := actionOf(pressed(t, h, f, "crew:horn:sound"))
	if a.Action != view.ActAutoscaleCrew || a.Crew != (view.CrewOrder{Min: 3, Max: 7, CPU: HornPitch}) {
		t.Fatalf("sound = %+v", a)
	}
}

func TestNewerClassesAreOwnedTagsOfTheSameTrade(t *testing.T) {
	got := NewerClasses(crewFrame(crewSel()), "porter:1.0")
	if len(got) != 1 || got[0] != "porter:1.1" {
		t.Fatalf("newer classes = %v", got)
	}
	if !soulstone.TagNewer("2.0", "1.1") || soulstone.TagNewer("1.0", "1.0") || !soulstone.TagNewer("1.10", "1.9") {
		t.Fatal("tag order")
	}
	if StepValue(CpuStep, CpuMost, 100, true) != 150 || StepValue(CpuStep, CpuMost, 0, false) != 0 || StepValue(CpuStep, CpuMost, 120, false) != 100 || StepValue(CpuStep, CpuMost, CpuMost, true) != CpuMost || StepValue(CpuStep, CpuMost, 100, false) != 50 {
		t.Fatal("step values")
	}
}

func TestCrewInspectEchoesIntoTheStoneAndOpensDetails(t *testing.T) {
	run := &fakeRunner{}
	h := New(nil, run.backend())
	intents := pressed(t, h, crewFrame(crewSel()), "crew:history")
	if len(run.commands) != 1 || run.commands[0] != "rollout history deployment/porters" {
		t.Fatalf("ran %v", run.commands)
	}
	if h.Overlay() != OverlayDetails || h.Details().Sel.Kind != SelectCrew {
		t.Fatalf("details = %+v", h.Details())
	}
	echoed := false
	for _, it := range intents {
		echoed = echoed || it.Kind == IntentEcho && it.Command == "rollout history deployment/porters"
	}
	if !echoed {
		t.Fatalf("intents = %+v", intents)
	}
	if got := DetailCommands(crewSel(), DetailYAML); len(got) != 1 || got[0] != "get deployment porters -n default -o yaml" {
		t.Fatalf("yaml tab = %v", got)
	}
	if got := DetailCommands(Selection{Kind: SelectForeman, ID: "default/porters-7d9f"}, DetailDescribe); got[0] != "describe replicaset porters-7d9f -n default" {
		t.Fatalf("foreman describe = %v", got)
	}
}

func TestCrewViewFitsAboveTheConsole(t *testing.T) {
	h := New(nil, Backend{})
	for _, mode := range []string{CrewModeChain, CrewModeImage, CrewModeRes, CrewModeHorn} {
		f := crewFrame(crewSel(), "soulvault")
		f.State.Unlocks[HornUnlock] = true
		f.State.Grimoire = []view.GrimoirePage{runes("resources.requests.cpu", "resources.limits.memory")}
		h.SetCrewMode(crewChain(), mode, f)
		h.View.Chain = crewChain().ID()
		rects := h.CrewRects(f)
		if len(rects) != 1 {
			t.Fatalf("%q shows %d panels", mode, len(rects))
		}
		box := rects[0]
		if box.Y < TopH || box.Y+box.H > ConsoleY || box.X+box.W > Width {
			t.Fatalf("%q box = %+v", mode, box)
		}
		for _, b := range h.CrewButtons(f) {
			if b.Rect.X < box.X || b.Rect.Y < box.Y || b.Rect.X+b.Rect.W > box.X+box.W || b.Rect.Y+b.Rect.H > box.Y+box.H {
				t.Errorf("%q: %s at %+v leaves the card %+v", mode, b.ID, b.Rect, box)
			}
		}
		if !h.Captures(box.X+2, box.Y+2, f) {
			t.Errorf("%q: the card does not hold its clicks", mode)
		}
	}
}

func TestCrewCardsForTheDeploymentAndItsForemen(t *testing.T) {
	d := CrewCard(crewFrame(crewSel()))
	if d.K8s != "Deployment/porters" || !strings.Contains(d.Lines[0].Text, "WANTS 3") || !strings.Contains(d.Lines[3].Text, "HISTORY 1 OF 10") {
		t.Fatalf("deployment card = %+v", d)
	}
	fm := CrewCard(crewFrame(Selection{Kind: SelectForeman, ID: "default/porters-54cd"}))
	if fm.K8s != "ReplicaSet/porters-54cd" || fm.Portrait != ForemanPortrait || !strings.Contains(fm.Lines[len(fm.Lines)-1].Text, "old revision") {
		t.Fatalf("foreman card = %+v", fm)
	}
	if SelectionTitle(Selection{Kind: SelectForeman, ID: "default/porters-54cd"}) != "Shift - porters-54cd" {
		t.Fatal("details title")
	}
	if !strings.Contains(DeployRowText(crewChain()), "RollingUpdate 25%/25%") || !strings.Contains(HistoryRowText(crewChain()), "rev 1") {
		t.Fatal("row texts")
	}
}

func TestTheCrewCardDrawsInBothNamings(t *testing.T) {
	lib := gfx.Load(art.Build())
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	for _, k8s := range []bool{false, true} {
		if k8s {
			ui.SetNamer(lexicon.TrueNames)
		}
		for _, mode := range []string{CrewModeChain, CrewModeImage, CrewModeRes, CrewModeHorn} {
			h := New(lib, Backend{})
			f := crewFrame(crewSel(), "soulvault")
			f.State.Grimoire = []view.GrimoirePage{runes("resources.requests.cpu")}
			h.SetCrewMode(crewChain(), mode, f)
			h.DrawCrewCard(dst, f)
			h.DrawSelection(dst, f)
		}
	}
	ui.SetNamer(nil)
}

func TestBareForemenAndHornedCrewsReadRight(t *testing.T) {
	bare := model.CrewChain{Bare: true, Namespace: "default", Foremen: []model.Foreman{{Namespace: "default", Name: "lantern", Newest: true, Desired: 2, Ready: 1}}}
	if title, kind := CrewTitle(bare); title != "SHIFT lantern" || !strings.Contains(kind, "no crew") {
		t.Fatalf("bare title %q %q", title, kind)
	}
	if sel := CrewSelection(bare); sel.Kind != SelectForeman || sel.ID != "default/lantern" {
		t.Fatalf("bare selection %+v", sel)
	}
	if !strings.Contains(ForemanRole(bare, bare.Foremen[0]), "bare ReplicaSet") || TallyInk(bare.Foremen[0]) != ui.Palette.GoldBright {
		t.Fatal("a bare shift's card")
	}
	if !strings.Contains(ForemanRole(crewChain(), crewChain().Foremen[0]), "Scale the crew") || TallyInk(model.Foreman{}) != ui.Palette.TextDim {
		t.Fatal("a standing shift's card")
	}
	c := crewChain()
	c.Horn = &model.Horn{Name: "porters", Target: "porters", Min: 2, Max: 8, Goal: 60, Current: 3, Utilization: 75, Measured: true}
	if p := FirstHorn(c); p != (HornPick{Min: 2, Max: 8, CPU: 60}) {
		t.Fatalf("horned pick %+v", p)
	}
	if b := HornOrder(c, FirstHorn(c)); b.ID != "crew:horn:silence" || !strings.Contains(b.Tip, "kubectl delete hpa porters") {
		t.Fatalf("horned order %+v", b)
	}
	if !strings.Contains(HornLine(c.Horn), "Floor 2, Ceiling 8, Pitch 60%") {
		t.Fatal(HornLine(c.Horn))
	}
	if tips := crewButtonMap(ScaleButtons(c, crewFrame(crewSel()))); !strings.Contains(tips["crew:scale:up"].Tip, "overrule") {
		t.Fatal("scaling a horned crew does not warn")
	}
	c.MaxSurge, c.Strategy = "", "Recreate"
	if StrategyLine(c) != "RECREATE" || !strings.HasSuffix(DeployRowText(c), "Recreate") {
		t.Fatal("recreate strategy")
	}
	if BareCommand("kubectl get pods") != "get pods" || BareCommand("etcdctl endpoint status") != "etcdctl endpoint status" {
		t.Fatal("bare command")
	}
	if _, ok := FieldNamed("nope"); ok {
		t.Fatal("unknown field")
	}
	h := New(nil, Backend{})
	if h.CrewMode(c) != CrewModeChain {
		t.Fatal("a fresh card opens on the chain")
	}
}

func TestHornedAndBareCardsDraw(t *testing.T) {
	lib := gfx.Load(art.Build())
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h := New(lib, Backend{})
	f := crewFrame(crewSel())
	f.Crews[0].Horn = &model.Horn{Name: "porters", Target: "porters", Min: 2, Max: 8, Goal: 60}
	f.Crews[0].Paused = true
	h.SetCrewMode(f.Crews[0], CrewModeHorn, f)
	h.DrawCrewCard(dst, f)
	bare := model.CrewChain{Bare: true, Namespace: "default", Foremen: []model.Foreman{{Namespace: "default", Name: "lantern", Newest: true, Pods: []string{"lantern-a"}}}}
	g := Frame{State: f.State, Selection: Selection{Kind: SelectForeman, ID: "default/lantern"}, Crews: []model.CrewChain{bare}}
	h.Hover, h.Hovering = Button{ID: "crew:pick:pod:lantern-a"}, true
	h.DrawCrewCard(dst, g)
	h.DrawSelection(dst, g)
}

func TestALongForemanNameSplitsBetweenTheTitleAndTheReferenceWithoutClipping(t *testing.T) {
	defer ui.SetNamer(nil)
	fm := model.Foreman{Name: "porters-7dd7c854b8", Desired: 3, Ready: 3}
	chars := ui.MaxChars(CardHeadWidth(), 1)
	for _, mode := range namesModes() {
		mode.apply()
		c := ForemanCard(model.CrewChain{}, fm)
		title, ref := SplitCardHead(c.Title, c.K8s, chars)
		for _, text := range []string{title, ref} {
			if strings.Contains(text, "...") || len([]rune(ui.Name(text))) > chars {
				t.Errorf("[%s] head %q / %q clips in %d chars", mode.name, title, ref, chars)
			}
		}
		if !strings.Contains(strings.ToLower(title+" "+ref), "porters-7dd7c854b8") {
			t.Errorf("[%s] the name is lost: %q / %q", mode.name, title, ref)
		}
	}
}

func TestTheBareForemanSubtitleUsesStoryWordsInStoryMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(nil)
	_, sub := CrewTitle(model.CrewChain{Bare: true, Foremen: []model.Foreman{{Name: "lantern"}}})
	if strings.Contains(sub, "ReplicaSet") {
		t.Errorf("story subtitle leaks Kubernetes: %q", sub)
	}
}
