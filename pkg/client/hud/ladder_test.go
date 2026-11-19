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

func TestGuildLinesAreThePerKindCrews(t *testing.T) {
	f := fullFrame(t)
	g := Guilds(f)
	if len(g) != 3 || g[0].Name != "PORTERS" || g[0].State != StateNeck || g[1].State != StateNoDepot || g[2].State != StateNoCrew || g[2].Share != .25 {
		t.Fatalf("guilds = %+v", g)
	}
	if g[0].Value != ui.IconGold+"6.0" || g[0].Crew != 3 || g[0].Share != .9 {
		t.Errorf("porters = %+v", g[0])
	}
	f.State.Bottleneck = ""
	f.State.Freight.Seals = view.SealsView{DepotOpen: true, CrewName: "sealwrights", ArrivalRate: .4, ServedRate: .4, Crew: 2}
	if g := Guilds(f); len(g) != 4 || g[0].State != "" || g[3].Name != "SEALWRIGHTS" || g[3].State != "" || g[3].Share != 1 {
		t.Errorf("with seals = %+v", g)
	}
	f.State.Freight.Seals.Crew = 0
	if g := Guilds(f); g[3].State != StateNoCrew {
		t.Errorf("an empty seals depot = %+v", g[3])
	}
	f.State.Freight.Seals.DepotOpen = false
	if g := Guilds(f); g[3].State != StateNoDepot {
		t.Errorf("no seals depot = %+v", g[3])
	}
}

func TestBoxGoldFitsTheValueColumn(t *testing.T) {
	cases := map[float64]string{1.25: ui.IconGold + "1.2", 9.94: ui.IconGold + "9.9", 13.5: ui.IconGold + "14", 512: ui.IconGold + "512", 1500: "1.5K", 64000: "64K"}
	for v, want := range cases {
		if got := BoxGold(v); got != want {
			t.Errorf("BoxGold(%v) = %q, want %q", v, got, want)
		}
		if w, _ := (ui.Font{}).Measure(BoxGold(v), 1); w > GuildBarXAt-GuildValueX-1 {
			t.Errorf("%q is %dpx, the column holds %d", BoxGold(v), w, GuildBarXAt-GuildValueX-1)
		}
	}
}

func TestCarriedShareAndItsColour(t *testing.T) {
	cases := []struct{ arrival, served, waiting, want float64 }{{0, 0, 0, 1}, {0, 0, 3, 0}, {2, 1, 5, .5}, {1, 3, 0, 1}, {1, -1, 0, 0}}
	for _, c := range cases {
		if got := CarriedShare(c.arrival, c.served, c.waiting); got != c.want {
			t.Errorf("share %+v = %v", c, got)
		}
	}
	if ShareColor(1) != ui.Palette.Green || ShareColor(.7) != ui.Palette.GoldBright || ShareColor(.2) != ui.Palette.RedText {
		t.Error("share colours")
	}
}

func TestCausesAndNext(t *testing.T) {
	f := fullFrame(t)
	for neck, want := range map[string]string{"crew": "No depot", "no-depot": "open ledgers", "labor": "out of food", "crowded": "crowd one Keep"} {
		f.State.Bottleneck = neck
		if got := Cause(f); !strings.Contains(ui.Plain(got), want) || len([]rune(got)) > cardChars {
			t.Errorf("%s cause = %q", neck, got)
		}
	}
	if NextLine(f) != "" {
		t.Error("while a task stands, the task card is the only voice")
	}
	f.State.Objectives = nil
	if NextLine(f) != f.State.Advice {
		t.Error("advice speaks once no task is active")
	}
	f.State.Advice = ""
	if NextLine(f) != "" {
		t.Error("empty next line")
	}
}

const cardChars = 42

func TestNextLineFollows(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Objectives = nil
	for advice, page := range map[string]string{
		"Add a Keep: a Stone Keep adds 4 Hands of Labor": PageBuild,
	} {
		f.State.Advice = advice
		h.PageID = PageMain
		clickOn(t, h, f, "card:next")
		if h.PageID != page {
			t.Errorf("%q opened page %q, want %q", advice, h.PageID, page)
		}
	}
	f.State.Advice = "The Tower is out of Hands: endow the leaner class of 2.0 or close a berth"
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyN}}, f)
	if h.Overlay() != OverlaySoulvault {
		t.Errorf("N follows the advice: %q", h.Overlay())
	}
	f.State.Advice = ""
	f.State.Objectives = []view.ObjectiveView{{ID: "speak-get", Text: "Speak", Command: "kubectl get nodes"}}
	h.OverlayID = OverlayNone
	h.FollowNext(f)
	if !h.TextFocus() {
		t.Error("a task with a command opens the Stone")
	}
}

