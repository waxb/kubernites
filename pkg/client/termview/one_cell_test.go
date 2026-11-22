package termview

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestOneCellNeverHoldsAnUnboundedCluster(t *testing.T) {
	for _, c := range []struct {
		name, chain string
	}{
		{"emoji joined by ZWJ", strings.Repeat("\U0001F468‍", 1000)},
		{"prepend letters", strings.Repeat("ൎ", 2000)},
		{"indic conjunct", strings.Repeat("क्", 1000)},
	} {
		cols, rows := 154, 27
		v := New(cols, rows, func([]byte) {})
		for y := 1; y <= rows; y++ {
			for x := 1; x <= cols; x += 2 {
				v.Write([]byte(fmt.Sprintf("\x1b[%d;%dH", y, x) + c.chain))
			}
		}
		most, mostRunes, runes := 0, 0, 0
		for y := range rows {
			for _, cell := range v.Row(y) {
				most = max(most, len(cell.Text))
				mostRunes = max(mostRunes, utf8.RuneCountInString(cell.Text))
				runes += utf8.RuneCountInString(cell.Text)
			}
		}
		v.Close()
		if most > 4*maxCluster || mostRunes > maxCluster || runes > cols*rows*maxCluster {
			t.Errorf("%s: one cell holds %d bytes and %d code points, and one frame of the Stone's terminal (%dx%d cells) holds %d code points", c.name, most, mostRunes, cols, rows, runes)
		}
	}
}
