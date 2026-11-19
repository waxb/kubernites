package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
)

func requesting(qos string, cpu, mem int64) model.PodView {
	return model.PodView{Namespace: "default", Name: "porters-a", Node: "lone-tower", QoS: qos, State: model.HearthReady,
		Requests: []model.ContainerRequest{{Name: "porter", CPUMilli: cpu, MemoryBytes: mem}}}
}

func TestPileUnitsBucketTheRequests(t *testing.T) {
	cases := []struct {
		v    int64
		want int
	}{{0, 0}, {-5, 0}, {1, 1}, {100, 1}, {101, 2}, {250, 2}, {251, 3}, {500, 3}, {501, 4}, {4000, 4}}
	for _, c := range cases {
		if got := PileUnits(c.v, cpuPileSteps); got != c.want {
			t.Errorf("%dm cpu = %d units, want %d", c.v, got, c.want)
		}
	}
	if PileUnits(64*Mebi, memPileSteps) != 1 || PileUnits(65*Mebi, memPileSteps) != 2 || PileUnits(1<<30, memPileSteps) != 4 {
		t.Error("memory buckets at 64, 128 and 256 MiB")
	}
}

func TestHutPilesScaleWithTheSumOfItsContainersRequests(t *testing.T) {
	one := requesting(model.QoSBurstable, 100, 64*Mebi)
	if wood, stone := HutPiles(one); wood != 1 || stone != 1 {
		t.Errorf("a modest worker = %d wood %d stone", wood, stone)
	}
	two := one
	two.Requests = append(append([]model.ContainerRequest(nil), one.Requests...), model.ContainerRequest{Name: "scribe", CPUMilli: 200, MemoryBytes: 200 * Mebi})
	if wood, stone := HutPiles(two); wood != 3 || stone != 4 {
		t.Errorf("two people pile %d wood %d stone, want 3 and 4", wood, stone)
	}
	cpuOnly := requesting(model.QoSBurstable, 600, 0)
	if wood, stone := HutPiles(cpuOnly); wood != 4 || stone != 0 {
		t.Errorf("cpu only = %d wood %d stone", wood, stone)
	}
}

func TestBestEffortAndBanishedHutsPileNothing(t *testing.T) {
	if wood, stone := HutPiles(requesting(model.QoSBestEffort, 0, 0)); wood+stone != 0 {
		t.Error("a vagrant shack reserves nothing")
	}
	gone := requesting(model.QoSGuaranteed, 500, 256*Mebi)
	gone.State = model.HearthBanished
	if wood, stone := HutPiles(gone); wood+stone != 0 {
		t.Error("a banished hut no longer holds land")
	}
}

func TestHutsDrawTheirPilesAndBestEffortDrawsNone(t *testing.T) {
	lib := testLib()
	small, n := PileSprites(requesting(model.QoSBurstable, 100, 64*Mebi), 1)
	if n != 2 || small[0].Name != "pile/wood/1" || small[1].Name != "pile/stone/1" {
		t.Fatalf("small = %+v (%d)", small, n)
	}
	big, n := PileSprites(requesting(model.QoSGuaranteed, 1000, 512*Mebi), 2)
	if n != 2 || big[0].Name != "pile/wood/4" || big[1].Name != "pile/stone/4" || big[0].DX != 2*small[0].DX {
		t.Fatalf("big = %+v (%d)", big, n)
	}
	for _, s := range append(small[:], big[:]...) {
		if !lib.Has(s.Name) {
			t.Errorf("the library lacks %s", s.Name)
		}
	}
	if small[0].DX >= small[1].DX || small[0].DX >= 0 {
		t.Errorf("the woodpile sits left of the stone stack, both at the door: %+v", small)
	}
	if _, n := PileSprites(requesting(model.QoSBestEffort, 0, 0), 1); n != 0 {
		t.Errorf("best effort draws %d piles", n)
	}
	if _, n := PileSprites(requesting(model.QoSBurstable, 100, 64*Mebi), familiarZoom/2); n != 0 {
		t.Errorf("far away the piles fade out, drew %d", n)
	}
	r := NewRenderer(lib)
	dst := ebiten.NewImage(120, 120)
	defer dst.Deallocate()
	r.drawRequestPiles(dst, requesting(model.QoSBurstable, 300, 100*Mebi), 60, 80, 1)
}
