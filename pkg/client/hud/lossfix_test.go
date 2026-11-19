package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestLossLinesFollowTheCalamityCardAndItsGates(t *testing.T) {
	st := view.GameState{Freight: fixtureFreight()}
	st.Unlocks = map[string]bool{}
	st.Freight.Losses = view.LossView{Spoiled: .4, Blocked: .1}
	scale := view.Action{Action: view.ActScaleCrew, Name: "porters", Replicas: 4}
	st.Calamities = []view.CalamityView{{ID: "spoiling:parcel", Kind: view.CalamitySpoiling, Remedies: []view.RemedyView{{Label: "Scale the porters to 4", Action: &scale}}}}
	lines := StateLossLines(st)
	if len(lines) != 2 || lines[0].Text != "Spoiled 0.40/s: scale porters to 4" || lines[1].Text != "Lost in flight 0.10/s: worker stopped" {
		t.Fatalf("lines = %+v", lines)
	}
	st.Unlocks[familiarsUnlock] = true
	if lines := StateLossLines(st); lines[1].Text != "Lost in flight 0.10/s: bind a Hound" {
		t.Fatalf("once Familiars are learned the Hound is named: %+v", lines)
	}
	for _, c := range []struct {
		rem  view.RemedyView
		want string
	}{
		{view.RemedyView{Label: "Close a berth: fewer ships until the crews cope", Action: &view.Action{Action: view.ActCloseBerth}}, "close a berth"},
		{view.RemedyView{Label: "Roll the leaner porter:2.0 out", Action: &view.Action{Action: view.ActCrewImage, Image: "porter:2.0"}}, "roll porter:2.0 out"},
		{view.RemedyView{Label: "Pull leaner porters at the registry (450 MiB)"}, "pull leaner porters at the registry"},
		{view.RemedyView{Label: "Switch to porter:1.1", Action: &view.Action{Action: view.ActReseatDoor}}, "switch the door porter"},
		{view.RemedyView{Label: "Raze a hut on node-a: its shift raises a new one on a Keep with room", Command: "kubectl delete pod porters-x"}, "raze a hut"},
		{view.RemedyView{Label: "Raze a hut on node-a: its shift raises a new one on a Keep with room", Command: "kubectl delete pod porters-x", Action: &view.Action{Action: view.ActMoveWorker, Name: "porters", ID: "porters-x"}}, "raze a hut"},
		{view.RemedyView{Label: "Open the ledgers depot", Action: &view.Action{Action: "openDepot", Kind: "ledger"}}, "open the ledgers"},
	} {
		if got := ShortFix(c.rem); got != c.want {
			t.Errorf("%q = %q, want %q", c.rem.Label, got, c.want)
		}
	}
}

func TestALossLineNamingTheFixReadsWholeInBothModes(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, mode := range namesModes() {
		mode.apply()
		st := view.GameState{Freight: view.FreightView{
			Kinds:  []view.KindView{{Kind: "ledger", Name: "ledgers", Depot: "ledgers", CrewName: "clerks", Losses: view.LossView{NoDepot: .24}}},
			Losses: view.LossView{NoDepot: .24}}}
		st.Calamities = []view.CalamityView{{ID: view.CalamityNoDepot + ":ledger", Remedies: []view.RemedyView{
			{Label: "Open the ledgers depot", Action: &view.Action{Action: "openDepot", Kind: "ledger"}}}}}
		for _, l := range StateLossLines(st) {
			if strings.Contains(l.Text, "...") || len([]rune(ui.Name(l.Text))) > lossLineChars {
				t.Errorf("[%s] the loss line reads %q", mode.name, l.Text)
			}
		}
	}
}
