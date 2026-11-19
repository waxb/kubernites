package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func fadedFrame(t *testing.T, mode string, fix *view.CommandOffer) Frame {
	t.Helper()
	f := flowFrame(t)
	f.State.Economy.FixMode, f.State.Economy.Fix = mode, fix
	return f
}

func keepFix() *view.CommandOffer {
	return &view.CommandOffer{ID: "buy-keep:porters", Label: "Add a Keep for more porters",
		Action: view.Action{Action: "buyKeep", KeepType: view.KeepTypeStone, For: "porters"}}
}

func TestTheHarborFixActsInItsFirstStage(t *testing.T) {
	h := New(nil, Backend{})
	f := fadedFrame(t, view.FixActs, scaleFix())
	if b, ok := HitID(h.Buttons(f), FlowbarFix); !ok || b.Label != "+2 PORTERS" {
		t.Fatalf("the acting fix = %+v %v", b, ok)
	}
	it, ok := hasIntent(clickOn(t, h, f, FlowbarFix), IntentAction)
	if !ok || it.Action.Action != view.ActScaleCrew || it.From != FromHarborBoard || !FixSource(it.From) {
		t.Fatalf("the fix acts = %+v %v", it, ok)
	}
}

func TestTheHarborFixShowsTheCrewInItsSecondStage(t *testing.T) {
	for _, c := range []struct {
		fix    *view.CommandOffer
		button string
		spot   string
	}{{scaleFix(), StepUpButton, GridActions[GridScale]}, {keepFix(), BuyKeepButton, ""}} {
		h := New(nil, Backend{})
		f := fadedFrame(t, view.FixShows, c.fix)
		b, ok := HitID(h.Buttons(f), FlowbarFix)
		if !ok || b.Label != "SHOW PORTERS" || !strings.Contains(b.Tip, "porters") {
			t.Fatalf("the SHOW fix = %+v %v", b, ok)
		}
		intents := clickOn(t, h, f, FlowbarFix)
		if _, acted := hasIntent(intents, IntentAction); acted {
			t.Fatalf("SHOW acted: %+v", intents)
		}
		it, ok := hasIntent(intents, IntentChoose)
		if !ok || it.Target != SelectCrew || it.Command != "default/porters" {
			t.Fatalf("SHOW selects %+v %v", it, ok)
		}
		f.Selection = Selection{Kind: SelectCrew, ID: "default/porters"}
		h.syncViews(f)
		if !h.ChainShown(f) || h.HintedButton(f) != c.button || h.HintedAction(f) != c.fix.Action.Action {
			t.Fatalf("SHOW opens the board %v and lights %q (%q), want %q", h.ChainShown(f), h.HintedButton(f), h.HintedAction(f), c.button)
		}
		if h.grid.spot != c.spot {
			t.Fatalf("SHOW spotlights the grid's %q, want %q", h.grid.spot, c.spot)
		}
		f.Clock += hintLife + 1
		if h.HintedButton(f) != "" {
			t.Fatal("the light never fades")
		}
	}
}

func TestTheHarborFixOnlyNamesTheBottleneckInItsLastStage(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, fix := range []*view.CommandOffer{scaleFix(), keepFix()} {
		h := New(nil, Backend{})
		f := fadedFrame(t, view.FixNames, fix)
		if b, ok := HitID(h.Buttons(f), FlowbarFix); ok {
			t.Fatalf("the last stage still offers %+v", b)
		}
		if got := NeckText(f); got != "HUTS (PORTERS)" {
			t.Errorf("the named bottleneck reads %q", got)
		}
		tip := NeckTip(EconomyOf(f.State))
		if strings.Contains(tip, "The button") || !strings.Contains(tip, "porters") || !strings.Contains(tip, "Go to the porters") {
			t.Errorf("the tip does not say where to go: %q", tip)
		}
	}
	ui.SetNamer(names.Kubernetes(64))
	show := fadedFrame(t, view.FixShows, scaleFix())
	if b, ok := HitID(New(nil, Backend{}).Buttons(show), FlowbarFix); !ok || (b.Label != "SHOW" && b.Label != "SHOW PORTERS") || strings.Contains(b.Label, "...") {
		t.Errorf("NAMES mode's SHOW reads %+v", b)
	}
	f := fadedFrame(t, view.FixNames, scaleFix())
	if got, tip := NeckText(f), NeckTip(EconomyOf(f.State)); got != "REPLICAS (PORTERS)" || !strings.Contains(tip, "Deployment porters") {
		t.Errorf("NAMES mode reads %q, tip %q", got, tip)
	}
}

