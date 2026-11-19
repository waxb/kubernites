package names

import (
	"strings"

	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

type bridge struct {
	story string
	k8s   string
}

var bridges = []bridge{
	{"Across the sea. Each trade climbs through its classes: endow a class once, and every hut of that trade can take it.",
		"Container images and their tags. Buy a tag once; every Pod that runs it pulls it when it starts."},
	{"The Gate is where the harbor's boxes enter the realm. Each box is a request. ",
		"The Gate is where the harbor's requests enter the cluster. "},
	{"Each isle stands on its own: a storm over one leaves the others standing.",
		"Each zone is a separate failure domain (topology.kubernetes.io/zone): an outage in one leaves the others standing."},
	{"Boxes are requests. Ships land them on your quay, where they wait.",
		"Ships land requests on your queue, where they wait."},
	{"Carry one and it pays gold. Leave one too long and its seal turns red:",
		"Carry one and it pays money. Leave one waiting too long and it times out:"},
	{"it spoils, a dropped request. The share delivered fresh is your reliability.",
		"a dropped request. The share delivered on time is your reliability."},
	{"is placed on a Keep by the Quartermaster", "is scheduled onto a Node"},
	{"are placed on Keeps by the Quartermaster", "are scheduled onto Nodes"},
	{"waits for its trade to arrive from overseas", "is pulling its image"},
	{"wait for their trade to arrive from overseas", "are pulling their image"},
	{"has received its trade", "has pulled its image"},
	{"have received their trade", "have pulled their image"},
	{"gets its worker", "has its container created"},
	{"get their worker", "have their container created"},
	{"is Burning: up and ready for boxes", "is Running"},
	{"are Burning: up and ready for boxes", "are Running"},
	{"is shutting down (Last Rites)", "is Terminating"},
	{"are shutting down (Last Rites)", "are Terminating"},
	{"is Guttering: its worker stops and is relit again and again", "is in CrashLoopBackOff, restarting"},
	{"are Guttering: their workers stop and are relit again and again", "are in CrashLoopBackOff, restarting"},
	{"is Unfit for Duty: not ready for boxes", "is not Ready"},
	{"are Unfit for Duty: not ready for boxes", "are not Ready"},
	{"was toppled by the Simian Imp", "was killed by the chaos monkey"},
	{"were toppled by the Simian Imp", "were killed by the chaos monkey"},
	{"is Voidtaken: killed for using too much housing", "is OOMKilled"},
	{"are Voidtaken: killed for using too much housing", "are OOMKilled"},
	{"is Banished: turned out of a crowded Keep", "is Evicted"},
	{"are Banished: turned out of a crowded Keep", "are Evicted"},
	{"is ready to host huts", "is Ready"},
	{"has gone dark: it hosts no new huts", "is NotReady"},
	{"cannot get started", "cannot create its Pod sandbox"},
	{"has finished its Quest", "has Completed"},
	{"have finished their Quest", "have Completed"},
	{"is pushed out to make room for a hut of higher Rank", "is preempted by a higher-priority Pod"},
	{"are pushed out to make room for huts of higher Rank", "are preempted by higher-priority Pods"},
	{"--image=<trade:class>", "--image=<image:tag>"},
	{"[--name=<horn>]", "[--name=<hpa>]"},
	{"The Codex answers etcdctl on the same Stone:", "etcd answers etcdctl in the same console:"},
	{"The realm's book: every hut, crew and order you give is written in it; what is not written did not happen.", "The cluster's database: every Pod, Deployment and order you give is stored in it; what is not stored did not happen."},
	{"Hire a porter", "Run a porter Pod"},
	{"Every hut has a room.", "Every Pod has room."},
	{"A Rank reaches a crew's huts through its rollout. When quarters run out, a higher Rank may displace a lower one.",
		"A PriorityClass reaches a Deployment's Pods through its rollout. When a Node is full, a higher PriorityClass may preempt a lower one."},
	{"The Augur sees nothing until the Observatory (metrics-server) stands.", "No metrics until metrics-server runs."},
	{"what the worker inside says", "what the container inside says"},
	{"what the worker said before its last restart", "what the container said before its last restart"},
	{"raises a hut", "creates a Pod"},
	{"sends a hut home", "deletes a Pod"},
}

func allBridges() []bridge {
	content := view.Content.NamesBridges()
	out := make([]bridge, 0, len(bridges)+len(content))
	out = append(out, bridges...)
	for _, b := range content {
		out = append(out, bridge{story: b.Story, k8s: b.K8s})
	}
	return out
}

func newBridger() *strings.Replacer {
	all := allBridges()
	pairs := make([]string, 0, 2*len(all))
	for _, b := range all {
		pairs = append(pairs, b.story, b.k8s)
	}
	return strings.NewReplacer(pairs...)
}

func Kubernetes(cache int) func(string) string {
	named, bridger := lexicon.TrueNamer(cache), newBridger()
	return func(text string) string { return named(bridger.Replace(text)) }
}
