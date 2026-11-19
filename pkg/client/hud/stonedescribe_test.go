package hud

import (
	"regexp"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

const describeNodeRaw = `Name:               lone-tower
Roles:              control-plane
Labels:             kubernetes.io/arch=amd64
                    kubernetes.io/hostname=lone-tower
                    kubernetes.io/os=linux
                    kubernites.dev/keep-type=tower
                    node-role.kubernetes.io/control-plane=
                    node.kubernetes.io/instance-type=general-large
                    topology.kubernetes.io/zone=isle-hearthmoor
Annotations:        kubernites.dev/ready-at: 3.00
CreationTimestamp:  Mon, 01 Jan 0001 00:00:00 +0000
Taints:             node-role.kubernetes.io/control-plane:NoSchedule
                    dedicated=x:NoExecute
Unschedulable:      false
Conditions:
  Type             Status   LastHeartbeatTime                 LastTransitionTime                Reason                       Message
  ----             ------   -----------------                 ------------------                ------                       -------
  MemoryPressure   False    Mon, 01 Jan 0001 00:00:10 +0000   Mon, 01 Jan 0001 00:00:10 +0000   KubeletHasSufficientMemory   kubelet has sufficient memory available
  PIDPressure      False    Mon, 01 Jan 0001 00:00:10 +0000   Mon, 01 Jan 0001 00:00:10 +0000   KubeletHasSufficientPID      kubelet has sufficient PID available
  Ready            True     Mon, 01 Jan 0001 00:00:03 +0000   Mon, 01 Jan 0001 00:00:03 +0000   KubeletReady                 kubelet is posting ready status
Addresses:
  InternalIP:  172.18.0.2
  Hostname:    lone-tower
Capacity:
  cpu:     2
  memory:  4Gi
System Info:
  Machine ID:    abc
  Kernel Version:  6.1
Events:
  Type    Reason     Age   From     Message
  ----    ------     ----  ----     -------
  Normal  NodeReady  42s   kubelet  Node lone-tower status is now: NodeReady
`

const describePodRaw = `Name:             porter-lone-tower
Namespace:        default
Priority:         0
Service Account:  <none>
Node:             lone-tower/172.18.0.2
Start Time:       Mon, 01 Jan 0001 00:00:00 +0000
Labels:           app=porters
Annotations:      kubernetes.io/config.hash: d4e113970f637624556c294002793db7
                  kubernetes.io/config.mirror: d4e113970f637624556c294002793db7
                  kubernetes.io/limit-ranger: LimitRanger plugin set: memory request for container porter
Status:           Running
Controlled By:    Node/lone-tower
Containers:
  porter:
    Image:          porter:1.0
    Port:           8080/TCP (http)
    State:          Running
      Started:      Mon, 01 Jan 0001 00:00:06 +0000
    Ready:          True
Conditions:
  Type              Status
  Initialized       True
  ContainersReady   True
  PodScheduled      True
Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             :NoExecute op=Exists
Events:                      <none>
`

const describeDeployRaw = `Name:                   porters
Namespace:              default
Selector:               app=porters
Replicas:               2 desired | 1 updated | 3 total | 2 available | 1 unavailable
StrategyType:           RollingUpdate
MinReadySeconds:        0
RollingUpdateStrategy:  25% max unavailable, 25% max surge
Conditions:
  Type           Status  Reason
  ----           ------  ------
  Available      True    MinimumReplicasAvailable
  Progressing    True    ReplicaSetUpdated
OldReplicaSets:  porters-59fcdcb77f (2/2 replicas created)
NewReplicaSet:   porters-58bf7cf7fb (1/1 replicas created)
Events:
  Type    Reason             Age   From                   Message
  ----    ------             ----  ----                   -------
  Normal  ScalingReplicaSet  35s   deployment-controller  Scaled up replica set porters-59fcdcb77f from 0 to 2
`

var describeLeak = regexp.MustCompile(`Kubelet\w*|NoSchedule|NoExecute|PreferNoSchedule|\w+Pressure\b|PIDPressure|LimitRanger|StrategyType|RollingUpdateStrategy|MinReadySeconds|ContainersReady|PodScheduled|Initialized|Start Time|Started:|\d+\.\d+\.\d+\.\d+|reeve-(mirror|scroll)|keep-make|keep-ways|keep-name|op=Exists|Depot Account|Service Account|OldReplicaSets|NewReplicaSet|ScalingReplicaSet|ReplicaSet|MinimumReplicasAvailable|Controlled By: \w+/|Machine ID|Kernel|InternalIP|Hostname`)

func storyOut(t *testing.T, raw string) string {
	t.Helper()
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	return strings.Join(StoryConsole(SplitOutput(raw)), "\n")
}

func TestStoryDescribeLeavesNoRawFieldsOnAKeepAHutOrACrew(t *testing.T) {
	for name, raw := range map[string]string{"keep": describeNodeRaw, "hut": describePodRaw, "crew": describeDeployRaw} {
		got := storyOut(t, raw)
		if leak := describeLeak.FindString(got); leak != "" {
			t.Errorf("%s: story describe keeps %q:\n%s", name, leak, got)
		}
		if leak := lexicon.KubernetesWords.FindString(got); leak != "" {
			t.Errorf("%s: story describe keeps the Kubernetes word %q:\n%s", name, leak, got)
		}
	}
}

func TestStoryDescribeKeepsTheBadgesThatTeachAndHidesTheSystemOnes(t *testing.T) {
	got := storyOut(t, describeNodeRaw)
	for _, want := range []string{"keep-size=general-large", "isle=isle-hearthmoor", "Wards:"} {
		if !strings.Contains(got, want) {
			t.Errorf("the keep describe lost %q:\n%s", want, got)
		}
	}
	for _, hidden := range []string{"Badges:  keep-make", "citadel-keep"} {
		if strings.Contains(got, hidden) {
			t.Errorf("the keep describe shows %q:\n%s", hidden, got)
		}
	}
}

func TestStoryDescribeSaysWhatAWardAndACharmDo(t *testing.T) {
	keep := storyOut(t, describeNodeRaw)
	if !strings.Contains(keep, "the Royal Ward (bars new huts)") || !strings.Contains(keep, "dedicated=x (turns huts out)") {
		t.Errorf("the keep's Wards are not explained:\n%s", keep)
	}
	hut := storyOut(t, describePodRaw)
	if !strings.Contains(hut, "the Unready Ward (turns huts out, for 300s)") || !strings.Contains(hut, "every Ward that turns huts out") {
		t.Errorf("the hut's Charms are not explained:\n%s", hut)
	}
}

func TestStoryDescribeAlignsAWardsContinuationUnderItsValue(t *testing.T) {
	lines := strings.Split(storyOut(t, describeNodeRaw), "\n")
	col, cont := -1, -1
	for i, l := range lines {
		if strings.HasPrefix(l, "Wards:") {
			col = len(l) - len(strings.TrimLeft(l[len("Wards:"):], " "))
			cont = len(lines[i+1]) - len(strings.TrimLeft(lines[i+1], " "))
		}
	}
	if col < 0 || cont != col {
		t.Errorf("the second Ward sits at column %d, its value at %d:\n%s", cont, col, strings.Join(lines, "\n"))
	}
}

func TestStoryDescribeNamesTheConditionsAndDropsTheirCodes(t *testing.T) {
	got := storyOut(t, describeNodeRaw)
	for _, want := range []string{"Crowding", "Throng", "Ready"} {
		if !strings.Contains(got, want) {
			t.Errorf("the conditions lack %q:\n%s", want, got)
		}
	}
	crew := storyOut(t, describeDeployRaw)
	for _, want := range []string{"Enough huts ready", "Shift swapping in", "Old shifts:", "New shift:", "Roll-out style:"} {
		if !strings.Contains(crew, want) {
			t.Errorf("the crew describe lacks %q:\n%s", want, crew)
		}
	}
}

func TestStoryDescribeDropsEmptyHeadersAndOrphanLines(t *testing.T) {
	got := storyOut(t, describeNodeRaw)
	for _, l := range strings.Split(got, "\n") {
		if l == "Addresses:" || l == "System Info:" {
			t.Errorf("an empty header stays: %q", l)
		}
	}
}

func TestAYamlAnswerInStoryModeUsesTheStoryKeys(t *testing.T) {
	got := storyOut(t, "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: porters\nspec:\n  replicas: 2\n  template:\n    spec:\n      containers:\n      - name: porter\n        image: porter:1.0\nstatus:\n  conditions:\n  - status: \"True\"\n    type: Available\n")
	for _, want := range []string{"book: crews/v1", "decree: crew", "count: 2", "workers:", "trade: porter:1.0", "report:"} {
		if !strings.Contains(got, want) {
			t.Errorf("the story yaml lacks %q:\n%s", want, got)
		}
	}
	for _, old := range []string{"apiVersion", "kind:", "replicas", "containers", "Deployment"} {
		if strings.Contains(got, old) {
			t.Errorf("the story yaml keeps %q:\n%s", old, got)
		}
	}
}

const describeServiceRaw = `Name:              parcels
Namespace:         default
Labels:            app=porters
Selector:          app=porters
Type:              ClusterIP
IP:                10.96.0.11
Port:              http  80/TCP
TargetPort:        8080/TCP
Endpoints:         10.244.0.2:8080,10.244.0.5:8080,10.244.0.6:8080
Session Affinity:  None
Events:            <none>
`

func TestStoryDescribeOfADepotShowsItsDoorAndHowManyHutsAnswer(t *testing.T) {
	got := storyOut(t, describeServiceRaw)
	for _, want := range []string{"Door:", "http 80", "Hut door:", "8080", "Ready huts:", "3"} {
		if !strings.Contains(got, want) {
			t.Errorf("the depot describe lacks %q:\n%s", want, got)
		}
	}
	for _, leak := range []string{"10.244", "10.96", "TCP", "ClusterIP", "Session"} {
		if strings.Contains(got, leak) {
			t.Errorf("the depot describe keeps %q:\n%s", leak, got)
		}
	}
	none := storyOut(t, strings.Replace(describeServiceRaw, "10.244.0.2:8080,10.244.0.5:8080,10.244.0.6:8080", "<none>", 1))
	if !strings.Contains(none, "none: boxes wait at the door") {
		t.Errorf("a depot with no ready hut says nothing:\n%s", none)
	}
	many := storyOut(t, strings.Replace(describeServiceRaw, "10.244.0.2:8080,10.244.0.5:8080,10.244.0.6:8080", "10.244.0.2:8080,10.244.0.5:8080,10.244.0.6:8080 + 9 more...", 1))
	if !regexp.MustCompile(`Ready huts:\s+12\b`).MatchString(many) {
		t.Errorf("the count of a long roll is wrong:\n%s", many)
	}
}
