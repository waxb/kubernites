package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
)

const (
	pageSize     = 500
	maxPages     = 200
	notOlderThan = "NotOlderThan"
)

type Snapshot struct {
	Context         ContextInfo
	Version         string
	TakenAt         time.Time
	Identity        Identity
	Denied          []string
	Missing         []string
	Refused         []Refusal
	Nodes           []corev1.Node
	Namespaces      []corev1.Namespace
	Pods            []corev1.Pod
	Services        []corev1.Service
	EndpointSlices  []discoveryv1.EndpointSlice
	Deployments     []appsv1.Deployment
	ReplicaSets     []appsv1.ReplicaSet
	DaemonSets      []appsv1.DaemonSet
	StatefulSets    []appsv1.StatefulSet
	Jobs            []batchv1.Job
	CronJobs        []batchv1.CronJob
	HPAs            []autoscalingv2.HorizontalPodAutoscaler
	PVCs            []corev1.PersistentVolumeClaim
	PVs             []corev1.PersistentVolume
	StorageClasses  []storagev1.StorageClass
	LimitRanges     []corev1.LimitRange
	PriorityClasses []schedulingv1.PriorityClass
	Ingresses       []networkingv1.Ingress
	Gateways        []Gateway
	CRDs            []CRD
	Custom          []CustomResource
	PDBs            []policyv1.PodDisruptionBudget `json:",omitempty"`
	Events          []corev1.Event
	Lists           map[string]ListMark `json:",omitempty"`
}

type ListMark struct {
	Version   string
	Namespace string `json:",omitempty"`
}

type Refusal struct {
	List      string
	Namespace string `json:",omitempty"`
	Message   string `json:",omitempty"`
}

type Progress struct {
	Step       string
	List       string
	Count      int
	Done       int
	Total      int
	Nodes      int
	Namespaces int
	Pods       int
}

type lister interface {
	getJSON(ctx context.Context, path string, query url.Values, into any) error
}

type metaLister interface {
	getMeta(ctx context.Context, path string, query url.Values, into any) error
}

var errTooManyPages = errors.New("the list is longer than the realm reads at once")

func listAll[T any](ctx context.Context, get func(context.Context, string, url.Values, any) error, path string, page func(int)) ([]T, error) {
	out, _, err := listAt[T](ctx, get, path, page, "")
	return out, err
}

func listAt[T any](ctx context.Context, get func(context.Context, string, url.Values, any) error, path string, page func(int), since string) ([]T, string, error) {
	var out []T
	cont, version := "", ""
	for n := 0; n < maxPages; n++ {
		var got struct {
			Metadata metav1.ListMeta `json:"metadata"`
			Items    []T             `json:"items"`
		}
		q := url.Values{"limit": {strconv.Itoa(pageSize)}}
		switch {
		case cont != "":
			q.Set("continue", cont)
		case since != "":
			q.Set("resourceVersion", since)
			q.Set("resourceVersionMatch", notOlderThan)
		}
		if err := get(ctx, path, q, &got); err != nil {
			return out, version, err
		}
		out = append(out, got.Items...)
		version = got.Metadata.ResourceVersion
		if page != nil {
			page(len(out))
		}
		if cont = got.Metadata.Continue; cont == "" {
			return out, version, nil
		}
	}
	return out, version, errTooManyPages
}

type kindRead struct {
	name       string
	root       string
	resource   string
	namespaced bool
	custom     bool
	accept     string
	read       func(ctx context.Context, r *reading, path string) error
	feed       func(s *Snapshot, typ string, raw []byte) (metav1.Object, error)
	items      func(s *Snapshot) []metav1.Object
	swap       func(dst, src *Snapshot)
}

func (k kindRead) path(ns string) string {
	res := k.name
	if k.resource != "" {
		res = url.PathEscape(k.resource)
	}
	if ns == "" {
		return k.root + "/" + res
	}
	return k.root + "/namespaces/" + url.PathEscape(ns) + "/" + res
}

