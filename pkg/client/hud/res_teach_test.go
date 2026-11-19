package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheResourcesDialogExplainsTheFieldInFocus(t *testing.T) {
	v := crewView{chain: crewChain(), mode: CrewModeRes, fields: resFields}
	h := New(nil, (&fakeRunner{}).backend())
	h.Crew.Res = v.chain.Resources
	if got := h.focusField(v); got.key != "mem-lim" {
		t.Fatalf("with nothing hovered or changed the focus is %q", got.key)
	}
	h.Crew.Res.CPURequest = 500
	if got := h.focusField(v); got.key != "cpu-req" {
		t.Fatalf("the changed field is not in focus: %q", got.key)
	}
	h.Hovering, h.Hover = true, Button{ID: CrewPrefix + ":res:mem-req:up"}
	if got := h.focusField(v); got.key != "mem-req" {
		t.Fatalf("the hovered field is not in focus: %q", got.key)
	}
	for _, rf := range resFields {
		if rf.story == "" || rf.k8s == "" {
			t.Errorf("%s has no effect line", rf.key)
		}
	}
	if !strings.Contains(resFields[3].story, "Voidtaken and rebuilt") || !strings.Contains(resFields[3].k8s, "OOMKilled") {
		t.Errorf("the memory binding does not say what it does: %q", resFields[3].story)
	}
}

func TestTheResourcesDialogCountsTheRoomLeft(t *testing.T) {
	keeps := []view.KeepView{
		{Name: "a", Ready: true, Allocatable: view.ResourcePair{CPU: 2000, Memory: 4 << 30}, Requested: view.ResourcePair{CPU: 1000, Memory: 1 << 30}},
		{Name: "b", Ready: true, Allocatable: view.ResourcePair{CPU: 4000, Memory: 8 << 30}, Requested: view.ResourcePair{CPU: 3500, Memory: 0}},
	}
	if got := pledgeRoom(keeps, model.Resources{CPURequest: 250, MemRequest: 64}); got != 6 {
		t.Fatalf("room for 250m/64Mi = %d, want 4 on a and 2 on b", got)
	}
	if got := pledgeRoom(keeps, model.Resources{}); got != -1 {
		t.Fatalf("with no pledge the room is unbounded, got %d", got)
	}
}
