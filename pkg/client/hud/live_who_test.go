package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestTheLiveCardsTipsKeepTheUserNameTheClusterGaveInBothModes(t *testing.T) {
	defer ui.SetNamer(nil)
	h := New(nil, Backend{})
	for _, who := range []string{"system:serviceaccount:shop:pod-reader", "system:serviceaccount:kube-system:pod-reader", "system:node:kind-worker"} {
		c := LiveCard{Context: "kind-kubernites", Who: who, Keeps: 2, Huts: 9, Fiefs: 5, Isles: 1, Watching: true,
			Refused: []string{"services"}, Denied: []string{"secrets"}, Listed: []ListedRead{{List: "pods", Every: 10}}}
		for _, mode := range []string{"story", "names"} {
			if mode == "story" {
				ui.SetStoryNamer(func(s string) string { return lexicon.StoryText(lexicon.StoryNames(s)) })
			} else {
				ui.SetNamer(names.Kubernetes(64))
			}
			tips := map[string]string{}
			for _, l := range c.lines() {
				if l.id == liveRefusedID || l.id == liveListedID {
					tips[l.id] = l.tip
				}
			}
			if len(tips) != 2 {
				t.Fatalf("[%s] the card has %d of its two tips: %v", mode, len(tips), tips)
			}
			for id, tip := range tips {
				if named := ui.Name(tip); !strings.Contains(named, who) {
					t.Errorf("[%s] %s's tip renames the user %s: %q", mode, id, who, named)
				}
				if lines, _ := h.layoutTip(tip); strings.Contains(strings.Join(lines, ""), "writ") {
					t.Errorf("[%s] %s's tip draws the user's name renamed: %q", mode, id, lines)
				}
			}
		}
	}
}
