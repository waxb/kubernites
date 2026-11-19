package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func archiveFrame(t *testing.T, pages ...string) Frame {
	f := fullFrame(t)
	f.Pods = append(f.Pods, model.PodView{Namespace: "default", Name: "archive-7f9c-x2k", Node: "lone-tower", Image: "archivist:1.0", State: model.HearthReady})
	f.State.RealmTime = 1000
	f.State.Archive = &view.ArchiveView{Unsettled: 1200000, Seals: 3400, SettleIn: 200, Pace: 20, Writers: 1, Books: 1}
	for _, id := range pages {
		f.State.Grimoire = append(f.State.Grimoire, view.GrimoirePage{ID: id, State: "unlocked", Rank: 1})
	}
	return f
}

func verbIDs(h *HUD, f Frame) []string {
	out := []string{}
	for _, b := range h.kindButtons(f) {
		if strings.HasPrefix(b.ID, KindKey+":"+verbPrefix) {
			out = append(out, strings.TrimPrefix(b.ID, KindKey+":"+verbPrefix))
		}
	}
	return out
}

func TestTheMunimentRoomRowOpensTheArchiveCard(t *testing.T) {
	f := archiveFrame(t)
	found := false
	for _, r := range CrewsRows(f) {
		found = found || r.Id() == archiveCardID
	}
	if !found {
		t.Fatal("the crews list has no archive row once the archive stands")
	}
	c, ok := KindCard(f, archiveCardID)
	if !ok || c.Title != "THE MUNIMENT ROOM" {
		t.Fatalf("card = %+v", c)
	}
	for _, want := range []string{"UNSETTLED 1.2M GOLD IN 3.4K SEALS", "SETTLE IN 3:20", "20 SEALS A SECOND", "LOOSE PAGES IN THE HUT"} {
		if !containsLine(c.Lines, want) {
			t.Errorf("no line with %q in %+v", want, c.Lines)
		}
	}
	f.State.Archive = nil
	if _, ok := KindCard(f, archiveCardID); ok {
		t.Fatal("the card shows before the Archive stands")
	}
}

