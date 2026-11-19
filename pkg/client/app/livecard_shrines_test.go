package app

import (
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

type builtLive struct {
	snap live.Snapshot
	w    *live.World
}

func (b builtLive) Snapshot() live.Snapshot { return b.snap }
func (b builtLive) World() *live.World      { return b.w }

func TestTheManagedLiveCardCountsEveryPodAndTheHutsAtShrines(t *testing.T) {
	snap := livetest.Managed()
	w, err := live.Build(snap)
	if err != nil {
		t.Fatal(err)
	}
	shrines := 0
	for _, h := range w.Huts {
		if h.Shrine {
			shrines++
		}
	}
	card := LiveCardOf(builtLive{snap: snap, w: w})
	if card.Huts != len(snap.Pods) || shrines == 0 || card.Shrines != shrines {
		t.Fatalf("card %d huts, %d at shrines; the cluster has %d Pods, %d of them drawn at shrines", card.Huts, card.Shrines, len(snap.Pods), shrines)
	}
}
