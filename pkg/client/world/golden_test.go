package world

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestGleamsCountDownAndSpreadAlongThePlank(t *testing.T) {
	g := Gleam{ID: "g", Fades: 110, Span: 10}
	if GleamLeft(g, 104) != 6 || GleamLeft(g, 200) != 0 || math.Abs(GleamShare(g, 104)-.6) > 1e-9 || GleamShare(Gleam{}, 1) != 0 {
		t.Fatalf("left %v share %v", GleamLeft(g, 104), GleamShare(g, 104))
	}
	p := HavenPlan{Side: Vec{1, 0}, Out: Vec{0, -1}}
	b := Berth{Plank: Vec{10, 10}}
	spots := GleamSpots(p, b, 3)
	if len(spots) != 3 {
		t.Fatalf("spots %v", spots)
	}
	for i, at := range spots {
		if want := 10 + gleamInland + float64(i)*gleamStep; math.Abs(at.Y-want) > 1e-9 || math.Abs(at.X-10) > gleamJog+1e-9 {
			t.Fatalf("gleam %d at %v, want inland along the pier at %v", i, at, want)
		}
	}
	p.Props = []HavenProp{{Kind: "sign", At: spots[1]}}
	for _, at := range GleamSpots(p, b, 3) {
		if Dist(at, spots[1]) < gleamClear {
			t.Fatalf("a gleam at %v crowds the sign at %v", at, spots[1])
		}
	}
	p.Props = nil
	for k := range gleamTries {
		p.Props = append(p.Props, HavenProp{Kind: "crane", At: gleamCandidate(p, b, k)})
	}
	if got := GleamSpots(p, b, 3); len(got) != 3 {
		t.Fatalf("a crowded pier still shows every gleam: %v", got)
	}
}

func TestGleamingBoxesAreClickable(t *testing.T) {
	sc, r := harborScene(t)
	cam := cameraOn(sc, r)
	h := r.HarborOf()
	sc.Golden = &Gilded{Harbor: h.Plans[0].ID, Gleams: []Gleam{{ID: "golden-1-1", Fades: 40, Span: 20}, {ID: "golden-1-2", Fades: 42, Span: 22}, {ID: "gone", Fades: 5, Span: 3}}}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	seen := 0
	for _, hit := range r.HitList {
		if hit.Kind != HitGolden {
			continue
		}
		seen++
		got, ok := r.Pick(hit.Rect.X+hit.Rect.W/2, hit.Rect.Y+hit.Rect.H/2)
		if !ok || got.Kind != HitGolden {
			t.Errorf("clicking %s picks %+v", hit.ID, got)
		}
		if _, ok := r.GleamSpot(hit.ID); !ok {
			t.Errorf("%s has no spot for its burst", hit.ID)
		}
	}
	if seen != 2 {
		t.Fatalf("%d gleaming boxes are clickable, want the 2 that have not faded", seen)
	}
	sc.Golden = nil
	r.Draw(dst, cam, sc)
	if _, ok := r.GleamSpot("golden-1-1"); ok {
		t.Fatal("a sailed ship leaves its gleams behind")
	}
}
