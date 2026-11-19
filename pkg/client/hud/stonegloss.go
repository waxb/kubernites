package hud

import (
	"regexp"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

type glossEntry struct {
	word *regexp.Regexp
	key  string
	term string
}

var stoneGlossary = []glossEntry{
	{regexp.MustCompile(`\bBurning\b`), "burning", "Burning"},
	{regexp.MustCompile(`\bKindling\b`), "kindling", "Kindling"},
	{regexp.MustCompile(`\bUnhoused\b`), "unhoused", "Unhoused"},
	{regexp.MustCompile(`\bGuttering\b`), "guttering", "Guttering"},
	{regexp.MustCompile(`\bBanished\b`), "banished", "Banished"},
	{regexp.MustCompile(`\bexposed\b`), "exposed", "exposed"},
	{regexp.MustCompile(`\bReeve\b`), "reeve", "Reeve"},
	{regexp.MustCompile(`\bWards?\b`), "ward", "Ward"},
	{regexp.MustCompile(`\b(?:badges? it looks for|looksFor)\b`), "rally", "badge it looks for"},
	{regexp.MustCompile(`(?i)\bseason(?:ed|ing)\b`), "seasoned", "seasoned"},
}

func (h *HUD) glossStone(lines []string) {
	if !ui.StoryNaming() {
		return
	}
	text := strings.Join(lines, "\n")
	for _, g := range stoneGlossary {
		if h.Stone.glossed[g.key] || !g.word.MatchString(text) {
			continue
		}
		if h.Stone.glossed == nil {
			h.Stone.glossed = map[string]bool{}
		}
		h.Stone.glossed[g.key] = true
		if word, ok := lexicon.PlainOf(g.term); ok {
			h.Stone.Append(LineHint, g.term+" means "+word.Means+".")
		}
	}
}
