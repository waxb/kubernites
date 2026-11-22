package app

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live/write"
	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/view"
)

const orderMore = "more"

type orderSpec struct {
	key      string
	label    string
	tip      string
	verb     string
	icon     string
	cell     int
	page     int
	chord    string
	access   write.Access
	inputs   []hud.OrderInput
	lines    []string
	build    func(values []string) (write.Order, error)
	plan     func() (write.Drain, error)
	asks     bool
	quick    bool
	running  bool
	off      string
	offWhy   string
	replaces []string
	title    string
	object   string
	stone    string
	stoneRun bool
	read     bool
	askFirst bool
}

func patchAccess(gvr write.GVR, ns, name, sub string) write.Access {
	return write.Access{Verb: "patch", Group: gvr.Group, Version: gvr.Version, Resource: gvr.Resource, Subresource: sub, Namespace: ns, Name: name}
}

func (s *Session) specsFor(kind, ns, name string) []orderSpec {
	switch kind {
	case "Deployment":
		if d, ok := cached[*appsv1.Deployment](s.Cache, store.KindDeployment, ns, name); ok {
			return s.crewSpecs(d)
		}
	case "Pod":
		if p, ok := cached[*corev1.Pod](s.Cache, store.KindPod, ns, name); ok {
			worker := ""
			if sel := s.Selection; sel.Kind == "walker" && sel.Pod.Namespace == ns && sel.Pod.Name == name {
				worker = sel.Pod.Focus
			}
			return s.hutSpecs(p, worker)
		}
	case "Node":
		if n, ok := cached[*corev1.Node](s.Cache, store.KindNode, "", name); ok {
			return s.keepSpecs(n)
		}
	case "HorizontalPodAutoscaler":
		if h, ok := cached[*autoscalingv2.HorizontalPodAutoscaler](s.Cache, store.KindHorizontalPodAutoscaler, ns, name); ok {
			if d, ok := cached[*appsv1.Deployment](s.Cache, store.KindDeployment, ns, h.Spec.ScaleTargetRef.Name); ok && h.Spec.ScaleTargetRef.Kind == "Deployment" {
				return s.crewSpecs(d)
			}
		}
	case "Job":
		if src, ok := s.Realm.(jobSource); ok {
			if j, ok := src.Job(ns, name); ok {
				return []orderSpec{deleteSpec(write.Jobs, "Job", j, -1, ui.Bridge("Remove the failed quest and its huts.", "Delete the Job and its Pods."))}
			}
		}
	default:
		return s.objectSpecs(view.ObjectRef{Kind: kind, Namespace: ns, Name: name})
	}
	return nil
}

func (s *Session) remedySpecs() map[string]orderSpec {
	out := map[string]orderSpec{}
	for _, v := range s.State.Calamities {
		if v.Mended != "" {
			continue
		}
		for _, r := range v.Remedies {
			if r.Order == "" {
				continue
			}
			full := hud.RemedyOrder(r)
			if _, done := out[full]; done {
				continue
			}
			for _, sp := range s.specsFor(r.Target.Kind, r.Target.Namespace, r.Target.Name) {
				if sp.key == r.Order {
					sp.key, sp.cell, sp.page, sp.chord, sp.replaces = full, -1, 0, "", nil
					out[full] = sp
				}
			}
		}
	}
	return out
}

func (s *Session) liveSpecs() []orderSpec {
	out := s.selectedSpecs()
	keys := []string{}
	extra := s.remedySpecs()
	for k := range extra {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		out = append(out, extra[k])
	}
	return out
}

func (s *Session) selectedSpecs() []orderSpec {
	sel := s.Selection
	switch {
	case sel.Kind == hud.SelectCrew:
		ns, name, _ := strings.Cut(sel.ID, "/")
		if d, ok := cached[*appsv1.Deployment](s.Cache, store.KindDeployment, ns, name); ok {
			return s.crewSpecs(d)
		}
	case sel.Kind == hud.SelectHearth || sel.Kind == "walker":
		if p, ok := cached[*corev1.Pod](s.Cache, store.KindPod, sel.Pod.Namespace, sel.Pod.Name); ok {
			worker := ""
			if sel.Kind == "walker" {
				worker = sel.Pod.Focus
			}
			return s.hutSpecs(p, worker)
		}
	case sel.Kind == "structure" && sel.Structure.Kind == view.StructKeep && sel.Structure.Object.Name != "":
		if n, ok := cached[*corev1.Node](s.Cache, store.KindNode, "", sel.Structure.Object.Name); ok {
			return s.keepSpecs(n)
		}
	case sel.Kind == "structure" && sel.Structure.Object.Name != "":
		return s.objectSpecs(sel.Structure.Object)
	}
	return nil
}

