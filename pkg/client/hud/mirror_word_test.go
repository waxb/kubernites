package hud

import (
	"strings"
	"testing"
)

func TestThePortersLoreSaysWhatAMirrorIsWhereItFirstNamesIt(t *testing.T) {
	for name, lore := range map[string][]string{"story": staticLore, "chapter": chapterStaticLore} {
		text := strings.Join(lore, " ")
		at := strings.Index(text, "mirror")
		if at < 0 {
			continue
		}
		if head := text[:min(at+40, len(text))]; !strings.Contains(head, "shows") && !strings.Contains(head, "posts") {
			t.Errorf("the %s lore says %q before it says what a mirror is", name, text)
		}
	}
}
