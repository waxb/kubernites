package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const stoneKeepCost = 200

func boardFrame() Frame {
	f := crewFrame(crewSel())
	chain := crewChain()
	chain.Resources = model.Resources{CPURequest: 250, MemRequest: 64}
	chain.QoS = model.QoSBurstable
	chain.Ready = 3
	f.Crews = []model.CrewChain{chain}
	for i := range f.Pods {
		f.Pods[i].Ready, f.Pods[i].Node = true, "lone-tower"
	}
	f.Metrics = view.MetricsSnapshot{Nodes: map[string]view.NodeMetrics{
		"lone-tower": {CPUAllocatableMilli: 2000, CPURequestedMilli: 750, MemoryAllocatableBytes: 4 << 30, MemoryRequestedBytes: 192 << 20}}}
	f.State.Freight.Kinds = []view.KindView{{Kind: "parcel", Name: "parcels", Open: true, CrewName: "porters", Depot: "parcels",
		ArrivalRate: 2, ServedRate: 1.2, SpoiledRate: .3, Waiting: 14, Value: 1}}
	f.State.KeepShop = []view.KeepOffer{{KeepType: "stone", Name: "Stone Keep", CPU: "4", Memory: "8Gi", Cost: stoneKeepCost, UpkeepPerSec: 1, Available: true}}
	f.State.Crews = []view.CrewView{{Name: "porters", Image: "porter:1.0", Replicas: 3, Ready: 3, Fits: 5,
		PerReplica: view.ReplicaCost{Income: .4, CPU: 250, Memory: 64 << 20, UpkeepShare: .1},
		KeepOffer:  &view.CrewKeepOffer{KeepOffer: f.State.KeepShop[0], Seats: 15}}}
	f.State.Keeps = []view.KeepView{{Name: "lone-tower", Open: true, Ready: true, Upkeep: .2,
		Allocatable: view.ResourcePair{CPU: 2000, Memory: 4 << 30}, Requested: view.ResourcePair{CPU: 750, Memory: 192 << 20}},
		{Name: "stone-keep-1", Upkeep: 1, Allocatable: view.ResourcePair{CPU: 4000, Memory: 8 << 30}}}
	return f
}

func openBoard(t *testing.T, f Frame) (*HUD, BoardView) {
	t.Helper()
	h := New(nil, Backend{})
	h.Press(ChainButton, f)
	h.Update(Input{}, f)
	v, ok := h.BoardView(f)
	if !ok {
		t.Fatal("CHAIN opened no crew board")
	}
	return h, v
}

func sliderAt(v BoardView, n int) (int, int) {
	return SliderX(v.Track, v.Top, float64(n)), v.Track.Y + v.Track.H/2
}

func TestTheSliderMapsToReplicas(t *testing.T) {
	track := ui.Rect{X: 100, Y: 0, W: 300, H: 16}
	for n := 0; n <= 12; n++ {
		if got := SliderValue(track, 12, SliderX(track, 12, float64(n))); got != n {
			t.Errorf("value %d maps back to %d", n, got)
		}
	}
	if SliderValue(track, 12, 0) != 0 || SliderValue(track, 12, 999) != 12 || SliderValue(ui.Rect{}, 12, 5) != 0 || SliderX(track, 0, 3) != track.X {
		t.Fatal("the slider clamps to its ends")
	}
	if SliderTop(2, 4) != 8 || SliderTop(0, 0) != SliderFloor || SliderTop(90, OpenLand) != MaxCrewSize {
		t.Fatalf("tops %d %d %d", SliderTop(2, 4), SliderTop(0, 0), SliderTop(90, OpenLand))
	}
}

