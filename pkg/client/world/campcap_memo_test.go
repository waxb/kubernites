package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
)

func TestPendingSpotAmongMatchesPendingSpot(t *testing.T) {
	sc, _ := sceneFor(t, .5)
	spots := campSpots(sc.Site)
	for i := -1; i < 3*maxCamps*len(campJitter); i++ {
		if got, want := pendingSpotAmong(sc.Site, spots, i), PendingSpot(sc.Site, i); got != want {
			t.Fatalf("spot %d = %v, want %v", i, got, want)
		}
	}
	if got, want := signpostSpotAmong(sc.Site, spots), SignpostSpot(sc.Site); got != want {
		t.Fatalf("signpost = %v, want %v", got, want)
	}
}

func BenchmarkPlanCampsForManyPods(b *testing.B) {
	sc, _ := sceneFor(b, .5)
	pods := campers(2000)
	for i := range pods {
		pods[i].State = model.HearthReady
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PlanCamps(sc.Site, pods, map[string]Vec{})
	}
}