func reader[T any](into func(*Snapshot) *[]T) func(context.Context, *reading, string) error {
	return func(ctx context.Context, r *reading, path string) error {
		items, version, err := listAt[T](ctx, r.l.getJSON, path, r.page, r.since)
		*into(r.s) = items
		r.version = version
		return err
	}
}

type metaPtr[T any] interface {
	*T
	metav1.Object
}

func listed[T any, P metaPtr[T]](name, root string, namespaced bool, into func(*Snapshot) *[]T) kindRead {
	return kindRead{name: name, root: root, namespaced: namespaced, read: reader(into), feed: feeder[T, P](into),
		items: func(s *Snapshot) []metav1.Object {
			list := *into(s)
			out := make([]metav1.Object, len(list))
			for i := range list {
				out[i] = P(&list[i])
			}
			return out
		},
		swap: func(dst, src *Snapshot) { *into(dst) = *into(src) }}
}

func feeder[T any, P metaPtr[T]](into func(*Snapshot) *[]T) func(*Snapshot, string, []byte) (metav1.Object, error) {
	return func(s *Snapshot, typ string, raw []byte) (metav1.Object, error) {
		obj := new(T)
		if err := json.Unmarshal(raw, obj); err != nil {
			return nil, err
		}
		P(obj).SetManagedFields(nil)
		list := into(s)
		uid := P(obj).GetUID()
		at := slices.IndexFunc(*list, func(o T) bool { return P(&o).GetUID() == uid })
		switch {
		case typ == watchDeleted && at >= 0:
			*list = append((*list)[:at], (*list)[at+1:]...)
		case typ == watchDeleted:
		case at >= 0:
			(*list)[at] = *obj
		default:
			*list = append(*list, *obj)
		}
		return P(obj), nil
	}
}

var kindReads = []kindRead{
	listed("nodes", "/api/v1", false, func(s *Snapshot) *[]corev1.Node { return &s.Nodes }),
	listed("namespaces", "/api/v1", false, func(s *Snapshot) *[]corev1.Namespace { return &s.Namespaces }),
	listed("pods", "/api/v1", true, func(s *Snapshot) *[]corev1.Pod { return &s.Pods }),
	listed("services", "/api/v1", true, func(s *Snapshot) *[]corev1.Service { return &s.Services }),
	listed("endpointslices", "/apis/discovery.k8s.io/v1", true, func(s *Snapshot) *[]discoveryv1.EndpointSlice { return &s.EndpointSlices }),
	listed("deployments", "/apis/apps/v1", true, func(s *Snapshot) *[]appsv1.Deployment { return &s.Deployments }),
	listed("replicasets", "/apis/apps/v1", true, func(s *Snapshot) *[]appsv1.ReplicaSet { return &s.ReplicaSets }),
	listed("daemonsets", "/apis/apps/v1", true, func(s *Snapshot) *[]appsv1.DaemonSet { return &s.DaemonSets }),
	listed("statefulsets", "/apis/apps/v1", true, func(s *Snapshot) *[]appsv1.StatefulSet { return &s.StatefulSets }),
	listed("jobs", "/apis/batch/v1", true, func(s *Snapshot) *[]batchv1.Job { return &s.Jobs }),
	listed("cronjobs", "/apis/batch/v1", true, func(s *Snapshot) *[]batchv1.CronJob { return &s.CronJobs }),
	listed("horizontalpodautoscalers", "/apis/autoscaling/v2", true, func(s *Snapshot) *[]autoscalingv2.HorizontalPodAutoscaler { return &s.HPAs }),
	listed("persistentvolumeclaims", "/api/v1", true, func(s *Snapshot) *[]corev1.PersistentVolumeClaim { return &s.PVCs }),
	listed("persistentvolumes", "/api/v1", false, func(s *Snapshot) *[]corev1.PersistentVolume { return &s.PVs }),
	listed("storageclasses", "/apis/storage.k8s.io/v1", false, func(s *Snapshot) *[]storagev1.StorageClass { return &s.StorageClasses }),
	listed("limitranges", "/api/v1", true, func(s *Snapshot) *[]corev1.LimitRange { return &s.LimitRanges }),
	listed("priorityclasses", "/apis/scheduling.k8s.io/v1", false, func(s *Snapshot) *[]schedulingv1.PriorityClass { return &s.PriorityClasses }),
	listed("ingresses", "/apis/networking.k8s.io/v1", true, func(s *Snapshot) *[]networkingv1.Ingress { return &s.Ingresses }),
	listed("gateways", "/apis/gateway.networking.k8s.io/v1", true, func(s *Snapshot) *[]Gateway { return &s.Gateways }),
	crdKind,
	listed("poddisruptionbudgets", "/apis/policy/v1", true, func(s *Snapshot) *[]policyv1.PodDisruptionBudget { return &s.PDBs }),
	listed("events", "/api/v1", true, func(s *Snapshot) *[]corev1.Event { return &s.Events }),
}

