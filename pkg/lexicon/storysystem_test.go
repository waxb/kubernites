package lexicon

import (
	"strings"
	"testing"
)

func TestStoryTextNamesTheRoyalWardWhereADescribeListsIt(t *testing.T) {
	for k8s, want := range map[string]string{
		"node-role.kubernetes.io/control-plane:NoSchedule":                        "the Royal Ward:NoSchedule",
		"node.kubernetes.io/not-ready:NoExecute op=Exists for 300s":               "the Unready Ward:NoExecute op=Exists for 300s",
		"node.kubernetes.io/memory-pressure=true:NoSchedule":                      "the Crowding Ward:NoSchedule",
		"Labels: node-role.kubernetes.io/control-plane=":                          "Badges: citadel-keep=",
		"kubectl taint nodes x node-role.kubernetes.io/control-plane:NoSchedule-": "speakstctl taint keeps x citadel-keep:NoSchedule-",
	} {
		if got := StoryText(k8s); got != want {
			t.Errorf("StoryText(%q)\n = %q\nwant %q", k8s, got, want)
		}
	}
	if back := CommandFromStory("speakstctl taint keeps x citadel-keep:NoSchedule-"); back != "kubectl taint nodes x node-role.kubernetes.io/control-plane:NoSchedule-" {
		t.Errorf("the taint command reads back as %q", back)
	}
}

func TestStoryHelpListsOnlyTheRealmsTypes(t *testing.T) {
	line := "Types: pods (po), nodes (no), deployments (deploy), replicasets (rs), services (svc), endpointslices, events (ev), namespaces (ns), configmaps (cm), limitranges (limits), horizontalpodautoscalers (hpa)"
	got := StoryText(line)
	want := "Types: huts, keeps, crews, shifts, depots, rolls, tidings, fiefs, scrolls, defaults, horns"
	if got != want {
		t.Fatalf("the help's types read\n %q\nwant %q", got, want)
	}
	for _, noun := range strings.Split(strings.TrimPrefix(want, "Types: "), ", ") {
		if kube := CommandFromStory("speakstctl get " + noun); kube == "kubectl get "+noun {
			t.Errorf("the story type %q does not read back as a Kubernetes type", noun)
		}
	}
}

func TestStoryModeNamesTheCitadelsOwnDepots(t *testing.T) {
	for k8s, want := range map[string]string{
		"kubernetes   ClusterIP   10.96.0.1    <none>   443/TCP   2min 32s": "chancery   ClusterIP   10.96.0.1    <none>   443/TCP   2min 32s",
		"Service/kube-dns":                       "Depot/herald",
		"kubectl describe service kubernetes":    "speakstctl describe depot chancery",
		"app=porters,kubernetes.io/service-name": "app=porters,depot-name",
	} {
		if got := StoryText(k8s); got != want {
			t.Errorf("StoryText(%q)\n = %q\nwant %q", k8s, got, want)
		}
	}
	for story, kube := range map[string]string{
		"speakstctl describe depot chancery":          "kubectl describe service kubernetes",
		"speakstctl describe depot herald -n citadel": "kubectl describe service kube-dns -n kube-system",
	} {
		if got := CommandFromStory(story); got != kube {
			t.Errorf("CommandFromStory(%q) = %q, want %q", story, got, kube)
		}
	}
}

func TestStoryModeWordsTheReevesMarksAndTheGraduatesFailure(t *testing.T) {
	for k8s, want := range map[string]string{
		"  Warning   Failed   8s   kubelet   Error: ErrImagePull":       "  Warning   Failed   8s   Reeve   Error: the graduate never arrived",
		"Pending (FailedScheduling)":                                    "Unhoused",
		"ErrImagePull / ImagePullBackOff":                               "The graduate never arrived",
		"  Warning   FailedScheduling   4s   0/1 nodes are available":   "  Warning   NoKeepHasRoom   4s   0/1 Keeps are available",
		"kubernetes.io/config.mirror: d4e113970f637624556c294002793db7": "reeve-mirror: d4e113970f637624556c294002793db7",
	} {
		got := StoryText(k8s)
		if got != want {
			t.Errorf("StoryText(%q)\n = %q\nwant %q", k8s, got, want)
		}
		if strings.Contains(got, "kubernetes.io") || strings.Contains(got, "ErrImagePull") || strings.Contains(got, "FailedScheduling") {
			t.Errorf("StoryText(%q) still says %q", k8s, got)
		}
	}
}

func TestAStoryCommandForRollsRunsWhatItsEchoSays(t *testing.T) {
	echo := StoryText("kubectl get endpoints")
	if echo != "speakstctl get rolls" {
		t.Fatalf("get endpoints echoes as %q", echo)
	}
	run := CommandFromStory("speakstctl get endpoints")
	if run != CommandFromStory(echo) {
		t.Fatalf("speakstctl get endpoints runs %q, but its echo %q runs %q", run, echo, CommandFromStory(echo))
	}
}
