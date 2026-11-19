package hud

import "testing"

func TestThePurseTabAndTheStepPanelSitApart(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	h.land(1)
	tab, _, ok := h.PurseTab()
	panel := h.ObjectiveRectOf(f)
	if !ok || Overlaps(tab, panel) {
		t.Errorf("the counter tab %+v and the NEXT STEP panel %+v overlap (debut-walk, land-cairn: the \"+32\" over the panel's border)", tab, panel)
	}
	if panel.X != Objective.X || panel.Y != TopH+6+purseTabH {
		t.Errorf("the NEXT STEP panel stands at %d,%d; the player asked it top-left, down by the counter tab's height", panel.X, panel.Y)
	}
}

func TestThePurseTabIsDrawnAboveTheStepPanel(t *testing.T) {
	at := map[string]int{}
	for i, l := range HudLayers {
		at[l.Name] = i
	}
	if at["purse"] < at["windows"] {
		t.Errorf("the counter tab's layer %d is drawn under the windows' %d (debut-walk: the \"+1\" under the amount cut off by the NEXT STEP panel's top edge)", at["purse"], at["windows"])
	}
}
