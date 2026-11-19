package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func fixtureFreight() view.FreightView {
	fv := view.FreightView{HarborView: view.HarborView{Tier: 2, Ship: "caravel", Multiplier: 4, UpkeepMultiplier: 60, Fresh: 12000, TierProgress: .4, NextTier: 30000,
		NextShip: "galleon", NextOpens: "dispatches, large boxes and sealed boxes", NextPort: "haven-0", NextPortName: "Gate Harbor", NextLevel: 3,
		TierAt: 100, FreshPerSec: 3.2, ValuePerSec: 18.5},
		Waiting: 14, QuayCap: 72, Docking: view.DockingView{Ships: 9, Last: 400, LastCargo: 12, Next: 420}}
	fv.Kinds = []view.KindView{
		{Kind: "parcel", Name: "parcels", Open: true, Waiting: 8, ArrivalRate: 2, ServedRate: 1.8, TTL: 28, Value: 4.8, Depot: "parcels", DepotOpen: true,
			Crew: 3, CrewName: "porters", Losses: view.LossView{Spoiled: .2},
			Cohorts: []view.CohortView{{Count: 5, Age: 22, Remaining: 6, TTL: 28}, {Count: 3, Age: 2, Remaining: 26, TTL: 28}}},
		{Kind: "ledger", Name: "ledgers", Open: true, Waiting: 4, ArrivalRate: .8, TTL: 36, Value: 9.6, Depot: "ledgers", CrewName: "clerks",
			Losses: view.LossView{NoDepot: .5}, Cohorts: []view.CohortView{{Count: 4, Age: 3, Remaining: 33, TTL: 36}}},
		{Kind: "crate", Name: "crates", Open: true, Waiting: 2, ArrivalRate: .8, ServedRate: .2, TTL: 45, Value: 7.2, Depot: "crates", DepotOpen: true,
			CrewName: "haulers", Losses: view.LossView{NoCrew: .3}},
		{Kind: "dispatch", Name: "dispatches", Opens: 3, TTL: 120, Depot: "dispatches", CrewName: "couriers"},
	}
	fv.Seals = view.SealsView{Depot: "seals", CrewName: "sealwrights"}
	fv.Losses = view.LossView{Spoiled: .2, NoDepot: .5, NoCrew: .3}
	return fv
}

func TestTheBoardReadsTheHarborLadder(t *testing.T) {
	head := FreightHeadOf(fixtureFreight(), true)
	if goal := "Gate Harbor: galleons, " + ui.IconGold + "30K"; head.Tier != 2 || head.Count != ui.IconGold+"12K/30K" || head.Goal != goal {
		t.Errorf("head = %+v", head)
	}
	if len(head.Chips) != 3 || head.Chips[0].Kind != "parcel" || head.Chips[0].Level != 3 || head.Chips[1].Level != 0 || head.Chips[2].Fresh != 1 {
		t.Fatalf("chips = %+v", head.Chips)
	}
	if !strings.Contains(head.Chips[0].Tip, "8 waiting") || !strings.Contains(head.Chips[1].Tip, "No ledgers depot") || !strings.Contains(head.Chips[2].Tip, "stands empty") {
		t.Errorf("chip tips %q / %q / %q", head.Chips[0].Tip, head.Chips[1].Tip, head.Chips[2].Tip)
	}
}

