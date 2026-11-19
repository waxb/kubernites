package hud

import "testing"

func TestGuideCardsStayOffTheCrewWindowsControls(t *testing.T) {
	for _, target := range []string{"cmd:build", "sel:close", MeterGold, SurfaceMinimap} {
		h := New(nil, Backend{})
		f := stagedFrame(t, "cmd:serve", "cmd:build", MeterGold)
		f.Selection = crewSel()
		f.Crews = crewFrame(crewSel()).Crews
		f.State.Reveals = append(f.State.Reveals, "crew")
		f.Guide.Intro = &Intro{ID: "keep", Title: "Raise a Stone Keep", Purpose: "A second Keep for huts once the Tower is spoken for", Try: "Press BUILD.", Target: target, Via: ViaAct}
		if len(h.crewControls(f)) == 0 {
			t.Fatalf("%s: the fixture opens no crew window: it proves nothing", target)
		}
		card := h.IntroCard(f)
		if got := blockedArea(card, h.crewControls(f)); got != 0 {
			t.Errorf("%s: the intro card %+v covers %d px of the crew window %+v", target, card, got, h.crewControls(f))
		}
	}
}

func TestTheFoundationLineCountsNinesInTheSingular(t *testing.T) {
	h := New(nil, Backend{})
	f := Frame{}
	f.State.SLO.Nines = 1
	if got := h.FoundationLines(f); len(got) != 2 || got[1] != "Held now: 1 nine (0.0% delivered fresh)." && got[1] != "Held now: 1 nine ("+Nines(0)+" delivered fresh)." {
		t.Fatalf("lines %q", got)
	}
	f.State.SLO.Nines = 2
	if got := h.FoundationLines(f); got[1][:19] != "Held now: 2 nines (" {
		t.Fatalf("lines %q", got)
	}
}
