package livescenes

import (
	"bytes"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/view"
)

type Replayed struct {
	List   string
	Type   string
	Object any
}

type Trouble struct {
	Name  string
	Kind  string
	Break func(live.Snapshot) []Replayed
	Mend  func(live.Snapshot) []Replayed
}

const (
	troubleNS   = "market"
	troubleCrew = "storefront"
	struckAt    = 14
)

func struck(snap live.Snapshot) metav1.Time {
	return metav1.NewTime(snap.TakenAt.Add(struckAt * time.Minute))
}

func crewPod(snap live.Snapshot, crew string) corev1.Pod {
	for _, p := range snap.Pods {
		if p.Namespace == troubleNS && strings.HasPrefix(p.Name, crew+"-") && len(p.Status.ContainerStatuses) > 0 {
			return *p.DeepCopy()
		}
	}
	panic("no pod of " + crew)
}

func notReady(p *corev1.Pod) {
	for i := range p.Status.Conditions {
		if p.Status.Conditions[i].Type == corev1.PodReady || p.Status.Conditions[i].Type == corev1.ContainersReady {
			p.Status.Conditions[i].Status = corev1.ConditionFalse
		}
	}
	p.Status.ContainerStatuses[0].Ready = false
}

func CrashLooping(snap live.Snapshot, oom bool) corev1.Pod {
	p := crewPod(snap, troubleCrew)
	notReady(&p)
	cs := &p.Status.ContainerStatuses[0]
	cs.RestartCount = 4
	cs.State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff",
		Message: "back-off 1m20s restarting failed container=" + cs.Name + " pod=" + p.Name + "_" + p.Namespace + "(" + string(p.UID) + ")"}}
	last := &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1, FinishedAt: struck(snap)}
	if oom {
		last.Reason, last.ExitCode = "OOMKilled", 137
		p.Spec.Containers[0].Resources.Limits = corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("128Mi")}
	}
	cs.LastTerminationState = corev1.ContainerState{Terminated: last}
	return p
}

func PullingNothing(snap live.Snapshot) corev1.Pod {
	p := crewPod(snap, troubleCrew)
	notReady(&p)
	cs := &p.Status.ContainerStatuses[0]
	image := strings.Split(cs.Image, ":")[0] + ":0.0-never-published"
	cs.Image = image
	cs.State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff", Message: "Back-off pulling image \"" + image + "\""}}
	return p
}

func warningOn(snap live.Snapshot, name, kind, ns, object string, uid types.UID, reason, message string) corev1.Event {
	return corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, UID: types.UID("trouble-ev-" + name), CreationTimestamp: struck(snap)},
		InvolvedObject: corev1.ObjectReference{Kind: kind, Namespace: ns, Name: object, UID: uid}, Reason: reason, Message: message, Type: corev1.EventTypeWarning,
		Count: 5, FirstTimestamp: struck(snap), LastTimestamp: struck(snap)}
}

func normalOn(snap live.Snapshot, name, kind, ns, object string, uid types.UID, reason, message string) corev1.Event {
	e := warningOn(snap, name, kind, ns, object, uid, reason, message)
	e.Type, e.Count = corev1.EventTypeNormal, 1
	return e
}

const schedulerWords = "0/3 nodes are available: 3 Insufficient memory. preemption: 0/3 nodes are available: 3 No preemption victims found for incoming pod."

func unhousedPod(snap live.Snapshot, crew, name, rs, hash string) corev1.Pod {
	p := crewPod(snap, crew)
	p.Name, p.UID, p.Spec.NodeName, p.CreationTimestamp = name, types.UID("trouble-"+name), "", struck(snap)
	if rs != "" {
		yes := true
		p.OwnerReferences = []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: rs, UID: types.UID("trouble-" + rs), Controller: &yes}}
		p.Labels = map[string]string{"app": crew, "pod-template-hash": hash}
	}
	p.Spec.Containers[0].Resources.Requests = corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("64Gi")}
	p.Status = corev1.PodStatus{Phase: corev1.PodPending, Conditions: []corev1.PodCondition{{Type: corev1.PodScheduled, Status: corev1.ConditionFalse,
		Reason: "Unschedulable", Message: schedulerWords, LastTransitionTime: struck(snap)}}}
	return p
}