type reading struct {
	l        lister
	s        *Snapshot
	ns       string
	progress func(Progress)
	at       Progress
	version  string
	since    string
}

func (r *reading) page(n int) {
	r.at.Count = n
	switch r.at.List {
	case "nodes":
		r.at.Nodes = n
	case "namespaces":
		r.at.Namespaces = n
	case "pods":
		r.at.Pods = n
	}
	r.tell()
}

func (r *reading) tell() {
	if r.progress != nil {
		r.progress(r.at)
	}
}

func Take(ctx context.Context, c *Conn, now time.Time, progress func(Progress)) (Snapshot, error) {
	s, err := take(ctx, c, now, c.Context.Namespace, progress)
	s.Context = c.Context
	return s, err
}

func take(ctx context.Context, l lister, now time.Time, ns string, progress func(Progress)) (Snapshot, error) {
	s := Snapshot{TakenAt: now.UTC().Truncate(time.Second)}
	if ns == "" {
		ns = corev1.NamespaceDefault
	}
	r := &reading{l: l, s: &s, ns: ns, progress: progress, at: Progress{List: "version", Total: len(kindReads) + 1}}
	var v version.Info
	if err := l.getJSON(ctx, "/version", nil, &v); err != nil {
		return s, NoAnswer(err)
	}
	s.Version = v.GitVersion
	r.at.Done = 1
	r.tell()
	for _, k := range kindReads {
		r.at.List, r.at.Count = k.name, 0
		if err := r.list(ctx, k); err != nil {
			return s, err
		}
		r.at.Done++
		r.tell()
	}
	r.readCustom(ctx)
	if slices.Contains(s.Denied, "namespaces") {
		s.seeNamespaces(ns)
	}
	s.trim()
	return s, nil
}

func (r *reading) mark(name, ns string) {
	if r.s.Lists == nil {
		r.s.Lists = map[string]ListMark{}
	}
	r.s.Lists[name] = ListMark{Version: r.version, Namespace: ns}
}

func (r *reading) list(ctx context.Context, k kindRead) error {
	r.version = ""
	err := k.read(ctx, r, k.path(""))
	switch {
	case err == nil:
		r.mark(k.name, "")
		return nil
	case IsForbidden(err) && k.namespaced:
		refusal := Refusal{List: k.name, Message: messageOf(err)}
		r.version = ""
		ferr := k.read(ctx, r, k.path(r.ns))
		switch {
		case ferr == nil:
			refusal.Namespace = r.ns
			r.mark(k.name, r.ns)
		case IsForbidden(ferr):
			r.s.Denied = append(r.s.Denied, k.name)
		case IsNotFound(ferr):
			r.s.Missing = append(r.s.Missing, k.name)
		default:
			return fmt.Errorf("list %s in %s: %w", k.name, r.ns, ferr)
		}
		r.s.Refused = append(r.s.Refused, refusal)
	case IsForbidden(err):
		r.s.Denied = append(r.s.Denied, k.name)
		r.s.Refused = append(r.s.Refused, Refusal{List: k.name, Message: messageOf(err)})
	case IsNotFound(err):
		r.s.Missing = append(r.s.Missing, k.name)
	default:
		return fmt.Errorf("list %s: %w", k.name, err)
	}
	return nil
}

func messageOf(err error) string {
	var se *StatusError
	if errors.As(err, &se) {
		return se.Message
	}
	return err.Error()
}

