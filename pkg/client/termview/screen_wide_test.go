package termview

import (
	"strings"
	"testing"
)

func TestScreenWideSequencesNeverHoldTheDrawForLong(t *testing.T) {
	cols, rows := 154, 27
	for _, seq := range []string{"\x1bc", "\x1b[2J", "\x1b[S", "\x1b[J", "\x1b[K", "\x1b[L", "\x1b[M", "\x1b[T", "\x1b[@", "\x1b[P", "\x1b[X", "\x1b[?1049h\x1b[?1049l", "\n", "\x1bD", "\x1bM"} {
		w := through([]string{strings.Repeat(seq, 8192)}, cols, rows)
		checkPieces(t, "a flood of "+seq, w)
		units := 0
		for _, p := range w.pieces {
			units += p.units
		}
		if per := units / 8192; per < cols && seq != "\x1b[X" {
			t.Errorf("each %q was charged %d units though it touches at least a whole line of %d cells", seq, per, cols)
		}
	}
}
