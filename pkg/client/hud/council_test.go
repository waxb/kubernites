package hud

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const gi = float64(1 << 30)

func caesarFixture() model.Caesar {
	return model.Caesar{
		Counsel: "Readiness wanes: the crew awaits its probe.",
		Ratings: model.Ratings{Reliability: 82, Efficiency: 41, Resilience: 12, Favor: 67},
		Rated:   true,
		Decrees: []model.Decree{
			{ID: "d-tribute", Title: "A probe of 500 MiB", Text: "The crew asks for memory before the rollout.", Kind: model.DecreeTribute, State: model.DecreeOffered, Deadline: 400, Progress: .2, Reward: 10},
			{ID: "d-lean", Title: "Keep the Codex lean", Text: "Hold the Codex under half its quota.", Kind: "lean-codex", State: model.DecreeActive, Deadline: 250, Progress: .6, Reward: 5},
			{ID: "d-old", Title: "Pledge every hut", Kind: "pledge-all", State: model.DecreeMet, Deadline: 90, Progress: 1, Reward: 8},
		},
		Codex: model.Codex{SizeBytes: 1.9 * gi, InUseBytes: .7 * gi, QuotaBytes: 2 * gi, Alarm: true},
		Middens: map[string]model.Midden{
			"lone-tower":   {Node: "lone-tower", UsedBytes: 57 * gi, CapacityBytes: 60 * gi, Pressure: true},
			"stone-keep-2": {Node: "stone-keep-2", UsedBytes: 12 * gi, CapacityBytes: 60 * gi},
		},
	}
}

func caesarFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Age = view.AgeFeudal
	f.State.Reveals = append(append([]string(nil), f.State.Reveals...), CouncilReveal, LensReveal, CodexReveal, QosReveal)
	f.State.RealmTime = 100
	f.State.GoldPerSec = 1.2
	f.State.KeepShop = []view.KeepOffer{{KeepType: view.KeepTypeStone, UpkeepPerSec: .35}, {KeepType: view.KeepTypeGreat, UpkeepPerSec: .7}}
	f.State.Keeps = []view.KeepView{{Name: "lone-tower", KeepType: view.KeepTypeTower, Upkeep: .02}, {Name: "stone-keep-2", KeepType: view.KeepTypeStone, Upkeep: .33}}
	f.Caesar = caesarFixture()
	f.Site.Structures = append(append([]view.Structure(nil), f.Site.Structures...),
		view.Structure{ID: "keep-1", Kind: view.StructKeep, Variant: "tower", X: 20, Y: 20, W: 3, H: 3, State: view.StateReady, Label: "The Lone Tower", Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}},
		view.Structure{ID: "keep-2", Kind: view.StructKeep, Variant: "stone", X: 30, Y: 20, W: 3, H: 3, State: view.StateReady, Label: "Stone Keep 2", Object: view.ObjectRef{Kind: "Node", Name: "stone-keep-2"}},
		view.Structure{ID: "codex-1", Kind: view.StructCodex, X: 24, Y: 20, W: 2, H: 2, State: view.StateReady, Label: "The Codex", Object: view.ObjectRef{Kind: "Pod", Namespace: "kube-system", Name: "etcd-lone-tower"}})
	f.Crews = []model.CrewChain{{Namespace: "default", Name: "porters", Image: "porter:1.0", Desired: 3, Ready: 2, Rank: model.RankHigh},
		{Namespace: "default", Name: "clerks", Image: "clerk:1.0", Desired: 1, Ready: 1}}
	f.Rolls = []model.GuildRoll{{Service: "parcels", Ready: 2, Endpoints: 3}, {Service: "archive", Ready: 0, Endpoints: 1}}
	f.Site.Havens = []view.Haven{{ID: "haven-0", Name: "Gate Harbor", Open: true, Tier: 2, Share: .75}, {ID: "haven-1", Name: "North Haven", Open: true, Tier: 0, Share: .25}, {ID: "haven-2", Name: "Dawnport"}}
	f.Pods = append(f.Pods, model.PodView{Namespace: "default", Name: "porter-c", State: model.HearthPending, QoS: model.QoSBestEffort},
		model.PodView{Namespace: "default", Name: "porter-d", State: model.HearthPending, QoS: model.QoSBurstable})
	f.Metrics.Augury = true
	f.Metrics.Nodes["stone-keep-2"] = view.NodeMetrics{CPUAllocatableMilli: 4000, CPURequestedMilli: 4000, CPUMilli: 1200, MemoryAllocatableBytes: 8 << 30, Pods: 7}
	return f
}