func TestTierGoalsNameThePortShipsAndPrice(t *testing.T) {
	cases := []struct {
		port        string
		level, tier int
		cost        float64
		wait        string
		want        string
	}{
		{"Gate Harbor", 2, 1, 800, "", "Gate Harbor: caravels, " + ui.IconGold + "800"},
		{"Frostwater Quay", 3, 2, 50000, "", "Frostwater Quay: galleons, " + ui.IconGold + "50K"},
		{"Gullcliff Landing", 4, 3, 5000000, "", "Gullcliff Landing: carracks, " + ui.IconGold + "5M"},
		{"", 0, 2, 0, "", "Tier 2: no port upgrade on offer now"},
		{"", 0, 1, 0, "after 99% ready", "Tier 2: after 99% ready"},
		{"", 0, 2, 0, "galleon masters coming", "Tier 3: galleon masters coming"},
		{"", 0, 4, 0, "", "Tier 4: the top of the harbor ladder"},
	}
	for _, c := range cases {
		fv := view.FreightView{}
		fv.Tier, fv.NextLevel, fv.NextTier, fv.NextPortName, fv.NextWait = c.tier, c.level, c.cost, c.port, c.wait
		if got := TierGoal(fv); got != c.want {
			t.Errorf("goal %q, want %q", got, c.want)
		}
		if w, _ := (ui.Font{}).Measure(TierGoal(fv), 1); w > BoardW-2*BoardPad {
			t.Errorf("goal %q is %dpx, the board holds %d", TierGoal(fv), w, BoardW-2*BoardPad)
		}
		ui.SetNamer(names.Kubernetes(64))
		if w := (ui.Font{}).Width(TierGoal(fv), 1); w > BoardW-2*BoardPad {
			t.Errorf("goal %q is %dpx with Kubernetes names, the board holds %d", ui.Name(TierGoal(fv)), w, BoardW-2*BoardPad)
		}
		ui.SetNamer(nil)
	}
	waiting := view.FreightView{}
	waiting.Tier, waiting.NextWait, waiting.NextWhy = 1, "after 99% ready", "caravels sail when the realm calls them: hold 99% ready first"
	if tip := TierTip(waiting, true); !strings.Contains(tip, "Tier 2 waits: caravels sail when the realm calls them") {
		t.Errorf("the TIER tip does not say what the next tier waits for: %q", tip)
	}
	top := view.FreightView{}
	top.Tier, top.Fresh = 4, 3000000
	if head := FreightHeadOf(top, true); head.Count != "3M fresh" || head.Tier != 4 {
		t.Errorf("top head %+v", head)
	}
}

func TestLossLinesNameTheCauseAndTheFix(t *testing.T) {
	lines := LossLines(fixtureFreight())
	want := []string{"No depot 0.50/s: open ledgers", "No crew 0.30/s: raise haulers", "Spoiled 0.20/s: carried too slowly"}
	if len(lines) != len(want) {
		t.Fatalf("lines = %+v", lines)
	}
	for i, l := range lines {
		if ui.Plain(l.Text) != want[i] {
			t.Errorf("line %d = %q, want %q", i, l.Text, want[i])
		}
		if w, _ := (ui.Font{}).Measure(l.Text, 1); w > BoardW-2*BoardPad {
			t.Errorf("%q is too wide for the panel", l.Text)
		}
	}
	fv := fixtureFreight()
	fv.Losses = view.LossView{QuayFull: 1.2, Blocked: .1, Spoiled: .005}
	fv.Kinds[0].Losses = view.LossView{QuayFull: 1.2}
	lines = LossLines(fv)
	if len(lines) != 2 || lines[0].Text != "Quay full 1.2/s: the quay overflows" || lines[1].Text != "Lost in flight 0.10/s: worker stopped" {
		t.Errorf("quay full lines = %+v", lines)
	}
	if LossTotal(fv) != 1.305 {
		t.Errorf("total %v", LossTotal(fv))
	}
}

func TestATierClimbFlashesTheBoard(t *testing.T) {
	fv := fixtureFreight()
	if !Flashing(fv, 103) || Flashing(fv, 100+FlashFor+1) || Flashing(fv, 99) {
		t.Error("the board flashes for a few seconds after a climb")
	}
	fv.Tier = 1
	if Flashing(fv, 101) {
		t.Error("tier 1 is where every realm starts: no flash")
	}
}

func TestTheBoardAndLossesFitTheMapWithoutOverlap(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	board, losses := FreightRect(f), LossesRect(f)
	for name, r := range map[string]ui.Rect{"board": board, "losses": losses, "siege": SiegeStrip} {
		if r.X < 0 || r.Y < TopH || r.X+r.W > Width || r.Y+r.H > ConsoleY {
			t.Errorf("%s %+v leaves the map", name, r)
		}
		if Overlaps(r, h.ObjectiveRectOf(f)) || Overlaps(r, TopBar) {
			t.Errorf("%s %+v overlaps the task card or the top bar", name, r)
		}
		free := 0
		for _, spot := range CalamitySpots(120) {
			if !Overlaps(board, spot) && !Overlaps(h.ObjectiveRectOf(f), spot) {
				free++
			}
		}
		if free < 2 {
			t.Errorf("only %d calamity spots stay clear of the board and the task card", free)
		}
		if rail := RailSlot(0); Overlaps(r, rail) {
			t.Errorf("%s overlaps the rail", name)
		}
	}
	if !Within(losses, board) || Overlaps(board, SiegeStrip) {
		t.Errorf("the losses fold into the board: board %+v losses %+v strip %+v", board, losses, SiegeStrip)
	}
	f.State.Siege.Phase = "active"
	if l := LossesRect(f); Overlaps(l, SiegeStrip) {
		t.Errorf("the losses sit under the spike strip: %+v", l)
	}
	for _, r := range h.ToastRects(f) {
		if Overlaps(r, board) {
			t.Errorf("a toast %+v covers the board", r)
		}
	}
}

