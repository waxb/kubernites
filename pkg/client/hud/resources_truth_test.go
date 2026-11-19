package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func defaultsFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.LimitRange = &view.LimitRangeView{CPU: 200, Memory: 64 * mebi, CPULimit: 500, MemoryLimit: 1024 * mebi}
	return f
}

func TestTheResourcesPageShowsThePledgeTheCrewCardShows(t *testing.T) {
	f := defaultsFrame(t)
	pick := model.Resources{}
	food, _ := FieldNamed("cpu-req")
	value, inherited := resValue(food, pick, f)
	if !strings.Contains(value, "200") || !inherited {
		t.Fatalf("the crew card says every worker pledges 200 rations, RESOURCES says %q (inherited %v)", value, inherited)
	}
	if PickQoS(EffectiveResources(pick, f)) == model.QoSBestEffort {
		t.Fatal("workers given the realm's default pledge are called Vagrant (BestEffort)")
	}
	if line := pledgeRoomLine(f, model.CrewChain{Name: "porters"}, pick); strings.HasPrefix(line, "No pledge") || strings.HasPrefix(line, "No requests") {
		t.Fatalf("the page says there is no pledge while the default applies: %q", line)
	}
	if got := EffectiveResources(model.Resources{MemLimit: 448}, f).MemRequest; got != 448 {
		t.Fatalf("a binding with no pledge of its own pledges the binding, as the apiserver defaults it: %d", got)
	}
}

func TestAStepFromTheDefaultStartsAtTheDefault(t *testing.T) {
	h := New(nil, Backend{})
	f := defaultsFrame(t)
	f.State.Grimoire = append(f.State.Grimoire, view.GrimoirePage{ID: "pledges", Ranks: []view.RankView{{Learned: true, Runes: []string{"resources.requests.cpu"}}}})
	c := model.CrewChain{Name: "clerks"}
	h.pressResources(c, "cpu-req:up", f)
	if got := h.Crew.Res.CPURequest; got <= 200 {
		t.Fatalf("the first + from the 200-ration default pledged %d", got)
	}
}

func TestUseSuggestsTheDaysMeasuredPeakAndNothingBeforeANoon(t *testing.T) {
	food, _ := FieldNamed("cpu-req")
	if got := food.measured(view.UsePeakView{CPU: 300, Window: 300}); got != 0 {
		t.Fatalf("before the crew has worked through a noon tide USE offers %d", got)
	}
	peak := view.UsePeakView{CPU: 300, Memory: 330 * mebi, Window: 720, Noon: true}
	if got := food.measured(peak); got != 300 {
		t.Fatalf("USE offers %d against a measured day peak of 300", got)
	}
	room, _ := FieldNamed("mem-req")
	if got := room.measured(peak); got != 352 {
		t.Fatalf("housing USE is the peak rounded up to a step: %d", got)
	}
	binding, _ := FieldNamed("cpu-lim")
	if got := binding.measured(peak); got != 0 {
		t.Fatalf("a Binding the workers never reach is offered a raise to %d", got)
	}
	bound := view.UsePeakView{CPU: 500, Window: 720, Noon: true, Bound: true, Limit: 500, Raise: 1000}
	if got := food.measured(bound); got > bound.Limit {
		t.Fatalf("USE offers %d above the Binding of %d", got, bound.Limit)
	}
	if got := binding.measured(bound); got != 1000 {
		t.Fatalf("workers at their Binding are not offered the raise first: %d", got)
	}
	if tip := useTip(food, 500, bound); !strings.Contains(tip, "raise the Binding first") {
		t.Fatalf("the USE tip for workers at their Binding does not say to raise it first: %q", tip)
	}
	if tip := useTip(food, 300, peak); !strings.Contains(tip, "last day") {
		t.Fatalf("the USE tip does not name its window: %q", tip)
	}
}

func TestABindingBelowWhatAWorkerTakesIsWarnedOf(t *testing.T) {
	use := useFacts{Pods: 2, Memory: 2 * 352 * mebi, CPU: 2 * 300}
	if warn := bindingWarning(model.Resources{MemLimit: 32}, use); !strings.Contains(warn, "352") {
		t.Fatalf("a Housing Binding of 32 against a use of 352 is not warned of: %q", warn)
	}
	if warn := bindingWarning(model.Resources{MemLimit: 448, CPULimit: 500}, use); warn != "" {
		t.Fatalf("a binding above use is warned of: %q", warn)
	}
}
