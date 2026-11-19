package lexicon

import "testing"

var alignedCommands = []string{
	"kubectl get nodes",
	"kubectl scale deployment porters --replicas=4",
	"kubectl scale deployment/porters --replicas=4",
	"kubectl set resources deployment porters -c porter --requests=cpu=200m,memory=128Mi --limits=cpu=200m",
	"kubectl set image deployment/porters porter=porter:1.1",
	"kubectl autoscale deployment porters --min=2 --max=10 --cpu=70%",
	"kubectl expose deployment porters --name=parcels --port=80 --target-port=8080",
	"kubectl run porter --image=porter:1.0 --labels=app=porters",
	"kubectl drain stone-keep-1 --ignore-daemonsets",
	"kubectl get nodes -L topology.kubernetes.io/zone",
	"kubectl get nodes -L node.kubernetes.io/instance-type",
	"kubectl describe node lone-tower",
	"kubectl get endpointslices -l kubernetes.io/service-name=parcels",
	"kubectl logs <pod> --previous",
	"kubectl top pods",
	"kubectl create deployment clerks --image=clerk:1.0 --replicas=2 --port=8080",
	"kubectl set resources deployment/clerks --limits=ephemeral-storage=2Gi",
	"etcdctl endpoint status --write-out=table",
	"etcdctl defrag",
	"kubectl cluster-info",
}

func TestTheAlignedCommandReadsLikeTheStoryCommand(t *testing.T) {
	for _, cmd := range alignedCommands {
		tokens := AlignCommand(cmd)
		if got, want := StoryLine(tokens), StoryText(cmd); got != want {
			t.Errorf("%q\n aligned %q\n   story %q", cmd, got, want)
		}
		kube := ""
		for i, tok := range tokens {
			if i > 0 {
				kube += " "
			}
			kube += tok.Kube
		}
		if kube != cmd {
			t.Errorf("the Kubernetes side of %q lost a token: %q", cmd, kube)
		}
	}
}

func TestTheAlignedCommandPairsEachKubernetesTokenWithItsStoryWord(t *testing.T) {
	tokens := AlignCommand("kubectl scale deployment porters --replicas=4")
	want := []CommandToken{{"speakstctl", "kubectl"}, {"scale", "scale"}, {"crew", "deployment"}, {"porters", "porters"}, {"--count=4", "--replicas=4"}}
	if len(tokens) != len(want) {
		t.Fatalf("tokens = %+v", tokens)
	}
	for i, w := range want {
		if tokens[i] != w {
			t.Errorf("token %d = %+v, want %+v", i, tokens[i], w)
		}
	}
	if !tokens[2].Mapped() || tokens[1].Mapped() {
		t.Error("only changed tokens are mapped")
	}
}