func TestSiegeCellAndModal(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Reveals = append(slices.Clone(f.State.Reveals), SiegeReveal)
	if _, ok := HitID(h.CommandButtons(f), "cmd:serve"); ok {
		t.Fatal("SERVE is retired: the boxes on the quay are clicked instead")
	}
	f.State.Objectives = []view.ObjectiveView{{ID: CrewServesObjective, Done: true}}
	if b, ok := HitID(h.CommandButtons(f), "cmd:siege"); !ok || b.Disabled || b.Rect != MainCell(MainHomes["cmd:siege"]) {
		t.Fatalf("once the crews carry, the spike card takes its home cell: %+v", b)
	}
	clickOn(t, h, f, "cmd:siege")
	if h.Overlay() != OverlaySiege {
		t.Fatal("the cell opens the modal")
	}
	if got := h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f); len(got) != 0 {
		t.Error("enter never answers the call")
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyG}}, f)
	f.State.Siege.Spare = f.State.Siege.Hands
	if it, _ := hasIntent(clickOn(t, h, f, "siege:answer"), IntentAction); it.Action.Action != "answerSiege" {
		t.Errorf("answer = %+v", it)
	}
	h.Open(OverlaySiege)
	clickOn(t, h, f, "siege:cancel")
	f.Joining = true
	if b := h.SiegeButtons(f)[1]; !b.Disabled || b.Tip != "wait for Ready" {
		t.Errorf("answer while joining = %+v", b)
	}
	f.State.Siege = SiegeRetry()
	if b := SiegeButton(f); b.Disabled || b.Note != "6:12" {
		t.Errorf("retry cell = %+v", b)
	}
	f.State.Siege.RetryIn, f.State.Siege.Available, f.State.Siege.Reason = 0, false, "no horn"
	if b := SiegeButton(f); b.Disabled || b.Tip != "no horn" {
		t.Errorf("blocked cell = %+v", b)
	}
	f.State.Siege.Phase = "active"
	if b := SiegeButton(f); !b.Selected || h.OpenSiege(f) != nil {
		t.Error("an active spike selects the cell and opens nothing")
	}
}

func SiegeRetry() (v gameSiege) {
	v.RetryIn, v.Target, v.SealsGoal = 372, .99, 8
	return v
}

func TestSiegeWords(t *testing.T) {
	if Duration(161) != "2:41" || Duration(-3) != "0:00" || CrowdText(2) != "For 3 minutes, up to 2 more boxes land each second." || !strings.Contains(CrowdText(5.12), " 5.12 ") {
		t.Error("spike words")
	}
	f := fullFrame(t)
	lines := SiegeReadiness(f)
	if !strings.Contains(lines[0].Text, "1 sounding (porters 2-10)") || lines[1].Clr != greenToken() {
		t.Errorf("readiness = %+v", lines)
	}
	if lines[2].Text != "The tide needs 2,400 rations spare; you have 1,900" || lines[2].Clr != redTextToken() {
		t.Errorf("hands line = %+v", lines[2])
	}
	f.State.Siege.Hands = 1
	if lines = SiegeReadiness(f); lines[2].Clr != greenToken() {
		t.Errorf("enough hands = %+v", lines[2])
	}
	f.Horns, f.Joining = []model.Horn{{Sounding: false}}, true
	lines = SiegeReadiness(f)
	if lines[0].Text != "Your War Horn is silent" || lines[0].Clr != redTextToken() || !strings.Contains(lines[1].Text, "joining") {
		t.Errorf("unready = %+v", lines)
	}
	f.Horns = []model.Horn{{}, {}, {}}
	if lines = SiegeReadiness(f); lines[0].Text != "Your 3 War Horns are silent" {
		t.Errorf("three silent horns = %+v", lines[0])
	}
	f.Horns = nil
	if lines = SiegeReadiness(f); lines[0].Text != "No War Horn stands" {
		t.Errorf("no horn = %+v", lines[0])
	}
}

