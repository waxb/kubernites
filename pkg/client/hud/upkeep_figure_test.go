package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAKeepsUpkeepReadsTheSameOnItsCardAndInItsSurvey(t *testing.T) {
	for _, v := range []float64{0.004, 0.123, 3.45, 12.04, 28.46, 150.3, 2400} {
		if card, survey := Rate(v), view.UpkeepText(v); card != survey {
			t.Errorf("an overhead of %v a second reads %s on the Keep's card and the build bar, and %s in the survey and the Keep offer", v, card, survey)
		}
	}
}
