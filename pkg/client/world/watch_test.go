package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func watchScene() *view.WatchView {
	return &view.WatchView{
		Crews: []view.CrewWatchView{
			{Crew: "porters", Readiness: &view.ProbeSetting{Path: "/ready"}, Liveness: &view.ProbeSetting{Path: "/healthz", Threshold: 3}},
			{Crew: "clerks"},
		},
		Huts: []view.HutWatchView{
			{Pod: "porters-a", Crew: "porters", Warming: true, Shut: true},
			{Pod: "porters-b", Crew: "porters", Knocks: 2},
			{Pod: "clerks-c", Crew: "clerks", Warming: true},
		},
	}
}

func spriteNames(list []incidentSprite) map[string]bool {
	out := map[string]bool{}
	for _, s := range list {
		out[s.name] = true
	}
	return out
}

func TestTheWatchShowsShuttersBellHourglassAndKnocks(t *testing.T) {
	r := NewRenderer(testLib())
	w := indexWatch(watchScene())
	pod := func(name, qos string) model.PodView { return model.PodView{Name: name, QoS: qos} }
	warming := spriteNames(r.watchSprites(pod("porters-a", "Burstable"), w, 0, 100, 100, 1, 2.0, false))
	look := model.QoSLook("Burstable")
	if !warming[art.ShutterName(look, 0, false)] || !warming[art.BellName(look, 0, 0)] || !warming[art.HourglassName(IncidentFrame(2.0, false))] {
		t.Fatalf("a warming hut with both probes draws %v", warming)
	}
	knocked := spriteNames(r.watchSprites(pod("porters-b", "Burstable"), w, 0, 100, 100, 1, 2.0, false))
	if !knocked[art.ShutterName(look, 0, true)] || !knocked[art.KnockName(2)] {
		t.Fatalf("a ready hut being knocked draws %v", knocked)
	}
	bare := spriteNames(r.watchSprites(pod("clerks-c", ""), w, 0, 100, 100, 1, 2.0, false))
	if len(bare) != 1 || !bare[art.HourglassName(IncidentFrame(2.0, false))] {
		t.Fatalf("a crew without probes shows only its warming: %v", bare)
	}
	for name := range warming {
		if !r.Lib.Has(name) {
			t.Errorf("%s is not in the atlas", name)
		}
	}
	if got := r.watchSprites(pod("nobody", ""), w, 0, 100, 100, 1, 2.0, false); len(got) != 0 {
		t.Fatalf("an unwatched hut draws %v", got)
	}
}

func TestAFrozenHutWithoutABellShowsFrost(t *testing.T) {
	r := NewRenderer(testLib())
	w := indexWatch(&view.WatchView{
		Crews: []view.CrewWatchView{{Crew: "porters"}, {Crew: "clerks", Liveness: &view.ProbeSetting{Path: "/healthz", Threshold: 3}}},
		Huts:  []view.HutWatchView{{Pod: "porters-a", Crew: "porters", Frozen: true}, {Pod: "clerks-b", Crew: "clerks", Frozen: true}},
	})
	frozen := spriteNames(r.watchSprites(model.PodView{Name: "porters-a"}, w, 0, 100, 100, 1, 2.0, false))
	if len(frozen) != 1 || !frozen[art.FrostName()] || !r.Lib.Has(art.FrostName()) {
		t.Fatalf("a frozen porter with no bell draws %v: the freeze must be visible before its card", frozen)
	}
	if belled := spriteNames(r.watchSprites(model.PodView{Name: "clerks-b"}, w, 0, 100, 100, 1, 2.0, false)); belled[art.FrostName()] {
		t.Fatalf("a belled hut's freeze is the watchman's to handle, it draws %v", belled)
	}
}
