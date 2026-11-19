package lexicon

import (
	"strings"
	"testing"
)

func TestTheCellarIsCountedInBarrelsAndHousingInRooms(t *testing.T) {
	for in, want := range map[string]string{
		"kubectl set resources deployment/clerks --limits=ephemeral-storage=2Gi":                     "speakstctl set resources crew/clerks --binding=cellar=2048",
		"kubectl set resources deployment/clerks --requests=memory=128Mi,ephemeral-storage=1Gi":      "speakstctl set resources crew/clerks --pledge=housing=128,cellar=1024",
		"Container porter was using 3Gi, request is 0, has larger consumption of ephemeral-storage.": "Worker porter was using 3,072 barrels, request is 0, has larger consumption of Cellar.",
		"The worker took 512Mi of memory.":                                                           "The worker took 512 rooms.",
		"0/3 nodes are available: 2 Insufficient ephemeral-storage.":                                 "0/3 Keeps are available: 2 Keeps short of ground for cellars.",
	} {
		if got := StoryText(in); got != want {
			t.Errorf("StoryText(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

func TestACellarLineAndAHousingLineEachKeepTheirUnit(t *testing.T) {
	got := StoryText("memory: 512Mi\nephemeral-storage: 2Gi of ephemeral-storage")
	if !strings.Contains(got, "512 rooms") || !strings.Contains(got, "2,048 barrels") {
		t.Errorf("units crossed lines: %q", got)
	}
	if got := StoryBarrels("512Mi (1%)"); got != "512 barrels (1%)" {
		t.Errorf("StoryBarrels = %q", got)
	}
}

func TestTheCellarKeyRoundTripsThroughTheStoryManifest(t *testing.T) {
	kube := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: clerk\nspec:\n  containers:\n  - name: clerk\n    image: clerk:1.0\n    resources:\n      limits:\n        ephemeral-storage: 1Gi"
	story := StoryManifest(kube)
	if !strings.Contains(story, "cellar: 1024") || strings.Contains(story, "ephemeral") {
		t.Fatalf("story manifest = %q", story)
	}
	if back := ManifestFromStory(story); !strings.Contains(back, "ephemeral-storage: 1024Mi") {
		t.Errorf("back to Kubernetes = %q", back)
	}
	if got := CommandFromStory("speakstctl set resources crew/clerks --binding=cellar=2048"); got != "kubectl set resources deployment/clerks --limits=ephemeral-storage=2048Mi" {
		t.Errorf("CommandFromStory = %q", got)
	}
}

func TestNamesModeReadsBarrelsAndTheCellarWardAsKubernetes(t *testing.T) {
	for in, want := range map[string]string{
		"Each hut pledges 512 barrels.":                "Each Pod requests 512Mi of ephemeral storage.",
		"The Cellar Ward bars new huts from the Keep.": "DiskPressure taint bars new Pods from the Node.",
	} {
		if got := TrueNames(in); got != want {
			t.Errorf("TrueNames(%q)\n got %q\nwant %q", in, got, want)
		}
	}
	if got := WardName("node.kubernetes.io/disk-pressure"); got != "the Cellar Ward" {
		t.Errorf("WardName = %q", got)
	}
}