func TestTheArchiveCardNamesItsLastLoss(t *testing.T) {
	f := archiveFrame(t)
	f.State.Archive.LastLoss, f.State.Archive.LostSeals, f.State.Archive.LostAt, f.State.Archive.LostCause = 15000000, 3400, 750, "emptyDir"
	c, _ := KindCard(f, archiveCardID)
	if !containsLine(c.Lines, "LAST LOSS 3.4K SEALS, -15M GOLD, 4:10 AGO") {
		t.Fatalf("lines = %+v", c.Lines)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	c, _ = KindCard(f, archiveCardID)
	if !containsLine(c.Lines, "emptyDir deleted with its Pod") {
		t.Fatalf("NAMES lines = %+v", c.Lines)
	}
}

func TestTheArchiveCardSaysWhereTheBooksAreKept(t *testing.T) {
	f := archiveFrame(t)
	f.Pods[len(f.Pods)-1].Claims = []string{"books"}
	c, _ := KindCard(f, archiveCardID)
	if !containsLine(c.Lines, "BOOKS IN DEED books") {
		t.Fatalf("lines = %+v", c.Lines)
	}
	f.State.StatefulSets = []view.StatefulSetView{{Name: view.ArchiveName, Replicas: 1, Ready: 1}}
	c, _ = KindCard(f, archiveCardID)
	if !containsLine(c.Lines, "EACH ARCHIVIST KEEPS ITS OWN DEED") {
		t.Fatalf("lines = %+v", c.Lines)
	}
}

func TestTheArchiveCardOffersEachStorageVerbOnlyOnceItsPageIsLearned(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.KindCard = archiveCardID
	if got := verbIDs(h, archiveFrame(t)); len(got) != 0 {
		t.Fatalf("verbs before any page = %v", got)
	}
	f := archiveFrame(t, view.VaultsPage)
	if got := verbIDs(h, f); !slices.Equal(got, []string{verbDig, verbDeed}) {
		t.Fatalf("verbs with the Vaults = %v", got)
	}
	f = archiveFrame(t, view.VaultsPage, view.StatefulCrewsPage, view.GuildPage)
	if got := verbIDs(h, f); !slices.Equal(got, []string{verbDig, verbDeed, verbStateful, verbGuild}) {
		t.Fatalf("verbs with every page = %v", got)
	}
	f.State.Storage.Classes = []view.StorageClassView{{Name: view.GuildClass, Default: true}}
	f.State.StatefulSets = []view.StatefulSetView{{Name: view.ArchiveName, Replicas: 1}}
	if got := verbIDs(h, f); len(got) != 0 {
		t.Fatalf("verbs once the guild and the Stateful Crew stand = %v", got)
	}
}

func TestTheStorageVerbsSendTheirRealmActions(t *testing.T) {
	f := archiveFrame(t, view.VaultsPage, view.StatefulCrewsPage, view.GuildPage)
	want := map[string]view.Action{
		verbDig:      {Action: view.ActDigVault, Shelves: booksShelves},
		verbDeed:     {Action: view.ActWriteDeed, Name: booksDeed, Shelves: booksShelves},
		verbStateful: {Action: view.ActRaiseStatefulSet, From: view.ArchiveName},
		verbGuild:    {Action: view.ActFoundGuild},
	}
	for verb, act := range want {
		h := New(nil, (&fakeRunner{}).backend())
		h.KindCard = archiveCardID
		intents := h.PressKind(verbPrefix+verb, f)
		if len(intents) == 0 || intents[0].Kind != IntentAction || intents[0].Action != act || intents[0].From != FromButton {
			t.Errorf("%s sent %+v, want %+v", verb, intents, act)
		}
	}
}

func TestTheStorageVerbsSpeakBothModes(t *testing.T) {
	f := archiveFrame(t, view.VaultsPage, view.StatefulCrewsPage, view.GuildPage)
	h := New(nil, (&fakeRunner{}).backend())
	h.KindCard = archiveCardID
	labels := func() []string {
		out := []string{}
		for _, b := range h.kindButtons(f) {
			if strings.HasPrefix(b.ID, KindKey+":"+verbPrefix) {
				out = append(out, ui.Name(b.Label))
				if b.Tip == "" {
					t.Errorf("%s has no tip", b.ID)
				}
			}
		}
		return out
	}
	if got := labels(); !slices.Equal(got, []string{"DIG VAULT", "WRITE DEED", "RAISE STATEFUL CREW", "FOUND GUILD"}) {
		t.Fatalf("story labels = %v", got)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if got := labels(); !slices.Equal(got, []string{"CREATE PV", "CREATE PVC", "TO STATEFULSET", "CREATE STORAGECLASS"}) {
		t.Fatalf("NAMES labels = %v", got)
	}
}

func TestTheArchiveCardDrawsItsVerbsInsideItsFrame(t *testing.T) {
	f := archiveFrame(t, view.VaultsPage, view.StatefulCrewsPage, view.GuildPage)
	h := New(nil, (&fakeRunner{}).backend())
	h.PressCrews("open:"+archiveCardID, f)
	if h.KindCard != archiveCardID {
		t.Fatalf("kind card = %q", h.KindCard)
	}
	box, ok := kindWindow(h, f)
	if !ok {
		t.Fatal("the card is not shown")
	}
	for _, b := range h.kindButtons(f) {
		r := b.Rect
		if r.X < box.X || r.Y < box.Y || r.X+r.W > box.X+box.W || r.Y+r.H > box.Y+box.H {
			t.Errorf("%s at %+v falls outside the card %+v", b.ID, r, box)
		}
	}
	h.Draw(ebiten.NewImage(Width, Height), f)
	h.PressKind("describe", f)
	if h.DetailsCard.Lines[0] != "$ kubectl describe deployment archive" {
		t.Fatalf("describe ran %q", h.DetailsCard.Lines[0])
	}
}

func TestTheCrewsListOffersToRaiseTheMunimentRoomOnceSealwrightsCame(t *testing.T) {
	f := fullFrame(t)
	has := func(f Frame) bool {
		for _, r := range CrewsRows(f) {
			if r.Id() == archiveRaiseID {
				return true
			}
		}
		return false
	}
	if has(f) {
		t.Fatal("the Muniment Room is offered before any sealwright came")
	}
	f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: "sealwright:1.0", Owned: true})
	if !has(f) {
		t.Fatal("no row to raise the Muniment Room once the sealwrights came")
	}
	h := New(nil, (&fakeRunner{}).backend())
	intents := h.PressCrews("raise-archive", f)
	if len(intents) == 0 || intents[0].Kind != IntentAction || intents[0].Action.Action != view.ActRaiseArchive {
		t.Fatalf("intents = %+v", intents)
	}
	f.State.Archive = &view.ArchiveView{}
	if has(f) {
		t.Fatal("the raise row stays once the archive stands")
	}
}

