package model

import (
	"image/color"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/waxb/kubernites/pkg/sim/podutil"
	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/soulstone"
)

const (
	RevisionAnnotation    = "deployment.kubernetes.io/revision"
	ChangeCauseAnnotation = "kubernetes.io/change-cause"
	TemplateHashLabel     = appsv1.DefaultDeploymentUniqueLabelKey
	defaultRollingShare   = "25%"
	defaultHistoryLimit   = 10
)

type Resources struct {
	CPURequest int64
	MemRequest int64
	CPULimit   int64
	MemLimit   int64
}

type Foreman struct {
	Namespace   string
	Name        string
	Hash        string
	Revision    int64
	Desired     int
	Current     int
	Ready       int
	Available   int
	Image       string
	Newest      bool
	ChangeCause string
	Pods        []string
	Shutters    bool
	Bell        bool
	Pledge      string
}

type CrewChain struct {
	Namespace      string
	Name           string
	Bare           bool
	Image          string
	Container      string
	Desired        int
	Ready          int
	Updated        int
	Available      int
	Total          int
	Strategy       string
	MaxSurge       string
	MaxUnavailable string
	Paused         bool
	Uncounted      bool
	Revision       int64
	HistoryLimit   int
	Resources      Resources
	QoS            string
	Depot          string
	Horn           *Horn
	Foremen        []Foreman
	Rank           string
}

func (c *Cache) Deployments() []*appsv1.Deployment {
	return list[*appsv1.Deployment](c, store.KindDeployment)
}

func (c *Cache) CrewChains() []CrewChain {
	sets, pods, services := c.ReplicaSets(), c.Pods(), c.Services()
	horns := c.Horns()
	out := []CrewChain{}
	owned := map[string]bool{}
	for _, d := range c.Deployments() {
		if d.Namespace == systemNamespace {
			continue
		}
		chain := ChainOf(d, sets, pods)
		chain.Depot = DepotFor(services, d.Namespace, d.Spec.Template.Labels)
		chain.Horn = HornFor(horns, chain.Name)
		for _, f := range chain.Foremen {
			owned[f.Namespace+"/"+f.Name] = true
		}
		out = append(out, chain)
	}
	for _, rs := range sets {
		if !owned[rs.Namespace+"/"+rs.Name] && podutil.ControllerRef(rs) == nil && rs.Namespace == defaultNamespace {
			out = append(out, foremanCrew(BareChain(rs, pods), rs, services))
		}
	}
	return out
}

func foremanCrew(c CrewChain, rs *appsv1.ReplicaSet, services []*corev1.Service) CrewChain {
	if _, crew := soulstone.DepotOfCrew(rs.Name); !crew {
		return c
	}
	next := c
	next.Name, next.Depot = rs.Name, DepotFor(services, rs.Namespace, rs.Spec.Template.Labels)
	return next
}

const defaultNamespace = "default"

func ChainOf(d *appsv1.Deployment, sets []*appsv1.ReplicaSet, pods []*corev1.Pod) CrewChain {
	c := CrewChain{Namespace: d.Namespace, Name: d.Name, Paused: d.Spec.Paused, Revision: RevisionOf(d.Annotations),
		Ready: int(d.Status.ReadyReplicas), Updated: int(d.Status.UpdatedReplicas), Available: int(d.Status.AvailableReplicas),
		Total: int(d.Status.Replicas), Desired: 1, HistoryLimit: defaultHistoryLimit, Uncounted: d.Status.ObservedGeneration == 0}
	if d.Spec.Replicas != nil {
		c.Desired = int(*d.Spec.Replicas)
	}
	if d.Spec.RevisionHistoryLimit != nil {
		c.HistoryLimit = int(*d.Spec.RevisionHistoryLimit)
	}
	c.Strategy, c.MaxSurge, c.MaxUnavailable = StrategyOf(d.Spec.Strategy)
	c.Container, c.Image, c.Resources = TemplateOf(d.Spec.Template.Spec)
	c.QoS, c.Rank = TemplateQoS(d.Spec.Template.Spec), RankOf(d.Spec.Template.Spec.PriorityClassName)
	for _, rs := range sets {
		if ref := podutil.ControllerRef(rs); ref != nil && ref.Kind == deploymentKind && ref.Name == d.Name && rs.Namespace == d.Namespace {
			c.Foremen = append(c.Foremen, ForemanOf(rs, pods))
		}
	}
	c.Foremen = markNewest(c.Foremen)
	return c
}

func BareChain(rs *appsv1.ReplicaSet, pods []*corev1.Pod) CrewChain {
	f := ForemanOf(rs, pods)
	f.Newest = true
	c := CrewChain{Namespace: rs.Namespace, Bare: true, Image: f.Image, Desired: f.Desired, Ready: f.Ready, Total: f.Current,
		Foremen: []Foreman{f}}
	c.Container, _, c.Resources = TemplateOf(rs.Spec.Template.Spec)
	c.QoS, c.Rank = TemplateQoS(rs.Spec.Template.Spec), RankOf(rs.Spec.Template.Spec.PriorityClassName)
	return c
}