func TestTheBoardStaysHiddenUntilItsIntro(t *testing.T) {
	f := stagedFrame(t, MeterGold, "cmd:serve")
	if BoardShown(f) || LossesShown(f) || len(FreightRects(f)) != 0 {
		t.Error("the board shows before the freight intro")
	}
	f.Guide.Intro = &Intro{ID: "freight", Target: SurfaceFreight, Via: ViaAck}
	if !BoardShown(f) {
		t.Error("the freight intro shows the board it points at")
	}
	h := New(nil, Backend{})
	if r, ok := h.TargetRect(SurfaceFreight, f); !ok || r != FreightRect(f) {
		t.Errorf("freight target %+v %v", r, ok)
	}
	f.Guide.Intro = &Intro{ID: "losses", Target: SurfaceLosses, Via: ViaAck}
	f.State.Freight.Losses = view.LossView{}
	if !LossesShown(f) {
		t.Error("the losses intro shows the panel even with nothing lost")
	}
}

func TestBoardButtonsTipAndOpenTheLedger(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	ids := map[string]string{}
	for _, b := range h.FreightButtons(f) {
		ids[b.ID] = b.Tip
	}
	for _, id := range []string{"freight:tier", "freight:rate", DemandRowID, "freight:kind:parcel", "freight:kind:crate", "freight:losses"} {
		if strings.Contains(ids[id], "ley") {
			t.Errorf("%s speaks of the retired ley", id)
		}
		if ids[id] == "" {
			t.Errorf("no tip on %s", id)
		}
	}
	if !strings.Contains(ids["freight:tier"], "Gate Harbor takes galleons") || !strings.Contains(ids["freight:tier"], "for 30K gold") || strings.Contains(ids["freight:tier"], "Press") || !strings.Contains(ids["freight:tier"], "deepest port") {
		t.Errorf("tier tip %q", ids["freight:tier"])
	}
	clickOn(t, h, f, "freight:rate")
	if h.Overlay() != OverlayLedger {
		t.Errorf("the board opens the ledger, got %q", h.Overlay())
	}
	if len(h.FreightButtons(f)) != 0 {
		t.Error("the board takes no clicks under an open window")
	}
	if !h.Captures(FreightRect(f).X+2, FreightRect(f).Y+2, fullFrame(t)) {
		t.Error("a click on the board never reaches the map")
	}
}

func TestTheBoardDrawsAtEveryTier(t *testing.T) {
	h := New(nil, Backend{})
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	for tier := 1; tier <= 4; tier++ {
		f := fullFrame(t)
		f.State.Freight.Tier, f.State.RealmTime = tier, 101
		f.State.Freight.TierAt = 100
		h.Draw(dst, f)
	}
	f := fullFrame(t)
	f.State.Freight.Losses = view.LossView{}
	f.Guide = Guide{Staged: true, Intro: &Intro{ID: "losses", Target: SurfaceLosses, Via: ViaAck}, Introduced: []string{SurfaceFreight}}
	h.Draw(dst, f)
}

func TestGateLinesSpeakOfTheQuay(t *testing.T) {
	f := fullFrame(t)
	f.State.Pilgrims.TotalServed, f.State.Pilgrims.TotalDropped = 12, 4
	text := strings.Join(GateLines(f), " | ")
	for _, want := range []string{"8 parcels wait", "delivered 12", "spoiled 4", "GOAL"} {
		if !strings.Contains(text, want) {
			t.Errorf("gate card %q lacks %q", text, want)
		}
	}
	f.State.Freight.Kinds[0].Waiting = 0
	f.State.RealmTime = 405
	if l := GateLines(f)[0]; l != "The quay is empty: the next ship docks in 0:15" {
		t.Errorf("empty quay %q", l)
	}
	f.State.RealmTime = 430
	if l := GateLines(f)[0]; l != "The quay is empty: a ship is on its way" {
		t.Errorf("late ship %q", l)
	}
	for _, l := range GateLines(f) {
		if len(l) > cardChars {
			t.Errorf("gate line too long: %q", l)
		}
	}
}

