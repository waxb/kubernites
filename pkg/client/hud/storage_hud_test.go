package hud

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func courierPod() model.PodView {
	return model.PodView{Namespace: "default", Name: "couriers-a", Node: "stone-keep-2", Image: "courier:1.0", State: model.HearthReady}
}

func storageFrame(t *testing.T) Frame {
	f := fullFrame(t)
	f.Pods = append(f.Pods, courierPod(),
		model.PodView{Namespace: "default", Name: "archive-0", Node: "lone-tower", Image: "archivist:1.0", State: model.HearthReady, Claims: []string{"books-archive-0"}})
	f.State.Keeps = []view.KeepView{{Name: "lone-tower", Daemons: []view.DaemonOnKeep{{Name: "warden-x", Ready: true}}}, {Name: "stone-keep-2", Stranded: 4}}
	f.State.DaemonSets = []view.DaemonSetView{{Name: "warden", Namespace: "default", Image: "warden:1.0", Desired: 2, Current: 2, Ready: 1, Updated: 2}}
	f.State.StatefulSets = []view.StatefulSetView{{Name: "archive", Replicas: 2, Ready: 1, Pods: []view.StatefulPodView{
		{Name: "archive-0", Ordinal: 0, Claim: "books-archive-0", Keep: "lone-tower", Phase: "Running"},
		{Name: "archive-1", Ordinal: 1, Claim: "books-archive-1", Phase: "Pending"},
	}}}
	f.State.Storage = view.StorageView{
		Vaults: []view.VaultView{{Name: "vault-1", Isle: "home", Zone: "isle-home", Shelves: 2, Phase: "Bound", Claim: "default/books-archive-0", Reclaim: "Retain"}},
		Deeds: []view.DeedView{
			{Name: "books-archive-0", Shelves: 2, Phase: "Bound", Volume: "vault-1", Class: "vault-wrights"},
			{Name: "books-archive-1", Shelves: 2, Phase: "Pending", Waiting: "waiting for first consumer to be created before binding"},
		},
	}
	return f
}

func vaultStructure() view.Structure {
	return view.Structure{ID: "vault-1", Kind: view.StructVault, State: view.StateReady, Label: "Vault vault-1", W: 2, H: 2,
		Object: view.ObjectRef{Kind: "PersistentVolume", Name: "vault-1"}}
}

func TestAWardenlessKeepOffersPlaceWardenOnceCouriersCame(t *testing.T) {
	f := selectStructure(storageFrame(t), stoneKeep())
	h := New(nil, (&fakeRunner{}).backend())
	wantCells(t, h, f, map[string]int{"act:" + KeyWarden: 7})
	a, ok := ActionByKey(SelectionActions(f), KeyWarden)
	if !ok || a.Act.Action != view.ActPlaceWarden || a.Act.Keep != "stone-keep-2" || !strings.HasPrefix(a.Command, "run warden-stone-keep-2 ") {
		t.Fatalf("warden action = %+v", a)
	}
	intents := h.RunAction(a, f.Selection, CardSource(f))
	if len(intents) == 0 || intents[0].Kind != IntentAction || intents[0].Action.Keep != "stone-keep-2" {
		t.Fatalf("intents = %+v", intents)
	}
	tower := stoneKeep()
	tower.Object.Name = "lone-tower"
	if _, ok := ActionByKey(SelectionActions(selectStructure(storageFrame(t), tower)), KeyWarden); ok {
		t.Error("a Keep with its warden offers no second one")
	}
	g := storageFrame(t)
	g.Pods = fixturePods()
	if _, ok := ActionByKey(SelectionActions(selectStructure(g, stoneKeep())), KeyWarden); ok {
		t.Error("no post before the couriers come")
	}
}

func TestThePlaceWardenLabelSpeaksBothModes(t *testing.T) {
	f := selectStructure(storageFrame(t), stoneKeep())
	a, _ := ActionByKey(SelectionActions(f), KeyWarden)
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	b, _ := ActionByKey(SelectionActions(f), KeyWarden)
	if a.Label != "PLACE WARDEN" || b.Label == a.Label || !strings.Contains(b.Tip, "nodeSelector") {
		t.Fatalf("story %q, names %q (%q)", a.Label, b.Label, b.Tip)
	}
}

