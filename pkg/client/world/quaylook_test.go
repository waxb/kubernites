package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

type heldCohortSim struct {
	id    int
	at    float64
	ttl   float64
	count int
}

func (c heldCohortSim) view(now float64) view.CohortView {
	return view.CohortView{Count: float64(c.count), Age: now - c.at, Remaining: max(0, c.at+c.ttl-now), TTL: c.ttl}
}

type shownLook struct {
	variant, size, column, level int
}

type heldQuay struct {
	cohorts []heldCohortSim
	nextID  int
}

func (q *heldQuay) arrive(now float64, count int) {
	q.arriveFor(now, count, 30)
}

func (q *heldQuay) arriveFor(now float64, count int, ttl float64) {
	q.nextID++
	q.cohorts = append(q.cohorts, heldCohortSim{id: q.nextID, at: now, ttl: ttl, count: count})
}

func (q *heldQuay) carryHead() {
	if len(q.cohorts) == 0 {
		return
	}
	q.cohorts[0].count--
	if q.cohorts[0].count <= 0 {
		q.cohorts = q.cohorts[1:]
	}
}

func (q *heldQuay) keepYoung(now float64) {
	for i := range q.cohorts {
		q.cohorts[i].at = max(q.cohorts[i].at, now-q.cohorts[i].ttl*.5)
	}
}

func (q *heldQuay) views(now float64) []view.CohortView {
	out := make([]view.CohortView, 0, len(q.cohorts))
	for _, c := range q.cohorts {
		out = append(out, c.view(now))
	}
	return out
}

func TestAShownBoxKeepsItsLookWhileTheRealmKeepsBoxesYoung(t *testing.T) {
	keepsEveryLook(t, func(int) float64 { return 30 })
}

func TestAShownBoxKeepsItsLookWhenHeldShipsKeepTheirBoxesForDifferentTimes(t *testing.T) {
	keepsEveryLook(t, func(ship int) float64 { return 27.7 + float64(ship%3)*1.35 })
}

func keepsEveryLook(t *testing.T, ttlOf func(ship int) float64) {
	t.Helper()
	var m Manifest
	q := &heldQuay{}
	shown := map[[2]int]shownLook{}
	for tick := 0; tick < 2400; tick++ {
		now := float64(tick) * .25
		if tick%12 == 0 {
			q.arriveFor(now, 2+tick/12%4, ttlOf(tick/12))
		}
		if tick%52 == 0 && tick > 300 {
			q.carryHead()
		}
		q.keepYoung(now)
		m.Update(kindsWith(q.views(now)), now, now, alwaysHaven(0))
		loads := m.Sorted()
		if len(loads) != len(q.cohorts) {
			t.Fatalf("tick %d: %d loads for %d cohorts: a held cohort was taken for a new load", tick, len(loads), len(q.cohorts))
		}
		seen := map[[2]int]bool{}
		for i, l := range loads {
			for _, b := range l.Boxes() {
				if b.Column == offYard {
					continue
				}
				key := [2]int{q.cohorts[i].id, b.Index}
				seen[key] = true
				look := shownLook{b.Look.Variant, b.Look.Size, b.Column, b.Level}
				if was, ok := shown[key]; ok && was != look {
					t.Fatalf("tick %d: box %d of cohort %d changed from %+v to %+v", tick, b.Index, q.cohorts[i].id, was, look)
				}
				shown[key] = look
			}
		}
		for key := range shown {
			if !seen[key] {
				delete(shown, key)
			}
		}
	}
}

func TestACarriedBoxOfAHeldPileIsNotMistakenForSpoilage(t *testing.T) {
	var m Manifest
	q := &heldQuay{}
	q.arrive(0, 1)
	q.arrive(.5, 4)
	for tick := 0; tick < 200; tick++ {
		now := float64(tick) * .25
		q.keepYoung(now)
		m.Update(kindsWith(q.views(now)), now, now, alwaysHaven(0))
	}
	q.carryHead()
	m.Update(kindsWith(q.views(50)), 50, 50, alwaysHaven(0))
	if len(m.Ghosts) != 0 {
		t.Errorf("carrying the last box of a held cohort left %d spoil ghosts", len(m.Ghosts))
	}
}
