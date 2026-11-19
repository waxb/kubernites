package app

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/waxb/kubernites/pkg/lexicon"
)

const (
	storyTemplates = 4096
	numberSlot     = "\x03"
)

var (
	storyNumber = regexp.MustCompile(`\b\d+(?:[.,]\d+)*\b`)
	storyUnit   = storyUnitK8s()
)

func storyUnitK8s() *regexp.Regexp {
	return regexp.MustCompile(`\d\s*(?:m\b|Mi|Gi|Ki|MiB|GiB|[Cc][Pp][Uu]|cores?|millicores?|memory)`)
}

type storyNamer struct {
	templates map[string]string
}

func newStoryNamer() func(string) string {
	n := &storyNamer{templates: make(map[string]string, storyTemplates)}
	return n.name
}

func hasLetter(s string) bool {
	return strings.IndexFunc(s, unicode.IsLetter) >= 0
}

func (n *storyNamer) name(text string) string {
	if !hasLetter(text) || strings.Contains(text, numberSlot) {
		return text
	}
	text = lexicon.StoryNames(text)
	numbers := storyNumber.FindAllString(text, -1)
	if len(numbers) == 0 || storyUnit.MatchString(text) || lexicon.NumberSensitive(text) {
		return lexicon.StoryText(text)
	}
	template := storyNumber.ReplaceAllString(text, numberSlot)
	story, ok := n.templates[template]
	if !ok {
		if len(n.templates) >= storyTemplates {
			n.templates = make(map[string]string, storyTemplates)
		}
		story = lexicon.StoryText(template)
		n.templates[template] = story
	}
	if strings.Count(story, numberSlot) != len(numbers) {
		return lexicon.StoryText(text)
	}
	var b strings.Builder
	for i, part := range strings.Split(story, numberSlot) {
		if i > 0 {
			b.WriteString(numbers[i-1])
		}
		b.WriteString(part)
	}
	return b.String()
}
