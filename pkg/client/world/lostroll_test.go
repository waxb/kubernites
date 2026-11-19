package world

import "testing"

func TestAFallenHutStillOnTheRollPuffsAtItsEmptyPlot(t *testing.T) {
	l := &living{bare: map[string]Vec{"porter-a": {X: 4, Y: 7}, "porter-b": {X: 6, Y: 7}}}
	live := map[string]bool{"porter-b": true}
	l.keepFallen(Scene{Fallen: []string{"porter-a"}}, live)
	if at, ok := l.fallen["porter-a"]; !ok || at != (Vec{X: 4, Y: 7}) {
		t.Fatalf("the fallen hut's plot was not kept for its lost boxes: %+v", l.fallen)
	}
	if _, ok := l.fallen["porter-b"]; ok {
		t.Fatal("a standing hut puffs as fallen")
	}
	l.keepFallen(Scene{}, live)
	if len(l.fallen) != 0 {
		t.Fatalf("a struck hut still puffs: %+v", l.fallen)
	}
	shown := 0
	for step := range 40 {
		if _, ok := FallenPuff(float64(step)*.05, 7, 0); ok {
			shown++
		}
	}
	if shown == 0 || shown == 40 {
		t.Fatalf("the puff does not come and go over two seconds: %d of 40 frames", shown)
	}
}