func councilHUD(t *testing.T, tab string) (*HUD, Frame) {
	t.Helper()
	h := New(nil, (&fakeRunner{}).backend())
	h.OpenCouncil(tab)
	return h, caesarFrame(t)
}

func findButton(buttons []Button, id string) (Button, bool) {
	for _, b := range buttons {
		if b.ID == id {
			return b, true
		}
	}
	return Button{}, false
}

func TestCouncilOpensWithCOnlyOnceRevealed(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	dark := fullFrame(t)
	dark.State.Age = view.AgeFeudal
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyC}}, dark)
	if h.Overlay() == OverlayCouncil {
		t.Fatal("the Council stays dark until it is revealed")
	}
	if _, ok := findButton(h.Buttons(dark), "cmd:council"); ok {
		t.Fatal("the COUNCIL button stays dark until revealed")
	}
	dark.State.Reveals = append(append([]string(nil), dark.State.Reveals...), CouncilReveal)
	if !CouncilOpen(dark) {
		t.Fatal("the council reveal opens the Council")
	}
	f := caesarFrame(t)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyC}}, f)
	if h.Overlay() != OverlayCouncil || h.CouncilTab() != TabQuartermaster {
		t.Fatalf("C opened %q on %q", h.Overlay(), h.CouncilTab())
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyC}}, f)
	if h.Overlay() != OverlayNone {
		t.Fatal("C again closes the Council")
	}
	clickOn(t, h, f, "cmd:council")
	if h.Overlay() != OverlayCouncil {
		t.Fatal("the COUNCIL button in the info well opens it")
	}
	clickOn(t, h, f, "council:tab:"+TabAugur)
	if h.CouncilTab() != TabAugur {
		t.Fatal("tabs switch advisors")
	}
}

func TestCouncilLayoutFitsAndNeverOverlaps(t *testing.T) {
	for _, a := range Advisors {
		h, f := councilHUD(t, a.Id)
		buttons := h.CouncilButtons(f)
		for i, b := range buttons {
			if !inside(b.Rect, Overlay) {
				t.Errorf("%s: %s %+v leaves the Council %+v", a.Id, b.ID, b.Rect, Overlay)
			}
			for _, c := range buttons[:i] {
				if Overlaps(b.Rect, c.Rect) {
					t.Errorf("%s: %s %+v overlaps %s %+v", a.Id, b.ID, b.Rect, c.ID, c.Rect)
				}
			}
		}
		rows := SectionRows(CouncilPageFor(a.Id, f), CouncilBody().Y+PageTop)
		for _, section := range rows {
			if last := section[len(section)-1]; last+ui.LineHeight > FootRect(0).Y {
				t.Errorf("%s: a row at %d runs into the footer at %d", a.Id, last, FootRect(0).Y)
			}
		}
	}
	for _, r := range []ui.Rect{ChiefFace(), CounselBox(), DialsBox(), CouncilBody()} {
		if !inside(r, Overlay) {
			t.Errorf("%+v leaves the Council", r)
		}
	}
	if Overlaps(CounselBox(), DialsBox()) || Overlaps(DialsBox(), CouncilTabID(len(Advisors)-1)) || Overlaps(CouncilTabID(0), CouncilBody()) {
		t.Error("the Council's header parts overlap")
	}
	for i := range Advisors {
		if w, _ := (ui.Font{}).Measure(Advisors[i].Label, 1); w > CouncilTabID(i).W-26 {
			t.Errorf("tab %s is too narrow for its label", Advisors[i].Label)
		}
	}
	for i, d := range RatingDials(Frame{}) {
		if w, _ := (ui.Font{}).Measure(d.Label, 1); w > DialRect(i).W {
			t.Errorf("dial %s label is wider than its dial", d.Label)
		}
	}
}

