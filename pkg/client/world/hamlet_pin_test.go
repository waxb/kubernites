package world

import "testing"

func TestARePlannedRealmReservesOnlyItsNewPins(t *testing.T) {
	h := NewHamlet()
	h.Pin(map[string]Point{"a": {1, 1}, "b": {2, 2}})
	if !h.reserved(Point{1, 1}) {
		t.Fatal("a pinned spot is not reserved")
	}
	h.Pin(map[string]Point{"a": {5, 5}, "b": {2, 2}})
	if h.reserved(Point{1, 1}) || !h.reserved(Point{5, 5}) {
		t.Fatalf("a new plan of the same size keeps the old reservations: old %v, new %v", h.reserved(Point{1, 1}), h.reserved(Point{5, 5}))
	}
	pins := map[string]Point{"a": {7, 7}}
	h.Pin(pins)
	h.reserved(Point{7, 7})
	pins["a"] = Point{8, 8}
	h.Pin(pins)
	if h.reserved(Point{7, 7}) || !h.reserved(Point{8, 8}) {
		t.Fatalf("pins changed in place keep the old reservations: old %v, new %v", h.reserved(Point{7, 7}), h.reserved(Point{8, 8}))
	}
	h.Pin(nil)
	if h.reserved(Point{8, 8}) {
		t.Fatal("a realm without pins still reserves a spot")
	}
}
