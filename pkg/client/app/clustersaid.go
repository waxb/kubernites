package app

import "github.com/waxb/kubernites/pkg/client/ui"

const storyWhy = "not done, switch NAMES to read why"

func clusterSaid(head, said string) string {
	story := head + storyWhy
	if head == "" {
		story = "Not done, switch NAMES to read why."
	}
	return clusterSaidOr(story, head, said)
}

func clusterSaidOr(story, head, said string) string {
	if said == "" {
		return head
	}
	if !ui.Naming() {
		return story
	}
	return ui.Verbatim(ui.Name(head) + said)
}