func inside(r, box ui.Rect) bool {
	return r.X >= box.X && r.Y >= box.Y && r.X+r.W <= box.X+box.W && r.Y+r.H <= box.Y+box.H
}

func TestCounselComesFromTheChiefThenTheAdvice(t *testing.T) {
	f := caesarFrame(t)
	if Counsel(f) != f.Caesar.Counsel {
		t.Error("the chief advisor speaks first")
	}
	f.Caesar.Counsel = ""
	if Counsel(f) != f.State.Advice {
		t.Errorf("without counsel, the realm's advice: %q", Counsel(f))
	}
	f.State.Advice = ""
	if Counsel(f) != QuietRealm {
		t.Error("a quiet realm")
	}
}

func TestDialMaths(t *testing.T) {
	if DialAngle(0) != 3.141592653589793 || DialAngle(100) != 0 || DialAngle(150) != 0 || DialAngle(-4) != DialAngle(0) {
		t.Error("the needle sweeps from left (0) to right (100)")
	}
	if DialLit(4, 0) || !DialLit(5, 0) || !DialLit(96, 9) || DialLit(94, 9) {
		t.Error("segments light at their midpoints")
	}
	if DialZone(0) != ToneBad || DialZone(5) != ToneWarn || DialZone(9) != ToneGood {
		t.Error("zones run red, amber, green")
	}
	f := caesarFrame(t)
	dials := RatingDials(f)
	if len(dials) != 4 || DialText(dials[0]) != Nines(f.State.SLO.Current) || DialText(dials[1]) != "41" || DialText(dials[3]) != "67" {
		t.Errorf("dials = %+v", dials)
	}
	f.Caesar.Rated = false
	if DialText(RatingDials(f)[1]) != NoValue {
		t.Error("unknown ratings show dashes")
	}
}

func cells(s Section, row int) []string {
	out := []string{}
	for _, c := range s.Rows[row] {
		out = append(out, c.Text)
	}
	return out
}

func TestQuartermasterCountsHandsAndUnhoused(t *testing.T) {
	f := caesarFrame(t)
	p := QuartermasterPage(f)
	keeps := p.Sections[0]
	if len(keeps.Rows) != 2 {
		t.Fatalf("keeps rows = %d", len(keeps.Rows))
	}
	tower := cells(keeps, 0)
	if tower[0] != "The Lone Tower" || tower[1] != "1,300 / 1,900" || tower[4] != "0" {
		t.Errorf("tower row = %v", tower)
	}
	stone := keeps.Rows[1]
	if stone[1].Text != "4,000 / 4,000" || stone[1].Tone != ToneBad || stone[2].Text != "1,200" {
		t.Errorf("full stone keep row = %+v", stone)
	}
	if got := cells(p.Sections[1], 0)[0]; got != "2 huts" {
		t.Errorf("unhoused = %q", got)
	}
	if !strings.HasPrefix(p.Line, "2 huts are Unhoused") {
		t.Errorf("line = %q", p.Line)
	}
	f.Pods = f.Pods[:3]
	if !strings.Contains(QuartermasterPage(f).Line, "Stone Keep 2 is fully pledged") {
		t.Errorf("line = %q", QuartermasterPage(f).Line)
	}
	f.Metrics.Augury = false
	if got := cells(QuartermasterPage(f).Sections[0], 1)[2]; got != NoValue {
		t.Errorf("used without the Augur = %q", got)
	}
}

func withRankReachable(t *testing.T, reachable bool) {
	t.Helper()
	prev := RankReachable
	RankReachable = func() bool { return reachable }
	t.Cleanup(func() { RankReachable = prev })
}

func TestTheRankColumnStaysHiddenUntilItsPageCanBeLearned(t *testing.T) {
	h, f := councilHUD(t, TabStewards)
	if RankReachable() {
		t.Fatal("no page unlocks the rank column yet")
	}
	for _, b := range h.CouncilButtons(f) {
		if strings.HasPrefix(b.ID, CouncilPrefix+":rank:") {
			t.Errorf("a locked Rank button %s shows for a page nobody can read", b.ID)
		}
	}
	s := StewardsPage(f).Sections[0]
	if slices.Contains(s.Head, "RANK") || len(s.Head) != len(s.Cols) || len(s.Rows[0]) != len(s.Cols) {
		t.Errorf("the Rank column is hidden: %+v", s)
	}
	if strings.Contains(StewardsPage(f).Line, "Rank") {
		t.Errorf("the line still teaches Rank: %q", StewardsPage(f).Line)
	}
}

