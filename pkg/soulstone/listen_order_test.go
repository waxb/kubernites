package soulstone

import (
	"strings"
	"testing"
)

func TestAServerListensBeforeItServes(t *testing.T) {
	for _, p := range catalog {
		listen := -1
		for i, line := range p.StartLines {
			if strings.HasPrefix(line, "listening for ") {
				listen = i
			}
		}
		if listen < 0 {
			continue
		}
		for i, line := range p.StartLines[:listen] {
			if strings.Contains(line, "first") || strings.HasPrefix(line, "shouldering") {
				t.Errorf("%s logs %q (line %d) before %q: a server serves only once it listens", p.Image, line, i, p.StartLines[listen])
			}
		}
	}
	for _, p := range catalog {
		if !p.Cursed {
			continue
		}
		if len(p.StartLines) > 0 && strings.HasPrefix(p.StartLines[0], "shouldering") {
			t.Errorf("the crash-looping %s logs %q before it reads its config and panics", p.Image, p.StartLines[0])
		}
	}
}
