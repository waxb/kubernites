package world

import (
	"sort"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

var reportedPersons = map[string]bool{"reeve": true, "helmbearer": true, "builder": true, foremanPerson: true, "clerk": true, "imp": true,
	debutPerson: true, fallenPerson: true, "quartermaster": true, "steward": true, "scribe": true, "archivist": true}

func personsOf(t *testing.T, items []Drawable, seen map[string]int) {
	t.Helper()
	for _, it := range items {
		if it.person == "" {
			continue
		}
		seen[it.person]++
		switch {
		case it.person == personWorker && (it.Hit == nil || it.Hit.Kind != HitWalker):
			seen["worker without a click"]++
		case it.person != personWorker && !reportedPersons[it.person]:
			seen["townsfolk:"+it.person]++
		}
	}
}

func TestEveryPersonInTheWorldIsAWorkerThatAnswersAClick(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	seen := map[string]int{}
	sc, cam := sceneFor(t, .5)
	sc.Pods = append(sc.Pods, model.PodView{Namespace: "default", Name: "waiting", Sprite: "porter", Tier: 1, State: model.HearthPending})
	sc.Metrics.Nodes = map[string]view.NodeMetrics{"lone-tower": {CPUMilli: 1500, CPUAllocatableMilli: 1900, MemoryBytes: 3 << 30, MemoryAllocatableBytes: 4 << 30}}
	if k, ok := FirstOfKind(sc.Site, view.StructKeep, false); ok {
		cam = cam.CenterOn(float64(k.X), float64(k.Y), 960, 540)
	}
	r := NewRenderer(testLib())
	for i := 0; i < 40; i++ {
		sc.Time += 3
		sc.Served += 30
		r.Draw(dst, cam, sc)
		personsOf(t, r.sceneItems(cam, sc), seen)
	}
	hsc, hcam, hr := bustlingHarbor(t)
	for i := 0; i < 200; i++ {
		hsc.Time += .25
		hsc.RealmTime += .25
		hsc.Freight = synthFreight(hsc.RealmTime, 2, 6)
		hr.Draw(dst, hcam, hsc)
		personsOf(t, hr.sceneItems(hcam, hsc), seen)
	}
	bad := []string{}
	for k, n := range seen {
		if k == "worker without a click" || len(k) > 10 && k[:10] == "townsfolk:" {
			bad = append(bad, k)
			_ = n
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Errorf("people drawn that are no worker answering a click: %v (all persons seen: %v)", bad, seen)
	}
	if seen[personWorker] == 0 {
		t.Error("no worker was drawn at all")
	}
}

func TestAWorkerOnAFerryAnswersAClick(t *testing.T) {
	r := NewRenderer(testLib())
	f := Figure{Pod: model.PodView{Namespace: "default", Name: "porters-1", Sprite: "porter", State: model.HearthReady}, role: "porter", Pose: Pose{Dir: "se"}}
	d := r.riderItem(f, 200, 200, 1, 0)
	if d.person != personWorker || d.Hit == nil || d.Hit.Kind != HitWalker || d.Hit.ID != "default/porters-1" || d.Hit.Rect.W == 0 {
		t.Fatalf("a worker riding a ferry answers no click: %+v", d.Hit)
	}
}
