package lexicon

import "strings"

var (
	rawMarks  = []string{"--type=json", "-p '{", "-p '["}
	echoMarks = append(rawMarks, "--overrides='")
)

func StoryCommandShown(command string) bool {
	for _, mark := range rawMarks {
		if strings.Contains(command, mark) {
			return false
		}
	}
	return true
}

func CommandHead(command string) string {
	cut := len(command)
	for _, mark := range echoMarks {
		if i := strings.Index(command, mark); i >= 0 && i < cut {
			cut = i
		}
	}
	return strings.TrimSpace(command[:cut])
}
