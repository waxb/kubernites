package live

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	reasonCrashLoop   = "CrashLoopBackOff"
	reasonOOMKilled   = "OOMKilled"
	reasonPullBackOff = "ImagePullBackOff"
	reasonErrPull     = "ErrImagePull"
	reasonUnschedule  = "Unschedulable"
	reasonFailedSched = "FailedScheduling"
	reasonEvicted     = "Evicted"
	reasonUnhealthy   = "Unhealthy"
	reasonDeadline    = "ProgressDeadlineExceeded"
	reasonTooMany     = "TooManyReplicas"
	reasonNotSched    = "NodeNotSchedulable"
	revisionNote      = "deployment.kubernetes.io/revision"
	podTemplateHash   = "pod-template-hash"
	hutsListed        = 4
	seeLabel          = "SEE IT IN KUBERNETES"
	clockFormat       = "15:04:05 UTC"
)

var waitingErrors = map[string]bool{
	"CreateContainerConfigError": true, "CreateContainerError": true, "InvalidImageName": true, "RunContainerError": true, "PreStartHookError": true, "PostStartHookError": true,
}

type crewKey struct {
	ns, kind, name string
}

func (k crewKey) id() string {
	if k.kind == "" {
		return k.ns + "/" + k.name
	}
	return k.ns + "/" + k.kind + "/" + k.name
}

func (k crewKey) words() string {
	if k.kind == "" {
		return "Pod " + k.ns + "/" + k.name
	}
	return strings.ToLower(k.kind) + " " + k.ns + "/" + k.name
}

func (k crewKey) ref() string {
	if k.kind == "" {
		return "pod/" + k.name
	}
	return strings.ToLower(k.kind) + "/" + k.name
}

type finding struct {
	tag    string
	crew   crewKey
	pod    string
	node   string
	cause  string
	why    string
	at     time.Time
	object string
	see    view.RemedyView
	fix    *view.RemedyView
}

type mirrorIndex struct {
	s       *Snapshot
	rsOwner map[string]*metav1.OwnerReference
	jobOwn  map[string]*metav1.OwnerReference
	events  map[types.UID][]*corev1.Event
	byName  map[string][]*corev1.Event
}

func indexMirror(s *Snapshot) *mirrorIndex {
	ix := &mirrorIndex{s: s, rsOwner: map[string]*metav1.OwnerReference{}, jobOwn: map[string]*metav1.OwnerReference{}, events: map[types.UID][]*corev1.Event{}, byName: map[string][]*corev1.Event{}}
	for i := range s.ReplicaSets {
		rs := &s.ReplicaSets[i]
		ix.rsOwner[rs.Namespace+"/"+rs.Name] = metav1.GetControllerOfNoCopy(rs)
	}
	for i := range s.Jobs {
		j := &s.Jobs[i]
		ix.jobOwn[j.Namespace+"/"+j.Name] = metav1.GetControllerOfNoCopy(j)
	}
	for i := range s.Events {
		e := &s.Events[i]
		if e.InvolvedObject.UID != "" {
			ix.events[e.InvolvedObject.UID] = append(ix.events[e.InvolvedObject.UID], e)
		}
		key := e.InvolvedObject.Kind + "/" + e.InvolvedObject.Namespace + "/" + e.InvolvedObject.Name
		ix.byName[key] = append(ix.byName[key], e)
	}
	return ix
}

func eventTime(e *corev1.Event) time.Time {
	switch {
	case e.Series != nil && !e.Series.LastObservedTime.IsZero():
		return e.Series.LastObservedTime.Time
	case !e.LastTimestamp.IsZero():
		return e.LastTimestamp.Time
	case !e.EventTime.IsZero():
		return e.EventTime.Time
	}
	return e.CreationTimestamp.Time
}

func eventCount(e *corev1.Event) int32 {
	if e.Series != nil && e.Series.Count > 0 {
		return e.Series.Count
	}
	return max(e.Count, 1)
}

