package hud

import (
	"fmt"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func chipRowY(h *HUD, f Frame) int {
	for _, b := range h.FreightButtons(f) {
		if strings.HasPrefix(b.ID, FreightTip+"kind:") {
			return b.Rect.Y
		}
	}
	return -1
}

func TestTheHarborBoardKeepsItsHeightWhenTheBottleneckComesAndGoes(t *testing.T) {
	h := New(nil, Backend{})
	f := flowFrame(t)
	withNeck, chips := FreightRect(f), chipRowY(h, f)
	for _, e := range []view.EconomyView{{Bottleneck: view.FlowCPU}, {Fix: scaleFix()}, {}} {
		calm := f
		calm.State.Economy = e
		if got := FreightRect(calm); got != withNeck {
			t.Errorf("economy %+v: the board is %+v, with a bottleneck %+v", e, got, withNeck)
		}
		if got := chipRowY(h, calm); got != chips {
			t.Errorf("economy %+v: the chips sit at y=%d, with a bottleneck at y=%d", e, got, chips)
		}
	}
	calm := f
	calm.State.Economy = view.EconomyView{Income: 2, Profit: 2, Arriving: 1, Carried: 1}
	assertBoardFits(t, "calm board with its bottleneck row", calm)
}

func buyFixFrame(t *testing.T, available bool, reason string) Frame {
	t.Helper()
	f := flowFrame(t)
	f.State.Economy.Bottleneck = view.FlowPending
	f.State.Economy.Fix = &view.CommandOffer{ID: "buy-keep:porters", Label: "Add a Keep for more porters",
		Action: view.Action{Action: "buyKeep", KeepType: view.KeepTypeStone, For: "porters"}}
	f.State.KeepShop = []view.KeepOffer{{KeepType: view.KeepTypeStone, Name: "Stone Keep", CPU: "8", Memory: "4Gi", Cost: 100, UpkeepPerSec: .119, Available: available, Reason: reason}}
	return f
}

func TestTheBuyKeepFixShowsItsPriceAndWhyItWaits(t *testing.T) {
	defer ui.SetNamer(nil)
	h := New(nil, Backend{})
	poor := buyFixFrame(t, false, "Needs 100 MiB")
	b, ok := HitID(h.Buttons(poor), FlowbarFix)
	if !ok || !strings.Contains(b.Label, Gold(100)) || !b.Disabled || !strings.Contains(b.Tip, "Needs 100 MiB") {
		t.Fatalf("an unaffordable Keep fix = %+v %v", b, ok)
	}
	if _, acts := hasIntent(clickOn(t, h, poor, FlowbarFix), IntentAction); acts {
		t.Error("a disabled fix still bought a Keep")
	}
	rich := buyFixFrame(t, true, "")
	b, _ = HitID(h.Buttons(rich), FlowbarFix)
	if b.Disabled || !strings.Contains(b.Label, Gold(100)) {
		t.Errorf("an affordable Keep fix = %+v", b)
	}
	ui.SetNamer(names.Kubernetes(64))
	b, _ = HitID(h.Buttons(rich), FlowbarFix)
	if !strings.Contains(b.Label, "$100") {
		t.Errorf("k8s Keep fix = %+v", b)
	}
	assertBoardFits(t, "board with a priced Keep fix", poor)
}

func TestTheHarborBoardsNumbersTickAtMostOnceASecond(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := flowFrame(t)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	shown, at, changes := "", 0.0, 0
	for i := range 60 {
		f.State.Economy.Arriving += .07
		f.State.Economy.Carried += .03
		f.State.Freight.Waiting += .4
		tick(h, &f, .1)
		s := h.Stage(f)
		board := fmt.Sprintf("%s waiting %.1f", segmentTexts(FlowSegments(EconomyOf(s.State), BoardW-2*BoardPad)), s.State.Freight.Waiting)
		switch {
		case i == 0:
		case board == shown:
			continue
		case h.Now-at < CounterTick-1e-6:
			t.Fatalf("the harbor board changed %.2f s after its last change (%q to %q), under %.0f s", h.Now-at, shown, board, CounterTick)
		default:
			changes++
		}
		shown, at = board, h.Now
	}
	if changes == 0 {
		t.Fatal("the harbor board never followed the realm")
	}
}
