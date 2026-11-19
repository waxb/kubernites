package hud

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheDigitsPressTheCellsInReadingOrder(t *testing.T) {
	want := []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9}
	if !slices.Equal(gridKeyCells, want) || gridKeyNames != "123456789" {
		t.Fatalf("grid keys = %v %q", gridKeyCells, gridKeyNames)
	}
}

func TestEachDigitPressesItsOwnCell(t *testing.T) {
	f := selectStructure(staged(fullFrame(t)), stoneKeep())
	for cell, key := range gridKeyCells {
		h := New(nil, (&fakeRunner{}).backend())
		b, ok := cellButton(h.CommandButtons(f), cell)
		if !ok {
			continue
		}
		want := h.pressCell(cell, f)
		got := New(nil, (&fakeRunner{}).backend()).Update(Input{Keys: []ebiten.Key{key}}, f)
		if len(got) != len(want) {
			t.Errorf("key %v (cell %d, %s): intents %v, want %v", key, cell, b.ID, got, want)
		}
	}
}

func TestTheOldGridLettersDoNothingOnAnObjectGrid(t *testing.T) {
	f := selectStructure(staged(fullFrame(t)), stoneKeep())
	for _, key := range []ebiten.Key{ebiten.KeyQ, ebiten.KeyW, ebiten.KeyE, ebiten.KeyA, ebiten.KeyS, ebiten.KeyD, ebiten.KeyZ, ebiten.KeyX} {
		runner := &fakeRunner{}
		h := New(nil, runner.backend())
		intents := h.Update(Input{Keys: []ebiten.Key{key}}, f)
		if len(runner.commands) != 0 || len(intents) != 0 || h.Overlay() != OverlayNone || len(h.Toasts()) != 0 {
			t.Errorf("%v still acts on the grid: %v %v", key, runner.commands, intents)
		}
	}
}

func TestTheDigitsDoNothingOnTheHomeGrid(t *testing.T) {
	f := fullFrame(t)
	for _, key := range gridKeyCells {
		runner := &fakeRunner{}
		h := New(nil, runner.backend())
		intents := h.Update(Input{Keys: []ebiten.Key{key}}, f)
		if len(runner.commands) != 0 || len(intents) != 0 || h.Overlay() != OverlayNone || h.Page() != PageMain {
			t.Errorf("%v acts on the home grid: %v %v", key, runner.commands, intents)
		}
	}
}
