package model

import (
	"hash/fnv"
	"maps"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/podutil"
	"github.com/waxb/kubernites/pkg/soulstone"
)

const (
	HearthReady     = "ready"
	HearthKindling  = "kindling"
	HearthGuttering = "guttering"
	HearthGhost     = "ghost"
	HearthBanished  = "banished"
	HearthPending   = "pending"

	StaticSourceKey  = "kubernetes.io/config.source"
	StaticSourceFile = "file"
	systemNamespace  = "kube-system"
)

type PodView struct {
	Namespace     string
	Name          string
	UID           string
	Node          string
	Image         string
	Profession    string
	Sprite        string
	Tier          int
	State         string
	Reason        string
	Ready         bool
	Restarts      int32
	Terminating   bool
	Untried       bool
	System        bool
	Owner         string
	Familiars     []string
	PetNames      []string
	LastExit      string
	Static        bool
	QoS           string
	Priority      int32
	Rank          string
	People        []Kubernite
	Focus         string
	Crew          string
	Depot         string
	Labels        map[string]string
	Requests      []ContainerRequest
	Claims        []string
	NodePort      int32
	HostNetwork   bool
	RestartPolicy string
	ReadySince    float64
	DeletedAt     float64
}

func untried(p *corev1.Pod) bool {
	if p.Spec.NodeName != "" {
		return false
	}
	c := podutil.GetCondition(p.Status.Conditions, corev1.PodScheduled)
	return c == nil || c.Status != corev1.ConditionFalse
}

func HearthState(p *corev1.Pod) string {
	reason := podutil.StatusReason(p)
	switch {
	case p.Status.Phase == corev1.PodFailed || reason == "Evicted" || p.DeletionTimestamp != nil:
		return HearthBanished
	case p.Spec.NodeName == "":
		return HearthPending
	case reason == "ErrImagePull" || reason == "ImagePullBackOff":
		return HearthGhost
	case reason == "CrashLoopBackOff" || reason == "Error" || reason == "OOMKilled":
		return HearthGuttering
	case reason == "ContainerCreating" || reason == "PodInitializing" || reason == "Pending":
		return HearthKindling
	}
	return HearthReady
}

func TierOf(image string) int {
	p, ok := soulstone.Lookup(image)
	if !ok {
		return 1
	}
	switch {
	case strings.HasPrefix(p.Tag, "2."):
		return 3
	case p.Tag == "1.0":
		return 1
	case strings.HasPrefix(p.Tag, "1."):
		return 2
	}
	return 1
}

func ViewOf(p *corev1.Pod) PodView {
	image := ""
	if len(p.Spec.Containers) > 0 {
		image = p.Spec.Containers[0].Image
	}
	profession, sprite := soulstone.ProfessionOf(image)
	owner := ""
	if ref := podutil.ControllerRef(p); ref != nil {
		owner = ref.Kind + "/" + ref.Name
	}
	return PodView{
		Namespace: p.Namespace, Name: p.Name, UID: string(p.UID), Node: p.Spec.NodeName, Image: image,
		Profession: profession, Sprite: sprite, Tier: TierOf(image), State: HearthState(p), Untried: untried(p),
		Reason: podutil.StatusReason(p), Ready: podutil.IsReady(p), Restarts: podutil.RestartCount(p),
		Terminating: p.DeletionTimestamp != nil, System: p.Namespace == systemNamespace, Owner: owner,
		Familiars: FamiliarsOf(p), PetNames: FamiliarNamesOf(p), LastExit: LastExit(p), Static: IsStatic(p),
		QoS: QoSOf(p), Priority: priorityOf(p), Rank: RankOf(p.Spec.PriorityClassName), People: PeopleOf(p),
		Labels: maps.Clone(p.Labels), Requests: RequestsOf(p), Claims: ClaimsOf(p),
		NodePort: NodePortOf(p), HostNetwork: p.Spec.HostNetwork, RestartPolicy: string(p.Spec.RestartPolicy),
		ReadySince: readySince(p), DeletedAt: deletedAt(p),
	}
}

func readySince(p *corev1.Pod) float64 {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return unixOf(c.LastTransitionTime)
		}
	}
	return 0
}

func deletedAt(p *corev1.Pod) float64 {
	if p.DeletionTimestamp == nil {
		return 0
	}
	return unixOf(*p.DeletionTimestamp)
}

func NodePortOf(p *corev1.Pod) int32 {
	for _, hp := range podutil.HostPorts(p) {
		return hp.Port
	}
	if !p.Spec.HostNetwork {
		return 0
	}
	for _, c := range p.Spec.Containers {
		for _, cp := range c.Ports {
			return cp.ContainerPort
		}
	}
	return 0
}

func ClaimsOf(p *corev1.Pod) []string {
	var out []string
	for _, v := range p.Spec.Volumes {
		if c := v.PersistentVolumeClaim; c != nil {
			out = append(out, c.ClaimName)
		}
	}
	return out
}

func IsStatic(p *corev1.Pod) bool {
	return p.Namespace != systemNamespace && p.Annotations[StaticSourceKey] == StaticSourceFile
}

func (c *Cache) PodViews() []PodView {
	pods := c.Pods()
	out := make([]PodView, 0, len(pods))
	book := c.crewBook()
	for _, p := range pods {
		v := ViewOf(p)
		v.Crew, v.Depot = book.crewOf(p), book.depotOf(p)
		out = append(out, v)
	}
	return out
}

func Hash(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}
