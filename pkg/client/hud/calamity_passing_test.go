package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAPassingTroubleNeverOpensAsANewCard(t *testing.T) {
	passing := noDepot
	passing.Passing = true
	var st CalamityState
	st = st.Step([]view.CalamityView{passing}, 10, CalamityGate{})
	st = st.Step([]view.CalamityView{passing}, 10+CalamitySettle+1, CalamityGate{})
	if ready := st.Ready([]view.CalamityView{passing}, 10+CalamitySettle+1); len(ready) != 0 {
		t.Fatalf("a trouble that is already passing opened a card the player never saw live: %+v", ready)
	}
}

func TestAnOpenCardStaysWhileItsTroublePasses(t *testing.T) {
	var st CalamityState
	st = st.Step([]view.CalamityView{noDepot}, 10, CalamityGate{})
	st = st.Step([]view.CalamityView{noDepot}, 10+CalamitySettle, CalamityGate{})
	passing := noDepot
	passing.Passing = true
	st = st.Step([]view.CalamityView{passing}, 11+CalamitySettle, CalamityGate{})
	if ready := st.Ready([]view.CalamityView{passing}, 11+CalamitySettle); len(ready) != 1 {
		t.Fatalf("a card the player saw live must stay up while its trouble passes, got %+v", ready)
	}
}
