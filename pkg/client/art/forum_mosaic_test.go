package art

import (
	"image/color"
	"testing"
)

func castleForumNames() []string {
	out := []string{}
	for v := 0; v < 3; v++ {
		out = append(out, ForumPaveName(ageCastle, v))
	}
	for _, side := range ForumSides {
		out = append(out, ForumRimName(ageCastle, side))
	}
	out = append(out, ForumBandName(ageCastle, "x"), ForumBandName(ageCastle, "y"))
	for dy := -ForumRoseReach; dy <= ForumRoseReach; dy++ {
		for dx := -ForumRoseReach; dx <= ForumRoseReach; dx++ {
			out = append(out, ForumRoseName(ageCastle, dx, dy))
		}
	}
	return out
}

func inRamp(c color.NRGBA, r ramp) bool {
	for _, x := range r {
		if c == x {
			return true
		}
	}
	return false
}

func TestTheUATForumIsARomanMosaicWithoutIceRinkBlue(t *testing.T) {
	a := buildBuildings()
	for _, name := range castleForumNames() {
		r := a.Regions[name]
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				if c := a.Image.NRGBAAt(x, y); c.A > 0 && inRamp(c, blueR) {
					t.Fatalf("%s paints ice-rink blue at %d,%d", name, x-r.X, y-r.Y)
				}
			}
		}
	}
}

func TestUATCompartmentsMeetAtDarkCrossletsOnALightField(t *testing.T) {
	for v := 0; v < 3; v++ {
		for _, uv := range [][2]float64{{0.02, 0.02}, {0.97, 0.04}, {0.05, 0.97}} {
			if c := uatPave(uv[0], uv[1], 7, 5, v); !inRamp(c, nigraR) {
				t.Errorf("variant %d at %v: a compartment corner is %v, not a dark crosslet", v, uv, c)
			}
		}
		for _, uv := range [][2]float64{{0.01, 0.5}, {0.3, 0.7}} {
			if c := uatPave(uv[0], uv[1], 7, 5, v); inRamp(c, nigraR) {
				t.Errorf("variant %d at %v: the field and its seams stay light, got %v", v, uv, c)
			}
		}
	}
}

func TestUATMotifsDifferAndThePlainCompartmentStaysCalm(t *testing.T) {
	centre := func(v int) color.NRGBA { return uatPave(0.62, 0.5, 9, 4, v) }
	if !inRamp(centre(0), tesseraR) {
		t.Errorf("the plain compartment should be cream near its centre, got %v", centre(0))
	}
	if inRamp(centre(1), tesseraR) || inRamp(centre(2), tesseraR) || centre(1) == centre(2) {
		t.Errorf("the lozenge and the rosette should fill the compartment with distinct colours: %v %v", centre(1), centre(2))
	}
}

func TestTheProductionForumIsOpusSectileNotCourtLines(t *testing.T) {
	a := buildBuildings()
	for _, name := range []string{ForumBandName(ageEmpire, "x"), ForumBandName(ageEmpire, "y")} {
		r := a.Regions[name]
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				if c := a.Image.NRGBAAt(x, y); c.A > 0 && (inRamp(c, verdeR) || c == goldR[2]) {
					t.Fatalf("%s draws a court line (%v) at %d,%d", name, c, x-r.X, y-r.Y)
				}
			}
		}
	}
	centre := func(v int) color.NRGBA { return productionPave(0.55, 0.5, 9, 4, v) }
	if !inRamp(centre(0), marbleR) || !inRamp(centre(1), porphyryR) || !inRamp(centre(2), verdeR) {
		t.Errorf("slab, porphyry roundel and serpentine square expected at the centres: %v %v %v", centre(0), centre(1), centre(2))
	}
}

func TestTheRosettePetalsLieAlongTheFloorAxes(t *testing.T) {
	for _, axis := range [][2]float64{{0.15, 0}, {-0.15, 0}, {0, 0.15}, {0, -0.15}} {
		if c := uatPave(0.5+axis[0], 0.5+axis[1], 8, 6, 2); !inRamp(c, ochreR) {
			t.Errorf("a petal should lie along the floor axis at %v, got %v", axis, c)
		}
	}
	for _, between := range [][2]float64{{0.1, -0.1}, {-0.1, 0.1}} {
		if c := uatPave(0.5+between[0], 0.5+between[1], 8, 6, 2); inRamp(c, ochreR) {
			t.Errorf("between the floor axes at %v the rosette should not bloom toward the camera, got %v", between, c)
		}
	}
}