func TestOnlyScaleAndKeepFixesFade(t *testing.T) {
	depot := &view.CommandOffer{ID: "fix:porters", Label: "Open the parcels depot", Action: view.Action{Action: "openDepot", Kind: "parcel"}}
	f := fadedFrame(t, view.FixNames, depot)
	if FixFade(f, *depot) != view.FixActs {
		t.Fatal("a depot fix faded: it already opens placement")
	}
	if _, ok := HitID(New(nil, Backend{}).Buttons(f), FlowbarFix); !ok {
		t.Fatal("the depot fix lost its button")
	}
}

func scaleCard() view.CalamityView {
	return view.CalamityView{ID: "spoiling:parcel", Kind: view.CalamitySpoiling, Object: "kind/parcel", Cause: "Parcels spoil faster than the porters carry them",
		Why: "Too few porters.", Remedies: []view.RemedyView{{Label: "Scale the porters to 6", Command: "kubectl scale deployment porters --replicas=6",
			Action: &view.Action{Action: view.ActScaleCrew, Name: "porters", Replicas: 6}}}}
}

func TestTheCardsScaleRemedyFadesToShowToo(t *testing.T) {
	for _, mode := range []string{view.FixActs, view.FixShows, view.FixNames} {
		h := New(nil, (&fakeRunner{}).backend())
		f := calamityFrame(t, scaleCard())
		f.State.Economy.FixMode = mode
		f = settled(h, f)
		b, ok := HitID(h.CalamityButtons(f), "line:cal:0:do")
		if !ok {
			t.Fatalf("%s: no remedy button", mode)
		}
		intents := clickOn(t, h, f, "line:cal:0:do")
		_, acted := hasIntent(intents, IntentAction)
		it, chose := hasIntent(intents, IntentChoose)
		switch {
		case mode == view.FixActs && (b.Label != doLabel || !acted):
			t.Errorf("%s: the card's remedy %q acted %v", mode, b.Label, acted)
		case mode != view.FixActs && (b.Label != showLabel || acted || !chose || it.Command != "default/porters"):
			t.Errorf("%s: the card's remedy %q acted %v, chose %+v", mode, b.Label, acted, it)
		}
	}
}

func TestTheLossLineStopsNamingTheNumberInTheLastStage(t *testing.T) {
	st := view.GameState{Calamities: []view.CalamityView{scaleCard()}}
	cases := map[string]string{view.FixActs: "scale porters to 6", view.FixShows: "scale porters to 6", view.FixNames: "too few porters"}
	for mode, want := range cases {
		st.Economy.FixMode = mode
		if got := cardFix(st, view.CalamitySpoiling, "parcel", "carried too slowly"); got != want {
			t.Errorf("%s: the loss line reads %q, want %q", mode, got, want)
		}
	}
}

func TestAButtonThatOpensPlacementSaysSo(t *testing.T) {
	h := New(nil, Backend{})
	f := fadedFrame(t, view.FixActs, keepFix())
	b, ok := HitID(h.Buttons(f), FlowbarFix)
	if !ok || strings.Contains(b.Tip, ActsNote) || !strings.Contains(b.Tip, placeNote) {
		t.Fatalf("the Keep fix's tip %q", b.Tip)
	}
	line := CommandLine{Label: "Add a Keep for more porters", Action: &keepFix().Action}
	if tip := doTip(line); !strings.Contains(tip, placeNote) || strings.Contains(tip, "does the same") {
		t.Fatalf("the card's Add a Keep tip %q", tip)
	}
	if tip := doTip(CommandLine{Label: "Scale", Action: &scaleFix().Action}); strings.Contains(tip, placeNote) {
		t.Fatalf("a scale opens no placement: %q", tip)
	}
}

func TestTheScaleAdviceSendsThePlayerToTheCrewNotToAPod(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, mode := range namesModes() {
		mode.apply()
		tip := whereToGo(scaleFix().Action)
		if strings.Contains(tip, "one of its Pods and scale") || strings.Contains(tip, "one of their huts and raise") {
			t.Errorf("[%s] scaling belongs to the crew: %q", mode.name, tip)
		}
	}
	ui.SetNamer(names.Kubernetes(64))
	if tip := whereToGo(scaleFix().Action); !strings.Contains(tip, "select the Deployment") || !strings.Contains(tip, "kubectl scale deployment porters") {
		t.Errorf("NAMES advice = %q", tip)
	}
}
