package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

type boxSpot struct{ column, level, lift, size, variant int }

func spotsOf(m *Manifest, at float64) map[float64]map[int]boxSpot {
	out := map[float64]map[int]boxSpot{}
	for _, l := range m.Sorted() {
		for _, b := range Yards([]*Load{l})[0].Boxes {
			if out[b.At] == nil {
				out[b.At] = map[int]boxSpot{}
			}
			out[b.At][b.Index] = boxSpot{b.Column, b.Level, b.Lift, b.Look.Size, b.Look.Variant}
		}
	}
	return out
}

func stayedPut(t *testing.T, before, after map[float64]map[int]boxSpot) {
	t.Helper()
	for at, boxes := range before {
		for i, was := range boxes {
			if now, ok := after[at][i]; ok && now != was {
				t.Errorf("box %d of the load landed at %v moved from %+v to %+v", i, at, was, now)
			}
		}
	}
}

func TestAWaitingBoxKeepsItsSpotWhenItsLoadGrowsBehindANewerOne(t *testing.T) {
	var m Manifest
	m.Update(kindsWith([]view.CohortView{{Count: 3, Age: 1, Remaining: 27, TTL: 28}}), 10, 0, alwaysHaven(0))
	m.Update(kindsWith([]view.CohortView{{Count: 3, Age: 1.2, Remaining: 26.8, TTL: 28}, {Count: 4, Age: .1, Remaining: 27.9, TTL: 28}}), 10.2, .2, alwaysHaven(0))
	before := spotsOf(&m, 10.2)
	m.Update(kindsWith([]view.CohortView{{Count: 6, Age: 1.4, Remaining: 26.6, TTL: 28}, {Count: 4, Age: .3, Remaining: 27.7, TTL: 28}}), 10.4, .4, alwaysHaven(0))
	after := spotsOf(&m, 10.4)
	stayedPut(t, before, after)
	if n := len(after[9]); n != 6 {
		t.Errorf("the grown load shows %d boxes, want 6", n)
	}
}

func TestTheNewestLoadGrowsWithoutLiftingItsOwnBoxes(t *testing.T) {
	var m Manifest
	m.Update(kindsWith([]view.CohortView{{Count: 3, Age: 1, Remaining: 27, TTL: 28}}), 10, 0, alwaysHaven(0))
	before := spotsOf(&m, 10)
	m.Update(kindsWith([]view.CohortView{{Count: 4, Age: 1.2, Remaining: 26.8, TTL: 28}}), 10.2, .2, alwaysHaven(0))
	stayedPut(t, before, spotsOf(&m, 10.2))
}

func TestAnEmptiedSpotIsNotRefilledWhileItsBoxesStillFade(t *testing.T) {
	var m Manifest
	full := []view.CohortView{{Count: 2 * YardColumns, Age: 20, Remaining: 8, TTL: 28}, {Count: 4, Age: 1, Remaining: 27, TTL: 28}}
	m.Update(kindsWith(full), 30, 1, alwaysHaven(0))
	served := []view.CohortView{{Count: 2*YardColumns - 2, Age: 20.2, Remaining: 7.8, TTL: 28}, {Count: 4, Age: 1.2, Remaining: 26.8, TTL: 28}}
	m.Update(kindsWith(served), 30.2, 1.2, alwaysHaven(0))
	for _, b := range Yards(m.Sorted())[0].Boxes {
		if b.At == 29 {
			t.Fatalf("a hidden stack took the spot its neighbour just emptied: %+v", b)
		}
	}
	later := []view.CohortView{{Count: 2*YardColumns - 2, Age: 20.2 + SpoilLinger, Remaining: 7.8 - SpoilLinger, TTL: 28}, {Count: 4, Age: 1.2 + SpoilLinger, Remaining: 26.8 - SpoilLinger, TTL: 28}}
	m.Update(kindsWith(later), 30.2+SpoilLinger, 1.2+SpoilLinger+.01, alwaysHaven(0))
	shown := 0
	for _, b := range Yards(m.Sorted())[0].Boxes {
		if b.At == 29 {
			shown++
		}
	}
	if shown != 2 {
		t.Errorf("after the linger the waiting stack shows %d boxes, want 2", shown)
	}
}

type fakeCohort struct {
	at, ttl     float64
	count, goal int
}

func (c fakeCohort) view(now float64) view.CohortView {
	return view.CohortView{Count: float64(c.count), Age: now - c.at, Remaining: c.at + c.ttl - now, TTL: c.ttl}
}

func TestNoShownBoxEverMovesWhileShipsUnloadAndCrewsCarry(t *testing.T) {
	var m Manifest
	cohorts := []fakeCohort{}
	shown := map[[2]float64]boxSpot{}
	seed := uint32(7)
	for tick := 0; tick < 3000; tick++ {
		now := float64(tick) * .2
		seed = mix32(seed + uint32(tick))
		if seed%9 == 0 {
			cohorts = append(cohorts, fakeCohort{at: now, ttl: 30, goal: 1 + int(seed%11)})
		}
		next := cohorts[:0]
		served := seed%3 == 0
		for _, c := range cohorts {
			if c.count < c.goal && (seed>>4)%2 == 0 {
				c.count++
			}
			if served && c.count > 0 {
				c.count--
				served = false
			}
			if now-c.at < c.ttl && (c.count > 0 || c.count < c.goal) {
				next = append(next, c)
			}
		}
		cohorts = next
		views := make([]view.CohortView, 0, len(cohorts))
		for _, c := range cohorts {
			views = append(views, c.view(now))
		}
		m.Update(kindsWith(views), now, now, alwaysHaven(0))
		seen := map[[2]float64]bool{}
		for _, y := range Yards(m.Sorted()) {
			for _, b := range y.Boxes {
				key := [2]float64{b.At, float64(b.Index)}
				seen[key] = true
				spot := boxSpot{b.Column, b.Level, b.Lift, b.Look.Size, b.Look.Variant}
				if was, ok := shown[key]; ok && was != spot {
					t.Fatalf("tick %d: box %d of the load at %v moved from %+v to %+v", tick, b.Index, b.At, was, spot)
				}
				shown[key] = spot
			}
		}
		for key := range shown {
			if !seen[key] {
				delete(shown, key)
			}
		}
	}
}

func TestAServedOutLoadStillCountsInItsShipsLandingOrder(t *testing.T) {
	var m Manifest
	both := []view.CohortView{{Count: 3, Age: 1, Remaining: 27, TTL: 28}, {Count: 4, Age: .9, Remaining: 27, TTL: 28}}
	m.Update(kindsWith(both), 10, 0, alwaysHaven(0))
	total := 0
	for _, l := range m.landed() {
		total += l.Total
	}
	m.Update(kindsWith([]view.CohortView{{Count: 4, Age: 1, Remaining: 26.9, TTL: 28}}), 10.1, .1, alwaysHaven(0))
	after := 0
	for _, l := range m.landed() {
		after += l.Total
	}
	if after != total {
		t.Errorf("the ship's landing order lost %d boxes when one load was served out: later boxes would jump", total-after)
	}
}
