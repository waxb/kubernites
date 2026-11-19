package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

func foremenOnlyFrame() Frame {
	f := foremanBoardFrame()
	f.State.RaiseKeeper = view.KeeperReplicaSet
	return f
}

func namesACrew(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "crew") || strings.Contains(l, "deployment")
}

func bareForemanTexts(t *testing.T, f Frame) []string {
	t.Helper()
	h, v := openBoard(t, f)
	title, kind := boardTitle(v, f)
	_, card := chainTitle(v.Chain, f)
	out := append([]string{title, kind, card, crewLegend(v.Chain, CrewModeChain, f)}, cardTexts(CrewCard(f))...)
	stall := f
	stall.Selection = Selection{Kind: "structure", Structure: view.Structure{Kind: "guildhall", Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}}
	chain, ok := h.chainCell(stall)
	if !ok {
		t.Fatal("the parcels stall shows no SHIFT button for the shift behind it")
	}
	return append(out, chain.Label, chain.Tip)
}

func TestABareForemanNamesNoCrewBeforeCrewsAreLearned(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, namer := range []func(string) string{nil, lexicon.TrueNames} {
		ui.SetNamer(namer)
		for _, text := range bareForemanTexts(t, foremenOnlyFrame()) {
			if namesACrew(text) {
				t.Errorf("a bare shift names the crew (Deployment) before its page is learned: %q", text)
			}
		}
		learned := foremenOnlyFrame()
		learned.State.Unlocks[view.DeploymentsPage] = true
		if _, kind := chainTitle(learned.Crews[0], learned); !namesACrew(kind) {
			t.Errorf("once crews are learned, a bare shift's title does not say no crew stands behind it: %q", kind)
		}
	}
}

func TestScalingPastTheKeepsSaysTheNewWorkerWouldWait(t *testing.T) {
	f := boardFrame()
	f.State.Crews[0].Fits, f.State.Crews[0].PerReplica.CPU = 2, 200
	more := ScaleButtons(f.Crews[0], f)[1]
	if more.Cost == summonNoRoom || !strings.Contains(more.Tip, "room for 2 more") {
		t.Fatalf("with room for two the +1 reads %q: %q", more.Cost, more.Tip)
	}
	f.State.Crews[0].Fits = 0
	more = ScaleButtons(f.Crews[0], f)[1]
	if more.Cost != summonNoRoom || more.Disabled || !strings.Contains(more.Tip, "Unhoused") {
		t.Fatalf("with no room the +1 reads %+v", more)
	}
	if got := stepLabel(f.Crews[0], f, 1); !strings.Contains(got, noRoomStep) {
		t.Fatalf("the board's +1 reads %q with no room", got)
	}
}
