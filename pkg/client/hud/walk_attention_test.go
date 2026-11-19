package hud

import (
	"reflect"
	"testing"
)

func TestAGuideWalkLeavesTheAttentionLogAndTheTopBarAsTheyWere(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	tick(h, &f, 5)
	before, top := h.Attention(), h.top.stats[statKey(GoldStat(f))]
	opened := len(h.attn.open)
	if w := h.WalkGuide(f); len(w.Steps) == 0 {
		t.Fatalf("the walk pressed nothing (%+v): the test needs a goal behind a click", w)
	}
	if after := h.Attention(); !reflect.DeepEqual(before, after) || len(h.attn.open) != opened {
		t.Fatalf("the guide walk's pretend clicks stayed in the attention log: %d boxes, %d frames, %d events before; %d, %d, %d after", len(before.Boxes), len(before.Frames), len(before.Events), len(after.Boxes), len(after.Frames), len(after.Events))
	}
	if got := h.top.stats[statKey(GoldStat(f))]; got != top {
		t.Fatalf("the guide walk's pretend clicks moved the top bar's counter: %+v, was %+v", got, top)
	}
}
