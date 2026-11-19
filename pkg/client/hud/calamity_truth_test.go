package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func troubleUnderIntro(t *testing.T, list ...view.CalamityView) Frame {
	t.Helper()
	f := calamityFrame(t, list...)
	f.Guide.Intro = &Intro{ID: "academy", Title: "The registry", Purpose: "Newer images for your crews.", Try: "Press REGISTRY to see its images.", Target: "cmd:soulvault", Via: ViaAck}
	return f
}

func TestALiveTroubleShowsItsCardWithinTenSecondsEvenUnderAnIntroCard(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := troubleUnderIntro(t, noDepot)
	shown := false
	for range 11 {
		h.Update(Input{MX: -5, MY: -5}, f)
		if _, ok := h.ShownCalamity(f); ok {
			shown = true
			break
		}
		f.Clock++
	}
	if !shown {
		t.Fatal("a live trouble waited behind an intro card for more than 10 s")
	}
	card, _ := h.CalamityRect(f)
	if h.IntroShown(f) && Overlaps(card, h.IntroCard(f)) {
		t.Fatalf("the trouble card %+v covers the intro card %+v", card, h.IntroCard(f))
	}
}

func TestASecondTroubleJoinsTheOpenCardWithoutWaitingAQuietGap(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, calamityFrame(t, noDepot))
	f.State.Calamities = []view.CalamityView{noDepot, oom}
	for range 10 {
		f.Clock++
		h.Update(Input{MX: -5, MY: -5}, f)
	}
	if _, _, count, ok := h.CurrentCalamity(f); !ok || count != 2 {
		t.Fatalf("a second trouble live for 10 s has no card: %d cards", count)
	}
}

func TestAnOpenCardFollowsTheLiveRealmAndNeverNamesAHutThatIsGone(t *testing.T) {
	hungry := view.CalamityView{ID: "throttled:lone-tower", Kind: view.TagThrottled, Object: "node/lone-tower",
		Cause:    "Hungry: lone-tower has too little food for every worker",
		Remedies: []view.RemedyView{{Label: "Raze a hut on lone-tower", Command: "kubectl delete pod porters-59fcdcb77f-2882g"}}}
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, calamityFrame(t, hungry))
	moved := hungry
	moved.Remedies = []view.RemedyView{{Label: "Raze a hut on lone-tower", Command: "kubectl delete pod porters-59fcdcb77f-5mm9j"}}
	f.State.Calamities = []view.CalamityView{moved}
	f.Clock += 10
	h.Update(Input{MX: -5, MY: -5}, f)
	for _, l := range h.calamityLines(f) {
		if strings.Contains(l.Command, "2882g") {
			t.Fatalf("10 s after the realm moved on, the card still offers to raze a hut that is gone: %+v", l)
		}
	}
}

func TestAResolvedTroubleShowsMendedThenLeaves(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, calamityFrame(t, noDepot))
	mended := noDepot
	mended.Remedies, mended.Mended = []view.RemedyView{}, "Your fix held: the depot stands open."
	f.State.Calamities = []view.CalamityView{mended}
	f.Clock++
	h.Update(Input{MX: -5, MY: -5}, f)
	v, ok := h.ShownCalamity(f)
	if !ok || v.Mended == "" {
		t.Fatalf("a mended trouble does not say so: %+v %v", v, ok)
	}
	cause, _ := h.CalamityText(f, v)
	if !strings.Contains(strings.Join(cause, " "), "fix held") {
		t.Fatalf("the card does not read Mended: %q", cause)
	}
	if _, found := HitID(h.CalamityButtons(f), CalamityOK); !found {
		t.Fatal("a mended card has no OK")
	}
	f.State.Calamities = nil
	f.Clock++
	h.Update(Input{MX: -5, MY: -5}, f)
	if v, ok := h.ShownCalamity(f); ok {
		t.Fatalf("a retired trouble keeps its card: %+v", v)
	}
}

func TestAMendedTroubleNeverOpensACardOfItsOwn(t *testing.T) {
	mended := noDepot
	mended.Remedies, mended.Mended = []view.RemedyView{}, "It passed: the depot stands open."
	h := New(nil, (&fakeRunner{}).backend())
	f := calamityFrame(t, mended)
	for range 10 {
		h.Update(Input{MX: -5, MY: -5}, f)
		f.Clock++
	}
	if v, ok := h.ShownCalamity(f); ok {
		t.Fatalf("a trouble that passed before its card opened shows only to say it passed: %+v", v)
	}
}
