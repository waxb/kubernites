package termview

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

type dispatch struct {
	kind string
	cmd  int
	vals []int
	more []bool
	data string
}

func (d dispatch) String() string {
	return fmt.Sprintf("%s %q %v %v %q", d.kind, ansi.Cmd(d.cmd).Final(), d.vals, d.more, d.data)
}

func recorder(cols, rows int, clamped bool, seen *[]dispatch) *ansi.Parser {
	p := ansi.NewParser()
	p.SetParamsSize(parser.MaxParamsSize)
	p.SetDataSize(maxString)
	c := &filter{cols: cols, rows: rows}
	params := func(cmd ansi.Cmd, ps ansi.Params) ([]int, []bool) {
		vals := paramValues(ps)
		if clamped {
			vals, _ = c.clamp(cmd, ps)
		}
		more := make([]bool, len(ps))
		for i, q := range ps {
			more[i] = q.HasMore() && i < len(ps)-1
		}
		return vals, more
	}
	p.SetHandler(ansi.Handler{
		Execute: func(b byte) {
			if b != 0 && b != ansi.ESC {
				*seen = append(*seen, dispatch{kind: "exec", cmd: int(b)})
			}
		},
		HandleCsi: func(cmd ansi.Cmd, ps ansi.Params) {
			if clamped && (cmd.Final() < 0x40 || cmd.Final() > 0x7e) {
				return
			}
			vals, more := params(cmd, ps)
			*seen = append(*seen, dispatch{kind: "csi", cmd: int(cmd), vals: vals, more: more})
		},
		HandleEsc: func(cmd ansi.Cmd) {
			if clamped && (cmd.Final() < 0x30 || cmd.Final() > 0x7e || cmd.Intermediate() == 0 && strings.IndexByte("[]PX^_", cmd.Final()) >= 0) {
				return
			}
			*seen = append(*seen, dispatch{kind: "esc", cmd: int(cmd)})
		},
		HandleDcs: func(cmd ansi.Cmd, ps ansi.Params, data []byte) {
			if clamped {
				return
			}
			vals := paramValues(ps)
			for i := range vals {
				vals[i] = min(vals[i], maxNumber)
			}
			*seen = append(*seen, dispatch{kind: "dcs", cmd: int(cmd), vals: vals, data: string(data)})
		},
		HandleOsc: func(cmd int, data []byte) {
			if cmd == oscCommand(data) && (!clamped || actedOSC[cmd]) {
				*seen = append(*seen, dispatch{kind: "osc", cmd: cmd, data: string(data)})
			}
		},
		HandleApc: func(data []byte) {
			if !clamped {
				*seen = append(*seen, dispatch{kind: "apc", data: string(data)})
			}
		},
		HandlePm: func(data []byte) {
			if !clamped {
				*seen = append(*seen, dispatch{kind: "pm", data: string(data)})
			}
		},
		HandleSos: func(data []byte) {
			if !clamped {
				*seen = append(*seen, dispatch{kind: "sos", data: string(data)})
			}
		},
	})
	return p
}

var hostileStreams = []string{
	"x\x1b\x1b[2147483646b", "\x1b\x1b[1;11r\n", "\x1b\x1b\x1b[300;300H", "\x9b2147483646b", "x\x9b300b",
	"\x1b]0;title\x1b\x1b[1;300r", "\x1b]0;title\x07\x1b[300B", "\x1bPq#0;2;0;0;0\x1b\x1b[999S", "\x1b_apc\x1b\x1b[999L",
	"\x1b^pm\x1b[999M", "\x1bXsos\x1b[999T", "\x1b[1;2\x18\x1b[999b", "\x1b[1;2\x1a\x1b[999J", "\x90q\x9c\x1b[300d",
	"\x9d0;t\x9c\x9b300G", "\x98s\x9c\x1b[300`", "\x9e\x9c\x9f\x9c\x1b[300e", "\x1b[?1049h\x1b[?47h\x1b[?1047l", "\x1bc\x1bD\x1bM\x1bE",
	"\x1b[" + strings.Repeat("1;", 40) + "b", "\x1b[" + strings.Repeat("9", 40) + "b", "\x1b[1:2:3;4m", "\x1b[38:2::1:2:3m",
	"\x1b[3;300r\x1b[?69h\x1b[1;300s", "\xe0\x1b[300b", "\xc3\x1b[999S", "\x1b[>c\x1b[6n\x1b[?6n\x1b[2$p", "a\u0301\u0302b",
	"\x1bP1$qm\x1b\\", "\x1b P\x1b(0\x1b)B", "\x1b[ 2q\x1b[!p", "\x1b[999@\x1b[999P\x1b[999X\x1b[999I\x1b[999Z", "\x1b[3g\x1b[?5W",
}

