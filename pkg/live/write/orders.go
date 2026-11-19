package write

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GVR struct {
	Group    string
	Version  string
	Resource string
}

var (
	Deployments  = GVR{"apps", "v1", "deployments"}
	StatefulSets = GVR{"apps", "v1", "statefulsets"}
	DaemonSets   = GVR{"apps", "v1", "daemonsets"}
	ReplicaSets  = GVR{"apps", "v1", "replicasets"}
	Pods         = GVR{"", "v1", "pods"}
	Nodes        = GVR{"", "v1", "nodes"}
	Services     = GVR{"", "v1", "services"}
	Jobs         = GVR{"batch", "v1", "jobs"}
	CronJobs     = GVR{"batch", "v1", "cronjobs"}
	HPAs         = GVR{"autoscaling", "v2", "horizontalpodautoscalers"}
)

func (g GVR) Path(namespace, name string, sub ...string) string {
	p := "/api/" + g.Version
	if g.Group != "" {
		p = "/apis/" + g.Group + "/" + g.Version
	}
	if namespace != "" {
		p += "/namespaces/" + namespace
	}
	p += "/" + g.Resource
	if name != "" {
		p += "/" + name
	}
	for _, s := range sub {
		p += "/" + s
	}
	return p
}

type Ref struct {
	GVR
	Kind            string
	Namespace       string
	Name            string
	UID             string
	ResourceVersion string
}

func RefOf(gvr GVR, kind string, m metav1.Object) Ref {
	return Ref{GVR: gvr, Kind: kind, Namespace: m.GetNamespace(), Name: m.GetName(), UID: string(m.GetUID()), ResourceVersion: m.GetResourceVersion()}
}

func (r Ref) String() string {
	kind := strings.ToLower(r.Kind)
	if r.Namespace == "" {
		return kind + "/" + r.Name
	}
	return kind + " " + r.Namespace + "/" + r.Name
}

func (r Ref) kubectl() string {
	s := strings.ToLower(r.Kind) + "/" + r.Name
	if r.Namespace != "" {
		s += " -n " + r.Namespace
	}
	return s
}

