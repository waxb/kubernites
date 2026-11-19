package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAReadyLineThatGrowsWhileReadIsNeverCut(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	step := func(progress string) []view.ObjectiveView {
		return []view.ObjectiveView{{ID: "augury", Text: "Read about Metrics in the manual", Purpose: "Metrics show how hungry each hut is", Progress: progress,
			Hint: "Open the manual, then read about Metrics."}}
	}
	f := directorFrame(t)
	f.State.Objectives = step("Fill it: 0/100 MiB")
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	tick(h, &f, 2)
	long := "ready: press SHOW ME, then click the Observatory, then press OMENS and pick the porters, then read their hunger"
	f.State.Objectives = step(long)
	whole := false
	for range 60 {
		tick(h, &f, .5)
		text := strings.Join(strings.Fields(SurfaceText(h.ObjectiveLines(h.Stage(f)))), " ")
		if strings.Contains(text, "press SHOW ME") && !strings.Contains(text, "read their hunger") {
			t.Fatalf("the step reads %q: its ready line is cut", text)
		}
		whole = whole || strings.Contains(text, "read their hunger")
	}
	if !whole {
		t.Error("the long ready line never showed")
	}
}

func TestAReadyLineFollowsTheCommandAtOnceWithoutGrowingTheCard(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	step := func(progress string) []view.ObjectiveView {
		return []view.ObjectiveView{{ID: "depot-badge", Text: "Let the depot look for a badge: the Stewards keep its roll", Purpose: "The Stewards keep the roll true by the badge each hut wears",
			Progress: progress, Hint: "Click the depot to read its roll."}}
	}
	f := directorFrame(t)
	f.State.Objectives = step("ready: click the depot, then LOOK FOR app=porters")
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	tick(h, &f, 1)
	rect := h.ObjectiveRectOf(h.Stage(f))
	f.State.Objectives = step("ready: click the depot, then TEAR UP the hand roll: the Stewards keep only the roll they write")
	for range 4 {
		tick(h, &f, .5)
	}
	text := strings.Join(strings.Fields(SurfaceText(h.ObjectiveLines(h.Stage(f)))), " ")
	if strings.Contains(text, "LOOK FOR") || !strings.Contains(text, "TEAR UP the hand roll") {
		t.Fatalf("two seconds after a kubectl set selector the step still reads %q: the card's ready line must follow the realm while it is read", text)
	}
	if now := h.ObjectiveRectOf(h.Stage(f)); now != rect {
		t.Fatalf("the step card resized while read, from %+v to %+v", rect, now)
	}
	whole := false
	for range 60 {
		tick(h, &f, .5)
		text := strings.Join(strings.Fields(SurfaceText(h.ObjectiveLines(h.Stage(f)))), " ")
		whole = whole || strings.Contains(text, "keep only the roll they write")
	}
	if !whole {
		t.Error("the ready line's reason never showed once the card was read")
	}
}
