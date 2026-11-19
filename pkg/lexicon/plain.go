package lexicon

import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

type Plain struct {
	Term  string
	Means string
	First string
	Quiet bool
	match *regexp.Regexp
}

var (
	plainLoaded  sync.Once
	plainEntries []Plain
)

func plainTable() []Plain {
	plainLoaded.Do(func() {
		lx, err := Load()
		if err != nil {
			return
		}
		for _, w := range lx.Plain {
			plainEntries = append(plainEntries, Plain{Term: w.Term, Means: w.Means, First: w.First, Quiet: w.Quiet, match: plainPattern(w.Term)})
		}
		sort.SliceStable(plainEntries, func(i, j int) bool { return len(plainEntries[i].Term) > len(plainEntries[j].Term) })
	})
	return plainEntries
}

func plainPattern(term string) *regexp.Regexp {
	stem, tail := term, `(?:'s|s|es)?`
	if strings.HasSuffix(term, "s") {
		stem, tail = term[:len(term)-1], `s?`
	}
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(stem) + tail + `\b`)
}

func PlainWords() []Plain {
	return append([]Plain(nil), plainTable()...)
}

func PlainOf(term string) (Plain, bool) {
	for _, w := range plainTable() {
		if w.Term == term {
			return w, true
		}
	}
	return Plain{}, false
}

func (p Plain) Line() string {
	return p.Term + ": " + p.Means
}

func (p Plain) Sentence() string {
	return strings.ToUpper(p.Means[:1]) + p.Means[1:] + "."
}

func PlainMatches(text string) []Plain {
	out := []Plain{}
	for _, w := range plainTable() {
		if w.Quiet || !w.match.MatchString(text) {
			continue
		}
		covered := false
		for _, kept := range out {
			if strings.Contains(kept.Term, w.Term) {
				covered = true
				break
			}
		}
		if !covered {
			out = append(out, w)
		}
	}
	return out
}
