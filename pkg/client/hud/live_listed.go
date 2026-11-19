package hud

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

type ListedRead struct {
	List  string
	Every int
	Cap   int
}

const liveListedID = "info:live:listed"

func init() {
	liveButtons[liveListedID] = true
}

func (c *LiveCard) listedLine() (liveLine, bool) {
	if len(c.Listed) == 0 {
		return liveLine{}, false
	}
	lo, hi := c.Listed[0].Every, c.Listed[0].Every
	var story, names, forbidStory, forbidNames, capStory, capNames []string
	capEvery := map[int]bool{}
	for _, l := range c.Listed {
		lo, hi = min(lo, l.Every), max(hi, l.Every)
		s := lexicon.StoryList(l.List)
		story, names = append(story, s), append(names, l.List)
		if l.Cap > 0 {
			capEvery[l.Every] = true
			continue
		}
		forbidStory, forbidNames = append(forbidStory, s), append(forbidNames, l.List)
	}
	for _, l := range c.Listed {
		switch {
		case l.Cap == 0:
		case len(capEvery) == 1:
			capStory, capNames = append(capStory, lexicon.StoryList(l.List)), append(capNames, l.List)
		default:
			capStory = append(capStory, fmt.Sprintf("%s every %d s", lexicon.StoryList(l.List), l.Every))
			capNames = append(capNames, fmt.Sprintf("%s every %d s", l.List, l.Every))
		}
	}
	if len(capEvery) == 1 {
		for every := range capEvery {
			capStory = []string{strings.Join(capStory, ", ") + fmt.Sprintf(" every %d s", every)}
			capNames = []string{strings.Join(capNames, ", ") + fmt.Sprintf(" every %d s", every)}
		}
	}
	every := fmt.Sprintf("%d S", lo)
	if hi != lo {
		every = fmt.Sprintf("%d-%d S", lo, hi)
	}
	var tellStory, tellNames []string
	if len(forbidStory) > 0 {
		tellStory = append(tellStory, fmt.Sprintf("%s may read the %s but not watch them, so the realm reads them again every %d s", c.who(), strings.Join(forbidStory, ", "), lo))
		tellNames = append(tellNames, fmt.Sprintf("%s may list %s but not watch them, so they are listed again every %d s", c.who(), strings.Join(forbidNames, ", "), lo))
	}
	if len(capStory) > 0 {
		limit := c.Listed[len(c.Listed)-1].Cap
		for _, l := range c.Listed {
			limit = max(limit, l.Cap)
		}
		tellStory = append(tellStory, fmt.Sprintf("The realm watches at most %d foreign kinds at once, so it reads the %s", limit, strings.Join(capStory, ", ")))
		tellNames = append(tellNames, fmt.Sprintf("At most %d custom resource kinds are watched at once, so these are listed: %s", limit, strings.Join(capNames, ", ")))
	}
	tip := ui.Bridge(strings.Join(tellStory, ". ")+". They stay fresh while each reading succeeds.",
		strings.Join(tellNames, ". ")+". Each list asks for resourceVersionMatch=NotOlderThan the last one, so the watch cache answers it; they turn stale only when the lists keep failing, by the same rule as a broken watch.")
	text := ui.Bridge("READ EVERY "+every+": "+strings.Join(story, ", "), "LISTED EVERY "+every+": "+strings.Join(names, ", "))
	return liveLine{text: liveFont.Clip(text, liveCardChars), literal: true, clr: ui.Palette.TextDim, id: liveListedID, tip: tip}, true
}
