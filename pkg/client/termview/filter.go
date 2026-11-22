package termview

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

const (
	pieceBudget = 2048
	maxString   = 64 << 10
	maxCluster  = 32
	maxMarks    = 32
	maxNumber   = 99999
)

type filter struct {
	p       *ansi.Parser
	cols    int
	rows    int
	out     func([]byte, int)
	piece   []byte
	cost    int
	marks   int
	cluster int
}

func newFilter() *filter {
	f := &filter{p: ansi.NewParser()}
	f.p.SetParamsSize(parser.MaxParamsSize)
	f.p.SetDataSize(maxString)
	f.p.SetHandler(ansi.Handler{
		Print:     f.print,
		Execute:   f.execute,
		HandleCsi: f.csi,
		HandleEsc: f.esc,
		HandleDcs: func(ansi.Cmd, ansi.Params, []byte) { f.drop() },
		HandleOsc: f.osc,
		HandleApc: func([]byte) { f.drop() },
		HandlePm:  func([]byte) { f.drop() },
		HandleSos: func([]byte) { f.drop() },
	})
	return f
}

func (f *filter) feed(p []byte, cols, rows int, out func([]byte, int)) {
	f.cols, f.rows, f.out = max(cols, 1), max(rows, 1), out
	for _, b := range p {
		f.p.Advance(b)
	}
	f.flush()
	f.out = nil
}

func (f *filter) flush() {
	if len(f.piece) > 0 {
		f.out(f.piece, f.cost)
		f.piece, f.cost = nil, 0
	}
	f.cluster, f.marks = 0, 0
}

func (f *filter) emit(b []byte, cost int) {
	if f.cost > 0 && f.cost+cost > pieceBudget {
		marks, cluster := f.marks, f.cluster
		f.flush()
		f.marks, f.cluster = marks, cluster
	}
	f.piece = append(f.piece, b...)
	f.cost += cost
}

func (f *filter) screen() int {
	return f.cols * f.rows
}

func (f *filter) print(r rune) {
	if r < utf8.RuneSelf {
		f.marks, f.cluster = 0, 1
		f.emit([]byte{byte(r)}, 1+f.rows)
		return
	}
	if extending(r) {
		f.marks++
	} else {
		f.marks = 0
	}
	if f.marks > maxMarks {
		return
	}
	if f.cluster >= maxCluster {
		f.emit([]byte{0}, 1)
		f.cluster = 0
	}
	f.cluster++
	f.emit(utf8.AppendRune(nil, r), 1+f.rows)
}

func (f *filter) boundary() {
	f.marks, f.cluster = 0, 0
}

func (f *filter) execute(b byte) {
	f.boundary()
	switch b {
	case ansi.ESC:
		f.emit([]byte{0}, 1)
	case ansi.LF, ansi.VT, ansi.FF, ansi.IND, ansi.NEL, ansi.RI:
		f.emit([]byte{b}, f.screen())
	default:
		f.emit([]byte{b}, 1)
	}
}

func (f *filter) drop() {
	f.boundary()
	f.emit([]byte{0}, 1)
}

func (f *filter) esc(cmd ansi.Cmd) {
	if fin := cmd.Final(); fin < 0x30 || fin > 0x7e || cmd.Intermediate() == 0 && strings.IndexByte("[]PX^_", fin) >= 0 {
		f.drop()
		return
	}
	f.boundary()
	seq := []byte{ansi.ESC}
	if i := cmd.Intermediate(); i != 0 {
		seq = append(seq, i)
	}
	seq = append(seq, cmd.Final())
	cost := 1
	if cmd.Intermediate() == 0 {
		switch cmd.Final() {
		case 'c', 'D', 'M':
			cost = f.screen()
		}
	}
	f.emit(seq, cost)
}

var actedOSC = map[int]bool{10: true, 11: true, 12: true, 110: true, 111: true, 112: true}

func (f *filter) osc(cmd int, data []byte) {
	if cmd != oscCommand(data) || !actedOSC[cmd] {
		f.drop()
		return
	}
	f.str("\x1b]", nil, data)
}