func TestAdviceOpensTheMatchingCommand(t *testing.T) {
	f := fullFrame(t)
	f.State.Objectives = nil
	f.State.Commands = []view.CommandOffer{
		{ID: "open-depot:ledger", Command: "kubectl expose deployment clerks --name=ledgers --port=80 --target-port=8080", Action: view.Action{Action: "openDepot", Kind: "ledger"}},
		{ID: "raise-crew:haulers", Command: "kubectl create deployment haulers --image=hauler:1.0 --replicas=2 --port=8080", Action: view.Action{Action: "raiseCrew", Name: "haulers"}},
	}
	f.State.Advice = "Raise haulers: the crates depot stands empty"
	if got := AdviceCommand(f); !strings.Contains(got, "create deployment haulers") {
		t.Errorf("haulers advice opens %q", got)
	}
	f.State.Advice = "Open the ledgers depot: ledgers spoil with nowhere to go"
	if got := AdviceCommand(f); !strings.Contains(got, "--name=ledgers") {
		t.Errorf("ledgers advice opens %q", got)
	}
	f.State.Advice = "Add a Keep"
	if AdviceCommand(f) != "" {
		t.Error("unrelated advice opens a command")
	}
	h := New(nil, Backend{})
	f.State.Advice = "Raise haulers: the crates depot stands empty"
	h.FollowNext(f)
	if !h.TextFocus() {
		t.Error("the advice opens the Stone with its command")
	}
}

func TestLossLinesFitWithKubernetesNames(t *testing.T) {
	defer ui.SetNamer(nil)
	fv := fixtureFreight()
	fv.Losses = view.LossView{QuayFull: 1.2, Blocked: .27, Spoiled: .5, NoDepot: .5, NoCrew: .3}
	for _, mode := range namesModes() {
		mode.apply()
		for _, l := range LossLines(fv) {
			if w := (ui.Font{}).Width(l.Text, 1); w > BoardW-2*BoardPad {
				t.Errorf("[%s] %q is %dpx, the panel holds %d", mode.name, ui.Name(l.Text), w, BoardW-2*BoardPad)
			}
		}
	}
}

func TestTheChipTipSpeaksOfSpoilingOnlyOnceBoxesCanSpoil(t *testing.T) {
	watching := view.ObjectiveView{ID: view.SpoilObjective, Progress: "leave a few boxes on the quay"}
	depot := view.ObjectiveView{ID: "open-depot", Done: true}
	for _, c := range []struct {
		name   string
		age    string
		steps  []view.ObjectiveView
		spoils bool
	}{
		{"the founding", view.AgeDark, []view.ObjectiveView{{ID: "serve-hand", Done: true}, {ID: "raise-keep", Progress: "ready: press BUILD"}}, false},
		{"the timeout step open", view.AgeDark, []view.ObjectiveView{depot, watching}, true},
		{"the timeout step done", view.AgeDark, []view.ObjectiveView{depot, {ID: view.SpoilObjective, Done: true}, {ID: view.HandPortersObjective}}, true},
		{"hand porters done", view.AgeDark, []view.ObjectiveView{depot, {ID: view.HandPortersObjective, Done: true}}, true},
		{"staging", view.AgeFeudal, nil, true},
	} {
		gs := view.GameState{}
		gs.Age, gs.Objectives = c.age, c.steps
		if got := BoxesSpoil(gs); got != c.spoils {
			t.Errorf("%s: boxes spoil %v, want %v", c.name, got, c.spoils)
		}
	}
	k := fixtureFreight().Kinds[0]
	if tip := ChipTip(k, .5, false); strings.Contains(tip, "spoils after") || !strings.Contains(tip, "50%") {
		t.Errorf("before the timeout step the tip still promises a spoil: %q", tip)
	}
	if tip := ChipTip(k, .5, true); !strings.Contains(tip, "a parcel spoils after 28s") {
		t.Errorf("once boxes spoil the tip lost their time to live: %q", tip)
	}
}

func TestTheTierTipNamesThePortsRowOnlyOnceItIsOnTheBoard(t *testing.T) {
	if tip := TierTip(view.FreightView{}, false); strings.Contains(tip, "PORTS row") {
		t.Fatalf("one port and no PORTS row, yet the tip says %q", tip)
	}
	if tip := TierTip(view.FreightView{}, true); !strings.Contains(tip, "PORTS row") {
		t.Fatalf("the PORTS row is up and the tip does not say how to use it: %q", tip)
	}
}
