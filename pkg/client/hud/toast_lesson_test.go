package hud

import "testing"

func TestAToastNeverCoversALessonCard(t *testing.T) {
	f := fullFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	h.Lesson("BURNING", "Up and ready for [box] boxes.")
	if !h.LessonShown(f) {
		t.Fatal("the hint card is not shown")
	}
	for _, text := range []string{"Help: click anything to open its page in the Almanac. Esc leaves.", "Depot/parcels created", "A depot named \"parcels\" already stands."} {
		h.Toast(text, false)
	}
	card := h.LessonRect(f)
	for i, r := range h.ToastRects(f) {
		if Overlaps(r, card) {
			t.Errorf("toast %q at %+v covers the hint card %+v", h.ToastList[i].Text, r, card)
		}
	}
}
