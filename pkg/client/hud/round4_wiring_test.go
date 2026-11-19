package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAFirstStrikeCardShowsBeforeTheRailIsKnown(t *testing.T) {
	first := oom
	first.First = true
	for _, c := range []struct {
		card view.CalamityView
		want bool
	}{{oom, false}, {first, true}} {
		h := New(nil, (&fakeRunner{}).backend())
		f := calamityFrame(t, c.card)
		f.Guide = Guide{Staged: true, Introduced: []string{SurfaceTidings}}
		f = settled(h, f)
		if _, shown := h.ShownCalamity(f); shown != c.want {
			t.Errorf("first=%v: shown %v before the rail, want %v", c.card.First, shown, c.want)
		}
	}
}

func TestOpeningTheAgePanelReadsTheNewAge(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.State.AgeMoment = &view.AgeMomentView{Age: view.AgeFeudal}
	intents := h.Press("cmd:age", f)
	read := false
	for _, in := range intents {
		read = read || in.Kind == IntentAction && in.Action.Action == view.ActReadAge
	}
	if h.Overlay() != OverlayAge || !read {
		t.Fatalf("the panel opened %q and sent readAge %v", h.Overlay(), read)
	}
	h = New(nil, (&fakeRunner{}).backend())
	f.State.AgeMoment.Read = true
	for _, in := range h.Press("cmd:age", f) {
		if in.Kind == IntentAction {
			t.Fatalf("a read tier is read again: %+v", in)
		}
	}
}

func TestClosingTheAgePanelTheAdvanceOpenedReadsTheNewAge(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.State.Age = view.AgeFeudal
	f.State.AgeMoment = &view.AgeMomentView{Age: view.AgeFeudal}
	h.Open(OverlayAge)
	if sentReadAge(h.Update(Input{MX: -10, MY: -10}, f)) {
		t.Fatal("the new tier was read before its panel closed")
	}
	h.Open(OverlayNone)
	if !sentReadAge(h.Update(Input{MX: -10, MY: -10}, f)) {
		t.Fatal("closing the tier panel the advance opened did not read the new tier")
	}
	if sentReadAge(h.Update(Input{MX: -10, MY: -10}, f)) {
		t.Fatal("the new tier was read twice")
	}
}

func sentReadAge(intents []Intent) bool {
	for _, in := range intents {
		if in.Kind == IntentAction && in.Action.Action == view.ActReadAge {
			return true
		}
	}
	return false
}

func TestAnOpenAdvanceNeverSaysMeetEveryRequirementFirst(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.State.AgeUp = view.AgeUpView{NextAge: view.AgeFeudal, Available: true}
	f.State.Reforge.Available = false
	h.Open(OverlayAge)
	b, ok := HitID(h.ageButtons(f), "age:advance")
	if !ok || b.Disabled || strings.Contains(b.Tip, "Meet every requirement") {
		t.Fatalf("an enabled ADVANCE reads %+v", b)
	}
}
