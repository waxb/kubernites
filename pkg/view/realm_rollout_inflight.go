package view

import (
	"slices"
)

type InFlightRoll struct {
	Crew        string        `json:"-"`
	Count       int           `json:"-"`
	Surge       int           `json:"-"`
	Unavailable int           `json:"-"`
	Placed      int           `json:"-"`
	Asks        ResourcePair  `json:"-"`
	Old         []StandingHut `json:"-"`
}

type StandingHut struct {
	Keep int          `json:"-"`
	Asks ResourcePair `json:"-"`
}

func (f InFlightRoll) Waits(keeps []KeepRoom) int {
	old, placed := f.Old, f.Placed
	for {
		for len(old) > max(0, f.Count-f.Unavailable-placed) {
			if k := old[0].Keep; k >= 0 {
				keeps[k].Free = AddPair(keeps[k].Free, old[0].Asks)
			}
			old = old[1:]
		}
		if placed >= f.Count {
			return 0
		}
		i, ok := Roomiest(keeps, f.Asks)
		if !ok {
			return max(0, min(f.Count, f.Count+f.Surge-len(old))-placed)
		}
		keeps[i].Free = SubPair(keeps[i].Free, f.Asks)
		placed++
	}
}

func (b RoomBook) PlacedBeside(rolls []InFlightRoll) []KeepRoom {
	keeps := slices.Clone(b.Keeps)
	for _, o := range b.Owed {
		if slices.ContainsFunc(rolls, func(f InFlightRoll) bool { return f.Crew == o.Crew }) {
			continue
		}
		for range o.Count {
			if i, ok := Roomiest(keeps, o.Asks); ok {
				keeps[i].Free = SubPair(keeps[i].Free, o.Asks)
			}
		}
	}
	return keeps
}
