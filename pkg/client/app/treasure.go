package app

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const chestOpenSeconds = 4.0

type treasureChests struct {
	primed  bool
	waiting []string
	opened  map[string]float64
}

func TreasureBanner(t view.TreasureView) (string, string) {
	title := ui.Bridge("NEW CARGO: "+strings.ToUpper(t.Boxes), "NEW TRAFFIC: "+strings.ToUpper(ui.Name(t.Boxes)))
	cause := ui.Bridge(fmt.Sprintf("%s wait at sea until their depot stands: raise the %s and open it, and the first ship's treasure is theirs", t.Boxes, t.Crew),
		fmt.Sprintf("%s wait: no Service and no Deployment serve them yet; a one-time bonus waits for the first %s Pod", ui.Name(t.Boxes), ui.Name(t.Crew)))
	return title, cause
}

func kindsOf(list []view.TreasureView) []string {
	out := make([]string, 0, len(list))
	for _, t := range list {
		out = append(out, t.Kind)
	}
	return out
}

func (c treasureChests) prime(list []view.TreasureView) treasureChests {
	return treasureChests{primed: true, waiting: kindsOf(list), opened: c.opened}
}

func (c treasureChests) step(list []view.TreasureView, now float64) (treasureChests, []view.TreasureView, []string) {
	if !c.primed {
		c = c.prime(nil)
	}
	next := treasureChests{primed: true, waiting: kindsOf(list), opened: map[string]float64{}}
	for k, at := range c.opened {
		if now-at < chestOpenSeconds {
			next.opened[k] = at
		}
	}
	fresh, opened := []view.TreasureView{}, []string{}
	for _, t := range list {
		if !slices.Contains(c.waiting, t.Kind) {
			fresh = append(fresh, t)
		}
	}
	for _, k := range c.waiting {
		if !slices.Contains(next.waiting, k) {
			next.opened[k] = now
			opened = append(opened, k)
		}
	}
	return next, fresh, opened
}

func (c treasureChests) marks(now float64) []world.ChestMark {
	out := []world.ChestMark{}
	for _, k := range c.waiting {
		out = append(out, world.ChestMark{Kind: k})
	}
	open := make([]string, 0, len(c.opened))
	for k := range c.opened {
		open = append(open, k)
	}
	sort.Strings(open)
	for _, k := range open {
		if age := now - c.opened[k]; age < chestOpenSeconds {
			out = append(out, world.ChestMark{Kind: k, Open: true, Age: age})
		}
	}
	return out
}

func (s *Session) treasureNotices() []sound.SFX {
	if !s.chests.primed {
		s.chests = s.chests.prime(s.State.Treasure)
		return nil
	}
	next, fresh, opened := s.chests.step(s.State.Treasure, s.Clocks.World)
	s.chests = next
	var out []sound.SFX
	for _, t := range fresh {
		title, cause := TreasureBanner(t)
		kind := t.Kind
		waits := func() bool { return slices.Contains(kindsOf(s.State.Treasure), kind) }
		s.Hud.BannerWhile(title, cause, waits)
		s.resultWhile(title, cause, waits)
		out = append(out, sound.SFXHorn)
	}
	if len(opened) > 0 {
		out = append(out, sound.SFXCoin)
	}
	return out
}