func cached[T store.Object](c interface {
	Get(kind, namespace, name string) (store.Object, bool)
}, kind, ns, name string) (T, bool) {
	var zero T
	o, ok := c.Get(kind, ns, name)
	if !ok {
		return zero, false
	}
	t, ok := o.(T)
	return t, ok
}

func storyKind(kind string) string {
	switch kind {
	case "Deployment":
		return ui.Bridge("crew", "Deployment")
	case "StatefulSet":
		return ui.Bridge("crew", "StatefulSet")
	case "DaemonSet":
		return ui.Bridge("shrine crew", "DaemonSet")
	case "Pod":
		return ui.Bridge("hut", "Pod")
	case "Node":
		return ui.Bridge("Keep", "Node")
	case "Service":
		return ui.Bridge("depot", "Service")
	case "CronJob":
		return ui.Bridge("quest board", "CronJob")
	case "HorizontalPodAutoscaler":
		return ui.Bridge("War Horn", "HorizontalPodAutoscaler")
	}
	return kind
}

func objectName(kind string, m metav1.Object) string {
	if m.GetNamespace() == "" {
		return storyKind(kind) + " " + m.GetName()
	}
	return storyKind(kind) + " " + m.GetNamespace() + "/" + m.GetName()
}

func deleteInputs() []hud.OrderInput {
	props := []string{string(write.Background), string(write.Foreground), string(write.Orphan), string(write.PropagationNone)}
	return []hud.OrderInput{
		{Label: "PROPAGATION", Value: string(write.Background), Choices: props, Hint: ui.Bridge("What becomes of what it owns: BACKGROUND (removed after it), FOREGROUND (removed before it), ORPHAN (left standing) or NONE (the server's default)", "propagationPolicy: Background (the default), Foreground, Orphan, or None (the server's default)")},
		{Label: "FORCE", Value: "off", Tick: true, Hint: ui.Bridge("Gone at once, without its grace: destroying more than one thing, with its own warning", "gracePeriodSeconds 0 (--force): a force deletion, with the destroying class warning")},
	}
}

func deleteBuild(ref write.Ref) func([]string) (write.Order, error) {
	return func(v []string) (write.Order, error) {
		prop, force := write.Background, false
		if len(v) > 0 && v[0] != "" {
			prop = write.Propagation(v[0])
		}
		if len(v) > 1 {
			force = v[1] == "on"
		}
		return write.Delete(ref, prop, force)
	}
}

func deleteSpec(gvr write.GVR, kind string, m metav1.Object, cell int, tip string) orderSpec {
	ref := write.RefOf(gvr, kind, m)
	return orderSpec{key: "delete", label: "DELETE", verb: "delete it", icon: "icon/close", cell: cell, chord: "D", asks: true,
		tip: tip, access: write.Access{Verb: "delete", Group: gvr.Group, Version: gvr.Version, Resource: gvr.Resource, Namespace: m.GetNamespace(), Name: m.GetName()},
		inputs: deleteInputs(), build: deleteBuild(ref), title: "DELETE", object: objectName(kind, m)}
}

func (s *Session) hornOf(kind string, m metav1.Object) (*autoscalingv2.HorizontalPodAutoscaler, bool) {
	for _, h := range s.Cache.HPAs() {
		if h.Namespace == m.GetNamespace() && h.Spec.ScaleTargetRef.Kind == kind && h.Spec.ScaleTargetRef.Name == m.GetName() {
			return h, true
		}
	}
	return nil, false
}

func hornLine(h *autoscalingv2.HorizontalPodAutoscaler) string {
	return ui.Bridge("A War Horn keeps this crew: it reckons again every 15 seconds from the crew's size and its load, so a cut it disagrees with is raised again, a raise drains back within about 5 minutes, and 0 puts the Horn to sleep until the crew is scaled again.",
		"An HPA ("+h.Name+") scales this target: it reconciles every 15 seconds from the replicas and the load, so a scale-down it disagrees with is raised again, a scale-up drains back within about 5 minutes (the scale-down window), and 0 disables the HPA until the target is scaled again.")
}

