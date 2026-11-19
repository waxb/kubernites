package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	KeyDescribe        = "describe"
	KeyTop             = "top"
	KeyLabel           = "label"
	KeyTaint           = "taint"
	KeyDrain           = "drain"
	KeyCordon          = "cordon"
	KeyUncordon        = "uncordon"
	KeyLogs            = "logs"
	KeyPrevious        = "previous"
	KeyYAML            = "yaml"
	KeyDelete          = "delete"
	keyEndpoints       = "endpoints"
	KeyStatus          = "status"
	KeyAlarms          = "alarms"
	KeyCompact         = "compact"
	KeyDefrag          = "defrag"
	KeyDisarm          = "disarm"
	keyGetPods         = "get-pods"
	keyEvents          = "events"
	KeyGet             = "get"
	KeyEtcdVer         = "etcd-version"
	keyVersion         = "version"
	keyCluster         = "cluster-info"
	keyResources       = "api-resources"
	keyTopPods         = "top-pods"
	keyTopNodes        = "top-nodes"
	keySummon          = "summon"
	summonClass        = "porter:1.0"
	NoMetricsTip       = "The Omens are silent: learn Augury in the Grimoire so the Observatory (metrics-server) reads them."
	etcdVersionCommand = "etcdctl version"
	StaticDelete       = "A static Pod: its Keep runs it from a file (/etc/kubernetes/manifests). Deleting the mirror only makes the Keep write it again."
)

func KeepActions(s view.Structure, f Frame) []CardAction {
	node := s.Object.Name
	n := f.Nodes[node]
	cordon := CardAction{Key: KeyCordon, Label: "CORDON", Command: "cordon " + node, Tip: "Mark the Keep unschedulable: no new hut is placed here (kubectl cordon)."}
	if n.Unschedulable {
		cordon = CardAction{Key: KeyUncordon, Label: "UNCORDON", Command: "uncordon " + node, Tip: "Open the Keep to new huts again (kubectl uncordon)."}
	}
	drain := CardAction{Key: KeyDrain, Label: "DRAIN", Command: drainCommand(node, f), Tip: "Cordon the Keep and evict its huts so they move elsewhere (kubectl drain).",
		Act: view.Action{Action: view.ActDrainKeep, ID: node}}
	if !RuneKnown(f, drainRune) {
		drain.Disabled, drain.Tip = true, drainLockTip
	}
	top := CardAction{Key: KeyTop, Label: "TOP", Command: "top node " + node, Details: detailOutput, Tip: "The food and housing this Keep uses right now (kubectl top node)."}
	if !f.Metrics.Augury {
		top.Disabled, top.Tip = true, NoMetricsTip
	}
	return withWarden(withTowerSummon([]CardAction{
		{Key: KeyDescribe, Label: "DESCRIBE", Command: "describe node " + node, Details: DetailDescribe, Tip: "kubectl describe: its capacity, its Wards, its huts and what they pledge"},
		cordon, drain,
		{Key: KeyTaint, Label: "TAINT", Tip: "Set or lift a Ward (taint): huts without a Charm for it keep away.", Choices: TaintChoices(node, n)},
		{Key: KeyLabel, Label: "LABEL", Tip: "Set or remove a badge (label) on this Keep.", Choices: NodeLabelChoices(node, n)},
		top,
	}, node, f), node, f)
}

func drainCommand(node string, f Frame) string {
	if k, ok := keepViewOf(f, node); ok && k.DrainCommand != "" {
		return strings.TrimPrefix(k.DrainCommand, "kubectl ")
	}
	return "drain " + node + " --ignore-daemonsets"
}

func withTowerSummon(acts []CardAction, node string, f Frame) []CardAction {
	if node != view.TowerName || !f.Revealed("summon") {
		return acts
	}
	summon := CardAction{Key: keySummon, Label: "HIRE", Command: "run porter-<name> --image=" + summonClass + " --labels=app=porters",
		Tip: "HIRE (kubectl run): one bare porter hut that no crew keeps. " + summonCost, Act: view.Action{Action: "summon", Soulstone: summonClass}}
	return append(append(append([]CardAction(nil), acts[:1]...), summon), acts[1:]...)
}