func randomStream(r *rand.Rand) string {
	pieces := []string{"\x1b", "[", "]", "P", "X", "^", "_", "\\", "\x07", "\x18", "\x1a", "\x9b", "\x90", "\x9d", "\x9c", "\x98", "\x9e", "\x9f",
		";", ":", "?", ">", "<", "=", " ", "$", "!", "1", "9", "300", "2147483646", "99999999999999999999", "b", "r", "s", "H", "S", "J", "K", "L", "m", "h", "l",
		"x", "\u0301", "\u200d", "\U0001F468", "\u0915\u094d", "\xe0", "\xc3", "\xf0\x9f", "\n", "\r", "\t", "\x7f", "\x00", "\x84", "\x85", "\x8d"}
	var b strings.Builder
	for range r.IntN(60) + 1 {
		b.WriteString(pieces[r.IntN(len(pieces))])
	}
	return b.String()
}

func checkOneParse(t *testing.T, input string, cols, rows int) {
	t.Helper()
	var want, got []dispatch
	recorder(cols, rows, true, &want).Parse([]byte(input))
	vt := recorder(cols, rows, false, &got)
	f := newFilter()
	f.feed([]byte(input), cols, rows, func(piece []byte, cost int) {
		vt.Parse(piece)
		if vt.State() != parser.GroundState {
			t.Fatalf("%q at %dx%d: a piece left the emulator's parser in %s: %q", input, cols, rows, vt.StateName(), piece)
		}
		if cost > pieceBudget && strings.Count(string(piece), "\x1b")+strings.Count(string(piece), "\n") > 1 {
			t.Errorf("%q: a piece charged %d units, past the budget of %d, holds more than one action: %q", input, cost, pieceBudget, piece)
		}
	})
	for _, d := range got {
		if d.kind != "csi" {
			continue
		}
		clamped, _ := (&filter{cols: cols, rows: rows}).clamp(ansi.Cmd(d.cmd), toParams(d.vals, d.more))
		for i := range d.vals {
			if d.vals[i] != clamped[i] {
				t.Errorf("%q at %dx%d: the emulator dispatched %v past its clamp %v", input, cols, rows, d, clamped)
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("%q at %dx%d: the emulator parsed\n%v\nthe filter parsed\n%v", input, cols, rows, got, want)
	}
	for i := range got {
		if got[i].String() != want[i].String() {
			t.Fatalf("%q at %dx%d: the emulator parsed %v where the filter parsed %v", input, cols, rows, got[i], want[i])
		}
	}
}

func toParams(vals []int, more []bool) ansi.Params {
	out := make(ansi.Params, len(vals))
	for i, v := range vals {
		if v < 0 {
			v = parser.MissingParam
		}
		out[i] = ansi.Param(ansi.Parameter(v, more[i]))
	}
	return out
}

func TestTheEmulatorParsesEveryByteStreamExactlyAsTheFilterDid(t *testing.T) {
	for _, size := range [][2]int{{154, 27}, {20, 5}, {5, 20}} {
		for _, s := range hostileStreams {
			checkOneParse(t, s, size[0], size[1])
			for cut := 1; cut < len(s); cut++ {
				var want, got []dispatch
				recorder(size[0], size[1], true, &want).Parse([]byte(s))
				vt := recorder(size[0], size[1], false, &got)
				f := newFilter()
				for _, part := range []string{s[:cut], s[cut:]} {
					f.feed([]byte(part), size[0], size[1], func(piece []byte, _ int) {
						vt.Parse(piece)
						if vt.State() != parser.GroundState {
							t.Fatalf("%q cut at %d: the emulator's parser was left in %s", s, cut, vt.StateName())
						}
					})
				}
				if fmt.Sprint(got) != fmt.Sprint(want) {
					t.Fatalf("%q cut at %d: the emulator parsed %v, the filter %v", s, cut, got, want)
				}
			}
		}
	}
	r := rand.New(rand.NewPCG(1, 2))
	for range 20000 {
		checkOneParse(t, randomStream(r), 154, 27)
	}
}

func FuzzNoSequenceTheEmulatorDispatchesPassesItsClamp(f *testing.F) {
	for _, s := range hostileStreams {
		f.Add([]byte(s), uint8(154), uint8(27))
	}
	f.Fuzz(func(t *testing.T, input []byte, cols, rows uint8) {
		checkOneParse(t, string(input), max(int(cols), 1), max(int(rows), 1))
	})
}