func scaleSpec(gvr write.GVR, kind string, m metav1.Object, current int32, horn *autoscalingv2.HorizontalPodAutoscaler, cell int) orderSpec {
	sp := orderSpec{key: "scale", label: "SCALE", verb: "scale it", icon: "icon/scale-up", cell: cell, chord: "N",
		tip: ui.Bridge("Set how many huts the crew keeps.", "Set spec.replicas through the scale subresource."), access: patchAccess(gvr, m.GetNamespace(), m.GetName(), "scale"),
		inputs: []hud.OrderInput{{Label: ui.Bridge("HUTS", "REPLICAS"), Value: strconv.Itoa(int(current)), Number: true, Hint: "now " + strconv.Itoa(int(current))}},
		title:  "SCALE", object: objectName(kind, m)}
	if horn != nil {
		sp.lines = []string{hornLine(horn)}
	}
	sp.build = func(v []string) (write.Order, error) {
		n, err := strconv.Atoi(strings.TrimSpace(first(v)))
		if err != nil || n < 0 || n > 10000 {
			return write.Order{}, errors.New(ui.Bridge("name a number of huts, 0 or more", "replicas must be a whole number, 0 or more"))
		}
		return write.Scale(kind, m, current, int32(n))
	}
	return sp
}

func first(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

func (s *Session) restartSpec(gvr write.GVR, kind string, m metav1.Object, paused bool, annotations map[string]string, cell int) orderSpec {
	sp := orderSpec{key: "restart", label: "RESTART", verb: "restart it", icon: "icon/roll", cell: cell, chord: "B",
		tip:    ui.Bridge("Raise every hut anew, one by one, with the same trade and class.", "kubectl rollout restart: stamp restartedAt on the pod template so every Pod is replaced."),
		access: patchAccess(gvr, m.GetNamespace(), m.GetName(), ""), title: "RESTART", object: objectName(kind, m)}
	if kind == "Deployment" && paused {
		sp.off, sp.offWhy = write.PausedRestart, "PAUSED"
	}
	now := s.writerNow
	sp.build = func([]string) (write.Order, error) {
		return write.Restart(kind, m, paused, annotations, now())
	}
	return sp
}

func (s *Session) crewSpecs(d *appsv1.Deployment) []orderSpec {
	current := int32(1)
	if d.Spec.Replicas != nil {
		current = *d.Spec.Replicas
	}
	horn, hasHorn := s.hornOf("Deployment", d)
	if !hasHorn {
		horn = nil
	}
	out := []orderSpec{scaleSpec(write.Deployments, "Deployment", d, current, horn, 2)}
	out = append(out, s.restartSpec(write.Deployments, "Deployment", d, d.Spec.Paused, d.Spec.Template.Annotations, 3))
	owned := s.ownedReplicaSets(d)
	back := orderSpec{key: "rollback", label: "UNDO", verb: "roll it back", icon: "icon/back", cell: 4, chord: "L",
		tip:    ui.Bridge("Return to the previous revision: the old shift is called back.", "kubectl rollout undo: restore the previous ReplicaSet's pod template."),
		access: patchAccess(write.Deployments, d.Namespace, d.Name, ""), title: "UNDO", object: objectName("Deployment", d),
		build: func([]string) (write.Order, error) { return write.Rollback(d, owned, 0) }}
	switch _, err := write.RollbackTarget(owned, 0); {
	case d.Spec.Paused:
		back.off, back.offWhy = write.PausedRollback, "PAUSED"
	case err != nil:
		back.off, back.offWhy = ui.Bridge("No earlier revision to return to.", "No rollout history: no earlier ReplicaSet revision."), "NO HISTORY"
	}
	out = append(out, back)
	pause := orderSpec{key: "pause", label: "PAUSE", verb: "pause it", icon: "icon/pause", cell: 5,
		tip:    ui.Bridge("Hold the crew's roll-outs: changes to its huts wait until it resumes.", "kubectl rollout pause: spec.paused true; template changes wait until resumed."),
		access: patchAccess(write.Deployments, d.Namespace, d.Name, ""), title: "PAUSE", object: objectName("Deployment", d),
		build: func([]string) (write.Order, error) { return write.Pause(d, true) }}
	if d.Spec.Paused {
		pause.key, pause.label, pause.verb, pause.icon, pause.title = "resume", "RESUME", "resume it", "icon/speed", "RESUME"
		pause.tip = ui.Bridge("Let the crew's held roll-out go on.", "kubectl rollout resume: spec.paused false.")
		pause.build = func([]string) (write.Order, error) { return write.Pause(d, false) }
	}
	out = append(out, pause)
	containers := d.Spec.Template.Spec.Containers
	names := make([]string, len(containers))
	for i, c := range containers {
		names[i] = c.Name
	}
	if len(containers) > 0 {
		image := orderSpec{key: "image", label: ui.Bridge("CLASS", "IMAGE"), verb: "set its image", icon: "icon/academy", cell: 6,
			tip:    ui.Bridge("Set the trade and class one worker of the crew runs.", "kubectl set image: a strategic merge patch on one container."),
			access: patchAccess(write.Deployments, d.Namespace, d.Name, ""), title: ui.Bridge("SET THE CLASS", "SET IMAGE"), object: objectName("Deployment", d),
			inputs: []hud.OrderInput{{Label: ui.Bridge("WORKER", "CONTAINER"), Value: names[0], Choices: names}, {Label: "IMAGE", Value: containers[0].Image, Hint: ui.Bridge("trade:class", "repository:tag")}}}
		image.build = func(v []string) (write.Order, error) {
			return write.SetImage("Deployment", d, containers, first(v), strings.TrimSpace(at(v, 1)))
		}
		out = append(out, image)
		res := orderSpec{key: "resources", label: "RESOURCES", verb: "set its resources", icon: "icon/wood", cell: 0, page: 1,
			tip:    ui.Bridge("Set what one worker pledges and may use.", "kubectl set resources: requests and limits of one container."),
			access: patchAccess(write.Deployments, d.Namespace, d.Name, ""), title: ui.Bridge("SET THE PLEDGE", "SET RESOURCES"), object: objectName("Deployment", d),
			inputs: []hud.OrderInput{{Label: ui.Bridge("WORKER", "CONTAINER"), Value: names[0], Choices: names}, {Label: "REQUESTS", Value: resourceText(containers[0].Resources.Requests), Hint: "cpu=100m,memory=64Mi"}, {Label: "LIMITS", Value: resourceText(containers[0].Resources.Limits), Hint: "memory=128Mi"}}}
		res.build = func(v []string) (write.Order, error) {
			req, err := write.ParseResources(at(v, 1))
			if err != nil {
				return write.Order{}, err
			}
			lim, err := write.ParseResources(at(v, 2))
			if err != nil {
				return write.Order{}, err
			}
			return write.SetResources("Deployment", d, containers, first(v), req, lim)
		}
		out = append(out, res)
	}
	out = append(out, orderSpec{key: orderMore, label: "MORE", icon: "icon/tech", cell: 7, tip: "More orders for this crew"})
	if horn != nil {
		min := int32(1)
		if horn.Spec.MinReplicas != nil {
			min = *horn.Spec.MinReplicas
		}
		h := horn
		out = append(out, orderSpec{key: "retune", label: ui.Bridge("WAR HORN", "HPA RANGE"), verb: "retune it", icon: "icon/warhorn", cell: 1, page: 1,
			tip:    ui.Bridge("Set the fewest and the most huts the War Horn may call.", "Patch the HPA's minReplicas and maxReplicas."),
			access: patchAccess(write.HPAs, h.Namespace, h.Name, ""), title: ui.Bridge("RETUNE THE WAR HORN", "RETUNE THE HPA"), object: objectName("HorizontalPodAutoscaler", h),
			inputs: []hud.OrderInput{{Label: "MIN", Value: strconv.Itoa(int(min)), Number: true}, {Label: "MAX", Value: strconv.Itoa(int(h.Spec.MaxReplicas)), Number: true}},
			build: func(v []string) (write.Order, error) {
				lo, err1 := strconv.Atoi(first(v))
				hi, err2 := strconv.Atoi(at(v, 1))
				if err1 != nil || err2 != nil {
					return write.Order{}, errors.New("name two whole numbers")
				}
				return write.Retune(h, int32(lo), int32(hi))
			}})
	}
	del := deleteSpec(write.Deployments, "Deployment", d, 2, ui.Bridge("Send the whole crew away: its shifts and huts go with it.", "Delete the Deployment: its ReplicaSets and Pods go with it (propagation)."))
	del.page = 1
	return append(out, del)
}

func at(v []string, i int) string {
	if i < len(v) {
		return v[i]
	}
	return ""
}

func resourceText(l corev1.ResourceList) string {
	var parts []string
	for k, q := range l {
		parts = append(parts, string(k)+"="+q.String())
	}
	slices.Sort(parts)
	return strings.Join(parts, ",")
}

func (s *Session) ownedReplicaSets(d *appsv1.Deployment) []appsv1.ReplicaSet {
	var out []appsv1.ReplicaSet
	for _, rs := range s.Cache.ReplicaSets() {
		for _, ref := range rs.OwnerReferences {
			if ref.UID == d.UID {
				out = append(out, *rs)
			}
		}
	}
	return out
}

func controllerOf(m metav1.Object) (metav1.OwnerReference, bool) {
	for _, ref := range m.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			return ref, true
		}
	}
	return metav1.OwnerReference{}, false
}