func (ix *mirrorIndex) latest(uid types.UID, kind, ns, name, reason, typ string) *corev1.Event {
	list := ix.events[uid]
	if uid == "" || len(list) == 0 {
		list = ix.byName[kind+"/"+ns+"/"+name]
	}
	var best *corev1.Event
	for _, e := range list {
		if reason != "" && e.Reason != reason || typ != "" && e.Type != typ {
			continue
		}
		if best == nil || eventTime(e).After(eventTime(best)) || eventTime(e).Equal(eventTime(best)) && e.Name > best.Name {
			best = e
		}
	}
	return best
}

func (ix *mirrorIndex) crewOf(p *corev1.Pod) crewKey {
	ref := metav1.GetControllerOfNoCopy(p)
	if ref == nil {
		return crewKey{ns: p.Namespace, name: p.Name}
	}
	top := crewKey{ns: p.Namespace, kind: ref.Kind, name: ref.Name}
	for range 3 {
		var up *metav1.OwnerReference
		switch top.kind {
		case "ReplicaSet":
			up = ix.rsOwner[top.ns+"/"+top.name]
		case "Job":
			up = ix.jobOwn[top.ns+"/"+top.name]
		}
		if up == nil {
			break
		}
		top = crewKey{ns: top.ns, kind: up.Kind, name: up.Name}
	}
	return top
}

func see(command string) view.RemedyView {
	return view.RemedyView{Label: seeLabel, Command: command}
}

func unwired(label, command string) *view.RemedyView {
	return &view.RemedyView{Label: label, Command: command, Unwired: true}
}

func ordered(r *view.RemedyView, key, kind, ns, name string) *view.RemedyView {
	if r == nil {
		return nil
	}
	r.Order, r.Target = key, view.ObjectRef{Kind: kind, Namespace: ns, Name: name}
	return r
}

func inNS(ns string) string {
	return " -n " + ns
}

func describePod(p *corev1.Pod) view.RemedyView {
	return see("kubectl describe pod " + p.Name + inNS(p.Namespace))
}

func rollable(k crewKey) bool {
	switch k.kind {
	case "Deployment", "StatefulSet", "DaemonSet":
		return true
	}
	return false
}

func undo(k crewKey) *view.RemedyView {
	if !rollable(k) {
		return nil
	}
	r := unwired("Roll "+k.words()+" back to its last revision", "kubectl rollout undo "+k.ref()+inNS(k.ns))
	if k.kind == "Deployment" {
		return ordered(r, "rollback", k.kind, k.ns, k.name)
	}
	return r
}

func lastWords(t *corev1.ContainerStateTerminated) string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("Last state: Terminated (%s, exit code %d) at %s.", t.Reason, t.ExitCode, t.FinishedAt.UTC().Format(clockFormat))
}

func containerStatuses(p *corev1.Pod) []corev1.ContainerStatus {
	return append(append([]corev1.ContainerStatus(nil), p.Status.InitContainerStatuses...), p.Status.ContainerStatuses...)
}

func memoryLimit(p *corev1.Pod, container string) string {
	for _, c := range append(append([]corev1.Container(nil), p.Spec.InitContainers...), p.Spec.Containers...) {
		if c.Name != container {
			continue
		}
		if q, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
			return q.String()
		}
	}
	return ""
}

func doubled(limit string) string {
	digits := strings.TrimRightFunc(limit, func(r rune) bool { return r < '0' || r > '9' })
	n, err := strconv.Atoi(digits)
	if err != nil || n == 0 {
		return ""
	}
	return strconv.Itoa(2*n) + limit[len(digits):]
}

func (ix *mirrorIndex) podFindings(p *corev1.Pod) []finding {
	crew := ix.crewOf(p)
	base := finding{crew: crew, pod: p.Name, node: p.Spec.NodeName, object: "pod/" + p.Namespace + "/" + p.Name, at: p.CreationTimestamp.Time}
	if p.Status.Phase == corev1.PodFailed && p.Status.Reason == reasonEvicted {
		f := base
		f.tag, f.cause = view.TagEvicted, reasonEvicted+": "+p.Status.Message
		f.why = "Node: " + p.Spec.NodeName + ". Phase: Failed."
		f.at = podStarted(p)
		f.see = describePod(p)
		f.fix = ordered(unwired("Delete the evicted Pod "+p.Name, "kubectl delete pod "+p.Name+inNS(p.Namespace)), "delete", "Pod", p.Namespace, p.Name)
		return []finding{f}
	}
	if p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
		return nil
	}
	var out []finding
	for _, cs := range containerStatuses(p) {
		if f, ok := ix.containerFinding(base, p, cs); ok {
			out = append(out, f)
		}
	}
	if len(out) > 0 {
		return out
	}
	if f, ok := ix.unhoused(base, p); ok {
		return []finding{f}
	}
	if f, ok := ix.unhealthy(base, p); ok {
		return []finding{f}
	}
	return nil
}

