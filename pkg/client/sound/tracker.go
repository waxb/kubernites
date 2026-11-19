package sound

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	tokenHold = "--"
	tokenOff  = ".."
	drumKinds = "KTSHJB"
)

type tokenKind uint8

const (
	tokHold tokenKind = iota
	tokOff
	tokNote
)

type token struct {
	kind  tokenKind
	value int
}

type noteEvent struct {
	step, length, note int
}

type hitEvent struct {
	step int
	kind byte
}

var semitones = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}

func parseNote(text string) (int, error) {
	if len(text) < 2 {
		return 0, fmt.Errorf("invalid note %q", text)
	}
	base, ok := semitones[text[0]]
	if !ok {
		return 0, fmt.Errorf("invalid note letter in %q", text)
	}
	rest := text[1:]
	switch rest[0] {
	case '#':
		base, rest = base+1, rest[1:]
	case 'b':
		base, rest = base-1, rest[1:]
	}
	octave, err := strconv.Atoi(rest)
	if err != nil || octave < 0 || octave > 9 {
		return 0, fmt.Errorf("invalid octave in %q", text)
	}
	return (octave+1)*12 + base, nil
}

func parseToken(field string, drum bool) (token, error) {
	switch field {
	case tokenHold:
		return token{kind: tokHold}, nil
	case tokenOff:
		return token{kind: tokOff}, nil
	}
	if drum {
		if len(field) != 1 || !strings.Contains(drumKinds, field) {
			return token{}, fmt.Errorf("invalid drum %q", field)
		}
		return token{kind: tokNote, value: int(field[0])}, nil
	}
	note, err := parseNote(field)
	if err != nil {
		return token{}, err
	}
	return token{kind: tokNote, value: note}, nil
}

func parseLine(line string, steps int, drum bool) ([]token, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil, fmt.Errorf("empty pattern")
	}
	if steps <= 0 || steps%len(fields) != 0 {
		return nil, fmt.Errorf("pattern of %d steps does not tile a %d step song", len(fields), steps)
	}
	tokens := make([]token, 0, len(fields))
	for _, f := range fields {
		t, err := parseToken(f, drum)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

func noteEvents(line string, steps, shift int) ([]noteEvent, error) {
	tokens, err := parseLine(line, steps, false)
	if err != nil {
		return nil, err
	}
	bounds := make([]int, 0, steps)
	for s := 0; s < steps; s++ {
		if tokens[s%len(tokens)].kind != tokHold {
			bounds = append(bounds, s)
		}
	}
	events := make([]noteEvent, 0, len(bounds))
	for i, s := range bounds {
		t := tokens[s%len(tokens)]
		if t.kind != tokNote {
			continue
		}
		next := bounds[0] + steps
		if i+1 < len(bounds) {
			next = bounds[i+1]
		}
		events = append(events, noteEvent{step: s, length: next - s, note: t.value + shift})
	}
	return events, nil
}

func hitEvents(line string, steps int) ([]hitEvent, error) {
	tokens, err := parseLine(line, steps, true)
	if err != nil {
		return nil, err
	}
	hits := make([]hitEvent, 0, steps)
	for s := 0; s < steps; s++ {
		if t := tokens[s%len(tokens)]; t.kind == tokNote {
			hits = append(hits, hitEvent{step: s, kind: byte(t.value)})
		}
	}
	return hits, nil
}

func seq(parts ...string) string {
	return strings.Join(parts, " ")
}

func hold(note string, steps int) string {
	if steps <= 1 {
		return note
	}
	return note + strings.Repeat(" "+tokenHold, steps-1)
}

func rest(steps int) string {
	return hold(tokenOff, steps)
}

func times(line string, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = line
	}
	return seq(parts...)
}

func arp4(a, b, c, d string) string {
	return seq(hold(a, 4), hold(b, 4), hold(c, 4), hold(d, 4))
}

func halves(a, b string) string {
	return seq(hold(a, 8), hold(b, 8))
}
