package lexicon

import (
	"strings"
	"testing"
)

func TestANameTheClusterGaveIsNeverRenamed(t *testing.T) {
	for _, name := range []string{
		"system:serviceaccount:shop:pod-reader",
		"system:serviceaccount:kube-system:pod-reader",
		"system:node:kind-worker",
		"system:kube-scheduler",
		"system:controller:replicaset-controller",
		"system:serviceaccounts:kube-system",
	} {
		text := name + " may list pods but not watch them, so they are listed again every 10 s."
		for mode, out := range map[string]string{"story": StoryText(StoryNames(text)), "names": TrueNames(text), "story names": StoryNames(text)} {
			if !strings.HasPrefix(out, name+" ") {
				t.Errorf("[%s] %q reads %q", mode, name, out)
			}
		}
		if out := StoryText(StoryNames(text)); !strings.Contains(out, "huts") {
			t.Errorf("the words around %q are no longer told: %q", name, out)
		}
	}
	if out := StoryText(StoryNames("Pods in kube-system run on Node kind-worker.")); strings.Contains(out, "kube-system") || strings.Contains(out, "Node") {
		t.Errorf("a sentence without a user name is told as before: %q", out)
	}
}
