package app

import (
	"sort"

	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const roundEvery = 8.0

type watchRounds struct {
	at  int
	due float64
}

func BellHuts(w *view.WatchView) []string {
	if w == nil {
		return nil
	}
	bells := map[string]bool{}
	for _, c := range w.Crews {
		if c.Liveness != nil {
			bells[c.Crew] = true
		}
	}
	out := []string{}
	for _, h := range w.Huts {
		if bells[h.Crew] {
			out = append(out, h.Pod)
		}
	}
	sort.Strings(out)
	return out
}

func (r *watchRounds) next(huts []string) (string, string, bool) {
	if len(huts) < 2 {
		return "", "", false
	}
	from := huts[r.at%len(huts)]
	r.at = (r.at + 1) % len(huts)
	return from, huts[r.at], true
}

func (s *Session) walkRounds() {
	if s.Clocks.World < s.rounds.due {
		return
	}
	s.rounds.due = s.Clocks.World + roundEvery
	from, to, ok := s.rounds.next(BellHuts(s.State.Watch))
	if !ok {
		return
	}
	a, okA := s.Renderer.Home(from)
	b, okB := s.Renderer.Home(to)
	if okA && okB {
		s.Town.send(world.Errand{Kind: world.ErrandSchedule, Role: "reeve", From: a, To: b}, s.Clocks.World)
	}
}
