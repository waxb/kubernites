package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestFanWatchShowsLinesOnlyAfterTheFirstAnswer(t *testing.T) {
	var w fanWatch
	w.observe(0, map[string]bool{"a": true})
	if w.visible("a", 1) {
		t.Error("a realm loaded mid-flow shows no lines")
	}
	var fresh fanWatch
	fresh.observe(0, nil)
	fresh.observe(5, map[string]bool{"a": true, "b": true})
	if !fresh.visible("a", 6) || !fresh.visible("b", 12) || fresh.visible("a", 5+fanShow+.1) {
		t.Error("the first answer draws every path for a while")
	}
	fresh.observe(30, map[string]bool{"a": true, "b": true, "c": true})
	if !fresh.visible("c", 31) || fresh.visible("a", 31) || fresh.visible("c", 30+fanFlash+.1) {
		t.Error("a newly enlisted hut flashes its own path")
	}
	if fresh.alpha("c", 30) <= fresh.alpha("c", 30+fanFlash*.9) {
		t.Error("paths fade out")
	}
}

func TestFanTargetsAreReadyDeclaredHearths(t *testing.T) {
	site := lifeSite()
	r := NewRenderer(nil)
	r.Homes = map[string]Vec{"a": {10, 10}, "b": {12, 10}, "door": {14, 10}}
	sc := Scene{Site: site, Pods: []model.PodView{
		{Namespace: "default", Name: "a", State: model.HearthReady},
		{Namespace: "default", Name: "b", State: model.HearthGuttering},
		{Namespace: "default", Name: "door", State: model.HearthReady, Static: true},
	}, Metrics: view.MetricsSnapshot{Pods: map[string]view.PodMetrics{"default/a": {ServedRate: .5}}}}
	fan := r.fanTargets(sc)
	if len(fan) != 1 || fan[0].name != "a" || fan[0].weight != .5 {
		t.Errorf("fan = %+v", fan)
	}
	site.Structures = site.Structures[:1]
	sc.Site = site
	if len(r.fanTargets(sc)) != 0 {
		t.Error("no Guildhall, no fan-out")
	}
}
