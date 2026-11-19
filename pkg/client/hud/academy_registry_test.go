package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

func registryFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Gold = 1e9
	f.State.Soulstones = []view.SoulstoneEntry{
		{Image: "porter:1.0", Name: "Porters, class of 1.0", Owned: true, Visible: true},
		{Image: "porter:1.1", Name: "Porters, class of 1.1", Cost: 150, Visible: true},
		{Image: "porter:2.0", Name: "Porters, class of 2.0", Cost: 3000, Visible: true},
		{Image: "porter:2.1", Name: "Porters, class of 2.1", Owned: true, Visible: true, Cursed: true},
		{Image: "porter:3.0", Name: "Porters, class of 3.0", Cost: 9000, Visible: true},
		{Image: "porter:latest", Name: "Porters, class of latest", Cost: 27000, Visible: true},
		{Image: "clerk:1.0", Name: "Clerks, class of 1.0", Cost: 200, Visible: true},
		{Image: "clerk:1.1", Name: "Clerks, class of 1.1", Cost: 400, Visible: true},
		{Image: "raven:1.0", Name: "Ravens, class of 1.0", Cost: 120, Visible: true},
	}
	f.Guide = Guide{}
	f.State.Objectives = nil
	return f
}

func TestTheRegistryListsTradesBeforeFamiliarsAndEachClassInItsProgression(t *testing.T) {
	f := registryFrame(t)
	repos := academyRepos(f)
	names := []string{}
	for _, r := range repos {
		names = append(names, r.Profession)
	}
	if strings.Join(names, ",") != "Porter,Clerk,Raven" {
		t.Fatalf("repositories = %v, want trades first and the familiar last", names)
	}
	tags := []string{}
	for _, c := range repos[0].Classes {
		tags = append(tags, tagOf(c.Image))
	}
	if strings.Join(tags, ",") != "1.0,1.1,2.0,2.1,3.0,latest" {
		t.Fatalf("the porter classes read %v, want the progression with the rumour after 2.0", tags)
	}
	if repos[0].Endowed != 1 {
		t.Fatalf("the rumoured class counts as pulled: %d", repos[0].Endowed)
	}
	if !repos[2].Familiar || repos[0].Familiar {
		t.Fatalf("familiar flags: %+v", repos)
	}
}

func TestTheFilterChipsKeepTradesAndFamiliarsApart(t *testing.T) {
	f := registryFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.Open(OverlaySoulvault)
	clickOn(t, h, f, "acad:filter:familiars")
	v := h.academyView(f)
	if len(v.Repos) != 1 || v.Repos[0].Profession != "Raven" || v.Filter != acadFilterFamiliars {
		t.Fatalf("FAMILIARS shows %+v", v.Repos)
	}
	if _, ok := HitID(h.AcademyButtons(f), "vault:raven:1.0"); !ok {
		t.Fatal("the familiar's class has no PULL button on the FAMILIARS filter")
	}
	clickOn(t, h, f, "acad:filter:trades")
	v = h.academyView(f)
	if len(v.Repos) != 2 {
		t.Fatalf("TRADES shows %d repositories", len(v.Repos))
	}
	clickOn(t, h, f, "acad:filter:all")
	if got := len(h.academyView(f).Repos); got != 3 {
		t.Fatalf("ALL shows %d repositories", got)
	}
}

func TestChoosingARepositoryShowsItsClassesAndTheTabsSwitch(t *testing.T) {
	f := registryFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.Open(OverlaySoulvault)
	if _, ok := HitID(h.AcademyButtons(f), "vault:porter:1.1"); !ok {
		t.Fatal("the first repository with a class on offer is not open on its CLASSES")
	}
	if _, ok := HitID(h.AcademyButtons(f), "vault:clerk:1.0"); ok {
		t.Fatal("the clerks' classes show while the porters are open")
	}
	clickOn(t, h, f, "acad:repo:Clerk")
	if _, ok := HitID(h.AcademyButtons(f), "vault:clerk:1.0"); !ok {
		t.Fatal("choosing the clerks did not show their classes")
	}
	clickOn(t, h, f, "acad:tab:overview")
	if _, ok := HitID(h.AcademyButtons(f), "vault:clerk:1.0"); ok {
		t.Fatal("OVERVIEW still lists the classes")
	}
	clickOn(t, h, f, "acad:tab:classes")
	if _, ok := HitID(h.AcademyButtons(f), "vault:clerk:1.0"); !ok {
		t.Fatal("CLASSES did not bring the table back")
	}
}

func TestEndowStillSendsTheSameAction(t *testing.T) {
	f := registryFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.Open(OverlaySoulvault)
	it, ok := hasIntent(clickOn(t, h, f, "vault:porter:1.1"), IntentAction)
	if !ok || it.Action.Action != "buySoulstone" || it.Action.Image != "porter:1.1" {
		t.Fatalf("PULL = %+v", it)
	}
}

