package app

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

type chordMods struct{ shift, ctrl, alt bool }

var modifierNames = map[string]func(*chordMods){
	"shift":   func(m *chordMods) { m.shift = true },
	"ctrl":    func(m *chordMods) { m.ctrl = true },
	"control": func(m *chordMods) { m.ctrl = true },
	"cmd":     func(m *chordMods) { m.ctrl = true },
	"meta":    func(m *chordMods) { m.ctrl = true },
	"alt":     func(m *chordMods) { m.alt = true },
}

func parseChord(name string) (ebiten.Key, chordMods, error) {
	parts := strings.Split(strings.TrimSpace(name), "+")
	var mods chordMods
	for _, p := range parts[:len(parts)-1] {
		set, ok := modifierNames[strings.ToLower(strings.TrimSpace(p))]
		if !ok {
			return 0, chordMods{}, fmt.Errorf("unknown modifier %q in %q: use Shift, Ctrl or Alt, as in Shift+Enter", p, name)
		}
		set(&mods)
	}
	k, err := keyNamed(strings.TrimSpace(parts[len(parts)-1]))
	return k, mods, err
}

func keyNamed(name string) (ebiten.Key, error) {
	var k ebiten.Key
	if err := k.UnmarshalText([]byte(name)); err == nil {
		return k, nil
	}
	if letter, ok := strings.CutPrefix(name, "Key"); ok && len(letter) == 1 {
		if err := k.UnmarshalText([]byte(letter)); err == nil {
			return k, nil
		}
	}
	return 0, fmt.Errorf("unknown key %q: use Ebitengine names such as Enter, Escape, Tab, Space, ArrowUp, O, Digit1, or a chord such as Shift+Enter", name)
}

var keyChars = map[string]rune{"Space": ' ', "Minus": '-', "Period": '.', "Comma": ',', "Slash": '/', "Equal": '='}

func keyChar(k ebiten.Key, mods chordMods) []rune {
	if mods.ctrl || mods.alt {
		return nil
	}
	name := k.String()
	switch {
	case len(name) == 1 && name[0] >= 'A' && name[0] <= 'Z':
		if mods.shift {
			return []rune(name)
		}
		return []rune(strings.ToLower(name))
	case strings.HasPrefix(name, "Digit") && len(name) == 6:
		return []rune{rune(name[5])}
	}
	if r, ok := keyChars[name]; ok {
		return []rune{r}
	}
	return nil
}