func TestDepotAndHornCards(t *testing.T) {
	f := fullFrame(t)
	parcels := view.Structure{Kind: view.StructGuildhall, Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	lines := DepotLines(parcels, f)
	if !lines[0].Bar || lines[0].Suffix != "1.8/2.0" || lines[0].Fill != greenToken() || lines[1].Text != "3 porters carry, 8 parcels waiting" {
		t.Errorf("parcels depot = %+v", lines)
	}
	crates := view.Structure{Kind: view.DepotStructure("crate"), Object: view.ObjectRef{Kind: "Service", Name: "crates"}}
	if l := DepotLines(crates, f); l[1].Text != "No haulers answer: crates pile up on the quay" || l[0].Fill != redTextToken() {
		t.Errorf("empty archive = %+v", l)
	}
	horn := view.Structure{Kind: view.StructWarHorn, Object: view.ObjectRef{Kind: "HorizontalPodAutoscaler", Name: "porters"}}
	if l := HornLines(horn, f.Horns); l[0].Text != "3 -> 5 huts, food 84% of target 70%" {
		t.Errorf("horn = %+v", l)
	}
	if l := HornLines(horn, []model.Horn{{Name: "porters", Max: 3, Current: 3, Sounding: true}}); !strings.Contains(l[1].Text, "at max (3)") {
		t.Errorf("maxed horn = %+v", l)
	}
	if l := HornLines(horn, []model.Horn{{Name: "porters", Message: "no metrics"}}); l[1].Text != "no metrics" {
		t.Errorf("silent horn = %+v", l)
	}
	if l := HornLines(horn, nil); len(l) != 1 {
		t.Error("no horn written")
	}
	if got := DetailCommands(Selection{Kind: "structure", Structure: horn}, DetailDescribe); got[0] != "describe hpa porters" {
		t.Errorf("horn describe = %v", got)
	}
	if DarkText(view.StructWarHorn) != "DARK: THE HORN IS SILENT" || DarkText("x") != "DARK" {
		t.Error("dark texts")
	}
}

func TestLoomTiers(t *testing.T) {
	f := fullFrame(t)
	rows := LoomLayout(f.State.Reforge.Patterns)
	headers := 0
	for _, r := range rows {
		if r.Header {
			headers++
		}
	}
	if headers != 3 || len(rows) != 8 {
		t.Fatalf("layout = %d rows, %d headers", len(rows), headers)
	}
	locked := f.State.Reforge.Patterns[3]
	if text, dim := TierHeader(locked, 7); !dim || text != "TIER II: weave 12 threads to open (7/12)" {
		t.Errorf("locked header = %q", text)
	}
	if text, dim := TierHeader(f.State.Reforge.Patterns[0], 7); dim || text != "TIER I" {
		t.Errorf("open header = %q", text)
	}
	if b := LoomButton(rows[5].Rect, locked); !b.Disabled || !strings.Contains(b.Tip, "12 threads") {
		t.Errorf("locked button = %+v", b)
	}
	h := New(nil, Backend{})
	h.Update(Input{}, f)
	next := f
	next.State.Reforge.Patterns = append([]gamePattern(nil), f.State.Reforge.Patterns...)
	next.State.Reforge.Patterns[3].Locked = false
	h.Update(Input{}, next)
	if h.Juice.Tiers[2] != h.Now {
		t.Error("a newly opened tier flashes")
	}
}

func TestBannerCause(t *testing.T) {
	h := New(nil, Backend{})
	h.BannerCause("THE GATE FELL", "92.1% served, 99% needed. The crowd broke on the full ledger.")
	if h.Juice.Cause == "" || h.Juice.Banner != "THE GATE FELL" {
		t.Fatal("banner cause")
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	f := fullFrame(t)
	f.State.Siege.Phase, f.State.Siege.TimeLeft = "countdown", 8
	h.Draw(dst, f)
	f.State.Siege.Phase = "active"
	h.Toast("x", false)
	h.Draw(dst, f)
}

func TestGuildValueNeverTouchesItsBarOrState(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, mode := range namesModes() {
		mode.apply()
		for _, v := range []float64{1.25, 13.5, 512, 1500, 64000, 640000} {
			value := BoxGold(v)
			if end := GuildValueX + (ui.Font{}).Width(value, 1); GuildBarX(value) < end+ui.Advance-1 {
				t.Errorf("[%s] %q ends at %d, its bar starts at %d", mode.name, ui.Name(value), end, GuildBarX(value))
			}
			if GuildBarX(value) > GuildBarXAt+2*ui.Advance {
				t.Errorf("[%s] %q pushes the bar to %d", mode.name, ui.Name(value), GuildBarX(value))
			}
		}
	}
}

func TestAnEmptyDepotWithARaisedCrewReadsStartingNotNoCrew(t *testing.T) {
	f := fullFrame(t)
	g := Guilds(f)
	var idx int
	for i, l := range g {
		if l.State == StateNoCrew {
			idx = i
		}
	}
	if g[idx].State != StateNoCrew {
		t.Skip("no empty depot in the fixture")
	}
	f.State.Crews = append(f.State.Crews, view.CrewView{Name: g[idx].Trade, Replicas: 2})
	if got := Guilds(f)[idx].State; got != StateStart {
		t.Fatalf("a crew of 2 is raised and still starting, the line says %q", got)
	}
}

func TestTheRealmRowCountsTheHutsThatCarryAndSaysSo(t *testing.T) {
	g := GuildLine{Name: "PORTERS", Crew: 13}
	if got := GuildCount(g); !strings.Contains(got, "13 CARRY") || strings.Contains(got, "READY") {
		t.Errorf("the realm row reads %q for the 13 huts that carry, beside \"14 of 16 huts ready\" (town-night: \"13 READY\" over \"14 of 16 huts ready\")", got)
	}
}
