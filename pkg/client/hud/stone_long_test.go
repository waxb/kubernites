package hud

import "testing"

func TestTheStoneTakesALongPatchWhole(t *testing.T) {
	s := NewStone()
	for i := 0; i < 400; i++ {
		s.Insert('a')
	}
	if len(s.Input) != 400 {
		t.Fatalf("the Stone kept %d of 400 typed characters", len(s.Input))
	}
}