func (s *Snapshot) seeNamespaces(ns string) {
	seen := map[string]bool{}
	for _, n := range s.Namespaces {
		seen[n.Name] = true
	}
	names := []string{ns}
	for _, m := range s.metas() {
		names = append(names, m.GetNamespace())
	}
	for _, name := range names {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		s.Namespaces = append(s.Namespaces, corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID("namespace/" + name)}})
	}
}

func (s Snapshot) clone() Snapshot {
	out := s
	out.Denied, out.Missing, out.Refused = slices.Clone(s.Denied), slices.Clone(s.Missing), slices.Clone(s.Refused)
	out.Nodes, out.Namespaces, out.Pods, out.Services = slices.Clone(s.Nodes), slices.Clone(s.Namespaces), slices.Clone(s.Pods), slices.Clone(s.Services)
	out.EndpointSlices, out.Deployments, out.ReplicaSets = slices.Clone(s.EndpointSlices), slices.Clone(s.Deployments), slices.Clone(s.ReplicaSets)
	out.DaemonSets, out.StatefulSets, out.Jobs, out.CronJobs = slices.Clone(s.DaemonSets), slices.Clone(s.StatefulSets), slices.Clone(s.Jobs), slices.Clone(s.CronJobs)
	out.HPAs, out.PVCs, out.PVs, out.StorageClasses = slices.Clone(s.HPAs), slices.Clone(s.PVCs), slices.Clone(s.PVs), slices.Clone(s.StorageClasses)
	out.LimitRanges, out.PriorityClasses, out.Ingresses = slices.Clone(s.LimitRanges), slices.Clone(s.PriorityClasses), slices.Clone(s.Ingresses)
	out.Gateways, out.CRDs, out.Custom, out.PDBs, out.Events = slices.Clone(s.Gateways), slices.Clone(s.CRDs), slices.Clone(s.Custom), slices.Clone(s.PDBs), slices.Clone(s.Events)
	out.Lists = maps.Clone(s.Lists)
	return out
}

func (s *Snapshot) Metas() []metav1.Object {
	return s.metas()
}

func (s *Snapshot) trim() {
	for _, m := range s.metas() {
		m.SetManagedFields(nil)
	}
}

func (s *Snapshot) metas() []metav1.Object {
	var out []metav1.Object
	add := func(m metav1.Object) { out = append(out, m) }
	for i := range s.Nodes {
		add(&s.Nodes[i])
	}
	for i := range s.Namespaces {
		add(&s.Namespaces[i])
	}
	for i := range s.Pods {
		add(&s.Pods[i])
	}
	for i := range s.Services {
		add(&s.Services[i])
	}
	for i := range s.EndpointSlices {
		add(&s.EndpointSlices[i])
	}
	for i := range s.Deployments {
		add(&s.Deployments[i])
	}
	for i := range s.ReplicaSets {
		add(&s.ReplicaSets[i])
	}
	for i := range s.DaemonSets {
		add(&s.DaemonSets[i])
	}
	for i := range s.StatefulSets {
		add(&s.StatefulSets[i])
	}
	for i := range s.Jobs {
		add(&s.Jobs[i])
	}
	for i := range s.CronJobs {
		add(&s.CronJobs[i])
	}
	for i := range s.HPAs {
		add(&s.HPAs[i])
	}
	for i := range s.PVCs {
		add(&s.PVCs[i])
	}
	for i := range s.PVs {
		add(&s.PVs[i])
	}
	for i := range s.StorageClasses {
		add(&s.StorageClasses[i])
	}
	for i := range s.LimitRanges {
		add(&s.LimitRanges[i])
	}
	for i := range s.PriorityClasses {
		add(&s.PriorityClasses[i])
	}
	for i := range s.Ingresses {
		add(&s.Ingresses[i])
	}
	for i := range s.Gateways {
		add(&s.Gateways[i].ObjectMeta)
	}
	for i := range s.PDBs {
		add(&s.PDBs[i])
	}
	for i := range s.Events {
		add(&s.Events[i])
	}
	return out
}
