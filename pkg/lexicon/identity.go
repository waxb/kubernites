package lexicon

import (
	"regexp"
	"strings"
)

const nameSlot = "\x04"

var identityName = regexp.MustCompile(`(^|[^A-Za-z0-9._/:@-])(system:[A-Za-z0-9._@:-]+)`)

func identityNames(text string) [][2]int {
	var out [][2]int
	for _, m := range identityName.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[4], m[5]
		end = start + len(strings.TrimRight(text[start:end], ".-_@:"))
		if end > start+len("system:") {
			out = append(out, [2]int{start, end})
		}
	}
	return out
}

func keepingNames(text string, tell func(string) string) string {
	spans := identityNames(text)
	if len(spans) == 0 || strings.Contains(text, nameSlot) {
		return tell(text)
	}
	var b strings.Builder
	names := make([]string, 0, len(spans))
	last := 0
	for _, s := range spans {
		b.WriteString(text[last:s[0]])
		b.WriteString(nameSlot)
		names = append(names, text[s[0]:s[1]])
		last = s[1]
	}
	b.WriteString(text[last:])
	told := tell(b.String())
	if strings.Count(told, nameSlot) != len(names) {
		return tell(text)
	}
	for _, n := range names {
		told = strings.Replace(told, nameSlot, n, 1)
	}
	return told
}