func TestStewardsRankCrewsWithSetRank(t *testing.T) {
	withRankReachable(t, true)
	h, f := councilHUD(t, TabStewards)
	if b, ok := findButton(h.CouncilButtons(f), "council:rank:porters:low"); !ok || !b.Disabled || b.Tip != RankLockTip {
		t.Fatalf("ranks wait for their page: %+v", b)
	}
	f.State.Grimoire = append(f.State.Grimoire, view.GrimoirePage{ID: view.RankPageID, Ranks: []view.RankView{{Learned: true, Runes: []string{view.RankRune}}}})
	p := StewardsPage(f)
	if got := cells(p.Sections[0], 0); got[1] != "2/3" || got[3] != "2-10, 3 -> 5" {
		t.Errorf("porters row = %v", got)
	}
	b, ok := findButton(h.CouncilButtons(f), "council:rank:porters:high")
	if !ok || !b.Selected || !strings.Contains(b.Tip, "realm-high") || !strings.Contains(b.Tip, "1000") {
		t.Fatalf("high rank button = %+v", b)
	}
	intents := clickOn(t, h, f, "council:rank:clerks:low")
	it, ok := hasIntent(intents, IntentAction)
	if !ok || it.Action.Action != "setRank" || it.Action.Name != "clerks" {
		t.Fatalf("rank click = %+v", intents)
	}
	if it.Action.Rank != model.RankLow {
		t.Errorf("setRank must carry the rank: %+v", it.Action)
	}
}

func TestWayfinderListsRollsAndHarbors(t *testing.T) {
	f := caesarFrame(t)
	p := WayfinderPage(f, func(h view.Haven) int { return int(h.Tier) + 1 })
	if got := cells(p.Sections[0], 0); got[0] != "parcels" || got[1] != "2 of 3 ready" {
		t.Errorf("roll = %v", got)
	}
	if p.Sections[0].Rows[1][1].Tone != ToneBad {
		t.Error("an empty Guild Roll is bad news")
	}
	gate := cells(p.Sections[1], 0)
	if gate[1] != "OPEN" || gate[2] != "WHARF" || gate[3] != "75%" || gate[4] != "0.38" || gate[5] != "3" {
		t.Errorf("gate harbor = %v", gate)
	}
	if cells(p.Sections[1], 2)[1] != "CLOSED" || !strings.HasPrefix(p.Line, "2 harbors open") {
		t.Errorf("closed harbor or line wrong: %q", p.Line)
	}
}

