package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAStepTheMetAgeCardSetAsideGivesWayToTheAgeStepOnItsFrameWithNoDoneBox(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	ledgers := view.ObjectiveView{ID: "ledgers", Text: "Raise clerks, then open the ledgers depot", Purpose: "Ledgers stop spoiling on the quay",
		Progress: "ready: press BUILD, then LEDGERS DEPOT", Build: "archive"}
	f.State.Objectives = []view.ObjectiveView{ledgers}
	f.Guide.Target = "build:archive"
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the ledgers step", shownStep("ledgers", false))
	tick(h, &f, 30)
	age := view.ObjectiveView{ID: "age-up", Text: "Add a second Keep to scale", Purpose: "More Keeps, more huts and new trades open with a second Keep",
		Progress: "ready: press NODE and add one", Target: "cmd:age"}
	f.State.Objectives = []view.ObjectiveView{age}
	f.Guide.Target = "cmd:age"
	tick(h, &f, 1.0/60)
	if !shownStep("age-up", false)(h.Stage(f)) {
		t.Fatalf("the step did not come on the frame its card was met: %s", h.StageDebug())
	}
	if g := h.Stage(f).Guide; g.Held != "" || g.Target != "cmd:age" {
		t.Fatalf("the button is not lit on that frame: held %q, target %q", g.Held, g.Target)
	}
	for range 120 {
		tick(h, &f, .25)
		if h.Director.Stage.DoneOn == "ledgers" || shownStep("ledgers", true)(h.Stage(f)) {
			t.Fatal("the ledgers step, set aside unmet, was shown done")
		}
	}
	noAttentionFaults(t, h)
}

func TestADoneStepStaysOnTheListAndIsShownDoneBeforeTheNextStep(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the keep step", shownStep("raise-keep", false))
	f.State.Objectives = []view.ObjectiveView{{ID: "raise-keep", Text: "Raise a Keep", Done: true},
		{ID: "door-hearth", Text: "Set a porter at the first Keep's door", Progress: "ready: press DOOR", Target: "cmd:door"}}
	waitUntil(t, h, &f, paceLimit, "the keep step's done box", shownStep("raise-keep", true))
}

func TestAnasStepSetAsideByAWardStepIsNeverShownDone(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	classes := view.ObjectiveView{ID: "clerk-classes", Text: "Pull clerk class 3.0 from the registry.", Purpose: "Each new clerk class serves more requests for the same CPU",
		Progress: "Pull clerk:3.0 from the registry: 4198/15000 MiB", Target: "cmd:soulvault"}
	f.State.Objectives = []view.ObjectiveView{classes}
	f.Guide.Target = "cmd:soulvault"
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	waitUntil(t, h, &f, paceLimit, "the clerk classes step", shownStep("clerk-classes", false))
	tick(h, &f, 30)
	ward := view.ObjectiveView{ID: "see-ward", Text: "See the Ward on the crowded Keep", Purpose: "A Keep behind a Ward takes no new huts",
		Progress: "ready: select the Keep", Target: "cmd:keep"}
	f.State.Objectives = []view.ObjectiveView{ward}
	f.Guide.Target = "cmd:keep"
	for range 240 {
		tick(h, &f, .25)
		if h.Director.Stage.DoneOn == "clerk-classes" || shownStep("clerk-classes", true)(h.Stage(f)) {
			t.Fatal("the clerk classes step, set aside unmet by the Ward's step, was shown STEP DONE")
		}
		if shownStep("see-ward", false)(h.Stage(f)) {
			return
		}
	}
	t.Fatalf("the Ward's step never took the card from the step it set aside: %s", h.StageDebug())
}