type cronSource interface {
	CronJobOf(namespace string, owners []metav1.OwnerReference) (*batchv1.CronJob, bool)
}

type jobSource interface {
	Job(namespace, name string) (*batchv1.Job, bool)
}

type budgetSource interface {
	Budgets(namespace string) []policyv1.PodDisruptionBudget
}

func (s *Session) hutSpecs(p *corev1.Pod, worker string) []orderSpec {
	tip := ui.Bridge("Delete this hut: its crew raises a new one in its place.", "Delete this Pod: its controller creates a replacement.")
	owner, owned := controllerOf(p)
	switch {
	case p.Annotations["kubernetes.io/config.mirror"] != "":
		tip = ui.Bridge("A mirror hut: its Keep runs it from a file and raises it again at once.", "A mirror Pod: the kubelet runs the static Pod from a file and creates the mirror again.")
	case !owned:
		tip = ui.Bridge("A hut no crew keeps: deleted, it is gone for good.", "A Pod with no controller: deleted, it is gone for good.")
	}
	del := deleteSpec(write.Pods, "Pod", p, 1, tip)
	del.replaces = []string{hud.ActionPrefix + ":" + hud.KeyDelete}
	kill := orderSpec{key: "kill", label: "KILL", verb: "kill it", icon: "icon/alert", cell: 2, chord: "K", quick: true,
		tip:    ui.Bridge("Delete this hut with one second of grace, as k9s's kill.", "Delete with gracePeriodSeconds 1, as k9s's kill (kubectl delete --now)."),
		access: write.Access{Verb: "delete", Version: "v1", Resource: "pods", Namespace: p.Namespace, Name: p.Name}, title: "KILL", object: objectName("Pod", p),
		build: func([]string) (write.Order, error) { return write.Kill(p) }}
	out := append([]orderSpec{del, kill}, hutReachSpecs(p, worker)...)
	if !owned {
		return out
	}
	switch owner.Kind {
	case "StatefulSet":
		if sts, ok := cached[*appsv1.StatefulSet](s.Cache, store.KindStatefulSet, p.Namespace, owner.Name); ok {
			current := int32(1)
			if sts.Spec.Replicas != nil {
				current = *sts.Spec.Replicas
			}
			horn, _ := s.hornOf("StatefulSet", sts)
			out = append(out, s.restartSpec(write.StatefulSets, "StatefulSet", sts, false, sts.Spec.Template.Annotations, 3), scaleSpec(write.StatefulSets, "StatefulSet", sts, current, horn, 5))
		}
	case "DaemonSet":
		if ds, ok := cached[*appsv1.DaemonSet](s.Cache, store.KindDaemonSet, p.Namespace, owner.Name); ok {
			out = append(out, s.restartSpec(write.DaemonSets, "DaemonSet", ds, false, ds.Spec.Template.Annotations, 3))
		}
	case "Job":
		if f, ok := s.Realm.(cronSource); ok {
			if cj, ok := f.CronJobOf(p.Namespace, p.OwnerReferences); ok {
				w := s.Writer
				out = append(out, orderSpec{key: "trigger", label: "TRIGGER", verb: "trigger it", icon: "icon/speed", cell: 3, chord: "T",
					tip:    ui.Bridge("Send this hut's quest board out once more, now.", "kubectl create job --from=cronjob/"+cj.Name+": a Job from the CronJob's template, now."),
					access: write.Access{Verb: "create", Group: "batch", Version: "v1", Resource: "jobs", Namespace: cj.Namespace}, title: "TRIGGER", object: objectName("CronJob", cj),
					build: func([]string) (write.Order, error) {
						suffix := "x"
						if w != nil {
							suffix = w.suffix()
						}
						return write.Trigger(cj, suffix)
					}})
			}
		}
	}
	return out
}

