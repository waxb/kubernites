package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/termview"
)

func TestTheTerminalTakesControlChordsTextAndSpecialKeysButNoCmdChord(t *testing.T) {
	cases := []struct {
		in   Input
		keys []termview.Key
		text string
	}{
		{Input{Chars: []rune("ls -la")}, nil, "ls -la"},
		{Input{Keys: []ebiten.Key{ebiten.KeyEnter, ebiten.KeyEscape, ebiten.KeyF11}}, []termview.Key{{Name: "enter"}, {Name: "escape"}, {Name: "f11"}}, ""},
		{Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyC}, Chars: []rune("c")}, []termview.Key{{Rune: 'c', Ctrl: true}}, ""},
		{Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyP, ebiten.KeyQ}}, []termview.Key{{Rune: 'p', Ctrl: true}, {Rune: 'q', Ctrl: true}}, ""},
		{Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyBracketLeft, ebiten.KeySpace}}, []termview.Key{{Rune: '[', Ctrl: true}, {Rune: ' ', Ctrl: true}}, ""},
		{Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyArrowUp}}, []termview.Key{{Name: "up", Ctrl: true}}, ""},
		{Input{Alt: true, Chars: []rune("b")}, []termview.Key{{Rune: 'b', Alt: true}}, ""},
		{Input{Shift: true, Keys: []ebiten.Key{ebiten.KeyTab}}, []termview.Key{{Name: "tab", Shift: true}}, ""},
		{Input{Ctrl: true, Keys: []ebiten.Key{ebiten.KeyC, ebiten.KeyV}, Chars: []rune("cv")}, nil, ""},
		{Input{Keys: []ebiten.Key{ebiten.KeyA, ebiten.KeySpace}, Chars: []rune("a ")}, nil, "a "},
	}
	for i, c := range cases {
		keys, text := TerminalKeys(c.in)
		if text != c.text || len(keys) != len(c.keys) {
			t.Errorf("case %d: keys %+v text %q, want %+v %q", i, keys, text, c.keys, c.text)
			continue
		}
		for j := range keys {
			if keys[j] != c.keys[j] {
				t.Errorf("case %d: key %d is %+v, want %+v", i, j, keys[j], c.keys[j])
			}
		}
	}
}