func deploymentOf(snap live.Snapshot, name string) appsv1.Deployment {
	for _, d := range snap.Deployments {
		if d.Namespace == troubleNS && d.Name == name {
			return *d.DeepCopy()
		}
	}
	panic("no deployment " + name)
}

func stalled(snap live.Snapshot) []Replayed {
	d := deploymentOf(snap, "checkout")
	want := int32(1)
	if d.Spec.Replicas != nil {
		want = *d.Spec.Replicas
	}
	rsName := d.Name + "-6f7d9c8b4"
	yes := true
	rs := appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: rsName, Namespace: troubleNS, UID: types.UID("trouble-" + rsName), CreationTimestamp: struck(snap),
		Labels: map[string]string{"app": d.Name, "pod-template-hash": "6f7d9c8b4"}, Annotations: map[string]string{"deployment.kubernetes.io/revision": "99"},
		OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: d.Name, UID: d.UID, Controller: &yes}}},
		Status: appsv1.ReplicaSetStatus{Replicas: 1}}
	p := unhousedPod(snap, d.Name, rsName+"-x4k2p", rsName, "6f7d9c8b4")
	d.Status.UpdatedReplicas, d.Status.Replicas = 1, want+1
	d.Status.Conditions = []appsv1.DeploymentCondition{{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded",
		Message: "ReplicaSet \"" + rsName + "\" has timed out progressing.", LastUpdateTime: struck(snap), LastTransitionTime: struck(snap)}}
	return []Replayed{{"replicasets", "ADDED", rs}, {"pods", "ADDED", p}, {"deployments", "MODIFIED", d}}
}

func cordoned(snap live.Snapshot) corev1.Node {
	n := *snap.Nodes[len(snap.Nodes)-1].DeepCopy()
	n.Spec.Unschedulable = true
	return n
}

func failedJob(snap live.Snapshot) batchv1.Job {
	j := *snap.Jobs[0].DeepCopy()
	j.Status.Failed, j.Status.Active, j.Status.Succeeded = 6, 0, 0
	j.Status.Conditions = append(j.Status.Conditions, batchv1.JobCondition{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: "BackoffLimitExceeded",
		Message: "Job has reached the specified backoff limit", LastTransitionTime: struck(snap)})
	return j
}

func pendingClaim(snap live.Snapshot) corev1.PersistentVolumeClaim {
	c := *snap.PVCs[0].DeepCopy()
	c.Status = corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending}
	return c
}

func hornAtMost(snap live.Snapshot) autoscalingv2.HorizontalPodAutoscaler {
	return autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: troubleCrew, Namespace: troubleNS, UID: "trouble-horn", CreationTimestamp: struck(snap)},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{MaxReplicas: 4, ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: troubleCrew}},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{CurrentReplicas: 4, DesiredReplicas: 4, Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{
			{Type: autoscalingv2.ScalingLimited, Status: corev1.ConditionTrue, Reason: "TooManyReplicas", Message: "the desired replica count is more than the maximum replica count", LastTransitionTime: struck(snap)}}}}
}

func one(list, typ string, obj any) []Replayed {
	return []Replayed{{list, typ, obj}}
}

