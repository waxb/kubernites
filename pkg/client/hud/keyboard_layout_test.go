package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/termview"
)

var usRows = []ebiten.Key{
	ebiten.KeyQ, ebiten.KeyW, ebiten.KeyE, ebiten.KeyR, ebiten.KeyT, ebiten.KeyY, ebiten.KeyU, ebiten.KeyI, ebiten.KeyO, ebiten.KeyP, ebiten.KeyBracketLeft, ebiten.KeyBracketRight,
	ebiten.KeyA, ebiten.KeyS, ebiten.KeyD, ebiten.KeyF, ebiten.KeyG, ebiten.KeyH, ebiten.KeyJ, ebiten.KeyK, ebiten.KeyL, ebiten.KeySemicolon, ebiten.KeyQuote,
	ebiten.KeyZ, ebiten.KeyX, ebiten.KeyC, ebiten.KeyV, ebiten.KeyB, ebiten.KeyN, ebiten.KeyM, ebiten.KeyComma, ebiten.KeyPeriod, ebiten.KeySlash,
	ebiten.KeyMinus, ebiten.KeyDigit6, ebiten.KeyBackslash,
}

var keyboardLayouts = map[string]string{
	"US":     "qwertyuiop[]asdfghjkl;'zxcvbnm,./-6\\",
	"Dvorak": "',.pyfgcrl/=aoeuidhtns-;qjkxbmwvz[6\\",
	"AZERTY": "azertyuiop^$qsdfghjklmùwxcvbn,;:!)-*",
	"QWERTZ": "qwertzuiopü+asdfghjklöäyxcvbnm,.-ß6#",
}

func layoutTable(labels string) func(ebiten.Key) string {
	runes := []rune(labels)
	table := map[ebiten.Key]string{}
	for i, k := range usRows {
		table[k] = string(runes[i])
	}
	return func(k ebiten.Key) string { return table[k] }
}

func labelledKey(t *testing.T, layout func(ebiten.Key) string, label rune) ebiten.Key {
	t.Helper()
	for _, k := range usRows {
		if layout(k) == string(label) {
			return k
		}
	}
	t.Fatalf("no key is labelled %q", label)
	return 0
}

func TestControlChordsInTheTerminalFollowTheLabelledKeysOfTheKeyboardLayout(t *testing.T) {
	for name, labels := range keyboardLayouts {
		layout := layoutTable(labels)
		for _, label := range "pqcdzxa" {
			k := labelledKey(t, layout, label)
			keys, text := TerminalKeys(Input{Ctrl: true, Control: true, Keys: []ebiten.Key{k}, Layout: layout})
			want := termview.Key{Rune: label, Ctrl: true}
			if text != "" || len(keys) != 1 || keys[0] != want {
				t.Errorf("%s: Control and the key labelled %q (Ebitengine's %v) gave %+v %q, want %+v", name, label, k, keys, text, want)
			}
		}
	}
}

func TestTheLeaveChordIsTheLabelledCtrlPCtrlQOnEveryLayout(t *testing.T) {
	for name, labels := range keyboardLayouts {
		layout := layoutTable(labels)
		var got []termview.Key
		for _, label := range "pq" {
			keys, _ := TerminalKeys(Input{Ctrl: true, Control: true, Keys: []ebiten.Key{labelledKey(t, layout, label)}, Layout: layout})
			got = append(got, keys...)
		}
		if len(got) != 2 || got[0] != (termview.Key{Rune: 'p', Ctrl: true}) || got[1] != (termview.Key{Rune: 'q', Ctrl: true}) {
			t.Errorf("%s: the keys labelled ctrl-p ctrl-q reach the terminal as %+v", name, got)
		}
	}
}

func TestTheCardsControlChordsFollowTheLabelledKeysOfTheKeyboardLayout(t *testing.T) {
	for name, labels := range keyboardLayouts {
		layout := layoutTable(labels)
		for letter := range ChordKeys {
			h, _ := liveHUD(t, true)
			h.OverlayID = ""
			h.SetLiveOrders([]LiveOrder{{Key: "order-" + letter, Label: letter, Chord: letter}})
			label := []rune(letter)[0] - 'A' + 'a'
			k := labelledKey(t, layout, label)
			intents, done := h.liveChord(Input{Ctrl: true, Control: true, Keys: []ebiten.Key{k}, Layout: layout}, Frame{})
			fired := ""
			for _, it := range intents {
				if it.Kind == IntentLiveOrder {
					fired = it.Target
				}
			}
			if !done || fired != "order-"+letter {
				t.Errorf("%s: Control and the key labelled %q fired %q, want the order whose chord is CTRL-%s", name, label, fired, letter)
			}
		}
	}
}
