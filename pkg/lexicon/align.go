package lexicon

import "strings"

type CommandToken struct {
	Story string
	Kube  string
}

func (t CommandToken) Mapped() bool { return t.Story != t.Kube }

func AlignCommand(command string) []CommandToken {
	fields := strings.Fields(command)
	out := make([]CommandToken, 0, len(fields))
	for i, tok := range fields {
		out = append(out, CommandToken{Story: alignedStory(fields, i), Kube: tok})
	}
	return out
}

func alignedStory(fields []string, i int) string {
	tok := fields[i]
	if i == 0 {
		return storyToken(tok)
	}
	if fields[0] == "etcdctl" && i == 1 && tok == "endpoint" {
		return ""
	}
	tok = storyIdentifiersIn(storySystemDepots(storyLabelKeys.Replace(tok)))
	return storyCommandWord(tok)
}

func StoryLine(tokens []CommandToken) string {
	words := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t.Story != "" {
			words = append(words, t.Story)
		}
	}
	return strings.Join(words, " ")
}
