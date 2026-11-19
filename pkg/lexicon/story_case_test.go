package lexicon

import "testing"

func TestStoryKindsMatchInAnyCaseAsKubectlDoes(t *testing.T) {
	for in, want := range map[string]string{
		StoryCommand + " get Keeps":                "kubectl get nodes",
		StoryCommand + " describe Keep lone-tower": "kubectl describe node lone-tower",
		StoryCommand + " get HUTS -A":              "kubectl get pods -A",
		StoryCommand + " describe keep lone-tower": "kubectl describe node lone-tower",
	} {
		if got := CommandFromStory(in); got != want {
			t.Errorf("%q speaks %q, want %q", in, got, want)
		}
	}
}