func podStarted(p *corev1.Pod) time.Time {
	if p.Status.StartTime != nil {
		return p.Status.StartTime.Time
	}
	return p.CreationTimestamp.Time
}

func (ix *mirrorIndex) containerFinding(base finding, p *corev1.Pod, cs corev1.ContainerStatus) (finding, bool) {
	f := base
	w, last := cs.State.Waiting, cs.LastTerminationState.Terminated
	if t := cs.State.Terminated; t != nil && t.Reason == reasonOOMKilled {
		last = t
	}
	down := cs.State.Waiting != nil || cs.State.Terminated != nil
	switch {
	case down && last != nil && last.Reason == reasonOOMKilled:
		f.tag = view.TagOOMKilled
		f.cause = fmt.Sprintf("%s: container %q was terminated with exit code %d at %s.", reasonOOMKilled, cs.Name, last.ExitCode, last.FinishedAt.UTC().Format(clockFormat))
		f.at = last.FinishedAt.Time
		limit := memoryLimit(p, cs.Name)
		f.why = "It has no memory limit: the node ran out of memory."
		if limit != "" {
			f.why = "Its memory limit is " + limit + "."
		}
		f.why += fmt.Sprintf(" Restart count: %d.", cs.RestartCount)
		f.see = describePod(p)
		if more := doubled(limit); more != "" && f.crew.kind != "" && rollable(f.crew) {
			f.fix = unwired("Raise the memory limit of "+cs.Name+" to "+more, fmt.Sprintf("kubectl set resources %s -c %s --limits=memory=%s%s", f.crew.ref(), cs.Name, more, inNS(p.Namespace)))
			if f.crew.kind == "Deployment" {
				f.fix = ordered(f.fix, "resources", f.crew.kind, f.crew.ns, f.crew.name)
			}
		}
	case w != nil && w.Reason == reasonCrashLoop:
		f.tag = view.TagCrashLoop
		f.cause = reasonCrashLoop + ": " + w.Message
		f.why = strings.TrimSpace(lastWords(last) + fmt.Sprintf(" Restart count: %d.", cs.RestartCount))
		if last != nil {
			f.at = last.FinishedAt.Time
		}
		f.see = view.RemedyView{Label: "Read what " + cs.Name + " wrote before it stopped", Command: "kubectl logs " + p.Name + " -c " + cs.Name + inNS(p.Namespace) + " --previous"}
		f.fix = undo(f.crew)
	case cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 && cs.RestartCount > 0 && p.Spec.RestartPolicy != corev1.RestartPolicyNever:
		t := cs.State.Terminated
		f.tag = view.TagCrashLoop
		f.cause = fmt.Sprintf("%s: container %q exited with code %d at %s.", t.Reason, cs.Name, t.ExitCode, t.FinishedAt.UTC().Format(clockFormat))
		f.why = fmt.Sprintf("Restart count: %d.", cs.RestartCount)
		f.at = t.FinishedAt.Time
		f.see = view.RemedyView{Label: "Read what " + cs.Name + " wrote before it stopped", Command: "kubectl logs " + p.Name + " -c " + cs.Name + inNS(p.Namespace) + " --previous"}
		f.fix = undo(f.crew)
	case w != nil && (w.Reason == reasonPullBackOff || w.Reason == reasonErrPull):
		f.tag = view.TagImagePull
		f.cause = w.Reason + ": " + w.Message
		f.why = "Image: " + cs.Image + "."
		if e := ix.latest(p.UID, "Pod", p.Namespace, p.Name, "Failed", corev1.EventTypeWarning); e != nil {
			f.at = eventTime(e)
		}
		f.see = describePod(p)
		if f.crew.kind == "Deployment" || f.crew.kind == "StatefulSet" || f.crew.kind == "DaemonSet" {
			f.fix = unwired("Give "+cs.Name+" an image that exists", fmt.Sprintf("kubectl set image %s %s=<image>%s", f.crew.ref(), cs.Name, inNS(p.Namespace)))
			if f.crew.kind == "Deployment" {
				f.fix = ordered(f.fix, "image", f.crew.kind, f.crew.ns, f.crew.name)
			}
		}
	case w != nil && waitingErrors[w.Reason]:
		f.tag = view.ReportHut
		f.cause = w.Reason + ": " + w.Message
		f.why = "Container: " + cs.Name + "."
		f.see = describePod(p)
	default:
		return finding{}, false
	}
	return f, true
}

