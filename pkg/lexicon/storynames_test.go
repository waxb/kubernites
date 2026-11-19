package lexicon

import "testing"

func TestStoryTextSpeaksOnlyTheRealmsWords(t *testing.T) {
	cases := map[string]string{
		"Ask the Chancery what runs: kubectl get pods":                    "Ask the Chancery what runs: speakstctl get huts",
		"At noon, kubectl top nodes shows the Tower's 2 CPUs at work.":    "At noon, speakstctl top keeps shows the Tower's 2 food at work.",
		"Speak it: kubectl scale deployment porters --replicas=3.":        "Speak it: speakstctl scale crew porters --count=3.",
		"kubectl set image deployment/porters porter=porter:1.1":          "speakstctl set trade crew/porters porter=porter:1.1",
		"kubectl set resources deployment/porters --requests=cpu=100m":    "speakstctl set resources crew/porters --pledge=food=100",
		"kubectl set resources deployment/porters --limits=memory=1Gi":    "speakstctl set resources crew/porters --binding=housing=1024",
		"3 porters wait for a Keep with room (Pending): no Keep can seat": "3 porters wait for a Keep with room: no Keep can seat",
		"A Pod is Pending: no Node has 200m of CPU free.":                 "A hut is Unhoused: no Keep has 200 rations free.",
		"Each Pod asks 64Mi of memory and 1Gi more.":                      "Each hut asks 64 rooms and 1,024 rooms more.",
		"An etcd member keeps the cluster's state.":                       "A Codex-keeper keeps the realm's state.",
		"Restore etcd: the cluster's memory.":                             "Restore the Codex: the realm's housing.",
		"The Service sends boxes to ready Pods.":                          "The depot sends boxes to ready huts.",
		"A hut falls; the Levies stay.":                                   "A hut falls; the huts stay.",
		"Pods are the smallest unit.":                                     "Huts are the smallest unit.",
		"TOP PODS":                                                        "TOP HUTS",
		"An identifier like app=porters stays as it is":                   "An identifier like app=porters stays as it is",
	}
	for in, want := range cases {
		if got := StoryText(in); got != want {
			t.Errorf("StoryText(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

func TestStoryCommandsRunAsKubectl(t *testing.T) {
	cases := map[string]string{
		"speakstctl get huts":                                                   "kubectl get pods",
		"speakstctl top keeps":                                                  "kubectl top nodes",
		"speakstctl scale crew porters --count=3":                               "kubectl scale deployment porters --replicas=3",
		"speakstctl set trade crew/porters porter=porter:1.1":                   "kubectl set image deployment/porters porter=porter:1.1",
		"speakstctl set resources crew/porters --pledge=food=100":               "kubectl set resources deployment/porters --requests=cpu=100m",
		"speakstctl set resources crew/porters --binding=housing=1024,food=500": "kubectl set resources deployment/porters --limits=memory=1024Mi,cpu=500m",
		"speakstctl autoscale crew porters --floor=2 --ceiling=10 --food=70%":   "kubectl autoscale deployment porters --min=2 --max=10 --cpu=70%",
		"kubectl get pods":                                                      "kubectl get pods",
		"speakstctl describe hut porters-abc":                                   "kubectl describe pod porters-abc",
	}
	for in, want := range cases {
		if got := CommandFromStory(in); got != want {
			t.Errorf("CommandFromStory(%q)\n got %q\nwant %q", in, got, want)
		}
	}
	for _, kube := range []string{"kubectl get pods", "kubectl scale deployment porters --replicas=3", "kubectl set resources deployment/porters --requests=cpu=100m", "kubectl top nodes"} {
		if back := CommandFromStory(StoryText(kube)); back != kube {
			t.Errorf("round trip %q -> %q -> %q", kube, StoryText(kube), back)
		}
	}
}

func TestStoryTextIsIdempotent(t *testing.T) {
	for _, in := range []string{"A Pod is Pending: kubectl get pods", "64Mi of memory and 200m of CPU", "The Service (a depot) and the Node."} {
		once := StoryText(in)
		if twice := StoryText(once); twice != once {
			t.Errorf("StoryText twice changed %q -> %q -> %q", in, once, twice)
		}
	}
}

func TestStoryTextHandlesTwoCommandsInOneSentence(t *testing.T) {
	got := StoryText("Try kubectl get pods and kubectl get nodes now")
	if want := "Try speakstctl get huts and speakstctl get keeps now"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStoryTextReadsTheCodexCommandAndFieldNames(t *testing.T) {
	cases := map[string]string{
		"speak: etcdctl defrag":                             "speak: codexctl defrag",
		"requests.cpu reserves food for scheduling.":        "The Food Pledge reserves food for scheduling.",
		"No Pledge (requests.cpu), no utilization.":         "No Pledge, no utilization.",
		"tainted node.kubernetes.io/unreachable after 50s.": "tainted the Storm Ward after 50s.",
	}
	for in, want := range cases {
		if got := StoryText(in); got != want {
			t.Errorf("StoryText(%q)\n got %q\nwant %q", in, got, want)
		}
	}
	if got := CommandFromStory("codexctl defrag"); got != "etcdctl defrag" {
		t.Fatalf("codexctl runs as %q", got)
	}
}

func TestStoryTextRenamesTheCitadelsHuts(t *testing.T) {
	if got := StoryText("kube-system   etcd-lone-tower   Running"); got != "citadel   codex-lone-tower   Burning" && got != "citadel   codex-lone-tower   Running" {
		t.Fatalf("got %q", got)
	}
	if got := CommandFromStory("speakstctl describe hut codex-lone-tower -n citadel"); got != "kubectl describe pod etcd-lone-tower -n kube-system" {
		t.Fatalf("got %q", got)
	}
}

func TestStoryTextTranslatesAManifestWholeAndLeavesJSON(t *testing.T) {
	if got := StoryText("apiVersion: v1\nkind: Pod\nspec:\n  containers: []\n"); got != "book: realm/v1\ndecree: hut\nplan:\n  workers: []\n" {
		t.Errorf("a manifest read %q", got)
	}
	if doc := "{\"kind\": \"Pod\"}"; StoryText(doc) != doc {
		t.Errorf("a JSON document was half translated: %q", StoryText(doc))
	}
}

func TestStoryTextNamesEveryWorkloadAndStorageReference(t *testing.T) {
	for k8s, want := range map[string]string{
		"daemonset.apps/warden created":                                "Shrine/warden created",
		"statefulset.apps/archive created":                             "Order/archive created",
		"the daemonset/warden rolls out":                               "the shrine/warden rolls out",
		"persistentvolume/vault-1 created":                             "Vault/vault-1 created",
		"persistentvolumeclaim/books created":                          "Deed/books created",
		"storageclass.storage.k8s.io/vault-wrights created":            "Vault kind/vault-wrights created",
		"replicaset.apps/porters scaled":                               "Shift/porters scaled",
		"horizontalpodautoscaler.autoscaling/clerks autoscaled":        "Horn/clerks horned",
		"priorityclass.scheduling.k8s.io/royal created":                "Rank/royal created",
		"service/parcels exposed":                                      "Depot/parcels exposed",
		"configmap/charter created":                                    "Scroll/charter created",
		"namespace/east created":                                       "Fief/east created",
		`Error from server (NotFound): replicasets.apps "x" not found`: `Error from server (NotFound): shifts "x" not found`,
	} {
		got := StoryText(k8s)
		if got != want || KubernetesWords.MatchString(got) {
			t.Errorf("StoryText(%q) = %q, want %q", k8s, got, want)
		}
	}
}

func TestStoryTextNamesWardsAndSchedulerReasons(t *testing.T) {
	for k8s, want := range map[string]string{
		"0/2 nodes are available: 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }, 1 node(s) had untolerated taint {node.kubernetes.io/memory-pressure: }.":                                                                                                                                "0/2 Keeps are available: 1 Keep behind the Royal Ward, 1 behind the Crowding Ward.",
		"0/7 nodes are available: 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }, 1 node(s) were unschedulable, 2 node(s) had untolerated taint {node.kubernetes.io/unreachable: }, 3 Insufficient cpu. preemption: 0/7 nodes are available: 7 Preemption is not helpful for scheduling.": "0/7 Keeps are available: 1 Keep behind the Royal Ward, 1 Keep was cordoned, 2 Keeps behind the Storm Ward, 3 Keeps short of food. Displacement: 0/7 Keeps are available: 7 Keeps hold no lower Rank to displace.",
		"The Stewards warded lone-tower with node.kubernetes.io/not-ready: its Reeve is not ready":                                                             "The Stewards warded lone-tower with the Unready Ward: its Reeve is not ready",
		"The Chancery warded lone-tower with node.kubernetes.io/not-ready as it joined: its Reeve is not ready (TaintNodesByCondition admission on new nodes)": "The Chancery warded lone-tower with the Unready Ward as it joined: its Reeve is not ready",
		"app=porters,pod-template-hash=678dcc6569":                                                "app=porters,hut-pattern=678dcc6569",
		"2 Keeps are cordoned: no new huts (spec.unschedulable)":                                  "2 Keeps are cordoned: no new huts",
		"Read what it said before it stopped: kubectl logs <pod> --previous":                      "Read what it said before it stopped: speakstctl logs <hut> --last-life",
		"The Chancery opens its doors (kubeadm init: the API server answers at 172.18.0.2:6443).": "The Chancery opens its doors.",
	} {
		got := StoryText(k8s)
		if got != want || KubernetesWords.MatchString(got) {
			t.Errorf("StoryText(%q)\n = %q\nwant %q", k8s, got, want)
		}
	}
	if got := CommandFromStory("speakstctl logs porter-a --last-life"); got != "kubectl logs porter-a --previous" {
		t.Errorf("the story flag reads back as %q", got)
	}
}

func TestStoryTextConvertsAmountsBeforeClosingPunctuation(t *testing.T) {
	got := StoryText("[or type: kubectl set resources deployment porters --requests=cpu=100m]")
	if got != "[or type: speakstctl set resources crew porters --pledge=food=100]" {
		t.Fatalf("got %q", got)
	}
}

func TestStoryTextMapsAFlagBeforeClosingPunctuation(t *testing.T) {
	if got := StoryText("[or type: kubectl drain stone-keep-1 --ignore-daemonsets]"); got != "[or type: speakstctl drain stone-keep-1 --ignore-shrines]" {
		t.Fatalf("got %q", got)
	}
}

func TestAPatchBodyReadsInTheRealmsKeysAndBack(t *testing.T) {
	k8s := `kubectl patch hpa porters -p '{"spec":{"minReplicas":2,"maxReplicas":40}}'`
	story := StoryText(k8s)
	if story != `speakstctl patch horns porters -p '{"plan":{"floor":2,"ceiling":40}}'` {
		t.Fatalf("story: %q", story)
	}
	if back := CommandFromStory(story); back != k8s {
		t.Fatalf("back: %q", back)
	}
}

func TestAnInnerHallCommandReadsInTheRealmsWordsAndBack(t *testing.T) {
	k8s := `kubectl create service clusterip archive --clusterip="None"`
	story := StoryText(k8s)
	if story != `speakstctl create depot inner archive --inner-door="None"` {
		t.Fatalf("story command = %q", story)
	}
	if back := CommandFromStory(story); back != k8s {
		t.Errorf("reads back as %q", back)
	}
}

func TestStoryTextNamesKeepTidingsAndHidesTheReevesShelf(t *testing.T) {
	for k8s, want := range map[string]string{
		"Keep lone-tower status is now: NodeHasSufficientMemory":                                                                  "Keep lone-tower status is now: KeepHasRoom",
		"Keep lone-tower status is now: NodeNotReady":                                                                             "Keep lone-tower status is now: KeepNotReady",
		"The Reeve of Lone Tower runs porter-lone-tower from its own shelf (/etc/kubernetes/manifests/porter.yaml, a static Pod)": "The Reeve of Lone Tower runs porter-lone-tower from its own shelf",
	} {
		got := StoryText(k8s)
		if got != want || KubernetesWords.MatchString(got) {
			t.Errorf("StoryText(%q) = %q, want %q", k8s, got, want)
		}
	}
	for _, leak := range []string{"replicaset.apps/porters", "Normal NodeHasSufficientPID", "/etc/kubernetes/scripts/porter.yaml"} {
		if !KubernetesWords.MatchString(leak) {
			t.Errorf("the leak check misses %q", leak)
		}
	}
}

func TestThreeWardsInARowNameTheKeepOnce(t *testing.T) {
	raw := "0/3 nodes are available: 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }, 1 node(s) had untolerated taint {node.kubernetes.io/memory-pressure: }, 1 node(s) had untolerated taint {node.kubernetes.io/not-ready: }."
	want := "0/3 Keeps are available: 1 Keep behind the Royal Ward, 1 behind the Crowding Ward, 1 behind the Unready Ward."
	if got := StoryText(raw); got != want {
		t.Fatalf("got %q", got)
	}
}

func TestTheV137TaintReasonReadsAWardWithoutItsName(t *testing.T) {
	for k8s, want := range map[string]string{
		"0/2 nodes are available: 1 node(s) had untolerated taint(s), 1 node(s) were unschedulable. preemption: 0/2 nodes are available: 2 Preemption is not helpful for scheduling.": "0/2 Keeps are available: 1 Keep behind a Ward, 1 Keep was cordoned. Displacement: 0/2 Keeps are available: 2 Keeps hold no lower Rank to displace.",
		"0/3 nodes are available: 2 node(s) had untolerated taint(s), 1 Insufficient cpu.":                                                                                            "0/3 Keeps are available: 2 Keeps behind Wards, 1 Keep short of food.",
	} {
		got := StoryText(k8s)
		if got != want || KubernetesWords.MatchString(got) {
			t.Errorf("StoryText(%q) = %q, want %q", k8s, got, want)
		}
	}
}
