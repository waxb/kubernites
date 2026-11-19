package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func rungOf(segs []Segment) string {
	first, _, _ := strings.Cut(segs[0].Text, " ")
	return first
}

func TestTheBoardKeepsItsWordsWhateverDigitsTheRatesHave(t *testing.T) {
	rates := []float64{0.5, 1.1, 9.5, 9.96, 12, 1500}
	for w := 60; w < 400; w++ {
		want := rungOf(FlowSegments(view.EconomyView{Arriving: rates[0], Carried: rates[0], Lost: rates[0]}, w))
		for _, v := range rates[1:] {
			if got := rungOf(FlowSegments(view.EconomyView{Arriving: v, Carried: v, Lost: v}, w)); got != want {
				t.Fatalf("width %d: the board reads %q at %v and %q at %v", w, got, v, want, rates[0])
			}
		}
	}
}
