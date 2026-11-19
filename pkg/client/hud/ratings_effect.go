package hud

import (
	"fmt"
	"math"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func favorEffect(f Frame) string {
	m := fmt.Sprintf("x%.2f", view.FavorMultiplier(f.Caesar.Ratings.Favor))
	return ui.Bridge("It draws "+m+" ships now.", "It scales incoming traffic "+m+" now.")
}

func reforgeEffect(f Frame) string {
	if !f.Revealed("reforge") {
		return ""
	}
	bonus := math.Round(view.ReforgeBonus(f.State.Ratings) * 100)
	return fmt.Sprintf("At a Reforge the four ratings add +%.0f%% Golden Threads.", bonus)
}

func withRatingEffects(f Frame, dials []Dial) []Dial {
	if !f.Caesar.Rated {
		return dials
	}
	out := make([]Dial, len(dials))
	for i, d := range dials {
		next := d
		if d.Label == "FAVOR" {
			next.Tip = withPayoffLine(next.Tip, favorEffect(f))
		}
		next.Tip = withPayoffLine(next.Tip, reforgeEffect(f))
		out[i] = next
	}
	return out
}
