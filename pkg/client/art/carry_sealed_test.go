package art

import "testing"

func TestASealwrightCarriesAWaxSealedBox(t *testing.T) {
	a := built().Walkers
	for size := 0; size < BoxSizes; size++ {
		for f := 0; f < WalkerFrames; f++ {
			sealed, ok := a.Regions[CarryName(size, SealedCarry, "se", f)]
			if !ok {
				t.Fatalf("no sealed carry %d/%d", size, f)
			}
			parcel := a.Regions[CarryName(size, "parcel", "se", f)]
			if redPixels(a, sealed) <= redPixels(a, parcel) {
				t.Errorf("size %d frame %d: the sealed box shows no wax (%d red against %d)", size, f, redPixels(a, sealed), redPixels(a, parcel))
			}
			if _, ok := a.Regions[CarryName(size, SealedCarry, "sw", f)]; !ok {
				t.Errorf("no south-west sealed carry %d/%d", size, f)
			}
		}
	}
}