type Access struct {
	Verb        string `json:"verb"`
	Group       string `json:"group,omitempty"`
	Version     string `json:"version,omitempty"`
	Resource    string `json:"resource"`
	Subresource string `json:"subresource,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Name        string `json:"name,omitempty"`
}

type Field struct {
	Label  string
	Path   []string
	Before string
	After  string
}

const (
	OrderScale        = "scale"
	OrderRestart      = "restart"
	OrderPause        = "pause"
	OrderResume       = "resume"
	OrderRollback     = "roll back"
	OrderSetImage     = "set image"
	OrderSetResources = "set resources"
	OrderRetune       = "retune"
	OrderDelete       = "delete"
	OrderKill         = "kill"
	OrderCordon       = "cordon"
	OrderUncordon     = "uncordon"
	OrderEvict        = "evict"
	OrderTrigger      = "trigger"
)

const (
	MergePatch     = "application/merge-patch+json"
	StrategicPatch = "application/strategic-merge-patch+json"
	JSONPatch      = "application/json-patch+json"
	JSONBody       = "application/json"
)

type Order struct {
	Verb        string
	Target      Ref
	Request     Request
	ContentType string
	Access      Access
	Classes     []Class
	Fields      []Field
	Command     string
}

func (o Order) Object() string {
	return o.Target.String()
}

type Propagation string

const (
	Background      Propagation = "Background"
	Foreground      Propagation = "Foreground"
	Orphan          Propagation = "Orphan"
	PropagationNone Propagation = "None"
)

var Propagations = []Propagation{Background, Foreground, Orphan, PropagationNone}

func fieldManager() url.Values {
	return url.Values{"fieldManager": {FieldManager}}
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func versioned(rv string, rest map[string]any) map[string]any {
	out := map[string]any{"metadata": map[string]any{"resourceVersion": rv}}
	for k, v := range rest {
		if k == "metadata" {
			meta := v.(map[string]any)
			meta["resourceVersion"] = rv
			out["metadata"] = meta
			continue
		}
		out[k] = v
	}
	return out
}

func patch(o Order, verb, contentType string, body any, sub string) Order {
	o.Request = Request{Method: http.MethodPatch, Path: o.Target.Path(o.Target.Namespace, o.Target.Name, subs(sub)...), Query: fieldManager(), Body: mustJSON(body)}
	o.ContentType = contentType
	o.Access = Access{Verb: "patch", Group: o.Target.Group, Version: o.Target.Version, Resource: o.Target.Resource, Subresource: sub, Namespace: o.Target.Namespace, Name: o.Target.Name}
	o.Verb = verb
	o.Classes = ClassesOf(o.Request)
	return o
}

func subs(sub string) []string {
	if sub == "" {
		return nil
	}
	return []string{sub}
}

var scalable = map[string]GVR{"Deployment": Deployments, "StatefulSet": StatefulSets, "ReplicaSet": ReplicaSets}

func Scale(kind string, m metav1.Object, current, replicas int32) (Order, error) {
	gvr, ok := scalable[kind]
	if !ok {
		return Order{}, fmt.Errorf("a %s cannot be scaled", kind)
	}
	if replicas < 0 {
		return Order{}, errors.New("the number of replicas cannot be negative")
	}
	o := Order{Target: RefOf(gvr, kind, m)}
	o = patch(o, OrderScale, MergePatch, versioned(o.Target.ResourceVersion, map[string]any{"spec": map[string]any{"replicas": replicas}}), "scale")
	o.Fields = []Field{{Label: "replicas", Path: []string{"spec", "replicas"}, Before: strconv.Itoa(int(current))}}
	o.Command = fmt.Sprintf("kubectl scale %s --replicas=%d", o.Target.kubectl(), replicas)
	return o, nil
}

var restartable = map[string]GVR{"Deployment": Deployments, "StatefulSet": StatefulSets, "DaemonSet": DaemonSets}

const restartedAt = "kubectl.kubernetes.io/restartedAt"

const PausedRestart = "can't restart paused deployment (run rollout resume first)"

func Restart(kind string, m metav1.Object, paused bool, template map[string]string, at time.Time) (Order, error) {
	gvr, ok := restartable[kind]
	if !ok {
		return Order{}, fmt.Errorf("a %s cannot be restarted", kind)
	}
	if kind == "Deployment" && paused {
		return Order{}, errors.New(PausedRestart)
	}
	stamp := at.Format(time.RFC3339)
	o := Order{Target: RefOf(gvr, kind, m)}
	o = patch(o, OrderRestart, StrategicPatch, versioned(o.Target.ResourceVersion, map[string]any{"spec": map[string]any{"template": map[string]any{"metadata": map[string]any{"annotations": map[string]any{restartedAt: stamp}}}}}), "")
	o.Fields = []Field{{Label: "restartedAt", Path: []string{"spec", "template", "metadata", "annotations", restartedAt}, Before: template[restartedAt]}}
	o.Command = "kubectl rollout restart " + o.Target.kubectl()
	return o, nil
}

func Pause(d *appsv1.Deployment, paused bool) (Order, error) {
	if d.Spec.Paused == paused {
		if paused {
			return Order{}, errors.New("the deployment is already paused")
		}
		return Order{}, errors.New("the deployment is not paused")
	}
	o := Order{Target: RefOf(Deployments, "Deployment", d)}
	verb, word := OrderPause, "pause"
	var value any = true
	if !paused {
		verb, word, value = OrderResume, "resume", nil
	}
	o = patch(o, verb, StrategicPatch, versioned(o.Target.ResourceVersion, map[string]any{"spec": map[string]any{"paused": value}}), "")
	o.Fields = []Field{{Label: "paused", Path: []string{"spec", "paused"}, Before: strconv.FormatBool(d.Spec.Paused)}}
	o.Command = "kubectl rollout " + word + " " + o.Target.kubectl()
	return o, nil
}

const (
	revisionAnnotation = "deployment.kubernetes.io/revision"
	hashLabel          = "pod-template-hash"
	PausedRollback     = "you cannot rollback a paused deployment; resume it first with 'kubectl rollout resume' and try again"
)

var annotationsToSkip = map[string]bool{
	"kubectl.kubernetes.io/last-applied-configuration": true,
	revisionAnnotation:                          true,
	"deployment.kubernetes.io/revision-history": true,
	"deployment.kubernetes.io/desired-replicas": true,
	"deployment.kubernetes.io/max-replicas":     true,
	"deprecated.deployment.rollback.to":         true,
}

func revisionOf(rs *appsv1.ReplicaSet) (int64, bool) {
	v, err := strconv.ParseInt(rs.Annotations[revisionAnnotation], 10, 64)
	return v, err == nil
}

func RollbackTarget(owned []appsv1.ReplicaSet, toRevision int64) (*appsv1.ReplicaSet, error) {
	var latest, previous *appsv1.ReplicaSet
	latestRev, previousRev := int64(-1), int64(-1)
	for i := range owned {
		rs := &owned[i]
		v, ok := revisionOf(rs)
		if !ok {
			continue
		}
		if toRevision > 0 {
			if v == toRevision {
				return rs, nil
			}
			continue
		}
		switch {
		case latestRev < v:
			previousRev, previous = latestRev, latest
			latestRev, latest = v, rs
		case previousRev < v:
			previousRev, previous = v, rs
		}
	}
	if toRevision > 0 {
		return nil, fmt.Errorf("unable to find specified revision %d in history", toRevision)
	}
	if previous == nil {
		return nil, errors.New("no rollout history found for this deployment")
	}
	return previous, nil
}

func Rollback(d *appsv1.Deployment, owned []appsv1.ReplicaSet, toRevision int64) (Order, error) {
	rs, err := RollbackTarget(owned, toRevision)
	if err != nil {
		return Order{}, err
	}
	if d.Spec.Paused {
		return Order{}, errors.New(PausedRollback)
	}
	template := *rs.Spec.Template.DeepCopy()
	delete(template.Labels, hashLabel)
	current := *d.Spec.Template.DeepCopy()
	delete(current.Labels, hashLabel)
	rev, _ := revisionOf(rs)
	if apiequality.Semantic.DeepEqual(template, current) {
		return Order{}, fmt.Errorf("skipped rollback (current template already matches revision %d)", rev)
	}
	annotations := map[string]string{}
	for k := range annotationsToSkip {
		if v, ok := d.Annotations[k]; ok {
			annotations[k] = v
		}
	}
	for k, v := range rs.Annotations {
		if !annotationsToSkip[k] {
			annotations[k] = v
		}
	}
	o := Order{Target: RefOf(Deployments, "Deployment", d)}
	ops := []map[string]any{
		{"op": "replace", "path": "/metadata/resourceVersion", "value": o.Target.ResourceVersion},
		{"op": "replace", "path": "/spec/template", "value": template},
		{"op": "replace", "path": "/metadata/annotations", "value": annotations},
	}
	o = patch(o, OrderRollback, JSONPatch, ops, "")
	o.Fields = []Field{{Label: "revision", Path: []string{"metadata", "annotations", revisionAnnotation}, Before: d.Annotations[revisionAnnotation]}}
	for _, c := range current.Spec.Containers {
		o.Fields = append(o.Fields, Field{Label: c.Name + " image", Path: []string{"spec", "template", "spec", "containers", "name=" + c.Name, "image"}, Before: c.Image})
	}
	o.Command = fmt.Sprintf("kubectl rollout undo %s --to-revision=%d", o.Target.kubectl(), rev)
	return o, nil
}

func templateOwner(kind string) (GVR, bool) {
	gvr, ok := restartable[kind]
	return gvr, ok
}

func containerNamed(containers []corev1.Container, name string) (corev1.Container, bool) {
	i := slices.IndexFunc(containers, func(c corev1.Container) bool { return c.Name == name })
	if i < 0 {
		return corev1.Container{}, false
	}
	return containers[i], true
}

func SetImage(kind string, m metav1.Object, containers []corev1.Container, container, image string) (Order, error) {
	gvr, ok := templateOwner(kind)
	if !ok {
		return Order{}, fmt.Errorf("a %s has no pod template to change", kind)
	}
	c, ok := containerNamed(containers, container)
	if !ok {
		return Order{}, fmt.Errorf("no container named %q", container)
	}
	if strings.TrimSpace(image) == "" {
		return Order{}, errors.New("name the image")
	}
	o := Order{Target: RefOf(gvr, kind, m)}
	body := versioned(o.Target.ResourceVersion, map[string]any{"spec": map[string]any{"template": map[string]any{"spec": map[string]any{"containers": []any{map[string]any{"name": container, "image": image}}}}}})
	o = patch(o, OrderSetImage, StrategicPatch, body, "")
	o.Fields = []Field{{Label: container + " image", Path: []string{"spec", "template", "spec", "containers", "name=" + container, "image"}, Before: c.Image}}
	o.Command = fmt.Sprintf("kubectl set image %s %s=%s", o.Target.kubectl(), container, image)
	return o, nil
}

func resourceMap(l corev1.ResourceList) map[string]any {
	out := map[string]any{}
	for k, v := range l {
		out[string(k)] = v.String()
	}
	return out
}

func resourceArg(l corev1.ResourceList) string {
	var parts []string
	for k, v := range l {
		parts = append(parts, string(k)+"="+v.String())
	}
	slices.Sort(parts)
	return strings.Join(parts, ",")
}

func quantity(l corev1.ResourceList, name corev1.ResourceName) string {
	if q, ok := l[name]; ok {
		return q.String()
	}
	return ""
}

func SetResources(kind string, m metav1.Object, containers []corev1.Container, container string, requests, limits corev1.ResourceList) (Order, error) {
	gvr, ok := templateOwner(kind)
	if !ok {
		return Order{}, fmt.Errorf("a %s has no pod template to change", kind)
	}
	c, ok := containerNamed(containers, container)
	if !ok {
		return Order{}, fmt.Errorf("no container named %q", container)
	}
	if len(requests) == 0 && len(limits) == 0 {
		return Order{}, errors.New("name a request or a limit")
	}
	res := map[string]any{}
	if len(requests) > 0 {
		res["requests"] = resourceMap(requests)
	}
	if len(limits) > 0 {
		res["limits"] = resourceMap(limits)
	}
	o := Order{Target: RefOf(gvr, kind, m)}
	body := versioned(o.Target.ResourceVersion, map[string]any{"spec": map[string]any{"template": map[string]any{"spec": map[string]any{"containers": []any{map[string]any{"name": container, "resources": res}}}}}})
	o = patch(o, OrderSetResources, StrategicPatch, body, "")
	base := []string{"spec", "template", "spec", "containers", "name=" + container, "resources"}
	for _, side := range []struct {
		name string
		list corev1.ResourceList
		was  corev1.ResourceList
	}{{"requests", requests, c.Resources.Requests}, {"limits", limits, c.Resources.Limits}} {
		var names []string
		for k := range side.list {
			names = append(names, string(k))
		}
		slices.Sort(names)
		for _, n := range names {
			o.Fields = append(o.Fields, Field{Label: container + " " + side.name + " " + n, Path: append(slices.Clone(base), side.name, n), Before: quantity(side.was, corev1.ResourceName(n))})
		}
	}
	cmd := fmt.Sprintf("kubectl set resources %s -c %s", o.Target.kubectl(), container)
	if len(limits) > 0 {
		cmd += " --limits=" + resourceArg(limits)
	}
	if len(requests) > 0 {
		cmd += " --requests=" + resourceArg(requests)
	}
	o.Command = cmd
	return o, nil
}

func ParseResources(spec string) (corev1.ResourceList, error) {
	out := corev1.ResourceList{}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("%q is not name=quantity", part)
		}
		q, err := resource.ParseQuantity(strings.TrimSpace(v))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
		out[corev1.ResourceName(strings.TrimSpace(k))] = q
	}
	return out, nil
}

func Retune(h *autoscalingv2.HorizontalPodAutoscaler, minReplicas, maxReplicas int32) (Order, error) {
	if minReplicas < 1 || maxReplicas < minReplicas {
		return Order{}, fmt.Errorf("the range %d to %d is not a range a War Horn can keep", minReplicas, maxReplicas)
	}
	o := Order{Target: RefOf(HPAs, "HorizontalPodAutoscaler", h)}
	o = patch(o, OrderRetune, MergePatch, versioned(o.Target.ResourceVersion, map[string]any{"spec": map[string]any{"minReplicas": minReplicas, "maxReplicas": maxReplicas}}), "")
	was := ""
	if h.Spec.MinReplicas != nil {
		was = strconv.Itoa(int(*h.Spec.MinReplicas))
	}
	o.Fields = []Field{
		{Label: "minReplicas", Path: []string{"spec", "minReplicas"}, Before: was},
		{Label: "maxReplicas", Path: []string{"spec", "maxReplicas"}, Before: strconv.Itoa(int(h.Spec.MaxReplicas))},
	}
	o.Command = fmt.Sprintf(`kubectl patch hpa %s -n %s -p '{"spec":{"minReplicas":%d,"maxReplicas":%d}}'`, h.Name, h.Namespace, minReplicas, maxReplicas)
	return o, nil
}

func deleteOptions(r Ref, p Propagation, grace *int64) map[string]any {
	opts := map[string]any{"kind": "DeleteOptions", "apiVersion": "v1", "preconditions": map[string]any{"uid": r.UID, "resourceVersion": r.ResourceVersion}}
	if p != "" && p != PropagationNone {
		opts["propagationPolicy"] = string(p)
	}
	if grace != nil {
		opts["gracePeriodSeconds"] = *grace
	}
	return opts
}

func Delete(r Ref, p Propagation, force bool) (Order, error) {
	if r.Name == "" || r.UID == "" || r.ResourceVersion == "" {
		return Order{}, errors.New("a delete names one object, with the uid and version it was drawn from")
	}
	if p != "" && !slices.Contains(Propagations, p) {
		return Order{}, fmt.Errorf("unknown propagation %q", p)
	}
	var grace *int64
	if force {
		zero := int64(0)
		grace = &zero
	}
	o := Order{Verb: OrderDelete, Target: r}
	o.Request = Request{Method: http.MethodDelete, Path: r.Path(r.Namespace, r.Name), Body: mustJSON(deleteOptions(r, p, grace))}
	o.ContentType = JSONBody
	o.Access = Access{Verb: "delete", Group: r.Group, Version: r.Version, Resource: r.Resource, Namespace: r.Namespace, Name: r.Name}
	o.Classes = ClassesOf(o.Request)
	o.Fields = []Field{{Label: "deletionTimestamp", Path: []string{"metadata", "deletionTimestamp"}}}
	cmd := "kubectl delete " + r.kubectl()
	if p == Foreground || p == Orphan {
		cmd += " --cascade=" + strings.ToLower(string(p))
	}
	if force {
		cmd += " --grace-period=0 --force"
	}
	o.Command = cmd
	return o, nil
}

func Kill(p *corev1.Pod) (Order, error) {
	r := RefOf(Pods, "Pod", p)
	if r.UID == "" || r.ResourceVersion == "" {
		return Order{}, errors.New("a kill names one hut, with the uid and version it was drawn from")
	}
	one := int64(1)
	o := Order{Verb: OrderKill, Target: r}
	o.Request = Request{Method: http.MethodDelete, Path: r.Path(r.Namespace, r.Name), Body: mustJSON(deleteOptions(r, "", &one))}
	o.ContentType = JSONBody
	o.Access = Access{Verb: "delete", Version: "v1", Resource: "pods", Namespace: r.Namespace, Name: r.Name}
	o.Classes = ClassesOf(o.Request)
	o.Fields = []Field{{Label: "deletionTimestamp", Path: []string{"metadata", "deletionTimestamp"}}}
	o.Command = "kubectl delete " + r.kubectl() + " --now"
	return o, nil
}

func Cordon(n *corev1.Node, cordon bool) (Order, error) {
	if n.Spec.Unschedulable == cordon {
		if cordon {
			return Order{}, errors.New("the node is already cordoned")
		}
		return Order{}, errors.New("the node is not cordoned")
	}
	o := Order{Target: RefOf(Nodes, "Node", n)}
	verb := OrderCordon
	var value any = true
	if !cordon {
		verb, value = OrderUncordon, nil
	}
	o = patch(o, verb, StrategicPatch, versioned(o.Target.ResourceVersion, map[string]any{"spec": map[string]any{"unschedulable": value}}), "")
	o.Fields = []Field{{Label: "unschedulable", Path: []string{"spec", "unschedulable"}, Before: strconv.FormatBool(n.Spec.Unschedulable)}}
	o.Command = "kubectl " + verb + " " + n.Name
	return o, nil
}

func Evict(p *corev1.Pod) (Order, error) {
	r := RefOf(Pods, "Pod", p)
	if r.UID == "" || r.ResourceVersion == "" {
		return Order{}, errors.New("an eviction names one hut, with the uid and version it was drawn from")
	}
	body := map[string]any{
		"apiVersion":    "policy/v1",
		"kind":          "Eviction",
		"metadata":      map[string]any{"name": p.Name, "namespace": p.Namespace},
		"deleteOptions": map[string]any{"preconditions": map[string]any{"uid": r.UID, "resourceVersion": r.ResourceVersion}},
	}
	o := Order{Verb: OrderEvict, Target: r}
	o.Request = Request{Method: http.MethodPost, Path: r.Path(r.Namespace, r.Name, "eviction"), Body: mustJSON(body)}
	o.ContentType = JSONBody
	o.Access = Access{Verb: "create", Version: "v1", Resource: "pods", Subresource: "eviction", Namespace: r.Namespace, Name: r.Name}
	o.Classes = ClassesOf(o.Request)
	return o, nil
}

func DrainCommand(node string) string {
	return "kubectl drain " + node + " --ignore-daemonsets"
}

const maxJobNameSize = 42

func TriggeredJobName(cronJob, suffix string) string {
	name := cronJob
	if len(name) >= maxJobNameSize {
		name = name[:maxJobNameSize]
	}
	return name + "-manual-" + suffix
}

func Trigger(cj *batchv1.CronJob, suffix string) (Order, error) {
	if cj.UID == "" {
		return Order{}, errors.New("a trigger names one CronJob, with the uid it was drawn from")
	}
	name := TriggeredJobName(cj.Name, suffix)
	annotations := map[string]string{"cronjob.kubernetes.io/instantiate": "manual"}
	for k, v := range cj.Spec.JobTemplate.Annotations {
		annotations[k] = v
	}
	controller := true
	job := batchv1.Job{
		TypeMeta: metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"},
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: cj.Namespace, Annotations: annotations, Labels: cj.Spec.JobTemplate.Labels,
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "batch/v1", Kind: "CronJob", Name: cj.Name, UID: cj.UID, Controller: &controller}},
		},
		Spec: cj.Spec.JobTemplate.Spec,
	}
	o := Order{Verb: OrderTrigger, Target: RefOf(CronJobs, "CronJob", cj)}
	o.Request = Request{Method: http.MethodPost, Path: Jobs.Path(cj.Namespace, ""), Query: fieldManager(), Body: mustJSON(job)}
	o.ContentType = JSONBody
	o.Access = Access{Verb: "create", Group: "batch", Version: "v1", Resource: "jobs", Namespace: cj.Namespace}
	o.Classes = ClassesOf(o.Request)
	o.Fields = []Field{{Label: "job", Path: []string{"metadata", "name"}}}
	o.Command = fmt.Sprintf("kubectl create job %s --from=cronjob/%s -n %s", name, cj.Name, cj.Namespace)
	return o, nil
}
