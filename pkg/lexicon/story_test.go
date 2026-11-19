package lexicon

import (
	"fmt"
	"testing"
)

func TestStoryGlossesAreTheParenthesesTheKubernetesFormDrops(t *testing.T) {
	cases := map[string][]string{
		"Builders begin The Codex (etcd).":                                 {"etcd"},
		"Your porters wait Unhoused (Pending)":                             {"Pending"},
		"BUILD the Quartermaster (kube-scheduler).":                        {"kube-scheduler"},
		"Open a harbor berth (2 more)":                                     nil,
		"Scale the porters: 3 of 10 lost in 30 s (a ship, or big boxes).":  nil,
		"The Stewards (kube-controller-manager) and the Codex (etcd) wake": {"kube-controller-manager", "etcd"},
	}
	for text, want := range cases {
		got := []string{}
		for _, g := range storyGlosses(text) {
			got = append(got, g.term)
		}
		if len(got) != len(want) {
			t.Errorf("%q: glosses %v, want %v", text, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%q: glosses %v, want %v", text, got, want)
			}
		}
	}
}

func TestAStoryLabelGlossesATermOnlyOnItsFirstIntroduction(t *testing.T) {
	label := StoryLabeler(8)
	first := "Builders begin The Codex (etcd)."
	steps := []struct{ in, want string }{
		{first, first},
		{"The Codex (etcd) swells.", "The Codex swells."},
		{first, first},
		{"Open a harbor berth (2 more)", "Open a harbor berth (2 more)"},
		{"Your porters wait Unhoused (Pending) beside the Codex (etcd)", "Your porters wait Unhoused (Pending) beside the Codex"},
		{"Clerks wait Unhoused (Pending)", "Clerks wait Unhoused"},
		{"", ""},
	}
	for _, s := range steps {
		if got := label(s.in); got != s.want {
			t.Errorf("label(%q) = %q, want %q", s.in, got, s.want)
		}
	}
}

func TestAStoryLabelerStaysBounded(t *testing.T) {
	labels := newStoryLabels(2)
	for i := range 50 {
		labels.label(fmt.Sprintf("The Codex (etcd) holds %d keys", i))
		if len(labels.seen) > 2 {
			t.Fatalf("the labeler remembers %d texts, limit 2", len(labels.seen))
		}
	}
	if got := labels.label("The Codex (etcd) holds 0 keys"); got != "The Codex holds 0 keys" {
		t.Fatalf("a text forgotten and labelled again = %q", got)
	}
}
