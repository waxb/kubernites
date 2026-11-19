package ui

import (
	"strings"
	"unicode"
)

var resourceIcons = map[string]string{
	"gold": IconGold, "money": IconGold,
	"wisdom": IconWisdom, "knowledge": IconWisdom,
	"food": IconFood, "ration": IconFood, "rations": IconFood, "cpu": IconFood, "cpus": IconFood,
	"housing": IconHousing, "room": IconHousing, "rooms": IconHousing, "memory": IconHousing,
	"renown": IconRenown, "reputation": IconRenown,
	"parcels": IconParcel, "parcel": IconParcel, "ledgers": IconLedger, "ledger": IconLedger,
	"crates": IconCrate, "crate": IconCrate, "dispatches": IconDispatch, "dispatch": IconDispatch,
	"boxes": IconBox, "box": IconBox, "sealed": IconSealed,
	"barrels": IconCellar, "barrel": IconCellar,
}

var commandWords = map[string]bool{"kubectl": true, "etcdctl": true, "speakstctl": true, "codexctl": true}

var boxWords = map[string]bool{"box": true, "boxes": true}

func Decorate(text string) string {
	runes := []rune(text)
	var out []rune
	last, quoted := 0, false
	for i := 0; i < len(runes); {
		if quoteMark(runes[i]) {
			quoted = !quoted
		}
		if !unicode.IsLetter(runes[i]) || quoted {
			i++
			continue
		}
		j := i
		for j < len(runes) && unicode.IsLetter(runes[j]) {
			j++
		}
		word := strings.ToLower(string(runes[i:j]))
		if commandWords[word] {
			i = commandEnd(runes, j)
			continue
		}
		if icon, ok := resourceIcons[word]; ok && standalone(runes, i, j) && counted(runes, i, j) && boxed(runes, word, i, j) {
			if at := iconSpot(runes, i); !decorated(runes, at) {
				out = append(append(append(out, runes[last:at]...), []rune(icon)...), gapRune)
				last = at
			}
		}
		i = j
	}
	if out == nil {
		return text
	}
	return string(append(out, runes[last:]...))
}

var countedOnly = map[string]bool{"barrel": true, "barrels": true, "room": true, "rooms": true, "parcel": true, "ledger": true, "crate": true, "dispatch": true, "box": true}

func boxed(runes []rune, word string, i, j int) bool {
	switch {
	case word == "sealed":
		return boxWords[strings.ToLower(nextWord(runes, j))]
	case boxWords[word]:
		return strings.ToLower(prevWord(runes, i)) != "sealed"
	}
	return true
}

func nextWord(runes []rune, j int) string {
	if j >= len(runes) || runes[j] != ' ' {
		return ""
	}
	k := j + 1
	for k < len(runes) && unicode.IsLetter(runes[k]) {
		k++
	}
	return string(runes[j+1 : k])
}

func prevWord(runes []rune, i int) string {
	if i < 2 || runes[i-1] != ' ' {
		return ""
	}
	k := i - 1
	for k > 0 && unicode.IsLetter(runes[k-1]) {
		k--
	}
	return string(runes[k : i-1])
}

func commandEnd(runes []rune, j int) int {
	for ; j < len(runes); j++ {
		switch {
		case runes[j] == '\n' || runes[j] == ')':
			return j
		case strings.ContainsRune(".,;:", runes[j]) && (j+1 == len(runes) || runes[j+1] == ' '):
			return j
		}
	}
	return j
}

func quoteMark(r rune) bool { return r == '"' || r == '\u201c' || r == '\u201d' }

func counted(runes []rune, i, j int) bool {
	if !countedOnly[strings.ToLower(string(runes[i:j]))] {
		return true
	}
	k := i - 1
	for k >= 0 && runes[k] == ' ' {
		k--
	}
	return k >= 0 && unicode.IsDigit(runes[k])
}

func standalone(runes []rune, i, j int) bool {
	if i > 0 && strings.ContainsRune("0123456789._=-/$@#{\\", runes[i-1]) {
		return false
	}
	if j == len(runes) {
		return true
	}
	next := runes[j]
	switch {
	case unicode.IsDigit(next) || strings.ContainsRune("_=(", next):
		return false
	case next == '.' || next == '-':
		return j+1 == len(runes) || !unicode.IsLetter(runes[j+1]) && !unicode.IsDigit(runes[j+1])
	}
	return true
}

func iconSpot(runes []rune, i int) int {
	if i < 2 || runes[i-1] != ' ' {
		return i
	}
	k := i - 1
	for k > 0 && runes[k-1] != ' ' && !iconRune(runes[k-1]) {
		k--
	}
	if quantityWithUnit(runes[k : i-1]) {
		return k
	}
	return i
}

func quantityWithUnit(token []rune) bool {
	digits, letters := 0, 0
	for n, r := range token {
		switch {
		case n == 0 && (r == '+' || r == '-'):
		case unicode.IsDigit(r) || r == '.' || r == ',':
			if letters > 0 {
				return false
			}
			digits++
		case unicode.IsLetter(r):
			letters++
		default:
			return false
		}
	}
	return digits > 0 && letters > 0
}

func decorated(runes []rune, at int) bool {
	return at >= 2 && runes[at-1] == gapRune && iconRune(runes[at-2])
}
