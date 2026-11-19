package hud

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func postmortem() view.ChronicleView {
	return view.ChronicleView{ID: "c-9", Kind: "crashloop", Calamity: "The guttering hut", K8s: "CrashLoopBackOff",
		Question: "Looking back: what was the root cause?", Incident: "inc-1",
		Options:  []view.ChronicleOption{{ID: "a", Text: "Its own logs end in a crash line"}, {ID: "b", Text: "The Stone lists it"}, {ID: "c", Text: "Its events say Back-off"}},
		Timeline: &view.IncidentTimeline{StruckAt: 100, CauseAt: 170, FixedAt: 255, BoxesLost: 34}}
}

func TestAPostmortemReadsItsOwnTimeline(t *testing.T) {
	rows := timelineRows(postmortem())
	icons, texts := []string{}, []string{}
	for _, r := range rows {
		icons, texts = append(icons, r.icon), append(texts, r.text)
	}
	if !slices.Equal(icons, []string{art.TimelineIconName("struck"), art.TimelineIconName("cause-found"), art.TimelineIconName("fixed"), art.TimelineIconName("boxes-lost")}) {
		t.Fatalf("icons = %v", icons)
	}
	if !slices.Equal(texts, []string{"STRUCK", "CAUSE FOUND 1:10 LATER", "MENDED 2:35 AFTER IT STRUCK", "34 BOXES LOST"}) {
		t.Fatalf("texts = %v", texts)
	}
	unfound := postmortem()
	unfound.Timeline.CauseAt = 0
	if got := timelineRows(unfound)[1].text; got != "CAUSE NEVER FOUND" {
		t.Fatalf("an unfound cause reads %q", got)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if got := timelineRows(postmortem())[2].text; got != "resolved 2:35 after detection" {
		t.Fatalf("NAMES fixed row = %q", got)
	}
	if len(timelineRows(view.ChronicleView{})) != 0 {
		t.Fatal("an old chronicle has a timeline")
	}
}

func TestThePostmortemTimelineSitsBesideASmallerPicture(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	l := h.ChronicleLayout(postmortem())
	if l.Scale != 1 || l.Timeline.W == 0 || l.Timeline.X < l.Picture.X+l.Picture.W || !contains(Overlay, l.Timeline) {
		t.Fatalf("layout = %+v", l)
	}
	if l.Timeline.H < 4*clueRowH {
		t.Fatalf("the timeline is %d high for four rows", l.Timeline.H)
	}
	plain := h.ChronicleLayout(view.ChronicleView{ID: "c-1", Kind: "imagepull", Question: "Why?"})
	if plain.Scale != chronVigS || plain.Timeline.W != 0 {
		t.Fatalf("an old chronicle's layout changed: %+v", plain)
	}
	f := fullFrame(t)
	f.State.Chronicles = []view.ChronicleView{postmortem()}
	h.OverlayID = OverlayChronicles
	h.Draw(ebiten.NewImage(Width, Height), f)
}