func TestDraggingTheSliderPreviewsThenScalesOnRelease(t *testing.T) {
	f := boardFrame()
	h, v := openBoard(t, f)
	x, y := sliderAt(v, 6)
	h.Update(Input{MX: x, MY: y}, f)
	if got := h.Update(Input{MX: x, MY: y, Click: true, Held: true}, f); len(got) != 1 || got[0].Kind != IntentSound {
		t.Fatalf("grab = %+v", got)
	}
	x5, _ := sliderAt(v, 5)
	h.Update(Input{MX: x5, MY: y, Held: true}, f)
	preview, _ := h.BoardView(f)
	if preview.Target != 5 || len(preview.Slots) != 5 || CountSlots(preview.Slots, SlotFits) != 2 {
		t.Fatalf("preview target %d slots %+v", preview.Target, preview.Slots)
	}
	a, from := actionOf(h.Update(Input{MX: x5, MY: y}, f))
	if a != ScaleTo(crewChain(), 5) || from != FromCrewCard || CommandOf(a) != "kubectl scale deployment porters --replicas=5" {
		t.Fatalf("release = %+v from %q (%s)", a, from, CommandOf(a))
	}
	if after, _ := h.BoardView(f); after.Target != 3 || h.Board.Drag != "" {
		t.Fatalf("after release target %d drag %q", after.Target, h.Board.Drag)
	}
	x3, _ := sliderAt(v, 3)
	h.Update(Input{MX: x3, MY: y}, f)
	h.Update(Input{MX: x3, MY: y, Click: true, Held: true}, f)
	if got := h.Update(Input{MX: x3, MY: y}, f); got != nil {
		t.Fatalf("releasing on the current count ordered %+v", got)
	}
}

func TestPlusAndMinusScaleAtOnceWithTheirEcho(t *testing.T) {
	f := boardFrame()
	h, v := openBoard(t, f)
	ids := crewButtonMap(h.BoardButtons(v))
	for _, id := range []string{StepUpButton, StepDownButton, SliderButton, "crew:close", "crew:pick:rs:porters-7d9f"} {
		if _, ok := ids[id]; !ok {
			t.Errorf("the board has no %s", id)
		}
	}
	want := map[string]string{StepUpButton: "kubectl scale deployment porters --replicas=4", StepDownButton: "kubectl scale deployment porters --replicas=2"}
	for id, cmd := range want {
		a, from := actionOf(h.Press(id, f))
		if a.Action != view.ActScaleCrew || from != FromCrewCard || CommandOf(a) != cmd || !strings.Contains(ids[id].Tip, cmd) {
			t.Errorf("%s = %+v from %q, tip %q", id, a, from, ids[id].Tip)
		}
	}
	h.Hover, h.Hovering = ids[StepUpButton], true
	if hover, _ := h.BoardView(f); hover.Target != 4 || CountSlots(hover.Slots, SlotFits) != 1 {
		t.Fatalf("hovering +1 previews %d", hover.Target)
	}
	h.Hover, h.Hovering = Button{ID: ScaleUpButton}, true
	if hover, _ := h.BoardView(f); hover.Target != 4 {
		t.Fatalf("hovering the grid's SCALE +1 previews %d", hover.Target)
	}
}

func TestGhostSlotsRespectFits(t *testing.T) {
	ready := model.PodView{Name: "a", State: model.HearthReady, Node: "k"}
	camp := model.PodView{Name: "b", State: model.HearthPending}
	slots := BoardSlots([]model.PodView{ready, camp}, 5, 2)
	kinds := []string{}
	for _, s := range slots {
		kinds = append(kinds, s.Kind)
	}
	if strings.Join(kinds, ",") != "standing,rising,fits,waits,waits" {
		t.Fatalf("kinds = %v", kinds)
	}
	if got := BoardSlots([]model.PodView{ready, camp}, 1, 0); got[1].Kind != SlotLeaving || got[0].Kind != SlotStanding {
		t.Fatalf("scale down = %+v", got)
	}
	f := boardFrame()
	f.State.Crews[0].Fits = 2
	h, v := openBoard(t, f)
	if v.Facts.Fits != 2 || v.Wall() != 5 {
		t.Fatalf("fits %d wall %d", v.Facts.Fits, v.Wall())
	}
	h.Board.Drag, h.Board.Pick, h.Board.Top = DragKnob, 7, v.Top
	full, _ := h.BoardView(f)
	if full.Waiting() != 2 || CountSlots(full.Slots, SlotFits) != 2 || len(full.Buy) != 1 {
		t.Fatalf("slots %+v buy %+v", full.Slots, full.Buy)
	}
}

