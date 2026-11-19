package hud

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	pileBeaconW = 40
	pileBeaconH = 60
	pileShift   = 6
)

func carryStepFrame(t *testing.T) Frame {
	t.Helper()
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "build-chancery", Text: "Raise the Chancery: kubectl is dark",
		Purpose: "The only way into the control plane: you and every crew speak through it", Progress: "CPU 2/11: carry the boxes on the quay", Build: "chancery"}}
	f.Guide.Target = "build:chancery"
	return f
}

func TestTheToastStackKeepsItsPlaceWhileBoxesAreCarriedAndNewToastsCome(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	f := carryStepFrame(t)
	tick(h, &f, 1)
	step := h.ObjectiveRectOf(h.Stage(f))
	pile := ui.Rect{X: Width/2 - pileBeaconW/2, Y: step.Y + step.H + 4, W: pileBeaconW, H: pileBeaconH}
	f.Beacons = []ui.Rect{pile}
	h.Toast("The Codex stands complete", false)
	h.Toast("Done: Check the control plane: nothing can be written down yet", false)
	tick(h, &f, .1)
	h.Draw(dst, f)
	before := slices.Clone(h.ToastRects(h.Stage(f)))
	for i, beacons := range [][]ui.Rect{{pile.Offset(pileShift, -pileShift)}, nil, {pile}} {
		f.Beacons = beacons
		f.State.Objectives[0].Progress = "CPU 3/11: carry the boxes on the quay, the next shift brings more"
		tick(h, &f, .1)
		h.Draw(dst, f)
		if got := h.ToastRects(h.Stage(f)); !slices.Equal(got, before) {
			t.Fatalf("click %d: the toasts moved from %v to %v", i, before, got)
		}
	}
	h.Toast("A box spoiled on the quay", false)
	h.Toast("The Codex stands complete", false)
	tick(h, &f, .1)
	h.Draw(dst, f)
	got := h.ToastRects(h.Stage(f))
	if len(got) != 3 {
		t.Fatalf("toasts %v", h.Toasts())
	}
	if got[0] != before[1] || got[2].Y != before[0].Y {
		t.Fatalf("new toasts moved the old ones from %v to %v", before, got)
	}
	if Overlaps(got[1], got[0]) || Overlaps(got[1], got[2]) {
		t.Fatalf("the new toast %v lies over the old ones %v", got[1], got)
	}
}

func TestABannerKeepsItsPlaceWhileTheBeaconsUnderItMove(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	f := carryStepFrame(t)
	tick(h, &f, 1)
	h.BannerCause("HARBOR TIER 2", "Bigger ships dock at your deepest port: they bring ledgers")
	for range 10 {
		tick(h, &f, .1)
	}
	h.Draw(dst, f)
	first, ok := h.BannerRect(h.Stage(f))
	if !ok {
		t.Fatal("no banner shows")
	}
	f.Beacons = []ui.Rect{first.Inset(-4)}
	tick(h, &f, .05)
	h.Draw(dst, f)
	if got, _ := h.BannerRect(h.Stage(f)); got != first {
		t.Fatalf("the banner jumped from %v to %v when a beacon lit under it", first, got)
	}
}

func TestANewsBoxWaitsForAPlaceClearOfAnOpenWindow(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := boardFrame()
	f.State.Objectives = []view.ObjectiveView{{ID: "harbor-two", Text: "Upgrade a port for caravels", Purpose: "Caravels carry more boxes, and more requests, to every berth of the Gate Harbor",
		Progress: "ready: click the Gate Harbor, then UPGRADE PORT", Hint: "Click the Gate Harbor and press UPGRADE PORT. Caravels are bigger ships: each brings more boxes, so your porters carry more and the realm serves more each second. More boxes also mean more load on your huts and Keeps."}}
	f.WorldBoxes = []ui.Rect{{X: 420, Y: 150, W: 120, H: 50}}
	h.Press(ChainButton, f)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	if _, ok := h.BoardView(h.Stage(f)); !ok {
		t.Fatal("CHAIN opened no crew board: the test proves nothing")
	}
	news := "The numbers are read: TOP PODS lists the food and housing each of your 7 huts really uses right now. Workers eat more while boxes wait: pledge what they eat at noon."
	h.News(news, false)
	ticks(h, &f, 30)
	noAttentionFaults(t, h)
	f.Selection = Selection{}
	waitUntil(t, h, &f, paceLimit, "the news once the board is closed", func(Frame) bool { return h.toastShown(news) })
	ticks(h, &f, 5)
	noAttentionFaults(t, h)
}