func oscCommand(data []byte) int {
	cmd := parser.MissingCommand
	for _, d := range data {
		if d < '0' || d > '9' {
			break
		}
		if cmd == parser.MissingCommand {
			cmd = 0
		}
		cmd = cmd*10 + int(d-'0')
	}
	return cmd
}

func (f *filter) str(intro string, head, data []byte) {
	f.boundary()
	seq := append([]byte(intro), head...)
	seq = append(seq, data...)
	seq = append(seq, ansi.ST)
	f.emit(seq, 1+len(data))
}

func extending(r rune) bool {
	switch {
	case unicode.In(r, unicode.Mn, unicode.Me, unicode.Mc, unicode.Cf):
		return true
	case r >= 0x1f3fb && r <= 0x1f3ff, r >= 0x1100 && r <= 0x11ff, r >= 0xa960 && r <= 0xa97f, r >= 0xd7b0 && r <= 0xd7ff:
		return true
	}
	return false
}

func paramValues(params ansi.Params) []int {
	vals := make([]int, len(params))
	for i, p := range params {
		vals[i] = p.Param(-1)
		if vals[i] == -1 && int(p)&parser.ParamMask == parser.MissingParam && int(p) != parser.MissingParam && int(p) != parser.MissingParam|parser.HasMoreFlag {
			vals[i] = maxNumber
		}
	}
	return vals
}

func (f *filter) clamp(cmd ansi.Cmd, params ansi.Params) ([]int, int) {
	vals := paramValues(params)
	cols, rows := f.cols, f.rows
	limitAll := func(n int) {
		for i := range vals {
			vals[i] = min(vals[i], n)
		}
	}
	count := func() int {
		if len(vals) == 0 || vals[0] < 0 {
			return 1
		}
		return max(vals[0], 1)
	}
	cost := 1
	if cmd.Prefix() == 0 && cmd.Intermediate() == 0 {
		switch cmd.Final() {
		case 'b':
			limitAll(f.screen())
			cost = count() * (1 + rows)
		case '@', 'P', 'K', 'g':
			limitAll(cols)
			cost = cols
		case 'X', 'I', 'Z':
			limitAll(cols)
			cost = count()
		case 'L', 'M', 'S', 'T', 'J':
			limitAll(rows)
			cost = f.screen()
		case 'A', 'B', 'E', 'F', 'd', 'e', 'r':
			limitAll(rows)
		case 'C', 'D', 'G', 'a', '`', 's':
			limitAll(cols)
		case 'H', 'f':
			limitAll(cols)
			if len(vals) > 0 {
				vals[0] = min(vals[0], rows)
			}
		}
	}
	if cmd.Prefix() == '?' && cmd.Intermediate() == 0 && (cmd.Final() == 'h' || cmd.Final() == 'l') {
		for _, v := range vals {
			switch v {
			case 47, 1047, 1049:
				cost += f.screen()
			}
		}
	}
	if cmd.Prefix() == '?' && cmd.Intermediate() == 0 && cmd.Final() == 'W' {
		cost = cols
	}
	limitAll(maxNumber)
	return vals, cost
}

func (f *filter) csi(cmd ansi.Cmd, params ansi.Params) {
	if fin := cmd.Final(); fin < 0x40 || fin > 0x7e {
		f.drop()
		return
	}
	f.boundary()
	vals, cost := f.clamp(cmd, params)
	seq := []byte{ansi.ESC, '['}
	if p := cmd.Prefix(); p != 0 {
		seq = append(seq, p)
	}
	seq = appendParams(seq, params, vals)
	if i := cmd.Intermediate(); i != 0 {
		seq = append(seq, i)
	}
	seq = append(seq, cmd.Final())
	f.emit(seq, cost)
}

func appendParams(seq []byte, params ansi.Params, vals []int) []byte {
	if vals == nil {
		vals = paramValues(params)
		for i := range vals {
			vals[i] = min(vals[i], maxNumber)
		}
	}
	for i, p := range params {
		if v := vals[i]; v >= 0 {
			seq = strconv.AppendInt(seq, int64(v), 10)
		}
		if i < len(params)-1 {
			if p.HasMore() {
				seq = append(seq, ':')
			} else {
				seq = append(seq, ';')
			}
		}
	}
	if len(params) == parser.MaxParamsSize {
		seq = append(seq, ';')
	}
	return seq
}