var Troubles = []Trouble{
	{"crashloop", view.TagCrashLoop,
		func(s live.Snapshot) []Replayed { return one("pods", "MODIFIED", CrashLooping(s, false)) },
		func(s live.Snapshot) []Replayed { return one("pods", "MODIFIED", crewPod(s, troubleCrew)) }},
	{"oomkilled", view.TagOOMKilled,
		func(s live.Snapshot) []Replayed { return one("pods", "MODIFIED", CrashLooping(s, true)) },
		func(s live.Snapshot) []Replayed { return one("pods", "MODIFIED", crewPod(s, troubleCrew)) }},
	{"imagepull", view.TagImagePull,
		func(s live.Snapshot) []Replayed { return one("pods", "MODIFIED", PullingNothing(s)) },
		func(s live.Snapshot) []Replayed { return one("pods", "MODIFIED", crewPod(s, troubleCrew)) }},
	{"unhoused", view.TagUnhoused,
		func(s live.Snapshot) []Replayed {
			p := unhousedPod(s, troubleCrew, troubleCrew+"-giant", "", "")
			return []Replayed{{"pods", "ADDED", p}, {"events", "ADDED", warningOn(s, "giant-sched", "Pod", troubleNS, p.Name, p.UID, "FailedScheduling", schedulerWords)}}
		},
		func(s live.Snapshot) []Replayed {
			return one("pods", "DELETED", unhousedPod(s, troubleCrew, troubleCrew+"-giant", "", ""))
		}},
	{"evicted", view.TagEvicted,
		func(s live.Snapshot) []Replayed {
			p := crewPod(s, troubleCrew)
			p.Status = corev1.PodStatus{Phase: corev1.PodFailed, Reason: "Evicted", Message: "Pod ephemeral local storage usage exceeds the total limit of containers 64Mi. ", StartTime: &metav1.Time{Time: struck(s).Time}}
			return one("pods", "MODIFIED", p)
		},
		func(s live.Snapshot) []Replayed { return one("pods", "DELETED", crewPod(s, troubleCrew)) }},
	{"stalled", view.CalamityStalled, stalled,
		func(s live.Snapshot) []Replayed {
			r := stalled(s)
			return []Replayed{{"pods", "DELETED", r[1].Object}, {"replicasets", "DELETED", r[0].Object}, {"deployments", "MODIFIED", deploymentOf(s, "checkout")}}
		}},
	{"report-hut", view.ReportHut,
		func(s live.Snapshot) []Replayed {
			p := crewPod(s, "catalog")
			notReady(&p)
			return []Replayed{{"pods", "MODIFIED", p}, {"events", "ADDED", warningOn(s, "catalog-probe", "Pod", troubleNS, p.Name, p.UID, "Unhealthy",
				"Readiness probe failed: HTTP probe failed with statuscode: 503")}}
		},
		func(s live.Snapshot) []Replayed { return one("pods", "MODIFIED", crewPod(s, "catalog")) }},
	{"report-keep", view.ReportKeep,
		func(s live.Snapshot) []Replayed {
			n := cordoned(s)
			return []Replayed{{"nodes", "MODIFIED", n}, {"events", "ADDED", normalOn(s, "cordon", "Node", "", n.Name, n.UID, "NodeNotSchedulable",
				"Node "+n.Name+" status is now: NodeNotSchedulable")}}
		},
		func(s live.Snapshot) []Replayed { return one("nodes", "MODIFIED", *s.Nodes[len(s.Nodes)-1].DeepCopy()) }},
	{"report-quest", view.ReportQuest,
		func(s live.Snapshot) []Replayed { return one("jobs", "MODIFIED", failedJob(s)) },
		func(s live.Snapshot) []Replayed { return one("jobs", "DELETED", failedJob(s)) }},
	{"report-deed", view.ReportDeed,
		func(s live.Snapshot) []Replayed {
			c := pendingClaim(s)
			return []Replayed{{"persistentvolumeclaims", "MODIFIED", c}, {"events", "ADDED", warningOn(s, "claim", "PersistentVolumeClaim", c.Namespace, c.Name, c.UID,
				"ProvisioningFailed", "storageclass.storage.k8s.io \"fast-ssd\" not found")}}
		},
		func(s live.Snapshot) []Replayed {
			return one("persistentvolumeclaims", "MODIFIED", *s.PVCs[0].DeepCopy())
		}},
	{"report-horn", view.ReportHorn,
		func(s live.Snapshot) []Replayed { return one("horizontalpodautoscalers", "ADDED", hornAtMost(s)) },
		func(s live.Snapshot) []Replayed { return one("horizontalpodautoscalers", "DELETED", hornAtMost(s)) }},
}

func TroubleNamed(name string) Trouble {
	for _, t := range Troubles {
		if t.Name == name {
			return t
		}
	}
	panic("no trouble " + name)
}

func Play(r *live.Realm, events []Replayed) error {
	for _, e := range events {
		if err := r.Replay(e.List, bytes.NewReader(live.WatchEvent(e.Type, e.Object))); err != nil {
			return err
		}
	}
	return nil
}