func (s *Session) keepSpecs(n *corev1.Node) []orderSpec {
	cordon := orderSpec{key: "cordon", label: "CORDON", verb: "cordon it", icon: "icon/sealed", cell: 4, chord: "O",
		tip:    ui.Bridge("Close the Keep to new huts: the Quartermaster places none here.", "kubectl cordon: spec.unschedulable true."),
		access: patchAccess(write.Nodes, "", n.Name, ""), title: "CORDON", object: objectName("Node", n),
		replaces: []string{hud.ActionPrefix + ":" + hud.KeyCordon, hud.ActionPrefix + ":" + hud.KeyUncordon},
		build:    func([]string) (write.Order, error) { return write.Cordon(n, true) }}
	if n.Spec.Unschedulable {
		cordon.key, cordon.label, cordon.verb, cordon.title = "uncordon", "UNCORDON", "uncordon it", "UNCORDON"
		cordon.tip = ui.Bridge("Open the Keep to new huts again.", "kubectl uncordon: spec.unschedulable cleared.")
		cordon.build = func([]string) (write.Order, error) { return write.Cordon(n, false) }
	}
	pods := s.Cache.Pods()
	drain := orderSpec{key: "drain", label: "DRAIN", verb: "drain it", icon: "icon/node", cell: 5, chord: "X", asks: true,
		tip:    ui.Bridge("Close the Keep, then send every hut away except shrine and mirror huts. A Quorum that blocks one is shown, and the drain waits.", "kubectl drain --ignore-daemonsets: cordon, then evict every Pod except DaemonSet and mirror Pods; a PodDisruptionBudget's 429 is shown and retried."),
		access: write.Access{Verb: "create", Version: "v1", Resource: "pods", Subresource: "eviction"}, title: "DRAIN", object: objectName("Node", n),
		replaces: []string{hud.ActionPrefix + ":" + hud.KeyDrain},
		plan: func() (write.Drain, error) {
			list := make([]corev1.Pod, len(pods))
			for i, p := range pods {
				list[i] = *p
			}
			return write.PlanDrain(n, list)
		}}
	if w := s.Writer; w != nil && w.drain != nil && !w.drain.over {
		if w.drain.node == n.Name {
			drain.label, drain.running = "DRAINING", true
			drain.tip = ui.Bridge("The drain runs: open it to see what waits, or STOP it.", "The drain runs: open it to see what blocks it, or STOP it.")
		} else {
			drain.off, drain.offWhy = oneDrain(w.drain.node), "DRAINING "+w.drain.node
		}
	}
	del := deleteSpec(write.Nodes, "Node", n, 6, ui.Bridge("Remove the Keep from the realm: destroying more than one thing.", "Delete the Node object: destroying more than one thing."))
	return []orderSpec{cordon, drain, del}
}

