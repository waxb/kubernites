package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/termview"
)

var nonLatinLayouts = map[string]string{
	"Greek":  ";ςερτυθιοπ[]ασδφγηξκλ΄'ζχψωβνμ,./-6\\",
	"Hebrew": "/'קראטוןםפ][שדגכעיחלךף,זסבהנמצתץ.-6\\",
}

func TestTheLeaveChordIsTypableOnLayoutsWithoutLatinLetters(t *testing.T) {
	for name, labels := range nonLatinLayouts {
		layout := layoutTable(labels)
		var got []termview.Key
		for _, k := range []ebiten.Key{ebiten.KeyP, ebiten.KeyQ} {
			keys, _ := TerminalKeys(Input{Ctrl: true, Control: true, Keys: []ebiten.Key{k}, Layout: layout})
			got = append(got, keys...)
		}
		if len(got) != 2 || got[0] != (termview.Key{Rune: 'p', Ctrl: true}) || got[1] != (termview.Key{Rune: 'q', Ctrl: true}) {
			t.Errorf("%s: a layout with no Latin letters keeps the US positions, but Control with the keys at P and Q reached the terminal as %+v, so ctrl-p ctrl-q cannot be typed and the terminal cannot be left from the keyboard", name, got)
		}
	}
}
