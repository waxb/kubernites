package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func hornedCrewFrame() Frame {
	f := crewFrame(crewSel(), "soulvault")
	f.State.Unlocks[HornUnlock] = true
	chain := crewChain()
	chain.Horn = &model.Horn{Name: "porters", Target: "porters", Min: 2, Max: 4, Current: 4, Goal: 70, Utilization: 95, Sounding: true, Measured: true}
	f.Crews = []model.CrewChain{chain}
	return f
}

func TestASoundingHornCanBeRetunedFromTheCrewCard(t *testing.T) {
	h := New(nil, Backend{})
	f := hornedCrewFrame()
	h.Press(ChainButton, f)
	pressed(t, h, f, "crew:mode:horn")
	buttons := crewButtonMap(append(h.CrewButtons(f), h.CommandButtons(f)...))
	retune, ok := buttons["crew:horn:retune"]
	if !ok || !retune.Disabled || buttons["crew:horn:silence"].ID == "" || buttons["crew:horn:max:up"].ID == "" {
		t.Fatalf("a sounding horn offers RETUNE (waiting for a change), SILENCE and steppers: %v", buttons)
	}
	if pressed(t, h, f, "crew:horn:retune") != nil {
		t.Fatal("an unchanged RETUNE does nothing")
	}
	pressed(t, h, f, "crew:horn:max:up")
	pressed(t, h, f, "crew:horn:max:up")
	retune = crewButtonMap(h.CommandButtons(f))["crew:horn:retune"]
	if retune.Disabled || !strings.Contains(retune.Tip, `kubectl patch hpa porters -p '{"spec":{"minReplicas":2,"maxReplicas":6`) {
		t.Fatalf("retune = %+v", retune)
	}
	a, from := actionOf(pressed(t, h, f, "crew:horn:retune"))
	if a.Action != view.ActRetuneHorn || a.Crew.Max != 6 || a.Crew.Min != 2 || a.Crew.CPU != 70 || from != FromCrewCard {
		t.Fatalf("retune action = %+v from %q", a, from)
	}
}
