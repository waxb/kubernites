package kubectl

import (
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/sim/volumeutil"
)

const (
	pvKeyWidth  = 17
	pvcKeyWidth = 15
	scKeyWidth  = 23
	stsKeyWidth = 20
)

func describeStorage(c Cluster, o store.Object) (string, bool) {
	switch obj := o.(type) {
	case *corev1.PersistentVolume:
		return describePV(c, obj), true
	case *corev1.PersistentVolumeClaim:
		return describePVC(c, obj), true
	case *storagev1.StorageClass:
		return describeSC(c, obj), true
	case *appsv1.StatefulSet:
		return describeSTS(c, obj), true
	}
	return "", false
}

func orNoneMap(m map[string]string) string {
	if len(m) == 0 {
		return "<none>"
	}
	return labels.FormatLabels(m)
}

func withEvents(b *block, c Cluster, o store.Object, width int) string {
	if len(eventsFor(c, o.GetUID())) == 0 {
		b.kv(0, width, "Events", "<none>")
		return b.String()
	}
	b.raw(eventsSection(c, o.GetUID()))
	return b.String()
}

func describePV(c Cluster, pv *corev1.PersistentVolume) string {
	b := &block{}
	claim := ""
	if ref := pv.Spec.ClaimRef; ref != nil {
		claim = ref.Namespace + "/" + ref.Name
	}
	size := pv.Spec.Capacity[corev1.ResourceStorage]
	for _, kv := range [][2]string{
		{"Name", pv.Name}, {"Labels", orNoneMap(pv.Labels)}, {"Annotations", orNoneMap(pv.Annotations)},
		{"Finalizers", fmt.Sprint(pv.Finalizers)}, {"StorageClass", pv.Spec.StorageClassName},
		{"Status", deletingStatus(c, pv, string(pv.Status.Phase))}, {"Claim", claim},
		{"Reclaim Policy", string(pv.Spec.PersistentVolumeReclaimPolicy)}, {"Access Modes", accessModes(pv.Spec.AccessModes)},
		{"VolumeMode", volumeModeOf(pv.Spec.VolumeMode)}, {"Capacity", size.String()},
	} {
		b.kv(0, pvKeyWidth, kv[0], kv[1])
	}
	describeNodeAffinity(b, pv)
	b.kv(0, pvKeyWidth, "Message", pv.Status.Message)
	if csi := pv.Spec.CSI; csi != nil {
		b.raw("Source:")
		b.kv(4, 19, "Type", "CSI (a Container Storage Interface (CSI) volume source)")
		b.kv(4, 19, "Driver", csi.Driver)
		b.kv(4, 19, "VolumeHandle", csi.VolumeHandle)
		b.kv(4, 19, "ReadOnly", fmt.Sprint(csi.ReadOnly))
	}
	return withEvents(b, c, pv, pvKeyWidth)
}

func describeNodeAffinity(b *block, pv *corev1.PersistentVolume) {
	if pv.Spec.NodeAffinity == nil || pv.Spec.NodeAffinity.Required == nil {
		b.kv(0, pvKeyWidth, "Node Affinity", "<none>")
		return
	}
	b.kv(0, pvKeyWidth, "Node Affinity", "")
	b.kv(2, 15, "Required Terms", "")
	for i, term := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
		parts := []string{}
		for _, req := range term.MatchExpressions {
			parts = append(parts, fmt.Sprintf("%s %s [%s]", req.Key, strings.ToLower(string(req.Operator)), strings.Join(req.Values, ", ")))
		}
		b.kv(4, 15, fmt.Sprintf("Term %d", i), strings.Join(parts, ", "))
	}
}