func (ix *mirrorIndex) unhoused(base finding, p *corev1.Pod) (finding, bool) {
	if p.Status.Phase != corev1.PodPending || p.Spec.NodeName != "" {
		return finding{}, false
	}
	message := ""
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodScheduled && c.Status == corev1.ConditionFalse && c.Reason == reasonUnschedule {
			message = c.Message
			base.at = c.LastTransitionTime.Time
		}
	}
	if e := ix.latest(p.UID, "Pod", p.Namespace, p.Name, reasonFailedSched, ""); e != nil {
		message = e.Message
		base.at = eventTime(e)
	}
	if message == "" {
		return finding{}, false
	}
	f := base
	f.tag, f.cause = view.TagUnhoused, reasonFailedSched+": "+message
	f.why = "Requests: " + requestsOf(p) + "."
	f.see = describePod(p)
	if f.crew.kind == "Deployment" || f.crew.kind == "StatefulSet" {
		f.fix = unwired("Lower the requests of "+f.crew.words(), "kubectl edit "+f.crew.ref()+inNS(p.Namespace))
	}
	return f, true
}

func requestsOf(p *corev1.Pod) string {
	var parts []string
	for _, c := range p.Spec.Containers {
		for _, r := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory, corev1.ResourceEphemeralStorage} {
			if q, ok := c.Resources.Requests[r]; ok {
				parts = append(parts, c.Name+" "+string(r)+" "+q.String())
			}
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func (ix *mirrorIndex) unhealthy(base finding, p *corev1.Pod) (finding, bool) {
	down := false
	for _, cs := range p.Status.ContainerStatuses {
		down = down || cs.State.Running != nil && !cs.Ready
	}
	e := ix.latest(p.UID, "Pod", p.Namespace, p.Name, reasonUnhealthy, corev1.EventTypeWarning)
	if !down || e == nil {
		return finding{}, false
	}
	f := base
	f.tag, f.cause = view.ReportHut, reasonUnhealthy+": "+e.Message
	f.why = fmt.Sprintf("Seen %d times, last at %s.", eventCount(e), eventTime(e).UTC().Format(clockFormat))
	f.at = eventTime(e)
	f.see = describePod(p)
	if rollable(f.crew) {
		f.fix = unwired("Fix the probe of "+f.crew.words(), "kubectl edit "+f.crew.ref()+inNS(p.Namespace))
	}
	return f, true
}

func newestRS(d *appsv1.Deployment, sets []appsv1.ReplicaSet) (*appsv1.ReplicaSet, int32) {
	var best *appsv1.ReplicaSet
	bestRev := -1
	var old int32
	for i := range sets {
		rs := &sets[i]
		ref := metav1.GetControllerOfNoCopy(rs)
		if rs.Namespace != d.Namespace || ref == nil || ref.UID != d.UID {
			continue
		}
		rev, _ := strconv.Atoi(rs.Annotations[revisionNote])
		if rev > bestRev {
			if best != nil {
				old += best.Status.Replicas
			}
			best, bestRev = rs, rev
			continue
		}
		old += rs.Status.Replicas
	}
	return best, old
}

func (ix *mirrorIndex) stalls(found []finding) ([]finding, []finding) {
	var stalled []finding
	held := map[string]bool{}
	for i := range ix.s.Deployments {
		d := &ix.s.Deployments[i]
		rs, old := newestRS(d, ix.s.ReplicaSets)
		want := int32(1)
		if d.Spec.Replicas != nil {
			want = *d.Spec.Replicas
		}
		if rs == nil || old == 0 && d.Status.UpdatedReplicas >= want {
			continue
		}
		crew := crewKey{ns: d.Namespace, kind: "Deployment", name: d.Name}
		var waiting []finding
		for _, f := range found {
			if f.tag == view.TagUnhoused && f.crew == crew && ix.podHash(f) == rs.Labels[podTemplateHash] {
				waiting = append(waiting, f)
			}
		}
		if len(waiting) == 0 || old == 0 {
			continue
		}
		held[crew.id()] = true
		f := waiting[0]
		f.tag = view.CalamityStalled
		rebuilt := max(0, int(d.Status.UpdatedReplicas)-len(waiting))
		f.cause = fmt.Sprintf("The roll-out of %s stalled at %d of %d: %d new %s Pending, and no Node has room. %s", crew.words(), rebuilt, want, len(waiting), podsAre(len(waiting)), strings.TrimPrefix(waiting[0].cause, reasonFailedSched+": "))
		f.why = ""
		for _, c := range d.Status.Conditions {
			if c.Type == appsv1.DeploymentProgressing {
				f.why = c.Reason + ": " + c.Message
				if c.Reason == reasonDeadline {
					f.at = c.LastUpdateTime.Time
				}
			}
		}
		f.see = see("kubectl rollout status deployment/" + d.Name + inNS(d.Namespace))
		f.fix = ordered(unwired("Roll "+crew.words()+" back: its old Pods fit where they stood", "kubectl rollout undo deployment/"+d.Name+inNS(d.Namespace)), "rollback", "Deployment", d.Namespace, d.Name)
		stalled = append(stalled, f)
	}
	var rest []finding
	for _, f := range found {
		if !(f.tag == view.TagUnhoused && held[f.crew.id()]) {
			rest = append(rest, f)
		}
	}
	return rest, stalled
}

func podsAre(n int) string {
	if n == 1 {
		return "Pod is"
	}
	return "Pods are"
}

func (ix *mirrorIndex) podHash(f finding) string {
	for i := range ix.s.Pods {
		p := &ix.s.Pods[i]
		if p.Name == f.pod && p.Namespace == f.crew.ns {
			return p.Labels[podTemplateHash]
		}
	}
	return ""
}

func (ix *mirrorIndex) nodeFindings() []finding {
	var out []finding
	for i := range ix.s.Nodes {
		n := &ix.s.Nodes[i]
		f := finding{tag: view.ReportKeep, crew: crewKey{kind: "Node", name: n.Name}, node: n.Name, object: "node/" + n.Name}
		f.see = see("kubectl describe node " + n.Name)
		var causes []string
		for _, c := range n.Status.Conditions {
			switch {
			case c.Type == corev1.NodeReady && c.Status != corev1.ConditionTrue:
				causes = append(causes, "NotReady: "+c.Reason+": "+c.Message)
				f.at = c.LastTransitionTime.Time
				f.fix = ordered(unwired("Drain "+n.Name+" so its Pods start elsewhere", "kubectl drain "+n.Name+" --ignore-daemonsets"), "drain", "Node", "", n.Name)
			case c.Type != corev1.NodeReady && c.Status == corev1.ConditionTrue && strings.HasSuffix(string(c.Type), "Pressure"):
				causes = append(causes, string(c.Type)+": "+c.Reason+": "+c.Message)
				f.at = c.LastTransitionTime.Time
				f.fix = ordered(unwired("Drain "+n.Name+" so its Pods start elsewhere", "kubectl drain "+n.Name+" --ignore-daemonsets"), "drain", "Node", "", n.Name)
			}
		}
		if n.Spec.Unschedulable {
			text := "SchedulingDisabled: the node is cordoned (spec.unschedulable is true)."
			if e := ix.latest(n.UID, "Node", "", n.Name, reasonNotSched, ""); e != nil {
				text = "SchedulingDisabled: " + e.Message
				if len(causes) == 0 {
					f.at = eventTime(e)
				}
			}
			causes = append(causes, text)
			if f.fix == nil {
				f.fix = ordered(unwired("Uncordon "+n.Name, "kubectl uncordon "+n.Name), "uncordon", "Node", "", n.Name)
			}
		}
		if len(causes) == 0 {
			continue
		}
		f.cause, f.why = causes[0], strings.Join(causes[1:], " ")
		out = append(out, f)
	}
	return out
}

func (ix *mirrorIndex) jobFindings() []finding {
	var out []finding
	for i := range ix.s.Jobs {
		j := &ix.s.Jobs[i]
		for _, c := range j.Status.Conditions {
			if c.Type != batchv1.JobFailed || c.Status != corev1.ConditionTrue {
				continue
			}
			crew := crewKey{ns: j.Namespace, kind: "Job", name: j.Name}
			f := finding{tag: view.ReportQuest, crew: crew, cause: "Failed: " + c.Reason + ": " + c.Message, at: c.LastTransitionTime.Time,
				why: fmt.Sprintf("Failed Pods: %d. Succeeded: %d.", j.Status.Failed, j.Status.Succeeded)}
			f.fix = ordered(unwired("Delete the failed Job "+j.Name, "kubectl delete job "+j.Name+inNS(j.Namespace)), "delete", "Job", j.Namespace, j.Name)
			f.see = see("kubectl get pods -l job-name=" + j.Name + inNS(j.Namespace))
			if p := ix.newestPodOf(j.Namespace, "Job", j.Name); p != nil {
				f.pod, f.object, f.node = p.Name, "pod/"+p.Namespace+"/"+p.Name, p.Spec.NodeName
				f.see = view.RemedyView{Label: "Read what its last Pod wrote", Command: "kubectl logs " + p.Name + inNS(p.Namespace)}
			}
			out = append(out, f)
		}
	}
	return out
}

func (ix *mirrorIndex) newestPodOf(ns, kind, name string) *corev1.Pod {
	var best *corev1.Pod
	for i := range ix.s.Pods {
		p := &ix.s.Pods[i]
		ref := metav1.GetControllerOfNoCopy(p)
		if p.Namespace != ns || ref == nil || ref.Kind != kind || ref.Name != name {
			continue
		}
		if best == nil || p.CreationTimestamp.After(best.CreationTimestamp.Time) || p.CreationTimestamp.Equal(&best.CreationTimestamp) && p.Name > best.Name {
			best = p
		}
	}
	return best
}

func (ix *mirrorIndex) firstPodOf(crew crewKey) *corev1.Pod {
	var best *corev1.Pod
	for i := range ix.s.Pods {
		p := &ix.s.Pods[i]
		if p.Namespace == crew.ns && ix.crewOf(p) == crew && (best == nil || p.Name < best.Name) {
			best = p
		}
	}
	return best
}

func (ix *mirrorIndex) claimFindings() []finding {
	var out []finding
	for i := range ix.s.PVCs {
		c := &ix.s.PVCs[i]
		if c.Status.Phase != corev1.ClaimPending {
			continue
		}
		e := ix.latest(c.UID, "PersistentVolumeClaim", c.Namespace, c.Name, "", "")
		if e == nil || e.Type != corev1.EventTypeWarning {
			continue
		}
		class := "none"
		if c.Spec.StorageClassName != nil {
			class = *c.Spec.StorageClassName
		}
		f := finding{tag: view.ReportDeed, crew: crewKey{ns: c.Namespace, kind: "PersistentVolumeClaim", name: c.Name}, cause: e.Reason + ": " + e.Message,
			why: "Phase: Pending. StorageClass: " + class + ".", at: eventTime(e)}
		f.see = see("kubectl describe pvc " + c.Name + inNS(c.Namespace))
		f.fix = ordered(unwired("Delete the claim and write it again with a StorageClass that exists", "kubectl delete pvc "+c.Name+inNS(c.Namespace)), "delete", "PersistentVolumeClaim", c.Namespace, c.Name)
		out = append(out, f)
	}
	return out
}

func (ix *mirrorIndex) hornFindings() []finding {
	var out []finding
	for i := range ix.s.HPAs {
		h := &ix.s.HPAs[i]
		for _, c := range h.Status.Conditions {
			if c.Type != autoscalingv2.ScalingLimited || c.Status != corev1.ConditionTrue || c.Reason != reasonTooMany {
				continue
			}
			f := finding{tag: view.ReportHorn, crew: crewKey{ns: h.Namespace, kind: "HorizontalPodAutoscaler", name: h.Name},
				cause: "ScalingLimited: " + c.Reason + ": " + c.Message, at: c.LastTransitionTime.Time,
				why: fmt.Sprintf("Replicas: %d, desired %d, maximum %d.", h.Status.CurrentReplicas, h.Status.DesiredReplicas, h.Spec.MaxReplicas)}
			f.see = see("kubectl describe hpa " + h.Name + inNS(h.Namespace))
			f.fix = ordered(unwired(fmt.Sprintf("Raise the maximum of %s to %d", h.Name, 2*h.Spec.MaxReplicas),
				fmt.Sprintf("kubectl patch hpa %s%s -p {\"spec\":{\"maxReplicas\":%d}}", h.Name, inNS(h.Namespace), 2*h.Spec.MaxReplicas)), "retune", "HorizontalPodAutoscaler", h.Namespace, h.Name)
			if p := ix.firstPodOf(crewKey{ns: h.Namespace, kind: h.Spec.ScaleTargetRef.Kind, name: h.Spec.ScaleTargetRef.Name}); p != nil {
				f.object = "pod/" + p.Namespace + "/" + p.Name
			}
			out = append(out, f)
		}
	}
	return out
}

type signal struct {
	view view.CalamityView
	at   time.Time
	crew crewKey
	pod  string
}

func findings(s *Snapshot) []finding {
	ix := indexMirror(s)
	var found []finding
	for i := range s.Pods {
		found = append(found, ix.podFindings(&s.Pods[i])...)
	}
	found, stalled := ix.stalls(found)
	found = append(found, stalled...)
	found = append(found, ix.nodeFindings()...)
	found = append(found, ix.jobFindings()...)
	found = append(found, ix.claimFindings()...)
	return append(found, ix.hornFindings()...)
}

func Signals(s *Snapshot) []view.CalamityView {
	sigs := signals(s)
	out := make([]view.CalamityView, len(sigs))
	for i, g := range sigs {
		out[i] = g.view
	}
	return out
}

func signals(s *Snapshot) []signal {
	byID := map[string][]finding{}
	for _, f := range findings(s) {
		id := f.tag + ":" + f.crew.id()
		byID[id] = append(byID[id], f)
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]signal, 0, len(ids))
	for _, id := range ids {
		list := byID[id]
		sort.SliceStable(list, func(i, j int) bool {
			if !list[i].at.Equal(list[j].at) {
				return list[i].at.After(list[j].at)
			}
			return list[i].pod < list[j].pod
		})
		out = append(out, signal{view: calamityOf(id, list), at: list[0].at, crew: list[0].crew, pod: list[0].pod})
	}
	return out
}

func calamityOf(id string, list []finding) view.CalamityView {
	f := list[0]
	v := view.CalamityView{ID: id, Kind: f.tag, Object: f.object, Cause: f.cause, Why: f.why, Remedies: []view.RemedyView{f.see}}
	if f.fix != nil {
		v.Remedies = append(v.Remedies, *f.fix)
	}
	if pods := podsOf(list); len(pods) > 1 || len(pods) == 1 && f.crew.kind != "" {
		v.Why = strings.TrimSpace(v.Why + " " + hutList(f.crew, pods))
	}
	return v
}

func podsOf(list []finding) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range list {
		if f.pod != "" && !seen[f.pod] {
			seen[f.pod] = true
			out = append(out, f.pod)
		}
	}
	sort.Strings(out)
	return out
}

func hutList(crew crewKey, pods []string) string {
	shown := pods[:min(len(pods), hutsListed)]
	text := fmt.Sprintf("%d %s of %s: %s", len(pods), podWord(len(pods)), crew.words(), strings.Join(shown, ", "))
	if len(pods) > len(shown) {
		text += fmt.Sprintf(" and %d more", len(pods)-len(shown))
	}
	return text + "."
}

func podWord(n int) string {
	if n == 1 {
		return "Pod"
	}
	return "Pods"
}
