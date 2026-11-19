package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func crewsLearned(f Frame) Frame {
	next := f
	next.State.Unlocks = map[string]bool{view.ForemenPage: true, view.DeploymentsPage: true}
	return next
}

func TestAForemansGridOffersDeployOnceCrewsAreLearned(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	if cell := gridCell(t, h, foremanBoardFrame(), view.DeployButton); cell >= 0 {
		t.Fatalf("DEPLOY stands in cell %d before the page of Crews", cell)
	}
	f := crewsLearned(foremanBoardFrame())
	wantCells(t, h, f, map[string]int{view.DeployButton: 2, ScaleUpButton: 4})
	b, _ := HitID(h.CommandButtons(f), view.DeployButton)
	if b.Label != "RAISE CREW" {
		t.Fatalf("the shift's grid raises a crew with one verb, RAISE CREW: %q", b.Label)
	}
	if !strings.Contains(b.Tip, "$ kubectl create deployment porters --image=porter:1.0 --replicas=3 --port=8080") {
		t.Fatalf("DEPLOY names its kubectl: %q", b.Tip)
	}
	a, from := actionOf(clickOn(t, h, f, view.DeployButton))
	if a.Action != view.ActDeployCrew || a.Name != "porters" || from != FromCrewCard {
		t.Fatalf("DEPLOY = %+v from %q", a, from)
	}
	g := guided(f, "crew:default/porters", view.DeployButton)
	if w := New(nil, (&fakeRunner{}).backend()).WalkGuide(g); !w.OK() || len(w.Steps) != 1 || w.Steps[0] != view.DeployButton {
		t.Errorf("the guide lights DEPLOY on the shift's crew: %+v", w)
	}
}

func TestALegacyReplicaSetOffersNoDeploy(t *testing.T) {
	f := crewsLearned(bareCrewFrame(Selection{Kind: SelectForeman, ID: "default/porters-7d9f"}))
	h := New(nil, (&fakeRunner{}).backend())
	if cell := gridCell(t, h, f, view.DeployButton); cell >= 0 {
		t.Fatalf("a ReplicaSet named for no trade offers DEPLOY in cell %d", cell)
	}
}
