package termview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

type piece struct {
	units   int
	actions int
}

type work struct {
	csi    []dispatch
	prints int
	pieces []piece
}

func through(parts []string, cols, rows int) work {
	var w work
	actions := 0
	p := ansi.NewParser()
	p.SetParamsSize(parser.MaxParamsSize)
	p.SetDataSize(maxString)
	p.SetHandler(ansi.Handler{
		Print:   func(rune) { w.prints++; actions++ },
		Execute: func(byte) { actions++ },
		HandleCsi: func(cmd ansi.Cmd, ps ansi.Params) {
			actions++
			vals := paramValues(ps)
			w.csi = append(w.csi, dispatch{kind: "csi", cmd: int(cmd), vals: vals})
			if cmd.Final() == 'b' && cmd.Prefix() == 0 && cmd.Intermediate() == 0 {
				n, _, _ := ps.Param(0, 1)
				w.prints += max(n, 1)
			}
		},
		HandleEsc: func(ansi.Cmd) { actions++ },
		HandleDcs: func(ansi.Cmd, ansi.Params, []byte) { actions++ },
		HandleOsc: func(int, []byte) { actions++ },
		HandleApc: func([]byte) { actions++ },
		HandlePm:  func([]byte) { actions++ },
		HandleSos: func([]byte) { actions++ },
	})
	f := newFilter()
	for _, part := range parts {
		f.feed([]byte(part), cols, rows, func(b []byte, units int) {
			actions = 0
			p.Parse(b)
			w.pieces = append(w.pieces, piece{units: units, actions: actions})
		})
	}
	return w
}

func splits(s string) [][]string {
	out := [][]string{{s}}
	for cut := 1; cut < len(s); cut++ {
		out = append(out, []string{s[:cut], s[cut:]})
	}
	return out
}

func introduced(seq string) []string {
	return []string{
		"\x1b[" + seq, "\x1b\x1b[" + seq, "\x9b" + seq, "\x1b\x1b\x9b" + seq,
		"\x1b]0;title\x1b\\\x1b[" + seq, "\x1b]0;title\x07\x1b[" + seq, "\x1b]0;title\x1b[" + seq, "\x1b]0;title\x9c\x9b" + seq,
		"\x1bPq#0;2;0;0;0\x1b[" + seq, "\x1bP1$qm\x1b\\\x9b" + seq, "\x1b_apc\x1b[" + seq, "\x1b^pm\x1b[" + seq, "\x1bXsos\x1b[" + seq,
		"\x90q\x9c\x1b[" + seq, "\x9d0;t\x07\x9b" + seq, "\x98s\x9c\x9b" + seq, "\x9ep\x9c\x9b" + seq, "\x9fa\x9c\x9b" + seq,
		"\x1b[1;2\x18\x1b[" + seq, "\x1b[1;2\x1a\x9b" + seq, "\x1b]0;t\x18\x1b[" + seq, "\x1bPq\x1a\x1b[" + seq,
		"\x1b[" + seq[:len(seq)/2] + "\n" + seq[len(seq)/2:], "\x1b[" + seq[:len(seq)/2] + "\x84" + seq[len(seq)/2:],
	}
}

func checkPieces(t *testing.T, what string, w work) {
	t.Helper()
	for _, p := range w.pieces {
		if p.units > pieceBudget && p.actions > 1 {
			t.Errorf("%s: one write to the emulator was charged %d units, past the budget of %d, and held %d actions", what, p.units, pieceBudget, p.actions)
		}
	}
}

func TestEveryRepeatAndScreenWideActionIsChargedTheCellsItTouches(t *testing.T) {
	cols, rows := 154, 27
	screen := cols * rows
	for _, c := range []struct {
		seq   string
		units int
	}{
		{"\x1bc", screen}, {"\x1b[J", screen}, {"\x1b[0J", screen}, {"\x1b[1J", screen}, {"\x1b[2J", screen}, {"\x1b[3J", screen},
		{"\x1b[K", cols}, {"\x1b[0K", cols}, {"\x1b[1K", cols}, {"\x1b[2K", cols},
		{"\x1b[?1049h", screen}, {"\x1b[?1049l", screen}, {"\x1b[?47h", screen}, {"\x1b[?1047l", screen},
		{"\x1b[S", screen}, {"\x1b[9S", screen}, {"\x1b[T", screen}, {"\x1b[9T", screen}, {"\x1b[L", screen}, {"\x1b[9L", screen},
		{"\x1b[M", screen}, {"\x1b[9M", screen}, {"\x1b[@", cols}, {"\x1b[9@", cols}, {"\x1b[P", cols}, {"\x1b[9P", cols},
		{"\x1b[X", 1}, {"\x1b[99X", 99}, {"\x1b[999X", cols}, {"\x1bD", screen}, {"\x1bM", screen}, {"\n", screen}, {"\x85", screen},
		{"x\x1b[999b", 1 + rows + 999*(1+rows)},
	} {
		w := through([]string{c.seq}, cols, rows)
		total := 0
		for _, p := range w.pieces {
			total += p.units
		}
		if total < c.units {
			t.Errorf("%q touches %d cells and was charged %d units", c.seq, c.units, total)
		}
		flood := through([]string{strings.Repeat(c.seq, 4096)}, cols, rows)
		checkPieces(t, "a flood of "+c.seq, flood)
	}
}

func TestOneRepeatSequenceNeverPinsTheView(t *testing.T) {
	for _, size := range [][2]int{{154, 27}, {20, 5}} {
		cols, rows := size[0], size[1]
		for _, count := range []string{"2147483646", "2000000", "99999999999999999999", ""} {
			for _, form := range introduced(count + "b") {
				for _, parts := range splits("x" + form) {
					w := through(parts, cols, rows)
					checkPieces(t, form, w)
					for _, d := range w.csi {
						if ansi.Cmd(d.cmd).Final() == 'b' && len(d.vals) > 0 && d.vals[0] > cols*rows {
							t.Fatalf("%q split %q at %dx%d: the emulator repeats %d times, past the %d cells of the screen", form, parts, cols, rows, d.vals[0], cols*rows)
						}
					}
					if w.prints > 1+cols*rows {
						t.Fatalf("%q split %q: the emulator prints %d times for one character and one repeat", form, parts, w.prints)
					}
				}
			}
		}
	}
	v := New(154, 27, func([]byte) {})
	defer v.Close()
	v.Write([]byte("x\x1b[2147483646b"))
	if got := strings.Count(strings.Join(v.Text(), ""), "x"); got < 154*26 {
		t.Fatalf("x and REP 2147483646 left %d x on a screen of %d cells", got, 154*27)
	}
}
