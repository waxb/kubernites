package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

const paceLimit = 300.0

func waitUntil(t *testing.T, h *HUD, f *Frame, limit float64, what string, cond func(Frame) bool) float64 {
	t.Helper()
	for waited := 0.0; waited <= limit; waited += .25 {
		if cond(h.Stage(*f)) {
			return h.Now
		}
		tick(h, f, .25)
	}
	t.Fatalf("%s never came within %.0f s: %s", what, limit, h.StageDebug())
	return 0
}

func noAttentionFaults(t *testing.T, h *HUD) {
	t.Helper()
	log := h.Attention().Closed(h.Now, 0)
	if faults := CheckAttention(log, AttnRules{}); len(faults) > 0 {
		lines := []string{}
		for _, f := range faults {
			lines = append(lines, f.String())
		}
		t.Fatalf("attention faults:\n  %s\nboxes %v", strings.Join(lines, "\n  "), boxKeys(log))
	}
}

func boxKeys(log AttnLog) []string {
	out := []string{}
	for _, b := range log.Boxes {
		if !b.Entry {
			out = append(out, b.Key)
		}
	}
	return out
}

func shownStep(id string, done bool) func(Frame) bool {
	return func(s Frame) bool {
		o, ok := s.ShownObjective()
		return ok && o.ID == id && o.Done == done
	}
}

func TestACardClosedBeforeItsReadingTimeFreesTheSlotFromWhenItLeft(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	h.Result("CPU, RATE AND OVERHEAD", "The number is your CPU. The +/S beside it is throughput: the requests huts answer each second. From the second view on a red -/S joins it: overhead, what your Keeps use each second. Throughput minus overhead is what your CPU gains.")
	waitUntil(t, h, &f, paceLimit, "the card", func(s Frame) bool { return h.LessonShown(s) })
	if read := h.Director.Stage.Slot.Read; read < 2*CalmGap {
		t.Fatalf("the card reads in %.1f s: too short to tell a closed card from a read one", read)
	}
	tick(h, &f, 2)
	clickOn(t, h, f, LessonAck)
	closed := h.Now
	h.FeatBanner("THE ROLLOUT IS COMPLETE", "Every crew holds its count and every shift reports its huts as ready")
	at := waitUntil(t, h, &f, paceLimit, "the banner after the closed card", func(Frame) bool { return h.BannerActive() })
	if at > closed+CalmGap+1 {
		t.Fatalf("the banner came %.1f s after the card was closed: the closed card held the slot through the rest of its reading time", at-closed)
	}
	noAttentionFaults(t, h)
}

func TestAStepsDoneBoxComesBeforeTheBannerItsStepRaised(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	tick(h, &f, CalmGap)
	f.State.Objectives = []view.ObjectiveView{{ID: "raise-keep", Text: "Raise a Keep", Done: true}}
	h.FeatBanner("THE ROLLOUT IS COMPLETE", "Every crew holds its count and every shift reports its huts as ready")
	waitUntil(t, h, &f, paceLimit, "the banner", func(Frame) bool { return h.BannerActive() })
	keys := boxKeys(h.Attention().Closed(h.Now, 0))
	if i := slices.Index(keys, "done:raise-keep"); i < 0 || i > slices.IndexFunc(keys, func(k string) bool { return strings.HasPrefix(k, ActBanner) }) {
		t.Fatalf("the banner came before the done box of the step that raised it: %v", keys)
	}
	noAttentionFaults(t, h)
}

func TestARevealIsNoBoxAndComesThroughTheCalmAFloorAfterTheLastBox(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f.State.Reveals = slices.DeleteFunc(slices.Clone(f.State.Reveals), func(id string) bool { return id == "stone" })
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	h.Result("WATCH THE KUBELET", "The kubelet restarts the container that stopped.")
	waitUntil(t, h, &f, paceLimit, "the card", func(s Frame) bool { return h.LessonShown(s) })
	tick(h, &f, 1)
	clickOn(t, h, f, LessonAck)
	closed := h.Now
	f.State.Reveals = append(slices.Clone(f.State.Reveals), "stone")
	at := waitUntil(t, h, &f, paceLimit, "the Stone's reveal", func(s Frame) bool { return s.Revealed("stone") })
	if at > closed+BoxFloor+1 {
		t.Fatalf("the Stone was revealed %.1f s after the card closed: a reveal is no box and waits only the floor", at-closed)
	}
}

func TestAResultTaughtOnceIsNotQueuedASecondTime(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.ResultOnce(GildedTaught, "A TRAFFIC SPIKE ARRIVED", "Answer one.", nil)
	h.LessonCard = LessonCard{}
	h.Results = nil
	h.ResultOnce(GildedTaught, "A TRAFFIC SPIKE ARRIVED", "Answer one.", nil)
	if len(h.Results) > 0 || h.LessonCard.Shown {
		t.Fatal("a result taught once came again")
	}
	if !h.TaughtChanged() {
		t.Fatal("teaching the result once does not ask the settings to save it")
	}
}

func TestABoxWaitsAFloorAfterAReveal(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f.State.Reveals = slices.DeleteFunc(slices.Clone(f.State.Reveals), func(id string) bool { return id == "wisdom" })
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	tick(h, &f, 30)
	f.State.Reveals = append(slices.Clone(f.State.Reveals), "wisdom")
	revealed := waitUntil(t, h, &f, paceLimit, "the points meter", func(s Frame) bool { return s.Revealed("wisdom") })
	h.Result("WATCH THE KUBELET", "The kubelet restarts the container that stopped.")
	card := waitUntil(t, h, &f, paceLimit, "the card", func(s Frame) bool { return h.LessonShown(s) })
	if card < revealed+BoxFloor {
		t.Fatalf("a card came %.1f s after a reveal, under the %.0f s floor: two new things at once", card-revealed, BoxFloor)
	}
}

func TestARevealWaitsAFloorAfterABoxCame(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f.State.Reveals = slices.DeleteFunc(slices.Clone(f.State.Reveals), func(id string) bool { return id == "wisdom" })
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	tick(h, &f, 30)
	h.News("A Pod restarted in the cluster", false)
	shown := waitUntil(t, h, &f, paceLimit, "the toast", func(Frame) bool { return len(h.Toasts()) > 0 })
	f.State.Reveals = append(slices.Clone(f.State.Reveals), "wisdom")
	revealed := waitUntil(t, h, &f, paceLimit, "the points meter", func(s Frame) bool { return s.Revealed("wisdom") })
	if revealed < shown+BoxFloor {
		t.Fatalf("a reveal came %.1f s after a box, under the %.0f s floor: two new things at once", revealed-shown, BoxFloor)
	}
}

func TestTheNextBoardCallsNothingWhileTheStepIsHeld(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f.Guide.Staged, f.Guide.Introduced, f.Guide.Target = true, nil, ""
	carry := view.ObjectiveView{ID: "serve-hand", Text: "Carry 5 parcels from the quay by hand", Progress: "0/5 carried"}
	keep := view.ObjectiveView{ID: "raise-keep", Text: "Raise a Keep", Progress: "ready: press BUILD", Build: "keep"}
	f.State.Objectives = []view.ObjectiveView{carry}
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	tick(h, &f, 30)
	carry.Done = true
	f.State.Objectives = []view.ObjectiveView{carry, keep}
	f.Guide.Target = "build:keep"
	tick(h, &f, .25)
	if s := h.Stage(f); s.Guide.Held != "" && s.Shown("cmd:build") {
		t.Fatal("BUILD shows through the NEXT board while the Keep's step is still held: the step's own button comes with its box")
	}
}
