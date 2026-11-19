package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAClassPayoffCountsThePriceTheAcademyCharges(t *testing.T) {
	st := porterState(1.2, 2, 8)
	if got := ClassPayoff(st, "porter:2.0"); !strings.Contains(got, "PAYS BACK") {
		t.Fatalf("at its list size of 3,000 porter:2.0 should settle slowly: %q", got)
	}
	st.Soulstones = []view.SoulstoneEntry{{Image: "porter:2.0", Cost: 450, Visible: true}}
	if got := ClassPayoff(st, "porter:2.0"); strings.Contains(got, "PAYS BACK") {
		t.Fatalf("the result ignores the 450 the registry charges: %q", got)
	}
}
