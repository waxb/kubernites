package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	offerSeatsPorters = 31
	offerSeatsClerks  = 15
)

func computeOffer() *view.CrewKeepOffer {
	return &view.CrewKeepOffer{Seats: offerSeatsPorters, KeepOffer: view.KeepOffer{KeepType: view.KeepTypeStone, Name: "Stone Keep", CPU: "8", Memory: "4Gi",
		Instance: "compute-xlarge", Class: "compute", Territory: 5, Cost: stoneKeepCost, UpkeepPerSec: .5, Available: true}}
}

func memoryOffer() *view.CrewKeepOffer {
	return &view.CrewKeepOffer{Seats: offerSeatsClerks, KeepOffer: view.KeepOffer{KeepType: view.KeepTypeStone, Name: "Stone Keep", CPU: "3", Memory: "16Gi",
		Instance: "memory-xlarge", Class: "memory", Territory: 5, Cost: stoneKeepCost, UpkeepPerSec: .25, Available: true}}
}

func clerksChain() model.CrewChain {
	fm := model.Foreman{Namespace: "default", Name: "clerks-5b6c", Hash: "5b6c", Revision: 1, Desired: 2, Ready: 1, Newest: true,
		Image: "clerk:1.0", Pods: []string{"clerks-5b6c-a", "clerks-5b6c-b"}}
	return model.CrewChain{Namespace: "default", Name: "clerks", Image: "clerk:1.0", Container: "clerk", Desired: 2, Ready: 1, Updated: 2,
		Strategy: "RollingUpdate", Revision: 1, HistoryLimit: 10, Depot: "ledgers", Foremen: []model.Foreman{fm},
		Resources: model.Resources{CPURequest: 100, MemRequest: 1024}, QoS: model.QoSBurstable}
}

func clerkPod(name string, state string) model.PodView {
	return model.PodView{Namespace: "default", Name: name, Owner: "ReplicaSet/clerks-5b6c", State: state, Crew: "clerks", Ready: state == model.HearthReady}
}

func twoCrewBoardFrame(crew string) Frame {
	f := boardFrame()
	f.Selection = Selection{Kind: SelectCrew, ID: "default/" + crew}
	f.Pods = append(f.Pods, model.PodView{Namespace: "default", Name: "porters-7d9f-d", State: model.HearthPending, Crew: "porters"},
		clerkPod("clerks-5b6c-a", model.HearthReady), clerkPod("clerks-5b6c-b", model.HearthPending))
	f.Crews[0].Foremen[0].Pods = append(f.Crews[0].Foremen[0].Pods, "porters-7d9f-d")
	f.Crews[0].Desired = 4
	f.Crews = append(f.Crews, clerksChain())
	f.State.Crews[0].Fits, f.State.Crews[0].Pending, f.State.Crews[0].KeepOffer = 0, 1, computeOffer()
	f.State.Crews = append(f.State.Crews, view.CrewView{Name: "clerks", Image: "clerk:1.0", Replicas: 2, Ready: 1, Pending: 1,
		PerReplica: view.ReplicaCost{Income: .2, CPU: 100, Memory: 1 << 30, UpkeepShare: .05}, KeepOffer: memoryOffer()})
	f.State.KeepShop = []view.KeepOffer{computeOffer().KeepOffer}
	return f
}

func boardBuy(t *testing.T, f Frame) (*HUD, Button) {
	t.Helper()
	h, v := openBoard(t, f)
	b, ok := crewButtonMap(h.BoardButtons(v))[BuyKeepButton]
	if !ok {
		t.Fatalf("the %s board has no keep offer button", v.Chain.Name)
	}
	return h, b
}

func TestEachCrewBoardSellsTheKeepItsOwnCrewGets(t *testing.T) {
	cases := []struct {
		crew  string
		offer *view.CrewKeepOffer
		want  []string
		not   string
	}{
		{"porters", computeOffer(), []string{"+8,000 rations, room for 31 porters (the 1 waiting move in first), upkeep up to 0.50 gold/s", "compute-xlarge (compute land), 8,000 rations of food and 4,096 rooms of housing"}, "memory-xlarge"},
		{"clerks", memoryOffer(), []string{"+3,000 rations, room for 15 clerks (the 1 waiting move in first), upkeep up to 0.25 gold/s", "memory-xlarge (memory land), 3,000 rations of food and 16,384 rooms of housing"}, "compute-xlarge"},
	}
	for _, c := range cases {
		f := twoCrewBoardFrame(c.crew)
		h, b := boardBuy(t, f)
		if want := "BUY KEEP " + Gold(c.offer.Cost) + ", " + Money(Rate(c.offer.UpkeepPerSec)) + "/S"; b.Label != want || b.Disabled {
			t.Errorf("%s: label %q (disabled %v), want %q", c.crew, b.Label, b.Disabled, want)
		}
		for _, w := range c.want {
			if !strings.Contains(b.Tip, w) {
				t.Errorf("%s: tip %q lacks %q", c.crew, b.Tip, w)
			}
		}
		if strings.Contains(b.Tip, c.not) {
			t.Errorf("%s: tip %q names the other crew's plot %s", c.crew, b.Tip, c.not)
		}
		a, _ := actionOf(h.Press(BuyKeepButton, f))
		if a != BuyKeepAction(model.CrewChain{Name: c.crew}, c.offer.KeepOffer) || a.For != c.crew || a.KeepType != c.offer.KeepType {
			t.Errorf("%s: pressed %+v", c.crew, a)
		}
	}
}

