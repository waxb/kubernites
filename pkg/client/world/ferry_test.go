package world

import (
	"math"
	"testing"
)

func testLane(length float64) FerryLane {
	home := Jetty{Head: Vec{10, 5}, Dock: Vec{10.9, 5}, Out: Vec{1, 0}}
	away := Jetty{Head: Vec{11.8 + length, 5}, Dock: Vec{10.9 + length, 5}, Out: Vec{-1, 0}}
	path := []Vec{home.Dock, away.Dock}
	return FerryLane{Isle: "fernholm", Home: home, Away: away, Path: path, Length: PathLength(path)}
}

func TestFerriesShuttleCalmlyBetweenTheJetties(t *testing.T) {
	l := testLane(9)
	if got := l.Period(); math.Abs(got-2*(l.Cross()+ferryDwell)) > 1e-9 || l.Cross() < l.Length/ferrySpeed-1e-9 {
		t.Fatalf("period %.2f, crossing %.2f", got, l.Cross())
	}
	at := func(t float64) FerryPose { return l.Boat(0, t) }
	if p := at(ferryDwell / 2); p.Moving || p.Pos != l.Home.Dock || p.Side != sideHome {
		t.Fatalf("at the start the ferry is %+v, want moored at home", p)
	}
	mid := at(ferryDwell + l.Cross()/2)
	if !mid.Moving || math.Abs(mid.Pos.X-(l.Home.Dock.X+l.Away.Dock.X)/2) > .01 || mid.Dir != ShipHeading(Vec{1, 0}) {
		t.Fatalf("halfway the ferry is %+v", mid)
	}
	if p := at(ferryDwell*1.5 + l.Cross()); p.Moving || p.Pos != l.Away.Dock || p.Side != sideAway {
		t.Fatalf("after the crossing the ferry is %+v", p)
	}
	back := at(2*ferryDwell + 1.5*l.Cross())
	if !back.Moving || back.Dir != ShipHeading(Vec{-1, 0}) {
		t.Fatalf("on the way home the ferry is %+v", back)
	}
	fastest := 0.0
	for tt := 0.0; tt < l.Period(); tt += .05 {
		fastest = max(fastest, Dist(at(tt).Pos, at(tt+.05).Pos)/.05)
	}
	if fastest > ferrySpeed*1.6 {
		t.Fatalf("the ferry rushes at %.2f tiles/s", fastest)
	}
	if a, b := l.Boat(0, 3), l.Boat(1, 3); a.Pos == b.Pos {
		t.Fatal("both ferries of a lane sail together")
	}
}

func TestAWalkerBoardsTheNextFerryFromHisSide(t *testing.T) {
	l := testLane(9)
	for _, fromHome := range []bool{true, false} {
		for arrive := 0.0; arrive < 2*l.Period(); arrive += .7 {
			s := l.NextSailing(fromHome, arrive)
			if s.Depart < arrive-1e-9 || s.Board < arrive-1e-9 || s.Board > s.Depart+1e-9 || math.Abs(s.Land-s.Depart-l.Cross()) > 1e-9 {
				t.Fatalf("arriving at %.1f (home %v): %+v", arrive, fromHome, s)
			}
			if s.Depart-arrive > l.Period()/float64(FerryBoats)+1e-9 {
				t.Fatalf("arriving at %.1f (home %v) waits %.1f s", arrive, fromHome, s.Depart-arrive)
			}
			side := sideHome
			if !fromHome {
				side = sideAway
			}
			boat := l.Boat(s.Boat, s.Board+1e-6)
			if boat.Moving || boat.Side != side {
				t.Fatalf("arriving at %.1f (home %v) boards %+v at %.2f", arrive, fromHome, boat, s.Board)
			}
			if p := l.Boat(s.Boat, s.Land+1e-6); p.Moving || p.Side == side {
				t.Fatalf("arriving at %.1f (home %v) lands with the ferry at %+v", arrive, fromHome, p)
			}
		}
	}
}
