package hud

import "testing"

func TestTheNoRoomMarkNeverSitsOnTheWaitingHutsPiles(t *testing.T) {
	f := boardFrame()
	f.State.Crews[0].Fits = 2
	h, v := openBoard(t, f)
	h.Board.Drag, h.Board.Pick, h.Board.Top = DragKnob, 7, v.Top
	full, _ := h.BoardView(f)
	if full.Waiting() == 0 {
		t.Fatal("no hut waits, so this does not test the mark beside waiting huts")
	}
	mark, shown := h.noRoomRect(full)
	if !shown {
		return
	}
	rects, mode := full.plots()
	if mode != plotSprites {
		t.Fatalf("the yard draws in mode %v, want sprites with piles", mode)
	}
	for i, r := range rects[:min(len(rects), len(full.Slots))] {
		if full.Slots[i].Kind == SlotWaits && Overlaps(mark, pileArea(r)) {
			t.Errorf("the NO ROOM mark at %+v sits on waiting hut %d's piles at %+v", mark, i, pileArea(r))
		}
	}
}
