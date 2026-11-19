package world

import (
	"image/color"
	"testing"
)

func TestTheNightOnlyDarkensAtEveryHour(t *testing.T) {
	for i := 0; i <= 200; i++ {
		phase := float64(i) / 200
		c := nightFill(phase)
		if c.R > c.A || c.G > c.A || c.B > c.A {
			clock := phase * 24
			t.Fatalf("at %02d:%02d the night pass fills %v: a colour brighter than its alpha adds light, so a realm opened at dawn starts pale and foggy until the night pass stops", int(clock), int(clock*60)%60, c)
		}
	}
	if c := nightFill(.5); c.A != 0 {
		t.Errorf("noon is dimmed by %v", c)
	}
}

func TestDeepNightKeepsItsColour(t *testing.T) {
	if c := nightFill(0); c != (color.RGBA{18, 24, 60, 140}) {
		t.Errorf("midnight fills %v, the night the captures show is {18 24 60 140}", c)
	}
}