func TestTheCrewsListCarriesDaemonSetsAndStatefulCrewsWithAKindBadge(t *testing.T) {
	f := storageFrame(t)
	rows := CrewsRows(f)
	kinds := map[string]string{}
	for _, r := range rows {
		kinds[r.Id()] = r.KindWord()
	}
	if kinds["ds:warden"] != "WARDENS" || kinds["sts:archive"] != "STATEFUL CREW" || kinds[f.Crews[0].ID()] != "CREW" {
		t.Fatalf("kinds = %v", kinds)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	for _, r := range CrewsRows(f) {
		if r.Id() == "ds:warden" && r.KindWord() != "DAEMONSET" || r.Id() == "sts:archive" && r.KindWord() != "STATEFULSET" {
			t.Errorf("NAMES badge for %s = %q", r.Id(), r.KindWord())
		}
	}
}

func TestTheCrewsListOffersToRaiseTheWardensOnceCouriersCame(t *testing.T) {
	f := storageFrame(t)
	f.State.DaemonSets = nil
	h := New(nil, (&fakeRunner{}).backend())
	var raise *CrewsRow
	for _, r := range CrewsRows(f) {
		if r.Id() == DsRaiseID {
			raise = &r
		}
	}
	if raise == nil {
		t.Fatal("no row to raise the wardens")
	}
	intents := h.PressCrews("raise-ds", f)
	if len(intents) == 0 || intents[0].Kind != IntentAction || intents[0].Action.Action != view.ActRaiseDaemonSet {
		t.Fatalf("intents = %+v", intents)
	}
	f.Pods = fixturePods()
	for _, r := range CrewsRows(f) {
		if r.Id() == DsRaiseID {
			t.Fatal("the wardens are offered before any courier came")
		}
	}
}

func TestTheDaemonSetCardShowsItsCountsAndRunsDescribeAndGet(t *testing.T) {
	f := storageFrame(t)
	run := &fakeRunner{}
	h := New(nil, run.backend())
	h.PressCrews("open:ds:warden", f)
	if h.KindCard != "ds:warden" {
		t.Fatalf("kind card = %q", h.KindCard)
	}
	c, ok := KindCard(f, "ds:warden")
	if !ok || c.Title != "WARDEN" || c.K8s != "daemonset.apps/warden" {
		t.Fatalf("card = %+v", c)
	}
	if !containsLine(c.Lines, "DESIRED 2  CURRENT 2  READY 1  UP-TO-DATE 2") || !containsLine(c.Lines, "warden:1.0") {
		t.Fatalf("lines = %+v", c.Lines)
	}
	h.PressKind("describe", f)
	h.PressKind("get", f)
	if !slices.Contains(run.commands, "describe daemonset warden") || !slices.Contains(run.commands, "get daemonset warden") {
		t.Fatalf("commands = %v", run.commands)
	}
}

func TestTheStatefulCrewCardListsItsOrdinalsWithKeepAndDeed(t *testing.T) {
	f := storageFrame(t)
	c, ok := KindCard(f, "sts:archive")
	if !ok || c.K8s != "statefulset.apps/archive" {
		t.Fatalf("card = %+v", c)
	}
	if !containsLine(c.Lines, "archive-0  lone-tower  books-archive-0") || !containsLine(c.Lines, "archive-1  (pending)  books-archive-1") {
		t.Fatalf("lines = %+v", c.Lines)
	}
}

func TestTheVaultCardNamesItsDeedAndRunsTheirVerbs(t *testing.T) {
	f := selectStructure(storageFrame(t), vaultStructure())
	c := StructureCard(f.Selection.Structure, f)
	if !containsLine(c.Lines, "BOUND  2,048 BARRELS  ISLE home") || !containsLine(c.Lines, "DEED books-archive-0") {
		t.Fatalf("vault card = %+v", c.Lines)
	}
	cmds := map[string]string{}
	for _, a := range SelectionActions(f) {
		cmds[a.Key] = a.Command
	}
	want := map[string]string{KeyDescribe: "describe pv vault-1", KeyGet: "get pv vault-1", KeyDescribeDeed: "describe pvc books-archive-0", KeyGetDeed: "get pvc books-archive-0"}
	for k, v := range want {
		if cmds[k] != v {
			t.Errorf("%s = %q, want %q", k, cmds[k], v)
		}
	}
	h := New(nil, (&fakeRunner{}).backend())
	wantCells(t, h, f, map[string]int{"act:" + KeyDescribe: 0, "act:" + KeyGet: 1, "act:" + KeyDescribeDeed: 3, "act:" + KeyGetDeed: 4})
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	c = StructureCard(f.Selection.Structure, f)
	if !containsLine(c.Lines, "Bound  2Gi  zone isle-home") || !containsLine(c.Lines, "CLAIM default/books-archive-0") {
		t.Fatalf("NAMES vault card = %+v", c.Lines)
	}
}

func TestAHutHoldingADeedShowsItAndDescribesIt(t *testing.T) {
	f := storageFrame(t)
	p := f.Pods[len(f.Pods)-1]
	f.Selection = Selection{Kind: "hearth", ID: "default/" + p.Name, Pod: p}
	h := New(nil, (&fakeRunner{}).backend())
	c := h.CardFor(f)
	if !containsLine(c.Lines, "DEED books-archive-0 IN vault-1") {
		t.Fatalf("hut card = %+v", c.Lines)
	}
	a, ok := ActionByKey(SelectionActions(f), KeyDescribeDeed)
	if !ok || a.Command != "describe pvc books-archive-0" {
		t.Fatalf("deed action = %+v", a)
	}
	wantCells(t, h, f, map[string]int{"act:" + KeyDescribeDeed: 7})
	f.Pods[len(f.Pods)-1].Claims = []string{"books-archive-1"}
	f.Selection.Pod = f.Pods[len(f.Pods)-1]
	if c := h.CardFor(f); !containsLine(c.Lines, "DEED books-archive-1 WAITING: waiting for first consumer to be created before binding") {
		t.Fatalf("waiting deed = %+v", c.Lines)
	}
}

func TestTheStoragePagesHaveTheirIconsInTheGrimoire(t *testing.T) {
	for _, id := range art.StoragePageIcons {
		if gfx.PageIconName(id) != art.PageIconName(id) {
			t.Errorf("page %s falls back to a blank icon", id)
		}
		if l := LabelOf(view.GrimoirePage{ID: id}); l.Tag == "" || l.Docs == "" {
			t.Errorf("page %s has no plaque label", id)
		}
	}
}

func containsLine(lines []Line, text string) bool {
	for _, l := range lines {
		if strings.Contains(l.Text, text) {
			return true
		}
	}
	return false
}

func TestTheKindCardDrawsWithItsButtons(t *testing.T) {
	f := storageFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.PressCrews("open:sts:archive", f)
	ids := []string{}
	for _, b := range h.Buttons(f) {
		ids = append(ids, b.ID)
	}
	if !slices.Contains(ids, KindKey+":describe") || !slices.Contains(ids, KindKey+":close") {
		t.Fatalf("buttons = %v", ids)
	}
	h.Draw(ebiten.NewImage(Width, Height), f)
	h.PressKind("close", f)
	if h.KindShown(f) {
		t.Fatal("the card stays open after close")
	}
}

func TestKindCardsKeepObjectNamesLiteralInNamesMode(t *testing.T) {
	f := storageFrame(t)
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	c, _ := KindCard(f, "ds:warden")
	if ui.Name(c.K8s) != "daemonset.apps/warden" || ui.Name(c.Title) != "WARDEN" || !containsLine([]Line{{Text: ui.Name(c.Lines[1].Text)}}, "warden:1.0") {
		t.Fatalf("names mode renamed the object: %q %q %q", ui.Name(c.K8s), ui.Name(c.Title), ui.Name(c.Lines[1].Text))
	}
	s, _ := KindCard(f, "sts:archive")
	if got := ui.Name(s.Lines[len(s.Lines)-1].Text); got != "archive-1  (pending)  books-archive-1" {
		t.Fatalf("pod line = %q", got)
	}
}

func TestTheKindCardIsTallEnoughForItsWrappedLines(t *testing.T) {
	f := storageFrame(t)
	c, _ := KindCard(f, "sts:archive")
	rows := 0
	for _, l := range c.Lines {
		rows += len(ui.Font{}.WrapLines(l.Text, KindW-2*KindPad, 1, 99))
	}
	if got := KindRect(f, c).H; got < KindHead+rows*ui.LineHeight+KindBtnH+2*KindPad {
		t.Fatalf("card height %d for %d rows", got, rows)
	}
}

func TestTheVaultCardShowsTheVaultItself(t *testing.T) {
	f := selectStructure(storageFrame(t), vaultStructure())
	if got := StructureCard(f.Selection.Structure, f).Portrait; got != SpritePortrait+art.VaultName("bound", 1) {
		t.Fatalf("portrait = %q", got)
	}
}

func TestALiveVaultMeasuresItsSizeInBarrels(t *testing.T) {
	defer ui.SetNamer(nil)
	f := selectStructure(storageFrame(t), vaultStructure())
	v, ok := VaultOf(f, f.Selection.Structure.Object.Name)
	if !ok {
		t.Fatal("no vault in the frame")
	}
	want := strings.ToUpper(view.Barrels(float64(v.Shelves * 1024)))
	if got := VaultLines(f.Selection.Structure, f)[0].Text; !strings.Contains(got, want) || strings.Contains(got, "SHELVES") {
		t.Errorf("a live Vault reads %q in story mode, want its size in barrels (%s)", got, want)
	}
	ui.SetNamer(names.Kubernetes(64))
	if got := VaultLines(f.Selection.Structure, f)[0].Text; !strings.Contains(got, fmt.Sprintf("%dGi", v.Shelves)) {
		t.Errorf("NAMES mode reads %q", got)
	}
}
