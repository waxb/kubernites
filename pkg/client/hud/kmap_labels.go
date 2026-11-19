package hud

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	plaqueTextX    = 40
	plaqueMarkW    = 12
	plaqueLine1Y   = 9
	plaqueLine2Y   = 22
	PlaqueTextW    = PlaqueW - plaqueTextX - plaqueMarkW
	docsPrefix     = "https://"
	FogTitle       = "?  ?  ?"
	FogTag         = "unrevealed"
	kmapTitle      = "GRIMOIRE"
	kmapLearnedFmt = "LEARNED %d/%d"
)

type plaqueLabel struct {
	Title, Tag, Docs string `json:"-"`
}

var PlaqueLabels = map[string]plaqueLabel{
	"daemons":        {"Shrines", "DaemonSet", "https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/"},
	"vaults":         {"Vaults", "PV and PVC", "https://kubernetes.io/docs/concepts/storage/persistent-volumes/"},
	"stateful-crews": {"Stateful Crews", "StatefulSet", "https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/"},
	"vault-wrights":  {"Vault Guild", "StorageClass", "https://kubernetes.io/docs/concepts/storage/storage-classes/"},
	"speaking-stone": {"Speaking Stone", "kubectl", "https://kubernetes.io/docs/reference/kubectl/"},
	"marks":          {"Badges", "Labels", "https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/"},
	"scriptorium":    {"Scriptorium", "YAML manifests", "https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/"},
	"augury":         {"Augury", "metrics-server", "https://kubernetes.io/docs/tasks/debug/debug-cluster/resource-metrics-pipeline/"},
	"pledges":        {"Pledges", "Requests", "https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/"},
	"keeps":          {"Keeps", "Nodes", "https://kubernetes.io/docs/concepts/architecture/nodes/"},
	"war-horns":      {"War Horns", "HPA", "https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/"},
	"bindings":       {"Bindings", "Limits", "https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/"},
	"masonry":        {"Masonry", "Instance types", "https://kubernetes.io/docs/reference/labels-annotations-taints/#nodekubernetesioinstance-type"},
	"charters":       {"Charters", "Zones", "https://kubernetes.io/docs/concepts/scheduling-eviction/topology-spread-constraints/"},
	"familiars":      {"Familiars", "Sidecars", "https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/"},
	view.RankPageID:  {"Rank", "PriorityClass", view.RankDocs},
	"keep-floor":     {"Cellars", "Ephemeral", "https://kubernetes.io/docs/concepts/storage/ephemeral-storage/"},
	"palimpsest":     {"Palimpsest", "etcd defrag", "https://kubernetes.io/docs/tasks/administer-cluster/configure-upgrade-etcd/#cluster-defragmentation"},
	view.WatchPage:   {"Shutters", "Readiness", "https://kubernetes.io/docs/concepts/workloads/pods/probes/"},
	view.BellPage:    {"Watchman's Bell", "Liveness", "https://kubernetes.io/docs/concepts/workloads/pods/probes/"},
}

func LabelOf(p view.GrimoirePage) plaqueLabel {
	if l, ok := PlaqueLabels[p.ID]; ok {
		return l
	}
	return plaqueLabel{Title: shortName(p.Name), Tag: shortK8s(p.K8s)}
}

func shortName(name string) string {
	name = strings.TrimPrefix(name, "The ")
	for _, cut := range []string{" & ", " and ", " of "} {
		name, _, _ = strings.Cut(name, cut)
	}
	return name
}

func shortK8s(k8s string) string {
	for _, cut := range []string{" (", " and "} {
		k8s, _, _ = strings.Cut(k8s, cut)
	}
	return k8s
}

func apiGroup(school string) string {
	return strings.TrimSuffix(school, ".k8s.io")
}

func PlaqueLines(p view.GrimoirePage, fog, trueNames bool) (string, string) {
	l := LabelOf(p)
	first, second := l.Title, l.Tag
	switch {
	case fog:
		first, second = FogTitle, FogTag
	case trueNames:
		first, second = l.Tag, apiGroup(p.School)
	case ui.StoryNaming():
		second = ""
	}
	n := ui.MaxChars(PlaqueTextW, 1)
	return ui.Clip(strings.ToUpper(first), n), ui.Clip(strings.ToUpper(second), n)
}

func DocsText(p view.GrimoirePage) string {
	return strings.TrimPrefix(LabelOf(p).Docs, docsPrefix)
}

func learnedCount(pages []view.GrimoirePage) (int, int) {
	learned, shown := 0, 0
	for _, p := range pages {
		if p.State == "hidden" {
			continue
		}
		shown++
		if p.State == "unlocked" {
			learned++
		}
	}
	return learned, shown
}

func learnedText(pages []view.GrimoirePage) string {
	learned, shown := learnedCount(pages)
	return fmt.Sprintf(kmapLearnedFmt, learned, shown)
}

func romanOf(i int) string {
	romans := []string{"I", "II", "III", "IV", "V", "VI"}
	if i < len(romans) {
		return romans[i]
	}
	return fmt.Sprint(i + 1)
}
