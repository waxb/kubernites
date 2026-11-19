package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestTheHeraldryViewIsAMapViewThatTheLiveGateLetsThrough(t *testing.T) {
	f := Frame{}
	f.State.Reveals = []string{"stone", "tidings", heraldryReveal}
	if !LensAvailable(LensHeraldry, f) {
		t.Fatal("the Heraldry view is not offered with its reveal")
	}
	for _, m := range []string{LensLabor, LensUnhoused, LensMidden, LensCodex} {
		if LensAvailable(m, f) {
			t.Errorf("%s is offered without the overlays", m)
		}
	}
	if got := NextLens(LensNone, f); got != LensHeraldry {
		t.Errorf("O from no view goes to %q, want the Heraldry view", got)
	}
	if got := NextLens(LensHeraldry, f); got != LensNone {
		t.Errorf("O from the Heraldry view goes to %q, want no view", got)
	}
	f.State.Reveals = []string{"stone", "tidings"}
	if LensAvailable(LensHeraldry, f) {
		t.Error("the Heraldry view is offered before its reveal")
	}
	for _, id := range []string{LensCycleID, lensPrefix + ":" + LensHeraldry} {
		if !LiveReadButton(id) {
			t.Errorf("%s is refused in a live realm", id)
		}
	}
}

func TestTheHeraldryViewSpeaksOnlyStoryWordsInStoryMode(t *testing.T) {
	info := lensInfos[LensHeraldry]
	f := Frame{}
	f.State.Reveals = []string{heraldryReveal}
	for _, text := range []string{info.label, info.legend, legendText(LensHeraldry), lensCycleTip(f)} {
		if leak := lexicon.KubernetesWords.FindString(text); leak != "" {
			t.Errorf("%q names %q", text, leak)
		}
	}
	if got := lexicon.TrueNames(legendText(LensHeraldry)); got != "MAP: NAMESPACES   O next, Esc clears" {
		t.Errorf("with Kubernetes names the legend reads %q", got)
	}
}

func TestTheCampaignShowsTheHeraldryButtonWithItsCalmCardAndNotBefore(t *testing.T) {
	f := Frame{}
	f.State.Reveals = []string{"minimap"}
	f.Guide = Guide{Staged: true, Introduced: []string{SurfaceMinimap}}
	if LensButtonShown(f) || LensAvailable(LensHeraldry, f) {
		t.Fatal("the map views showed with one fief")
	}
	f.State.Reveals = append(f.State.Reveals, heraldryReveal)
	if LensButtonShown(f) {
		t.Fatal("the Heraldry button showed before its card")
	}
	f.Guide.Intro = &Intro{ID: "heraldry", Target: SurfaceHeraldry, Via: ViaAck}
	if !LensButtonShown(f) || !LensAvailable(LensHeraldry, f) {
		t.Fatal("the Heraldry card points at a button that is not there")
	}
	if rect, ok := surfaceRects[SurfaceHeraldry]; !ok || rect(&HUD{}, f) != LensButtonRect() {
		t.Fatal("the Heraldry card does not point at the map button")
	}
	f.Guide.Intro, f.Guide.Introduced = nil, []string{SurfaceMinimap, SurfaceHeraldry}
	if !LensButtonShown(f) {
		t.Fatal("the Heraldry button went once its card was read")
	}
}
