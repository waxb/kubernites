package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheFirstStoneStepOffersNoDoIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := offerFrame(t, raiseClerks)
	f.State.Objectives[0].ID = view.AskObjective
	f.State.Objectives[0].Command = "kubectl get nodes"
	if _, ok := HitID(h.ObjectiveButtons(f), "obj:speak"); !ok {
		t.Fatal("the first Stone step offers no command to write on the Stone")
	}
	if _, ok := HitID(h.ObjectiveButtons(f), "obj:doit"); ok {
		t.Fatal("the first Stone step speaks for the player: DO IT is offered")
	}
}