func ForemanOf(rs *appsv1.ReplicaSet, pods []*corev1.Pod) Foreman {
	f := Foreman{Namespace: rs.Namespace, Name: rs.Name, Hash: rs.Labels[TemplateHashLabel], Revision: RevisionOf(rs.Annotations),
		Current: int(rs.Status.Replicas), Ready: int(rs.Status.ReadyReplicas), Available: int(rs.Status.AvailableReplicas),
		ChangeCause: rs.Annotations[ChangeCauseAnnotation], Desired: 1}
	if rs.Spec.Replicas != nil {
		f.Desired = int(*rs.Spec.Replicas)
	}
	if cs := rs.Spec.Template.Spec.Containers; len(cs) > 0 {
		f.Image = cs[0].Image
		f.Shutters, f.Bell = cs[0].ReadinessProbe != nil, cs[0].LivenessProbe != nil
		f.Pledge = cs[0].Resources.Requests.Cpu().String() + "/" + cs[0].Resources.Requests.Memory().String()
	}
	for _, p := range pods {
		if ref := podutil.ControllerRef(p); ref != nil && ref.Kind == replicaSetKind && ref.Name == rs.Name && p.Namespace == rs.Namespace {
			f.Pods = append(f.Pods, p.Name)
		}
	}
	sort.Strings(f.Pods)
	return f
}

func markNewest(foremen []Foreman) []Foreman {
	out := append([]Foreman(nil), foremen...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Revision != out[j].Revision {
			return out[i].Revision > out[j].Revision
		}
		return out[i].Name < out[j].Name
	})
	if len(out) > 0 {
		out[0].Newest = true
	}
	return out
}

func RevisionOf(annotations map[string]string) int64 {
	v, err := strconv.ParseInt(annotations[RevisionAnnotation], 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func StrategyOf(s appsv1.DeploymentStrategy) (string, string, string) {
	kind := string(s.Type)
	if kind == "" {
		kind = string(appsv1.RollingUpdateDeploymentStrategyType)
	}
	if kind != string(appsv1.RollingUpdateDeploymentStrategyType) {
		return kind, "", ""
	}
	surge, unavailable := defaultRollingShare, defaultRollingShare
	if ru := s.RollingUpdate; ru != nil {
		surge, unavailable = shareOf(ru.MaxSurge, surge), shareOf(ru.MaxUnavailable, unavailable)
	}
	return kind, surge, unavailable
}

func shareOf(v *intstr.IntOrString, fallback string) string {
	if v == nil {
		return fallback
	}
	return v.String()
}

func TemplateOf(spec corev1.PodSpec) (string, string, Resources) {
	if len(spec.Containers) == 0 {
		return "", "", Resources{}
	}
	c := spec.Containers[0]
	req, lim := c.Resources.Requests, c.Resources.Limits
	return c.Name, c.Image, Resources{CPURequest: req.Cpu().MilliValue(), MemRequest: req.Memory().Value() / mebibyte,
		CPULimit: lim.Cpu().MilliValue(), MemLimit: lim.Memory().Value() / mebibyte}
}

func TemplateQoS(spec corev1.PodSpec) string {
	return string(podutil.QoSClass(&corev1.Pod{Spec: spec}))
}

func DepotFor(services []*corev1.Service, namespace string, labels map[string]string) string {
	for _, svc := range services {
		if svc.Name != apiService && svc.Namespace == namespace && selects(svc.Spec.Selector, labels) {
			return svc.Name
		}
	}
	return ""
}

func HornFor(horns []Horn, crew string) *Horn {
	for _, h := range horns {
		if h.Target == crew {
			found := h
			return &found
		}
	}
	return nil
}

func (c CrewChain) ID() string {
	if c.Bare && len(c.Foremen) > 0 {
		return c.Foremen[0].ID()
	}
	return c.Namespace + "/" + c.Name
}

func (f Foreman) ID() string { return f.Namespace + "/" + f.Name }

func (c CrewChain) Rolling() bool {
	n := 0
	for _, f := range c.Foremen {
		if f.Desired > 0 || f.Current > 0 {
			n++
		}
	}
	return n > 1
}

func (c CrewChain) Standing() []Foreman {
	out := []Foreman{}
	for _, f := range c.Foremen {
		if f.Newest || f.Desired > 0 || f.Current > 0 {
			out = append(out, f)
		}
	}
	return out
}

func (c CrewChain) History() []Foreman {
	out := []Foreman{}
	for _, f := range c.Foremen {
		if !f.Newest && f.Desired == 0 && f.Current == 0 {
			out = append(out, f)
		}
	}
	return out
}

func (c CrewChain) ForemanNamed(name string) (Foreman, bool) {
	for _, f := range c.Foremen {
		if f.Name == name {
			return f, true
		}
	}
	return Foreman{}, false
}

var revisionColors = []color.RGBA{
	{232, 186, 74, 255}, {92, 170, 222, 255}, {210, 96, 88, 255}, {120, 196, 104, 255}, {186, 120, 210, 255}, {236, 142, 64, 255},
}

func RevisionColor(revision int64) color.RGBA {
	if revision < 1 {
		return color.RGBA{200, 200, 190, 255}
	}
	return revisionColors[(revision-1)%int64(len(revisionColors))]
}
