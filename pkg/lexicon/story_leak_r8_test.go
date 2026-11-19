package lexicon

import (
	"strings"
	"testing"
)

func TestStoryTextNamesTheHandRollInStoryWords(t *testing.T) {
	cases := []string{
		"endpointslice.discovery.k8s.io/parcels-by-hand patched",
		`endpointslice.discovery.k8s.io "parcels-by-hand" deleted from default namespace`,
		"endpointslices.discovery.k8s.io/parcels-by-hand created",
	}
	for _, in := range cases {
		out := StoryText(in)
		if KubernetesWords.MatchString(out) {
			t.Errorf("StoryText(%q) = %q: a Kubernetes word leaks into story mode", in, out)
		}
	}
	if got := StoryText("endpointslice.discovery.k8s.io/parcels-by-hand patched"); got != "Roll/parcels-by-hand patched" {
		t.Errorf("hand roll toast = %q, want the roll named like depot/ and crew/", got)
	}
}

func TestRolloutStatusCountsHutsInStoryWords(t *testing.T) {
	cases := map[string]string{
		`Waiting for deployment "porters" rollout to finish: 2 old replicas are pending termination...`:     `Waiting for crew "porters" roll-out to finish: 2 old huts are pending termination...`,
		`Waiting for deployment "porters" rollout to finish: 5 out of 11 new replicas have been updated...`: `Waiting for crew "porters" roll-out to finish: 5 out of 11 new huts have been updated...`,
		`Waiting for deployment "porters" rollout to finish: 3 of 6 updated replicas are available...`:      `Waiting for crew "porters" roll-out to finish: 3 of 6 updated huts are available...`,
	}
	for in, want := range cases {
		if got := StoryText(in); got != want {
			t.Errorf("StoryText(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

func TestNamesSwapsOnlyTheNounScrollAndTheNounKeep(t *testing.T) {
	cases := map[string]string{
		"PgUp/PgDn scroll":                 "PgUp/PgDn scroll",
		"Scroll the map to the north gate": "Scroll the map to the north gate",
		"RAISE A SHIFT TO KEEP 3 PORTERS":  "RAISE A REPLICASET TO KEEP 3",
		"Raise a shift to keep 3 porters":  "Raise a ReplicaSet to keep 3 porters",
		"open the Scrolls":                 "open the ConfigMaps",
		"The Reeve of Lone Tower re-reads its scroll and runs porter-lone-tower as porter:1.1": "manifest",
	}
	for in, want := range cases {
		got := TrueNames(in)
		if want == "manifest" {
			if !strings.Contains(got, "manifest") || strings.Contains(got, "ConfigMap") {
				t.Errorf("TrueNames(%q) = %q, want the static Pod's manifest", in, got)
			}
			continue
		}
		if !strings.Contains(got, want) {
			t.Errorf("TrueNames(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStoryCommaListsNameEveryKind(t *testing.T) {
	cases := map[string]string{
		"speakstctl get crews,shifts":      "kubectl get deployments,replicasets",
		"speakstctl get huts,depots,rolls": "kubectl get pods,services,endpointslices",
	}
	for in, want := range cases {
		if got := CommandFromStory(in); got != want {
			t.Errorf("CommandFromStory(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlainMeaningsAvoidTheKeepHomograph(t *testing.T) {
	for _, w := range PlainWords() {
		lower := " " + w.Means + " "
		for _, bad := range []string{" keeps ", " keep ", " razes "} {
			if strings.Contains(lower, bad) {
				t.Errorf("%s means %q: %q reads as a Keep or a raise", w.Term, w.Means, strings.TrimSpace(bad))
			}
		}
	}
	for _, term := range []string{"Last Rites", "fief"} {
		if _, ok := PlainOf(term); !ok {
			t.Errorf("%s has no plain meaning", term)
		}
	}
}

func TestNamesNeverRenamesTheDepotInDataOrCommands(t *testing.T) {
	cases := map[string]string{
		"    kubernetes.io/service-name: parcels":          "    kubernetes.io/service-name: parcels",
		"kubectl set selector service parcels app=porters": "kubectl set selector service parcels app=porters",
		"kubectl describe svc parcels":                     "kubectl describe svc parcels",
	}
	for in, want := range cases {
		if got := TrueNames(in); got != want {
			t.Errorf("TrueNames(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStoryTextTellsRepliesWithoutRawNames(t *testing.T) {
	cases := []string{
		"Warning: resource deployments/porters was previously managed with 'kubectl apply'. Rolling back will not update the kubectl.kubernetes.io/last-applied-configuration annotation, which may cause unexpected behavior on future 'kubectl apply' operations.",
		`endpointslice.discovery.k8s.io "parcels-by-hand" deleted from default namespace`,
		`pod "porter-x" deleted from default namespace`,
		"Finished defragmenting etcd member[127.0.0.1:2379]. took 1.76s",
		"node/stone-keep-1 cordoned\nevicting pod default/clerks-64bcc844bf-6swq8\npod/clerks-64bcc844bf-6swq8 evicted",
	}
	for _, in := range cases {
		out := StoryText(in)
		for _, raw := range []string{"kubernetes.io", "last-applied", "annotation", "default fief", "default namespace", "127.0.0.1", "default/"} {
			if strings.Contains(out, raw) {
				t.Errorf("StoryText(%q) = %q: %q stays raw in story mode", in, out, raw)
			}
		}
	}
}
