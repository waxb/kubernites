package world

import "testing"

func TestUATPavingAlternatesPlainAndMotifCompartments(t *testing.T) {
	for y := 0; y < 6; y++ {
		for x := 0; x < 6; x++ {
			got := paveVariant(forumTile{at: Point{X: x, Y: y}}, tierUAT)
			want := 0
			if (x+y)%2 == 1 {
				want = 1 + x%2
			}
			if got != want {
				t.Fatalf("tile %d,%d: variant %d, want %d (plain compartments alternate with lozenges and rosettes)", x, y, got, want)
			}
		}
	}
}

func TestProductionPavingAlternatesRoundelsAndSquares(t *testing.T) {
	for y := 0; y < 6; y++ {
		for x := 0; x < 6; x++ {
			got := paveVariant(forumTile{at: Point{X: x, Y: y}}, tierProd)
			want := 2 - (x+y+1)%2
			if got != want {
				t.Fatalf("tile %d,%d: variant %d, want %d (porphyry roundels alternate with serpentine squares)", x, y, got, want)
			}
		}
	}
}
