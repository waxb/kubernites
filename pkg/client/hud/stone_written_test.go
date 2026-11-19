package hud

import "testing"

func TestTheStoneCountsEveryLineEvenPastItsCap(t *testing.T) {
	s := NewStone()
	start := s.Written
	for range maxStoneLines + 50 {
		s.Append(LineOutput, "line")
	}
	if len(s.Output) != maxStoneLines {
		t.Fatalf("the Stone keeps %d lines, want %d", len(s.Output), maxStoneLines)
	}
	if got := s.Written - start; got != maxStoneLines+50 {
		t.Fatalf("the Stone counted %d lines written, want %d", got, maxStoneLines+50)
	}
}
