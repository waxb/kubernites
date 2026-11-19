package view

type Liberty struct {
	Area  string
	Bends string
}

const LibertyApply = "Apply"

const LibertyScheduler = "Scheduler"

const LibertyCPU = "CPU"

const LibertyKubelet = "Kubelet"

const LibertyFamiliars = "Familiars"

const LibertyBoxes = "Boxes in flight"

const LibertyAllocatable = "Allocatable"

const LibertySystemPods = "System pods"

const LibertyNodes = "Node lifecycle"

const LibertyDrain = "Drain"

const LibertyEndpoints = "Endpoints"

const LibertyChaos = "Chaos"

const LibertyRunes = "Runes"

const LibertyMetrics = "Metrics"

const LibertyHPA = "War Horn (HPA)"

const LibertyRoyalWard = "Control-plane taint"

const LibertySeasoned = "Seasoned crews"

const LibertyDoor = "The door's porter"

var Liberties = map[string]string{
	LibertyApply: "Here apply replaces the fields you send. Real kubectl apply merges three ways with the last applied " +
		"configuration, and the server tracks who owns each field (managedFields).",
	LibertyScheduler: "The Quartermaster runs only a few scheduler plugins: taints, nodeSelector, resource fit (LeastAllocated) " +
		"and topology spread. Real kube-scheduler also weighs node and Pod affinity, balanced allocation and image locality.",
	LibertyCPU: "Food is shared as a weighted max-min fair share of the Keep's allocatable CPU, an approximation of CFS " +
		"shares. Real throttling happens in 100 ms CFS periods.",
	LibertyKubelet: "Each hut runs one app container plus its sidecars. Image pulls and start times are fixed per class, " +
		"and probes are never really called: their outcome follows the class's warm-up.",
	LibertyFamiliars: "Only sidecars, init containers with restartPolicy: Always, are accepted. Init containers that run " +
		"to completion are refused until a later Age.",
	LibertyBoxes: "When a hut falls, the boxes it was carrying are dropped and a Hound leads them back. In a real mesh the " +
		"caller's proxy or the gateway retries them; the dying Pod's own sidecar can only drain.",
	LibertyAllocatable: "A Keep's allocatable is its capacity minus 100m CPU and 512Mi memory. A real Node subtracts " +
		"kube-reserved, system-reserved and the hard eviction threshold.",
	LibertySystemPods: "The control plane runs as kubeadm's static Pods with their mirror Pods, but its manifests carry no command, " +
		"volumes or probes. Outside LEARN KUBERNETES the Speaking Stone refuses to change kube-system.",
	LibertyNodes: "Keeps join NotReady, then turn Ready after a fixed delay with no Pod network add-on: a kubeadm Node stays " +
		"NotReady until one runs, as LEARN KUBERNETES shows. Heartbeats are not Lease objects in kube-node-lease.",
	LibertyDrain: "There are no PodDisruptionBudgets yet, so an eviction is never refused with 429, and drain does not wait " +
		"for the Pods to leave.",
	LibertyEndpoints: "Only EndpointSlices are served; the deprecated v1 Endpoints API is not.",
	LibertyChaos:     "The Simian Imp deletes running Pods gracefully, the way a chaos tool would.",
	LibertyRunes: "Kinds and fields such as ReplicaSet and resources.requests open with Grimoire pages. That is a game rule, " +
		"not Kubernetes.",
	LibertyMetrics: "The Augur reports a Pod's CPU as the rate between its last two scrapes, 15 s apart, while kubectl top " +
		"shows the usage of this very moment.",
	LibertyHPA: "A War Horn reads only CPU Resource metrics, on Deployments and ReplicaSets, and forgets its " +
		"recommendations on save and load, like a restarted kube-controller-manager.",
	LibertyRoyalWard: "The Lone Tower starts without kubeadm's node-role.kubernetes.io/control-plane:NoSchedule taint, as if " +
		"you had removed it to run work on a one-Node cluster, and the game puts it back once a second Keep stands. " +
		"Kubernetes never puts it back by itself: kubeadm sets it once, at init.",
	LibertyDoor: "In the campaign the door's porter is a static Pod on the Pod network that holds hostPort 80, so it has a Pod " +
		"IP of its own. In the chapters it runs on the host network at the Node's port 80 and shares the Node's IP.",
	LibertySeasoned: "A crew's ready huts carry x2 as fast at 5 ready, x3 at 10 and x4.5 at 25, until the count falls " +
		"below half that mark. This is a game bonus: in Kubernetes a Pod serves no faster for having more replicas.",
}

var chapterLiberties = map[string]string{
	LibertyNodes:     "Heartbeats are not Lease objects in kube-node-lease.",
	LibertyRoyalWard: "",
	LibertyDoor:      "",
}

func ChapterBends(bends []Liberty) []Liberty {
	out := []Liberty{}
	for _, l := range bends {
		text, own := chapterLiberties[l.Area]
		switch {
		case !own:
			out = append(out, l)
		case text != "":
			out = append(out, Liberty{Area: l.Area, Bends: text})
		}
	}
	return out
}

func LibertyOf(area string) (Liberty, bool) {
	text, ok := Liberties[area]
	return Liberty{Area: area, Bends: text}, ok
}