func deleteHutTip(p model.PodView) string {
	if strings.HasPrefix(p.Owner, "ReplicaSet/") {
		return "Delete this hut. Its shift raises a new one to keep its count (kubectl delete)."
	}
	return "Delete this hut. Nobody raises a bare hut again (kubectl delete)."
}

func HearthActions(p model.PodView) []CardAction {
	ns := nsFlag(p.Namespace)
	if p.Namespace == defaultNS {
		ns = ""
	}
	del := CardAction{Key: KeyDelete, Label: "DELETE", Command: "delete pod " + p.Name + ns, Tip: deleteHutTip(p)}
	switch {
	case p.Namespace == systemNS:
		del.Disabled, del.Tip = true, citadelLockTip
	case p.Static:
		del.Tip = StaticDelete
	}
	previous := CardAction{Key: KeyPrevious, Label: "PREVIOUS", Command: "logs " + p.Name + ns + " --previous", Details: DetailLogs,
		Tip: "kubectl logs --previous: what the worker said before its last restart"}
	if p.Restarts == 0 {
		previous.Disabled, previous.Tip = true, "It has never restarted: there is no previous container to read (kubectl logs --previous)."
	}
	label := CardAction{Key: KeyLabel, Label: "LABEL", Tip: "Set or remove a badge (label). The depot sends boxes to huts wearing its app label.", Choices: PodLabelChoices(p, ns)}
	if p.Namespace == systemNS {
		label.Disabled, label.Tip, label.Choices = true, "The Citadel's own huts (kube-system) keep their badges.", nil
	}
	return []CardAction{
		{Key: KeyDescribe, Label: "DESCRIBE", Command: "describe pod " + p.Name + ns, Tip: "kubectl describe: its state, its Keep and its tidings", Details: DetailDescribe},
		{Key: KeyLogs, Label: "LOGS", Command: "logs " + p.Name + ns, Tip: "kubectl logs: what the worker inside says", Details: DetailLogs},
		del, label, previous,
		{Key: KeyYAML, Label: "YAML", Command: "get pod " + p.Name + ns + " -o yaml", Details: DetailYAML,
			Tip: "kubectl get -o yaml: its whole spec and status, the way to look inside without exec"},
	}
}

func DepotActions(service string) []CardAction {
	closeDepot := "delete service " + service
	return []CardAction{
		{Key: KeyDescribe, Label: "DESCRIBE", Command: "describe service " + service, Details: DetailDescribe, Tip: "kubectl describe service: its selector, ports and endpoints"},
		{Key: keyEndpoints, Label: ui.Bridge("ROLLS", "SLICES"), Command: "get endpointslices -l kubernetes.io/service-name=" + service, Details: detailOutput,
			Tip: "The depot's Guild Roll (EndpointSlice): every hut it may send a box to"},
		{Key: KeyDelete, Label: "DELETE", Tip: "Close this depot (delete its Service). Its boxes wait on the quay until a depot opens again.",
			Choices: []CardAction{{Key: closeDepot, Label: "Yes, close the " + service + " depot", Command: closeDepot, Tip: "kubectl " + closeDepot}}},
	}
}

