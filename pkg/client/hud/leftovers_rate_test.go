package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheBoardAndTheCardsFormatARateTheSameWay(t *testing.T) {
	for _, v := range []float64{0, 0.004, 0.05, 0.25, 0.999, 1, 1.15, 9.96, 10, 12.4, 999, 1500} {
		if got, want := Rate(v), view.RateFigure(v); got != want {
			t.Errorf("%v: the board reads %q, the card %q", v, got, want)
		}
	}
}
