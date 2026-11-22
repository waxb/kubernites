package termview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestADoubledEscapeNeverCarriesAnUnclampedMargin(t *testing.T) {
	v := New(154, 10, func([]byte) {})
	defer v.Close()
	v.Write([]byte("\x1b\x1b[1;11r" + strings.Repeat("\n", 20)))
	if err := v.Broken(); err != nil {
		t.Fatalf("ESC ESC [1;11r reached the emulator unclamped: %v", err)
	}
	for _, d := range through([]string{"\x1b\x1b[1;11r"}, 154, 10).csi {
		if ansi.Cmd(d.cmd).Final() == 'r' && len(d.vals) > 1 && d.vals[1] > 10 {
			t.Fatalf("ESC ESC [1;11r reached the emulator with a bottom margin of %d on a screen of 10 rows", d.vals[1])
		}
	}
}

func TestADoubledEscapeNeverPinsTheView(t *testing.T) {
	cols, rows := 154, 27
	for _, parts := range splits("x\x1b\x1b[2147483646b") {
		w := through(parts, cols, rows)
		checkPieces(t, "x ESC ESC [2147483646b", w)
		if w.prints > 1+cols*rows {
			t.Fatalf("x ESC ESC [2147483646b split %q: the emulator prints %d times, past the %d cells of the screen", parts, w.prints, cols*rows)
		}
	}
	v := New(cols, rows, func([]byte) {})
	defer v.Close()
	v.Write([]byte("x\x1b\x1b[2147483646b"))
	v.Row(0)
	v.Key(Key{Rune: 'p', Ctrl: true})
}