func TestTheBoardShowsItsOwnCrewsReasonWhenTheShopCrewCanBuy(t *testing.T) {
	f := twoCrewBoardFrame("clerks")
	closed := memoryOffer()
	closed.Available, closed.Reason = false, "No open land holds enough forest and stone for a Stone Keep: charter an isle"
	f.State.Crews[1].KeepOffer = closed
	if _, b := boardBuy(t, f); !b.Disabled || b.Tip != closed.Reason {
		t.Fatalf("the clerks' closed offer shows %+v", b)
	}
	f.State.Crews[1].KeepOffer = nil
	h, v := openBoard(t, f)
	if _, ok := crewButtonMap(h.BoardButtons(v))[BuyKeepButton]; ok {
		t.Fatal("a crew with no Keep offer is sold the shop's Keep")
	}
	if got := h.Press(BuyKeepButton, f); got != nil {
		t.Fatalf("a crew with no Keep offer bought %+v", got)
	}
}

func TestTheWaitingHutsSayWhenTheOfferedKeepHoldsNone(t *testing.T) {
	f := twoCrewBoardFrame("clerks")
	_, v := openBoard(t, f)
	if got := waitCaptionsOf(v)[0][0]; got != "WAITING: NEEDS A KEEP" {
		t.Fatalf("a Keep that seats 15 clerks: %q", got)
	}
	f.State.Crews[1].KeepOffer.Seats = 0
	_, v = openBoard(t, f)
	if got := waitCaptionsOf(v)[0][0]; got != "WAITING: TOO BIG FOR A KEEP" {
		t.Fatalf("a Keep that seats no clerk: %q", got)
	}
	tip := v.Buy[0].Tip
	if !strings.Contains(tip, "room for 0 clerks") || !strings.Contains(tip, "each worker pledges more than it holds") || strings.Contains(tip, "move in") {
		t.Fatalf("a Keep that seats no clerk promises %q", tip)
	}
}

func twoCrewOffers(f Frame) Frame {
	next := f
	next.State.Crews = []view.CrewView{
		{Name: "porters", PerReplica: view.ReplicaCost{Income: .4, CPU: 250, Memory: 64 << 20}, KeepOffer: computeOffer()},
		{Name: "clerks", PerReplica: view.ReplicaCost{Income: .2, CPU: 100, Memory: 1 << 30}, KeepOffer: memoryOffer()}}
	next.State.KeepShop = []view.KeepOffer{computeOffer().KeepOffer}
	return next
}

func buyFor(crew string) view.Action {
	return view.Action{Action: "buyKeep", KeepType: view.KeepTypeStone, For: crew}
}

const clerksPayoff = "+3,000 rations, room for 15 clerks, upkeep up to 0.25 gold/s"

func TestTheFlowFixSellsTheKeepOfTheCrewItBuysFor(t *testing.T) {
	f := twoCrewOffers(flowFrame(t))
	f.State.Economy.Bottleneck = view.FlowPending
	f.State.Economy.Fix = &view.CommandOffer{ID: "buy-keep:clerks", Label: "Add a Keep for more clerks", Action: buyFor("clerks")}
	row := BoardRowRect(FreightRect(f), FlowbarFirst(f))
	b, ok := FixButton(f, row)
	if !ok || b.Disabled || !strings.Contains(b.Tip, clerksPayoff) || strings.Contains(b.Tip, "porters") {
		t.Fatalf("the clerks' fix = %+v", b)
	}
	closed := memoryOffer()
	closed.Cost, closed.Available, closed.Reason = 900, false, "Needs 900 MiB"
	f.State.Crews[1].KeepOffer = closed
	if b, _ := FixButton(f, row); !b.Disabled || b.Tip != closed.Reason || !strings.Contains(b.Label, Gold(900)) {
		t.Fatalf("the clerks' closed offer sizes the fix %+v", b)
	}
}

func TestTheCalamityRemedySellsTheKeepOfItsCrew(t *testing.T) {
	a := buyFor("clerks")
	pending := view.CalamityView{ID: "pending:clerks", Kind: view.CalamitySpoiling, Object: "kind/ledger", Cause: "Clerks wait for room",
		Why: "No Keep has room.", Remedies: []view.RemedyView{{Label: "Add a Keep for more clerks", Action: &a}}}
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, twoCrewOffers(calamityFrame(t, pending)))
	do, ok := HitID(h.CalamityButtons(f), "line:cal:0:do")
	if !ok || !strings.Contains(do.Tip, clerksPayoff) || strings.Contains(do.Tip, "porters") {
		t.Fatalf("the clerks' remedy = %+v", do)
	}
}

func TestACrewThatPledgesNothingIsSoldHandsAndUpkeepOnly(t *testing.T) {
	f := twoCrewOffers(flowFrame(t))
	f.State.Crews[1].PerReplica.CPU, f.State.Crews[1].PerReplica.Memory = 0, 0
	f.State.Crews[1].KeepOffer.Seats = 0
	f.State.Economy.Fix = &view.CommandOffer{ID: "buy-keep:clerks", Label: "Add a Keep for more clerks", Action: buyFor("clerks")}
	b, _ := FixButton(f, BoardRowRect(FreightRect(f), FlowbarFirst(f)))
	if !strings.Contains(b.Tip, "\n+3,000 rations, upkeep up to 0.25 gold/s\n") || strings.Contains(b.Tip, "room for") {
		t.Fatalf("a crew that pledges nothing is sold %q", b.Tip)
	}
}

func TestTheCrewBoardSellsItsOwnKeepInKubernetesNames(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	_, b := boardBuy(t, twoCrewBoardFrame("clerks"))
	if b.Label != "ADD NODE $200, $0.25/S" || !strings.HasPrefix(b.Tip, "+3 CPU, room for 15 clerks (the 1 Pending land first), upkeep up to $0.25/s\n") || !strings.Contains(b.Tip, "memory-xlarge") {
		t.Fatalf("the clerks' Node = %q, %q", b.Label, b.Tip)
	}
}