func TestTreasurerTracesUpkeepToTheKeepsBuilt(t *testing.T) {
	f := caesarFrame(t)
	upkeep, keeps := Upkeep(f)
	if keeps != 2 || math.Abs(upkeep-.35) > 1e-9 {
		t.Fatalf("overhead = %v over %d Keeps, want the built Keeps' own bill (the next plot's figure is %v)", upkeep, keeps, f.State.KeepShop[0].UpkeepPerSec)
	}
	f.State.KeepShop[0].UpkeepPerSec = .9
	if again, _ := Upkeep(f); math.Abs(again-.35) > 1e-9 {
		t.Fatalf("the next plot's figure moved the bill of Keeps already built: %v", again)
	}
	f.State.GoldPerSec = 1.42
	p := TreasurerPage(f)
	if got := cells(p.Sections[0], 1)[1]; got != "+1.8" {
		t.Errorf("rate = %q", got)
	}
	if got := cells(p.Sections[0], 3)[1]; got != "+1.4" {
		t.Errorf("net = %q", got)
	}
	if got := cells(p.Sections[0], 2)[1]; got != "-0.35  (2 Keeps)" {
		t.Errorf("overhead = %q", got)
	}
	if held := p.Sections[1]; held.Title != "HELD CONTRACTS" || len(held.Rows) != 0 {
		t.Errorf("no contract held yet = %+v", held)
	}
	reward := view.ContractReward{Gold: 300, Favor: 6}
	f.State.Contracts = []view.ContractView{{ID: "c1", Title: "Sealed ledgers", State: "active", Deadline: f.State.RealmTime + 300, Reward: reward}}
	held := TreasurerPage(f).Sections[1]
	if len(held.Rows) != 1 || held.Rows[0][1].Text != "5:00" || held.Rows[0][2].Text != RewardText(reward) {
		t.Errorf("held contracts = %+v", held.Rows)
	}
	f.State.GoldPerSec, f.State.Age = -.4, view.AgeEmpire
	if !strings.Contains(TreasurerPage(f).Line, "drains") {
		t.Error("a draining balance is named")
	}
	f.State.GoldPerSec, f.State.Age, f.State.Economy.Income = 0, view.AgeFeudal, .2
	p = TreasurerPage(f)
	if strings.Contains(p.Line, "drains") || !strings.Contains(p.Line, "eats everything") || cells(p.Sections[0], 1)[1] != "+0.20" {
		t.Errorf("before Production the bill only eats the rate: %q, rate %q", p.Line, cells(p.Sections[0], 1)[1])
	}
}

func TestAugurReadsTheCellarsAndTheCodex(t *testing.T) {
	f := caesarFrame(t)
	p := AugurPage(f)
	if !strings.Contains(p.Line, "Codex is full") {
		t.Errorf("line = %q", p.Line)
	}
	midden := p.Sections[1]
	if len(midden.Rows) != 2 || midden.Rows[0][2].Text != "CELLAR WARD" || midden.Rows[0][0].Text != "The Lone Tower" {
		t.Errorf("midden = %+v", midden.Rows)
	}
	if got := midden.Rows[1][1].Text; got != "12,288 of 61,440 barrels (20%)" {
		t.Errorf("stone keep floor = %q", got)
	}
	codex := p.Sections[2]
	if got := codex.Rows[0][1].Text; got != "1.9 GiB of 2.0 GiB quota (95%)" {
		t.Errorf("codex size = %q", got)
	}
	if got := codex.Rows[1][1].Text; got != "717 MiB, 1.2 GiB to reclaim" {
		t.Errorf("codex in use = %q", got)
	}
	f.Caesar.Codex.Alarm = false
	if !strings.Contains(AugurPage(f).Line, "DiskPressure") {
		t.Errorf("pressure line = %q", AugurPage(f).Line)
	}
	f.Caesar = model.Caesar{}
	f.Metrics.Augury = false
	blind := AugurPage(f)
	if blind.Line != BlindAugur || blind.Sections[0].Empty != BlindOmens || blind.Sections[1].Empty != MiddenUnknown || blind.Sections[2].Empty != CodexUnknown {
		t.Errorf("blind augur = %+v", blind)
	}
}

func TestAugurRebindsAndOffersTheDisarmChip(t *testing.T) {
	h, f := councilHUD(t, TabAugur)
	intents := clickOn(t, h, f, "council:rebind")
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != "rebindCodex" {
		t.Fatalf("rebind = %+v", intents)
	}
	clickOn(t, h, f, "council:disarm")
	if h.Overlay() != OverlayStone || string(h.Stone.Input) != DisarmCommand {
		t.Fatalf("disarm chip opened %q with %q", h.Overlay(), string(h.Stone.Input))
	}
	if PromptFor(string(h.Stone.Input)) != NodePrompt || PromptFor("get pods") != "$ kubectl " {
		t.Error("etcdctl is spoken without the kubectl prompt")
	}
	if EchoFor("etcdctl alarm disarm") != NodePrompt+"etcdctl alarm disarm" || EchoFor("get pods") != "> kubectl get pods" {
		t.Error("echo names the right tool")
	}
	h.Open(OverlayCouncil)
	f.Caesar.Codex = model.Codex{SizeBytes: .1 * gi, QuotaBytes: 2 * gi}
	if _, ok := findButton(h.CouncilButtons(f), "council:rebind"); ok {
		t.Error("a lean Codex needs no rebind")
	}
}