func CodexActions(f Frame) []CardAction {
	c := f.State.Codex
	disarm := CardAction{Key: KeyDisarm, Label: "DISARM", Command: view.DisarmCommand, Tip: "Clear the NOSPACE alarm once the Codex is rebound (etcdctl alarm disarm)."}
	if !c.Alarm {
		disarm.Disabled, disarm.Tip = true, "No alarm is raised: the Codex accepts writes (etcdctl alarm list)."
	}
	compact := CardAction{Key: KeyCompact, Label: "COMPACT", Command: view.CompactCommand(c.Revision),
		Tip: "Scrape every revision older than the current one (etcdctl compact). The file keeps its size until a defrag."}
	if c.Revision <= c.CompactRevision {
		compact.Disabled, compact.Tip = true, "Nothing to compact: the Codex is compacted to its current revision."
	}
	status := CardAction{Key: KeyStatus, Label: "STATUS", Command: view.StatusCommand, Details: detailOutput, Tip: "The Codex's size, quota, revision and alarms (etcdctl endpoint status)"}
	alarms := CardAction{Key: KeyAlarms, Label: "ALARMS", Command: view.AlarmsCommand, Details: detailOutput, Tip: "Is an alarm raised? (etcdctl alarm list)"}
	defrag := CardAction{Key: KeyDefrag, Label: "DEFRAG", Command: view.DefragCommand, Tip: "Rebind the Codex: write it afresh and give back the space old pages hold (etcdctl defrag). It blocks for a moment."}
	version := CardAction{Key: KeyEtcdVer, Label: "VERSION", Command: etcdVersionCommand, Details: detailOutput, Tip: "Which etcd keeps the Codex (etcdctl version)"}
	if c.Alarm {
		return []CardAction{defrag, disarm, status, alarms, compact, version}
	}
	return []CardAction{status, alarms, compact, defrag, disarm, version}
}

func ChanceryActions() []CardAction {
	return []CardAction{
		{Key: keyGetPods, Label: "GET PODS", Command: "get pods", Details: detailOutput, Tip: "Ask the Chancery what runs (kubectl get pods)"},
		{Key: KeyGet, Label: "GET", Tip: "Ask the Chancery for any kind it keeps (kubectl get).", Choices: GetChoices()},
		{Key: keyEvents, Label: "EVENTS", Command: "get events", Details: detailOutput, Tip: "What the realm's servants report: the tidings panel is this list (kubectl get events)"},
		{Key: keyVersion, Label: "VERSION", Command: "version", Details: detailOutput, Tip: "Which Kubernetes the Chancery speaks (kubectl version)"},
		{Key: keyCluster, Label: "CLUSTER", Command: "cluster-info", Details: detailOutput, Tip: "Where the Chancery and the Herald answer (kubectl cluster-info)"},
		{Key: keyResources, Label: "RESOURCES", Command: "api-resources", Details: detailOutput, Tip: "Every kind the Chancery knows (kubectl api-resources)"},
	}
}

var ListedKinds = []struct{ plural, story string }{
	{"nodes", "the Keeps"}, {"deployments", "the crews"}, {"replicasets", "the shifts"}, {"services", "the depots"},
	{"endpointslices", "the Guild Rolls"}, {"namespaces", "the Fiefs"}, {"hpa", "the War Horns"},
	{"configmaps", "the Scrolls"}, {"priorityclasses", "the Ranks"}, {"pods -A", "every hut"},
}

func GetChoices() []CardAction {
	out := make([]CardAction, 0, len(ListedKinds))
	for _, k := range ListedKinds {
		cmd := "get " + k.plural
		out = append(out, CardAction{Key: cmd, Label: cmd + "  (" + k.story + ")", Command: cmd, Details: detailOutput, Tip: "kubectl " + cmd})
	}
	return out
}

func ObservatoryActions(f Frame) []CardAction {
	out := []CardAction{
		{Key: keyTopPods, Label: "TOP PODS", Command: "top pods", Details: detailOutput, Tip: "The food and housing each worker uses (kubectl top pods)"},
		{Key: keyTopNodes, Label: "TOP NODES", Command: "top nodes", Details: detailOutput, Tip: "The food and housing each Keep uses (kubectl top nodes)"},
	}
	if f.Metrics.Augury {
		return out
	}
	for i := range out {
		out[i].Disabled, out[i].Tip = true, NoMetricsTip
	}
	return out
}

func hasTaint(n model.NodeView, spec string) bool {
	return slices.Contains(n.TaintSpecs, spec)
}

func wardChoice(node, effect, label string, n model.NodeView) CardAction {
	spec := wardKey + "=true:" + effect
	a := CardAction{Key: spec, Label: label, Command: fmt.Sprintf("taint node %s %s", node, spec), Tip: "kubectl taint node " + node + " " + spec}
	if hasTaint(n, spec) {
		a.Disabled, a.Tip = true, "This Ward already stands on "+node+"."
	}
	return a
}
