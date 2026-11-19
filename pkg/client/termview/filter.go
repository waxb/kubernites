package termview

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	ground = iota
	escape
	control
)

const (
	pieceBudget = 2048
	maxSequence = 128
)

type filter struct {
	state int
	need  int
	r     []byte
	marks int
	seq   []byte
	piece []byte
	cost  int
}

func (f *filter) feed(p []byte, cols, rows int, out func([]byte)) {
	flush := func() {
		if len(f.piece) > 0 {
			out(f.piece)
			f.piece, f.cost = nil, 0
		}
	}
	emit := func(b []byte, cost int) {
		if f.cost > 0 && f.cost+cost > pieceBudget {
			flush()
		}
		f.piece = append(f.piece, b...)
		f.cost += cost
		if f.cost >= pieceBudget {
			flush()
		}
	}
	for _, b := range p {
		switch f.state {
		case ground:
			f.ground(b, emit)
		case escape:
			if b == '[' {
				f.state, f.seq = control, append(f.seq, b)
				continue
			}
			f.state = ground
			emit(append(f.seq, b), 1)
		case control:
			switch {
			case b >= 0x40 && b <= 0x7e:
				f.state = ground
				seq, cost := clamp(append(f.seq, b), cols, rows)
				emit(seq, cost)
			case b >= 0x20 && b <= 0x3f:
				if len(f.seq) < maxSequence {
					f.seq = append(f.seq, b)
				}
			case b == 0x1b:
				f.seq = append(f.seq[:0], b)
				f.state = escape
			case b == 0x18 || b == 0x1a:
				f.state = ground
				emit([]byte{b}, 1)
			case b < 0x20:
				emit([]byte{b}, 1)
			case b == 0x7f:
			default:
				f.state = ground
				f.ground(b, emit)
			}
		}
	}
	flush()
}

func (f *filter) ground(b byte, emit func([]byte, int)) {
	if f.need > 0 {
		if b&0xc0 == 0x80 {
			f.need--
			f.r = append(f.r, b)
			if f.need == 0 {
				f.char(emit)
			}
			return
		}
		f.need = 0
		emit(f.r, len(f.r))
		f.r = f.r[:0]
	}
	switch {
	case b == 0x1b:
		f.marks = 0
		f.state, f.seq = escape, append(f.seq[:0], b)
	case b == 0x9b:
		f.marks = 0
		f.state, f.seq = control, append(f.seq[:0], 0x1b, '[')
	case b >= 0xc2 && b <= 0xdf:
		f.need, f.r = 1, append(f.r[:0], b)
	case b >= 0xe0 && b <= 0xef:
		f.need, f.r = 2, append(f.r[:0], b)
	case b >= 0xf0 && b <= 0xf4:
		f.need, f.r = 3, append(f.r[:0], b)
	default:
		f.marks = 0
		emit([]byte{b}, 1)
	}
}

func (f *filter) char(emit func([]byte, int)) {
	r, _ := utf8.DecodeRune(f.r)
	if extending(r) {
		f.marks++
	} else {
		f.marks = 0
	}
	if f.marks <= maxMarks {
		emit(f.r, 1)
	}
	f.r = f.r[:0]
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

func clamp(seq []byte, cols, rows int) ([]byte, int) {
	body := string(seq[2:])
	final := body[len(body)-1]
	params := body[:len(body)-1]
	if params != "" && strings.ContainsRune("<=>?", rune(params[0])) || strings.IndexFunc(params, func(r rune) bool { return r >= 0x20 && r <= 0x2f }) >= 0 {
		return capDigits(seq), 1
	}
	limit, per := 0, 0
	switch final {
	case 'b':
		limit, per = cols*rows, 1
	case 'I', 'Z', 'X', '@', 'P':
		limit, per = cols, 1
	case 'L', 'M', 'S', 'T':
		limit, per = rows, cols
	case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'd', 'e', 'a', '`', 'H', 'f', 'r', 's':
		limit = max(cols, rows)
	default:
		return capDigits(seq), 1
	}
	limit = max(limit, 1)
	cost := 1
	var out strings.Builder
	out.WriteString("\x1b[")
	start := 0
	for i := 0; i <= len(params); i++ {
		if i < len(params) && params[i] != ';' && params[i] != ':' {
			continue
		}
		field := params[start:i]
		if n, ok := number(field); ok {
			n = min(n, limit)
			field = strconv.Itoa(n)
			cost = max(cost, n*per)
		}
		out.WriteString(field)
		if i < len(params) {
			out.WriteByte(params[i])
		}
		start = i + 1
	}
	out.WriteByte(final)
	return []byte(out.String()), cost
}

func number(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		if n < 1<<20 {
			n = n*10 + int(r-'0')
		}
	}
	return n, true
}

func capDigits(seq []byte) []byte {
	var out []byte
	run := 0
	for _, b := range seq {
		if b >= '0' && b <= '9' {
			run++
			if run > 5 {
				continue
			}
		} else {
			run = 0
		}
		out = append(out, b)
	}
	return out
}

const maxMarks = 32
