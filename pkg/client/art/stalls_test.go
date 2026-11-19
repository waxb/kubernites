package art

import (
	"math"
	"testing"
)

func TestStallSpritesCoverEveryFaceStateAgeAndVariant(t *testing.T) {
	a := built().Buildings
	for _, kind := range stallContract() {
		names := []string{buildingName(kind, "construction-0")}
		for age := 0; age < AgeTiers; age++ {
			for v := 0; v < BuildingVariants; v++ {
				names = append(names, BuildingLookName(kind, "ready", age, v))
			}
			for _, st := range []string{"dark", "construction-1", "construction-2"} {
				names = append(names, BuildingLookName(kind, st, age, 0))
			}
		}
		for _, n := range names {
			if _, ok := a.Regions[n]; !ok {
				t.Fatalf("%s is missing", n)
			}
		}
	}
}

func TestStallFramesAnchorAtTheFootprintsNearCorner(t *testing.T) {
	for _, kind := range StallKinds {
		for _, face := range StallFaces {
			sz := stallSize(kind, face)
			w, h := StallFootprint(face)
			fr := isoFrameWH(newCanvas(sz.w, sz.h), w, h, sz.ax, sz.ay)
			if p := fr.p(float64(w), float64(h), 0); p.x != float64(sz.ax) || p.y != float64(sz.ay) {
				t.Fatalf("%s %s: the near corner lands at %v, anchor %d,%d", kind, face, p, sz.ax, sz.ay)
			}
			if left, right := fr.p(0, float64(h), 0).x, fr.p(float64(w), 0, 0).x; left < 1 || right > float64(sz.w-2) {
				t.Fatalf("%s %s: the footprint spans %v..%v of %d", kind, face, left, right, sz.w)
			}
		}
	}
}

func TestStallFacesTurnTheStallIntoItsEdge(t *testing.T) {
	want := map[string][2]float64{"sw": {0, 1}, "se": {1, 0}, "ne": {0, -1}, "nw": {-1, 0}}
	for face, front := range want {
		st := stall{face: face}
		w, h := StallFootprint(face)
		x0, y0, x1, y1 := st.grid(0, 0, 2, 1)
		if x0 != 0 || y0 != 0 || x1 != float64(w) || y1 != float64(h) {
			t.Fatalf("%s: the stall spans %v,%v..%v,%v of a %dx%d footprint", face, x0, y0, x1, y1, w, h)
		}
		if dx, dy := st.dir(0, 1); math.Abs(dx-front[0]) > 1e-9 || math.Abs(dy-front[1]) > 1e-9 {
			t.Fatalf("%s: the counter faces %v,%v", face, dx, dy)
		}
		if st.sees(frontAt(1)) != st.facesCamera() || st.sees(backAt(0)) == st.facesCamera() {
			t.Fatalf("%s: the camera sees the wrong side", face)
		}
	}
}

func TestStallsStayWithinTheirDeclaredHeight(t *testing.T) {
	a := built().Buildings
	for _, kind := range StallKinds {
		tallest := 0
		for _, face := range StallFaces {
			w, h := StallFootprint(face)
			for age := 0; age < AgeTiers; age++ {
				r := a.Regions[BuildingLookName(StallArtKind(kind, face), "ready", age, 0)]
				rise := contentHeight(a, r) - (w+h)*TileHeight/2
				if rise > StallHeight(kind)+2 {
					t.Errorf("%s %s age %d rises %d px, declared %d", kind, face, age, rise, StallHeight(kind))
				}
				tallest = max(tallest, rise)
			}
		}
		if tallest < StallHeight(kind)-14 {
			t.Errorf("%s rises at most %d px, declared %d", kind, tallest, StallHeight(kind))
		}
	}
}

func TestStallKindsLookApart(t *testing.T) {
	a := built().Buildings
	for _, age := range []int{0, AgeTiers - 1} {
		for i, k := range StallKinds {
			for _, o := range StallKinds[i+1:] {
				x := a.Regions[BuildingLookName(StallArtKind(k, "sw"), "ready", age, 0)]
				y := a.Regions[BuildingLookName(StallArtKind(o, "sw"), "ready", age, 0)]
				if differingShare(a, x, y) < 0.2 {
					t.Errorf("age %d: %s and %s look alike", age, k, o)
				}
			}
		}
	}
}

func TestStallFacesShowFrontsAndBacks(t *testing.T) {
	a := built().Buildings
	for _, kind := range StallKinds {
		sw := a.Regions[BuildingLookName(StallArtKind(kind, "sw"), "ready", 1, 0)]
		ne := a.Regions[BuildingLookName(StallArtKind(kind, "ne"), "ready", 1, 0)]
		if differingShare(a, sw, ne) < 0.2 {
			t.Errorf("%s shows the same face front and back", kind)
		}
	}
}

func TestEveryBoxKindHasAStall(t *testing.T) {
	for box, want := range map[string]string{"parcel": "stall-parcel", "ledger": "stall-ledger", "crate": "stall-crate", "dispatch": "stall-dispatch", "sealed": "stall-seal", "mystery": "stall-parcel"} {
		if got := StallForBox(box); got != want {
			t.Fatalf("StallForBox(%q) = %q", box, got)
		}
	}
	if !isStallKind("stall-herald") || isStallKind("codex") || StallHeight("stall-archive") <= StallHeight("stall-parcel") {
		t.Fatal("stall kinds or heights are wrong")
	}
}

func TestStallPortraitsAgeWithTheRealm(t *testing.T) {
	s := built()
	for _, kind := range StallKinds {
		for age := 0; age < AgeTiers; age++ {
			if _, ok := s.Portraits.Regions[portraitName(AgedPortrait(kind, age))]; !ok {
				t.Fatalf("portrait of %s at age %d is missing", kind, age)
			}
			if _, ok := s.Portraits.Regions[BuildIconName(kind, age)]; !ok {
				t.Fatalf("build icon of %s at age %d is missing", kind, age)
			}
		}
	}
}