func TestTheDeltasComeFromTheContract(t *testing.T) {
	v := BoardView{Chain: crewChain(), Target: 7, Pods: make([]model.PodView, 3),
		Facts: CrewFacts{Fits: 2, Fed: 2, Seats: 2, Per: ReplicaFacts{Income: .4, CPU: 250, Memory: 64 << 20, UpkeepShare: .1}}}
	for i := range v.Pods {
		v.Pods[i] = model.PodView{State: model.HearthReady}
	}
	d := BoardDeltaOf(v)
	want := BoardDelta{From: 3, To: 7, Earning: 2, Waits: 2, Income: .8, CPU: 500, Memory: 128 << 20, Upkeep: .2}
	if d != want {
		t.Fatalf("delta = %+v, want %+v", d, want)
	}
	v.Target = 1
	if d := BoardDeltaOf(v); d.Earning != -2 || d.CPU != -500 || d.Waits != 0 {
		t.Fatalf("scale down delta = %+v", d)
	}
	f := boardFrame()
	h, bv := openBoard(t, f)
	h.Board.Drag, h.Board.Pick, h.Board.Top = DragKnob, 5, bv.Top
	drag, _ := h.BoardView(f)
	texts := []string{}
	for _, p := range DeltaPieces(drag, f) {
		texts = append(texts, p.Text)
	}
	joined := strings.Join(texts, "|")
	for _, part := range []string{"3 -> 5 HUTS", "+0.8 GOLD/S", "+500 RATIONS", "+128 ROOMS", "UPKEEP +0.20 GOLD/S"} {
		if !strings.Contains(joined, part) {
			t.Errorf("deltas %q lack %q", joined, part)
		}
	}
}

func TestBuyKeepSendsBuyKeepForTheCrew(t *testing.T) {
	f := boardFrame()
	f.Pods = append(f.Pods, model.PodView{Namespace: "default", Name: "porters-7d9f-d", State: model.HearthPending, Crew: "porters"})
	f.Crews[0].Foremen[0].Pods = append(f.Crews[0].Foremen[0].Pods, "porters-7d9f-d")
	f.Crews[0].Desired = 4
	f.State.Crews[0].Fits, f.State.Crews[0].Pending = 0, 1
	h, v := openBoard(t, f)
	buy, ok := crewButtonMap(h.BoardButtons(v))[BuyKeepButton]
	if !ok || buy.Disabled || buy.Label != "BUY KEEP "+ui.IconGold+"200, "+ui.IconGold+"1.0/S" || !strings.HasPrefix(buy.Tip, "+4,000 rations, room for 15 porters (the 1 waiting move in first), upkeep up to 1.0 gold/s\n") {
		t.Fatalf("buy keep = %+v", buy)
	}
	a, from := actionOf(h.Press(BuyKeepButton, f))
	if a.Action != "buyKeep" || a.KeepType != "stone" || a.For != "porters" || from != FromCrewCard {
		t.Fatalf("buy = %+v from %q", a, from)
	}
	closed := *f.State.Crews[0].KeepOffer
	closed.Available, closed.Reason = false, "Needs a second Keep: the first holds two huts"
	f.State.Crews[0].KeepOffer = &closed
	_, v = openBoard(t, f)
	if b := v.Buy[0]; !b.Disabled || b.Tip != "Needs a second Keep: the first holds two huts" {
		t.Fatalf("a closed shop = %+v", b)
	}
	f.State.Crews[0].KeepOffer = nil
	if got := h.Press(BuyKeepButton, f); got != nil {
		t.Fatalf("no offer bought %+v", got)
	}
}