func oneDrain(node string) string {
	return ui.Bridge("One drain at a time: Keep "+node+" is draining. STOP it, or wait until it ends.", "One drain at a time: node "+node+" is draining. STOP it, or wait until it ends.")
}

var objectGVRs = map[string]write.GVR{
	"Service": write.Services, "Deployment": write.Deployments, "StatefulSet": write.StatefulSets, "DaemonSet": write.DaemonSets,
	"ReplicaSet": write.ReplicaSets, "PersistentVolumeClaim": {Version: "v1", Resource: "persistentvolumeclaims"},
	"PersistentVolume": {Version: "v1", Resource: "persistentvolumes"}, "ConfigMap": {Version: "v1", Resource: "configmaps"},
	"StorageClass": {Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"}, "HorizontalPodAutoscaler": write.HPAs,
}

func (s *Session) objectSpecs(ref view.ObjectRef) []orderSpec {
	gvr, ok := objectGVRs[ref.Kind]
	if !ok {
		return nil
	}
	o, ok := s.Cache.Get(ref.Kind, ref.Namespace, ref.Name)
	if !ok {
		return nil
	}
	m, ok := o.(metav1.Object)
	if !ok {
		return nil
	}
	cell := 7
	sp := deleteSpec(gvr, ref.Kind, m, cell, ui.Bridge("Remove it from the realm.", "Delete this "+ref.Kind+"."))
	if ref.Kind == "Service" {
		sp.cell, sp.replaces = 2, []string{hud.ActionPrefix + ":" + hud.KeyDelete}
		sp.tip = ui.Bridge("Close this depot: its boxes wait on the quay until a depot opens again.", "Delete this Service: traffic to it stops.")
	}
	return []orderSpec{sp}
}

func (s *Session) asks(sp orderSpec) bool {
	return sp.asks || (s.Writer != nil && s.Writer.level == write.Seamless && len(sp.inputs) == 0 && !sp.quick)
}

func (s *Session) orderForm(run *orderRun) hud.OrderForm {
	w := s.Writer
	sp := run.spec
	form := hud.OrderForm{ID: run.id, Key: sp.key, Title: sp.title, Object: sp.object, Level: w.level.String(), Lines: sp.lines}
	form.Inputs = make([]hud.OrderInput, len(sp.inputs))
	copy(form.Inputs, sp.inputs)
	for i := range form.Inputs {
		if i < len(run.values) {
			form.Inputs[i].Value = run.values[i]
		}
	}
	if w.hints && run.order.Command != "" {
		form.Command = run.order.Command
	}
	form.Note, form.Bad = run.note, run.bad
	switch run.stage {
	case stageInput:
		form.Go, form.GoKey = "PREVIEW [ENTER]", ebiten.KeyEnter
		form.GoTip = ui.Bridge("Ask the Chancery what would change, without changing anything.", "Send it with dryRun=All: the server answers with the object as it would be, and changes nothing.")
		if w.level == write.Seamless {
			form.Go, form.GoTip = "SEND [ENTER]", ui.Bridge("Send the order now.", "Send the request now.")
			if s.asks(sp) {
				if len(form.Lines) == 0 && len(form.Inputs) == 0 {
					form.Lines = []string{sp.tip}
				}
				form.Go, form.GoKey, form.GoTip = sp.label+" [Y]", ebiten.KeyY, ui.Bridge("Send it now: DANGEROUSLY SEAMLESS asks no more.", "Send the "+sp.label+" now: DANGEROUSLY SEAMLESS asks no more.")
			}
		}
	case stageClass:
		form.Title, form.Warning, form.Locked = "! "+strings.ToUpper(classTitle(run.need))+" !", true, true
		form.Lines = classText(run.need)
		form.Go, form.GoKey = "I UNDERSTAND [Y]", ebiten.KeyY
		form.GoTip = "Pass this warning once for this context; it is remembered until RESET LIVE CHOICES"
	case stageAsking:
		form.Locked, form.Note, form.Bad = true, ui.Bridge("Asking the Chancery what would change...", "Sending the dry run (dryRun=All)..."), false
	case stagePreview:
		form.Locked = true
		for _, f := range run.fields {
			form.Changes = append(form.Changes, hud.OrderChange{Label: f.Label, Before: f.Before, After: f.After})
		}
		form.Progress = run.change
		form.Go, form.GoKey = "CONFIRM [Y]", ebiten.KeyY
		form.GoTip = ui.Bridge("Send the order for real: this changes the cluster.", "Issue the one-use ticket for exactly this request and send it.")
		if run.order.Request.Method == "" && run.drain == nil {
			form.GoOff = true
		}
	case stageSending:
		form.Locked, form.Note, form.Bad = true, ui.Bridge("Sent; waiting for the Chancery's answer...", "Sent; waiting for the server's answer..."), false
	case stageDraining:
		form.Locked = true
		form.Title = "DRAIN"
		if dr := w.drain; dr != nil {
			form.Object = objectName("Node", &metav1.ObjectMeta{Name: dr.node})
			form.Progress = append(form.Progress, fmt.Sprintf(ui.Bridge("SENT AWAY %d OF %d HUTS", "EVICTED %d OF %d PODS"), dr.done, len(dr.plan.Evict)))
			if dr.blocked != "" {
				form.Progress = append(form.Progress, "WAITING: "+dr.blocked)
			}
			if dr.budget != "" {
				form.Progress = append(form.Progress, dr.budget)
			}
			for _, f := range dr.failed {
				form.Progress = append(form.Progress, f.shown())
			}
			form.Stop = !dr.over
			if dr.over {
				form.Note, form.Bad = "DONE", len(dr.failed) > 0 || dr.err != ""
				switch {
				case strings.HasPrefix(dr.err, "STOPPED"):
					form.Note = dr.err
				case dr.err != "":
					form.Note = clusterSaid("", dr.err)
				}
			}
		}
	}
	if run.drain != nil && run.stage != stageDraining {
		form.Lines = append(form.Lines, drainPlanLines(*run.drain)...)
	}
	return form
}

func classTitle(c write.Class) string {
	switch c {
	case write.Secrets:
		return ui.Bridge("Sealed scrolls", "Secrets")
	case write.Workloads:
		return ui.Bridge("Reaching into huts", "Reaching into workloads")
	case write.Destroy:
		return "Destroying more than one thing"
	case write.Unpreviewed:
		return ui.Bridge("No preview", "No dry run")
	case write.Tokens:
		return ui.Bridge("Making a writ's token", "Minting a token")
	}
	return string(c)
}

func classText(c write.Class) []string {
	switch c {
	case write.Secrets:
		return []string{ui.Bridge("This order reads, creates or changes sealed scrolls: passwords, keys and tokens. Their values stay masked on screen until revealed key by key, and the ledger never holds them.",
			"This touches Secrets: reading, creating or changing them. Values stay masked on screen until revealed key by key, and the ledger never holds them.")}
	case write.Workloads:
		return []string{ui.Bridge("This order reaches into a hut: a shell, attach, port-forward, copy, debug or proxy works inside the workload with these credentials.",
			"This reaches into a workload: exec, attach, port-forward, cp, debug or proxy run inside it with these credentials.")}
	case write.Destroy:
		return []string{ui.Bridge("This order destroys more than one thing: a fief, a Keep, a volume, a rule of who may do what, a whole kind, everything a selector names, several things named at once or written in decrees, a Keep drained by force, or a removal without grace.",
			"This destroys more than one thing: a namespace, Node, PersistentVolume, CRD, RBAC object or admission webhook, a collection or selector delete, a delete of several names, kinds or objects in files, --all, replace or apply --force of several objects, apply --prune, a drain with --disable-eviction or --force, or a force deletion (grace period 0).")}
	case write.Unpreviewed:
		return []string{ui.Bridge("A guard of the Chancery has side effects, so it refuses a preview. The order can go without one: what changes is seen only after it is sent.",
			"An admission webhook with sideEffects refused the dry run. The request can be sent without a preview: what changes is seen only after it is sent.")}
	case write.Tokens:
		return []string{tokenWarning(write.Writes)}
	}
	return nil
}

func drainPlanLines(d write.Drain) []string {
	var huts []string
	for _, o := range d.Evict {
		huts = append(huts, o.Target.Namespace+"/"+o.Target.Name)
	}
	shown := huts[:min(len(huts), 4)]
	line := fmt.Sprintf(ui.Bridge("Close the Keep, then send away %d huts: %s", "Cordon the Node, then evict %d Pods: %s"), len(huts), strings.Join(shown, ", "))
	if len(huts) > len(shown) {
		line += fmt.Sprintf(" and %d more", len(huts)-len(shown))
	}
	if d.Cordon == nil {
		line = fmt.Sprintf(ui.Bridge("The Keep is closed already; send away %d huts: %s", "Already cordoned; evict %d Pods: %s"), len(huts), strings.Join(shown, ", "))
	}
	out := []string{line}
	if len(d.Skipped) > 0 {
		out = append(out, fmt.Sprintf(ui.Bridge("%d shrine or mirror huts stay.", "%d DaemonSet or mirror Pods stay."), len(d.Skipped)))
	}
	if len(d.Orphans) > 0 {
		out = append(out, fmt.Sprintf(ui.Bridge("%d huts no crew keeps are gone for good.", "%d Pods without a controller are gone for good (as --force)."), len(d.Orphans)))
	}
	return out
}

func (s *Session) budgetLine(hut string) string {
	src, ok := s.Realm.(budgetSource)
	if !ok {
		return ""
	}
	ns, name, _ := strings.Cut(hut, "/")
	p, ok := cached[*corev1.Pod](s.Cache, store.KindPod, ns, name)
	if !ok {
		return ""
	}
	for _, b := range src.Budgets(ns) {
		if b.Spec.Selector == nil {
			continue
		}
		sel, err := metav1.LabelSelectorAsSelector(b.Spec.Selector)
		if err != nil || !sel.Matches(k8slabels.Set(p.Labels)) {
			continue
		}
		return budgetWords(b)
	}
	return ""
}

func budgetWords(b policyv1.PodDisruptionBudget) string {
	story, rule := "", ""
	switch {
	case b.Spec.MinAvailable != nil:
		story, rule = "at least "+b.Spec.MinAvailable.String()+" must stand", "minAvailable "+b.Spec.MinAvailable.String()
	case b.Spec.MaxUnavailable != nil:
		story, rule = "at most "+b.Spec.MaxUnavailable.String()+" may be away", "maxUnavailable "+b.Spec.MaxUnavailable.String()
	}
	return ui.Bridge(fmt.Sprintf("QUORUM %s/%s: %s, %d of %d standing, %d may leave now", b.Namespace, b.Name, story, b.Status.CurrentHealthy, b.Status.DesiredHealthy, b.Status.DisruptionsAllowed),
		fmt.Sprintf("PodDisruptionBudget %s/%s: %s, %d healthy of %d desired, %d disruptions allowed", b.Namespace, b.Name, rule, b.Status.CurrentHealthy, b.Status.DesiredHealthy, b.Status.DisruptionsAllowed))
}