func TestTheCommandUnderTheTableNamesTheRealCrewAndOpensTheStone(t *testing.T) {
	f := registryFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.Open(OverlaySoulvault)
	if b, ok := HitID(h.AcademyButtons(f), "acad:cmd"); ok {
		t.Fatalf("the command names porter:1.1, which nobody pulled yet: %q", b.Label)
	}
	clickOn(t, h, f, "acad:pick:porter:1.0")
	b, ok := HitID(h.AcademyButtons(f), "acad:cmd")
	if !ok {
		t.Fatalf("no command under the table for a pulled class: %v", buttonIDs(h.AcademyButtons(f)))
	}
	if b.Label != "kubectl set image deployment/<crew> porter=porter:1.0" {
		t.Fatalf("command = %q", b.Label)
	}
	clickOn(t, h, f, "acad:pick:porter:2.0")
	if b, ok := HitID(h.AcademyButtons(f), "acad:cmd"); ok {
		t.Fatalf("picking an unpulled class still offers %q", b.Label)
	}
	clickOn(t, h, f, "acad:pick:porter:1.0")
	clickOn(t, h, f, "acad:cmd")
	if h.Overlay() != OverlayStone {
		t.Fatalf("pressing the command opened %q, not the Speaking Stone", h.Overlay())
	}
}

func TestAFamiliarClassGetsTheSidecarCommand(t *testing.T) {
	f := registryFrame(t)
	cmd := ClassCommand(f.State, "raven:1.0")
	want := view.FamiliarCommand(acadCrewHole, "raven", "raven:1.0")
	if cmd != want {
		t.Fatalf("raven command = %q, want %q", cmd, want)
	}
	if ClassCommand(f.State, "porter:2.1") != "" {
		t.Fatal("a rumoured class has a command")
	}
}

func TestTheGuidedClassStaysOnScreenWhateverWasOpenAndThePlayerKeepsControlAfterAClick(t *testing.T) {
	f := registryFrame(t)
	f.Guide.Button = view.VaultPrefix + "clerk:1.1"
	h := New(nil, (&fakeRunner{}).backend())
	h.Open(OverlaySoulvault)
	h.OpenAcademy(acadTabOverview, acadFilterFamiliars)
	if _, ok := HitID(h.AcademyButtons(f), "vault:clerk:1.1"); !ok {
		t.Fatal("the guided class is hidden behind the player's tab and filter")
	}
	clickOn(t, h, f, "acad:repo:Porter")
	if _, ok := HitID(h.AcademyButtons(f), "vault:porter:1.1"); !ok {
		t.Fatal("the guide took the registry back after the player chose another repository")
	}
	f.Guide.Button = view.VaultPrefix + "clerk:1.0"
	if _, ok := HitID(h.AcademyButtons(f), "vault:clerk:1.0"); !ok {
		t.Fatal("a new guided class does not follow")
	}
}

func TestTheRegistryScrollsItsClassTableToTheGuidedClass(t *testing.T) {
	f := registryFrame(t)
	for i := range academyVisible() {
		f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: "hauler:" + string(rune('a'+i)), Cost: 1, Visible: true})
	}
	f.Guide.Button = view.VaultPrefix + "hauler:" + string(rune('a'+academyVisible()-1))
	h := New(nil, (&fakeRunner{}).backend())
	h.Open(OverlaySoulvault)
	if _, ok := HitID(h.AcademyButtons(f), f.Guide.Button); !ok {
		t.Fatal("the last class of a long table is not on screen when the guide names it")
	}
}

func TestEveryPlayableRepositoryCanFillACard(t *testing.T) {
	f := Frame{}
	for _, p := range soulstone.Playable() {
		f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: p.Image, Name: p.Name, Visible: true})
	}
	for _, r := range academyRepos(f) {
		if soulstone.Tagline(r.Classes[0].Image) == "" {
			t.Errorf("repository %s has no tagline", r.Repository)
		}
	}
}

func TestAFilterWithNoRepositoryDrawsAndPressesWithoutAPanic(t *testing.T) {
	f := registryFrame(t)
	f.State.Soulstones = f.State.Soulstones[:2]
	h := New(nil, (&fakeRunner{}).backend())
	h.Open(OverlaySoulvault)
	clickOn(t, h, f, "acad:filter:familiars")
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	h.academyWheel(Input{MX: academyLeft().X, Wheel: -1}, f, 3)
	h.academyWheel(Input{MX: academyRight().X, Wheel: -1}, f, 3)
	if len(h.AcademyButtons(f)) == 0 {
		t.Fatal("the empty Registry lost its buttons")
	}
}