func TestTheHornBandShowsAndRetunes(t *testing.T) {
	f := hornedCrewFrame()
	h, v := openBoard(t, f)
	if !v.Band || len(v.Steps) != 1 || v.Steps[0].ID != RetuneButton || !v.Steps[0].Disabled {
		t.Fatalf("band %v steps %+v", v.Band, v.Steps)
	}
	x, y := SliderX(v.Track, v.Top, 4), v.Track.Y+v.Track.H/2
	h.Update(Input{MX: x, MY: y}, f)
	h.Update(Input{MX: x, MY: y, Click: true, Held: true}, f)
	x6 := SliderX(v.Track, v.Top, 6)
	h.Update(Input{MX: x6, MY: y, Held: true}, f)
	if got := h.Update(Input{MX: x6, MY: y}, f); got != nil || h.Crew.Horn.Max != 6 || h.Crew.Horn.Min != 2 {
		t.Fatalf("drag ceiling: %+v horn %+v", got, h.Crew.Horn)
	}
	after, _ := h.BoardView(f)
	if after.Steps[0].Disabled {
		t.Fatal("RETUNE stays disabled after the band moved")
	}
	a, _ := actionOf(h.Press(RetuneButton, f))
	if a.Action != view.ActRetuneHorn || a.Crew.Max != 6 || a.Crew.Min != 2 {
		t.Fatalf("retune = %+v", a)
	}
	if BandPick(HornPick{Min: 3, Max: 5}, DragMin, 9).Min != 5 || BandPick(HornPick{Min: 3, Max: 5}, DragMax, 0).Max != 3 {
		t.Fatal("the band's ends cannot cross")
	}
	if NearerHandle(v, HornPick{Min: 2, Max: 4}, v.Track.X) != DragMin || NearerHandle(v, HornPick{Min: 2, Max: 4}, v.Track.X+v.Track.W) != DragMax {
		t.Fatal("grab picks the nearer end")
	}
}

func TestHutsPopInOnTheRealClock(t *testing.T) {
	f := boardFrame()
	h, _ := openBoard(t, f)
	f.Clock = 20
	h.Update(Input{}, f)
	if dy, fade := h.PopOffset(0); dy != 0 || fade != 1 {
		t.Fatalf("a standing hut pops: %v %v", dy, fade)
	}
	f.Crews[0].Desired = 4
	h.Update(Input{}, f)
	if dy, _ := h.PopOffset(3); dy >= 0 {
		t.Fatalf("a new hut does not drop in: %v", dy)
	}
	f.Clock += PopTime
	h.Update(Input{}, f)
	if dy, _ := h.PopOffset(3); dy != 0 {
		t.Fatalf("the drop does not settle: %v", dy)
	}
	h.Reduce = true
	if dy, fade := h.PopOffset(3); dy != 0 || fade != 1 {
		t.Fatal("reduced motion still animates")
	}
}

func TestTheBoardsGiveWayToOtherModesAndBareCrews(t *testing.T) {
	f := boardFrame()
	h, _ := openBoard(t, f)
	pressed(t, h, f, "crew:mode:"+CrewModeImage)
	if _, ok := h.BoardView(f); ok {
		t.Fatal("the board stays over the UPGRADE page")
	}
	pressed(t, h, f, "crew:mode:")
	if _, ok := h.BoardView(f); !ok {
		t.Fatal("BACK does not return to the board")
	}
	if got := h.Press(CrewBoardButton, f); len(got) == 0 || got[0].Kind != IntentChoose || got[0].Command != "default/porters" {
		t.Fatalf("crew board button = %+v", got)
	}
	if got := h.Press("crew:close", f); len(got) == 0 || got[0].Kind != IntentSelect {
		t.Fatalf("X deselects the crew: %+v", got)
	}
	f.Selection = Selection{}
	h.Update(Input{}, f)
	if h.Board.Crew != "" {
		t.Fatal("a closed board keeps its state")
	}
}