func describePVC(c Cluster, pvc *corev1.PersistentVolumeClaim) string {
	b := &block{}
	capacity, modes := "", ""
	if pvc.Spec.VolumeName != "" {
		size := pvc.Status.Capacity[corev1.ResourceStorage]
		capacity, modes = size.String(), accessModes(pvc.Status.AccessModes)
	}
	for _, kv := range [][2]string{
		{"Name", pvc.Name}, {"Namespace", pvc.Namespace}, {"StorageClass", volumeutil.ClassOf(pvc)},
		{"Status", deletingStatus(c, pvc, string(pvc.Status.Phase))}, {"Volume", pvc.Spec.VolumeName},
		{"Labels", orNoneMap(pvc.Labels)}, {"Annotations", orNoneMap(pvc.Annotations)}, {"Finalizers", fmt.Sprint(pvc.Finalizers)},
		{"Capacity", capacity}, {"Access Modes", modes}, {"VolumeMode", volumeModeOf(pvc.Spec.VolumeMode)}, {"Used By", usedBy(c, pvc)},
	} {
		b.kv(0, pvcKeyWidth, kv[0], kv[1])
	}
	return withEvents(b, c, pvc, pvcKeyWidth)
}

func usedBy(c Cluster, pvc *corev1.PersistentVolumeClaim) string {
	out := []string{}
	for _, p := range store.View[*corev1.Pod](c.Store(), store.KindPod, pvc.Namespace) {
		for _, name := range volumeutil.ClaimNames(p) {
			if name == pvc.Name {
				out = append(out, p.Name)
			}
		}
	}
	if len(out) == 0 {
		return "<none>"
	}
	sort.Strings(out)
	return strings.Join(out, "\n"+strings.Repeat(" ", pvcKeyWidth))
}

func describeSC(c Cluster, sc *storagev1.StorageClass) string {
	b := &block{}
	isDefault := "No"
	if volumeutil.IsDefaultClass(sc) {
		isDefault = "Yes"
	}
	row := scRow(c, sc, false)
	for _, kv := range [][2]string{
		{"Name", sc.Name}, {"IsDefaultClass", isDefault}, {"Annotations", orNoneMap(sc.Annotations)}, {"Provisioner", sc.Provisioner},
		{"Parameters", orNoneMap(sc.Parameters)}, {"AllowVolumeExpansion", "<unset>"}, {"MountOptions", "<none>"},
		{"ReclaimPolicy", row[2]}, {"VolumeBindingMode", row[3]},
	} {
		b.kv(0, scKeyWidth, kv[0], kv[1])
	}
	return withEvents(b, c, sc, scKeyWidth)
}

func describeSTS(c Cluster, s *appsv1.StatefulSet) string {
	b := &block{}
	running, waiting := statefulPodCounts(c, s)
	desired := int32(1)
	if s.Spec.Replicas != nil {
		desired = *s.Spec.Replicas
	}
	for _, kv := range [][2]string{
		{"Name", s.Name}, {"Namespace", s.Namespace}, {"Selector", selectorString(s.Spec.Selector)}, {"Labels", orNoneMap(s.Labels)},
		{"Replicas", fmt.Sprintf("%d desired | %d total", desired, s.Status.Replicas)}, {"Update Strategy", string(s.Spec.UpdateStrategy.Type)},
		{"Pods Status", fmt.Sprintf("%d Running / %d Waiting / 0 Succeeded / 0 Failed", running, waiting)},
	} {
		b.kv(0, stsKeyWidth, kv[0], kv[1])
	}
	b.raw("Volume Claims:")
	for _, t := range s.Spec.VolumeClaimTemplates {
		size := t.Spec.Resources.Requests[corev1.ResourceStorage]
		b.kv(2, 15, "Name", t.Name)
		b.kv(2, 15, "StorageClass", volumeutil.ClassOf(&t))
		b.kv(2, 15, "Capacity", size.String())
		b.kv(2, 15, "Access Modes", fmt.Sprint(t.Spec.AccessModes))
	}
	return withEvents(b, c, s, stsKeyWidth)
}

func statefulPodCounts(c Cluster, s *appsv1.StatefulSet) (int, int) {
	running, waiting := 0, 0
	for _, p := range store.View[*corev1.Pod](c.Store(), store.KindPod, s.Namespace) {
		ref := p.OwnerReferences
		if len(ref) == 0 || ref[0].UID != s.UID {
			continue
		}
		switch p.Status.Phase {
		case corev1.PodRunning:
			running++
		case corev1.PodPending:
			waiting++
		}
	}
	return running, waiting
}

func deletingStatus(c Cluster, o store.Object, phase string) string {
	if ts := o.GetDeletionTimestamp(); ts != nil {
		return "Terminating (lasts " + age(c, *ts) + ")"
	}
	return phase
}