func TestTheRaiseRowNamesWhatUnregisteredSealsCost(t *testing.T) {
	f := fullFrame(t)
	if got := unregisteredNote(f); got != "no Muniment Room yet" {
		t.Fatalf("note = %q", got)
	}
	f.State.SealsUnpaid = 12000
	if got := unregisteredNote(f); got != "seals unregistered: -12K gold/s" {
		t.Fatalf("note = %q", got)
	}
}

func TestTheArchiveCardNamesTheGuildOnceItStands(t *testing.T) {
	f := archiveFrame(t)
	c, _ := KindCard(f, archiveCardID)
	if containsLine(c.Lines, "GUILD") {
		t.Fatal("a guild line before the guild")
	}
	f.State.Storage.Classes = []view.StorageClassView{{Name: view.GuildClass, BindingMode: "WaitForFirstConsumer", Reclaim: "Delete", Default: true}}
	c, _ = KindCard(f, archiveCardID)
	if !containsLine(c.Lines, "THE VAULT-WRIGHTS' GUILD DIGS A VAULT FOR EACH DEED") {
		t.Fatalf("lines = %+v", c.Lines)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	c, _ = KindCard(f, archiveCardID)
	if !containsLine(c.Lines, "StorageClass vault-wrights (default): WaitForFirstConsumer, reclaimPolicy Delete") {
		t.Fatalf("NAMES lines = %+v", c.Lines)
	}
}

func TestTheGuideLightsTheBooksRowThenTheVerb(t *testing.T) {
	f := archiveFrame(t, view.VaultsPage)
	f.Guide.Button = KindKey + ":" + verbPrefix + verbDig
	h := New(nil, (&fakeRunner{}).backend())
	h.CrewsList.Open = true
	lit := func(buttons []Button, id string) bool {
		for _, b := range buttons {
			if b.ID == id {
				return b.Selected
			}
		}
		return false
	}
	if rows := h.crewsButtons(f); len(rows) == 0 || !lit(rows, CrewsKey+":open:"+archiveCardID) {
		t.Fatalf("with the card closed the BOOKS row is not lit: %+v", rows)
	}
	h.KindCard = archiveCardID
	if lit(h.crewsButtons(f), CrewsKey+":open:"+archiveCardID) {
		t.Fatal("the BOOKS row stays lit once the card is open")
	}
	if !lit(h.kindButtons(f), f.Guide.Button) {
		t.Fatal("the open card does not light DIG VAULT")
	}
}

func TestTheGuidePointsAtTheBooksRowThenAtTheVerb(t *testing.T) {
	f := archiveFrame(t, view.VaultsPage)
	f.Guide.Button = KindKey + ":" + verbPrefix + verbDig
	h := New(nil, (&fakeRunner{}).backend())
	h.CrewsList.Open = true
	if got := h.chooserGoal(CmdCrew, f); got != CrewsKey+":open:"+archiveCardID {
		t.Fatalf("with the card closed the guide points at %q", got)
	}
	h.KindCard = archiveCardID
	if got := h.chooserGoal(CmdCrew, f); got != f.Guide.Button {
		t.Fatalf("with the card open the guide points at %q", got)
	}
	if r, ok := h.StepRect(f.Guide.Button, f); !ok || r.W == 0 {
		t.Fatal("the guide cannot find DIG VAULT on screen")
	}
	f.Pods = append(f.Pods, model.PodView{Namespace: "default", Name: "couriers-1", Image: courierImage + ":1.0", State: model.HearthReady})
	f.Guide.Button = CrewsKey + ":raise-ds"
	h.KindCard = ""
	if _, ok := HitID(h.crewsButtons(f), f.Guide.Button); !ok {
		t.Fatal("the fixture shows no Daemons row")
	}
	if got := h.chooserGoal(CmdCrew, f); got != f.Guide.Button {
		t.Fatalf("a try's crews button is not the guide's goal: %q", got)
	}
}
