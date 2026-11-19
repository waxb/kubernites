package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestFlagsDoNotWaveInLockstep(t *testing.T) {
	frames := map[int]int{}
	for i := range 40 {
		s := view.Structure{ID: "k", X: 10 + i*7, Y: 20 + (i*13)%31}
		frames[FlagFrame(12.5, s, 0)]++
	}
	if len(frames) < 4 {
		t.Fatalf("40 flags at one moment show only %d frames: %v", len(frames), frames)
	}
	s := view.Structure{X: 30, Y: 40}
	if FlagFrame(3.2, s, 0) != FlagFrame(3.2, s, 0) {
		t.Fatal("a flag's frame is not deterministic")
	}
	if FlagFrame(3.2, s, 0) == FlagFrame(3.2, s, 1) && FlagFrame(5.1, s, 0) == FlagFrame(5.1, s, 1) && FlagFrame(7.7, s, 0) == FlagFrame(7.7, s, 1) {
		t.Fatal("two flags on one building wave together")
	}
}

func TestAFlagWavesAndTheGustChangesItsPace(t *testing.T) {
	s := view.Structure{X: 12, Y: 9}
	seen := map[int]bool{}
	changes := 0
	prev := FlagFrame(0, s, 0)
	for step := 1; step <= 1200; step++ {
		f := FlagFrame(float64(step)*0.05, s, 0)
		seen[f] = true
		if f != prev {
			changes++
		}
		prev = f
	}
	if len(seen) < 6 || changes < 200 {
		t.Fatalf("a flag shows %d frames and changes %d times in a minute", len(seen), changes)
	}
	early, late := paceBetween(s, 0, 20), paceBetween(s, 20, 40)
	if early == late {
		t.Fatalf("the wind never changes pace: %d then %d frame changes", early, late)
	}
}

func paceBetween(s view.Structure, from, to float64) int {
	changes, prev := 0, FlagFrame(from, s, 0)
	for t := from; t < to; t += 0.02 {
		if f := FlagFrame(t, s, 0); f != prev {
			changes++
			prev = f
		}
	}
	return changes
}