func TestTheBoardReadsTheContractAndFallsBackWithoutIt(t *testing.T) {
	f := boardFrame()
	k := CrewFactsOf(f, f.Crews[0])
	if k.Per != (ReplicaFacts{Income: .4, CPU: 250, Memory: 64 << 20, UpkeepShare: .1}) || k.Fits != 5 || len(k.Keeps) != 1 || k.Keeps[0].Upkeep != .2 {
		t.Fatalf("contract facts = %+v", k)
	}
	if cpu, mem := k.LandTotals(); cpu != [2]int64{750, 2000} || mem != [2]int64{192 << 20, 4 << 30} {
		t.Fatalf("land totals %v %v", cpu, mem)
	}
	f.State.Crews, f.State.Keeps = nil, nil
	f.Crews[0].Resources = model.Resources{CPURequest: 250, MemRequest: 64}
	f.Nodes = map[string]model.NodeView{"lone-tower": {Name: "lone-tower"}, "tainted": {Name: "tainted", TaintSpecs: []string{"node-role.kubernetes.io/control-plane:NoSchedule"}}}
	f.Metrics.Nodes["tainted"] = view.NodeMetrics{CPUAllocatableMilli: 8000, MemoryAllocatableBytes: 8 << 30}
	k = CrewFactsOf(f, f.Crews[0])
	if k.Per.CPU != 250 || k.Per.Memory != 64<<20 || k.Fits != 5 || len(k.Keeps) != 1 || k.Per.Income != 0 {
		t.Fatalf("fallback facts = %+v", k)
	}
	f.Crews[0].Resources = model.Resources{}
	if k = CrewFactsOf(f, f.Crews[0]); k.Bounded() {
		t.Fatalf("a crew with no requests has bounded land: %+v", k)
	}
	if !Schedulable(model.NodeView{}) || Schedulable(model.NodeView{Unschedulable: true}) || RoomFor([2]int64{0, 100}, 0) != OpenLand {
		t.Fatal("schedulable land")
	}
}

func TestRolloutOrdersFoldIntoOnePage(t *testing.T) {
	f := boardFrame()
	h, v := openBoard(t, f)
	ids := crewButtonMap(h.CommandButtons(f))
	if _, ok := ids[CrewModeOpen+CrewModeRoll]; !ok {
		t.Fatalf("no ROLLOUT among %v", ids)
	}
	for _, id := range []string{"crew:roll:restart", "crew:roll:undo"} {
		if _, ok := ids[id]; ok {
			t.Errorf("%s still sits on the crew's grid", id)
		}
	}
	pressed(t, h, f, CrewModeOpen+CrewModeRoll)
	page := crewButtonMap(h.CommandButtons(f))
	for _, id := range []string{"crew:roll:restart", "crew:roll:undo", "crew:roll:pause", "crew:history", "crew:mode:"} {
		if _, ok := page[id]; !ok {
			t.Errorf("the ROLLOUT page lacks %s", id)
		}
	}
	if !strings.Contains(RolloutLine(f.Crews[0]), "Revision 2, 1 kept") {
		t.Fatalf("rollout line %q", RolloutLine(f.Crews[0]))
	}
	x, y, ok := h.SliderPoint(f, 0)
	if ok || x != 0 || y != 0 {
		t.Fatal("the slider has a point while the ROLLOUT page shows")
	}
	pressed(t, h, f, "crew:mode:")
	if x, _, ok := h.SliderPoint(f, 3); !ok || x != SliderX(v.Track, v.Top, 3) {
		t.Fatalf("slider point %d %v", x, ok)
	}
}

func TestTheCalamityCardAndTheCrewBoardNeverCoverEachOther(t *testing.T) {
	f := boardFrame()
	f.State.Reveals = append(append([]string(nil), allReveals...), "stone")
	f.State.Calamities = []view.CalamityView{noDepot, oom}
	h := New(nil, Backend{})
	h.Press(ChainButton, f)
	f = settled(h, f)
	card, shown := h.CalamityRect(f)
	boards := h.CrewRects(f)
	if !shown || len(boards) == 0 {
		t.Fatalf("card %v boards %v", shown, boards)
	}
	if Overlaps(card, boards[0]) {
		t.Errorf("the calamity card %+v and the crew board %+v cover each other", card, boards[0])
	}
}