func TestContractsTabShowsHeldThenOffered(t *testing.T) {
	h, f := councilHUD(t, TabDecrees)
	f.State.RealmTime = 100
	f.State.Contracts = []view.ContractView{
		{ID: "c-offer", Merchant: "Vendor B", Title: "Crates for the fair", Goal: "Deliver 40 crates", State: "offered", Deadline: 400, Reward: view.ContractReward{Gold: 300}},
		{ID: "c-held", Merchant: "Vendor D", Title: "Sealed ledgers", Goal: "Deliver 20 sealed ledgers", State: "active", Done: 5, Total: 20, Deadline: 150},
		{ID: "c-done", Title: "Old news", State: "met"},
	}
	cards := DecreeCards(f)
	if len(cards) != 2 || cards[0].ID != "c-held" || cards[1].ID != "c-offer" {
		t.Fatalf("cards = %+v", cards)
	}
	if cards[0].State != "HELD" || cards[1].State != "OFFERED" || cards[0].Progress != .25 || cards[0].Tone != ToneWarn {
		t.Errorf("card state = %+v", cards)
	}
	if cards[0].Tribute || cards[1].Tribute {
		t.Error("contracts are never settled as tributes")
	}
	if _, ok := findButton(h.CouncilButtons(f), "council:pay:c-offer"); ok {
		t.Error("the contracts tab has no settle buttons")
	}
	if DecreesPage(Frame{}).Line != NoDecrees || DecreesPage(f).Line != DecreesLine {
		t.Error("the page line follows whether any contract is open")
	}
	if last := Advisors[len(Advisors)-1]; last.Label != "CONTRACTS" {
		t.Errorf("tab label = %q", last.Label)
	}
}

func TestLensCyclesThroughWhatIsKnown(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := caesarFrame(t)
	want := []string{LensLabor, LensUnhoused, LensMidden, LensCodex, LensNone}
	for _, mode := range want {
		h.Update(Input{Keys: []ebiten.Key{ebiten.KeyO}}, f)
		if h.Lens() != mode {
			t.Fatalf("O gave %q, want %q", h.Lens(), mode)
		}
	}
	f.Caesar = model.Caesar{}
	if NextLens(LensUnhoused, f) != LensNone {
		t.Error("without the cellars or the Codex the cycle ends")
	}
	h.SetLens(LensMidden)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEscape}}, f)
	if h.Lens() != LensNone {
		t.Error("Esc clears the map overlay")
	}
	h.SetLens(LensLabor)
	clickOn(t, h, f, "lens:"+LensLabor)
	if h.Lens() != LensNone {
		t.Error("clicking the legend clears it")
	}
}

func TestLensButtonsInTheCouncilShowTheMap(t *testing.T) {
	h, f := councilHUD(t, TabAugur)
	clickOn(t, h, f, "lens:"+LensMidden)
	if h.Lens() != LensMidden || h.Overlay() != OverlayNone {
		t.Fatalf("map button: lens %q overlay %q", h.Lens(), h.Overlay())
	}
}

func TestLegendStaysClearOfTheHUD(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := railFrame(t, 9)
	f.Caesar = caesarFixture()
	for _, mode := range LensOrder {
		h.SetLens(mode)
		r := LegendRect(mode)
		for name, other := range map[string]ui.Rect{"rail": h.RailRect(f), "console": ConsoleBox, "objective": h.ObjectiveRectOf(f), "top": TopBar} {
			if Overlaps(r, other) {
				t.Errorf("%s legend %+v overlaps the %s %+v", mode, r, name, other)
			}
		}
		if r.X < 0 || r.X+r.W > Width {
			t.Errorf("%s legend leaves the screen", mode)
		}
	}
	h.SetLens(LensLabor)
	found := false
	for _, b := range h.BlockedRects(f) {
		found = found || b == LegendRect(LensLabor)
	}
	if !found {
		t.Error("alert arrows do not step around the legend")
	}
	f.Placing = "keep"
	if h.LegendShown(f) {
		t.Error("the power overlay owns the map while placing a Keep")
	}
}
