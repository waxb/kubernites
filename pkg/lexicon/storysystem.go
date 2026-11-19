package lexicon

import (
	"regexp"
	"strings"
)

var (
	taintForm    = regexp.MustCompile(`\b(node-role\.kubernetes\.io/control-plane|node\.kubernetes\.io/[a-z-]+)(?:=[^\s:,]*)?:(NoSchedule|PreferNoSchedule|NoExecute)\b`)
	typeListLine = regexp.MustCompile(`(?m)^Types: (.+)$`)
	typeListItem = regexp.MustCompile(`^([a-z]+)(?: \([a-z]+\))?$`)
	chanceryDoor = regexp.MustCompile(`(?m)(^|[\s/"])kubernetes([\s,"]|$)`)
)

const chanceryDepot = "chancery"

var storyMerged = map[string]string{"endpoints": "endpointslices"}

func storyTaints(text string) string {
	var b strings.Builder
	last := 0
	for _, span := range commandSpans(text) {
		b.WriteString(wardsIn(text[last:span[0]]))
		b.WriteString(text[span[0]:span[1]])
		last = span[1]
	}
	b.WriteString(wardsIn(text[last:]))
	return b.String()
}

func wardsIn(seg string) string {
	return taintForm.ReplaceAllStringFunc(seg, func(m string) string {
		parts := taintForm.FindStringSubmatch(m)
		return WardName(parts[1]) + ":" + parts[2]
	})
}

func storyTypeList(text string) string {
	return typeListLine.ReplaceAllStringFunc(text, func(line string) string {
		items := strings.Split(strings.TrimPrefix(line, "Types: "), ", ")
		out, seen := []string{}, map[string]bool{}
		for _, item := range items {
			name := item
			if m := typeListItem.FindStringSubmatch(item); m != nil {
				name = m[1]
			}
			if story, ok := storyCommandNouns[name]; ok {
				name = story
			}
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
		return "Types: " + strings.Join(out, ", ")
	})
}

func storySystemDepots(text string) string {
	return chanceryDoor.ReplaceAllString(text, "${1}"+chanceryDepot+"${2}")
}
