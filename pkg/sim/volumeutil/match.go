package volumeutil

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func BoundToClaim(pv *corev1.PersistentVolume, pvc *corev1.PersistentVolumeClaim) bool {
	ref := pv.Spec.ClaimRef
	return ref != nil && ref.Namespace == pvc.Namespace && ref.Name == pvc.Name && (ref.UID == "" || ref.UID == pvc.UID)
}

func MatchesClaim(pv *corev1.PersistentVolume, pvc *corev1.PersistentVolumeClaim) bool {
	want := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	have := pv.Spec.Capacity[corev1.ResourceStorage]
	if have.Cmp(want) < 0 || pv.DeletionTimestamp != nil || pv.Spec.StorageClassName != ClassOf(pvc) {
		return false
	}
	if !ModesCover(pv.Spec.AccessModes, pvc.Spec.AccessModes) {
		return false
	}
	if pvc.Spec.Selector == nil {
		return true
	}
	sel, err := metav1.LabelSelectorAsSelector(pvc.Spec.Selector)
	return err == nil && sel.Matches(labels.Set(pv.Labels))
}

func FindMatchingVolume(pvc *corev1.PersistentVolumeClaim, volumes []*corev1.PersistentVolume, node *corev1.Node, delay bool) *corev1.PersistentVolume {
	var best *corev1.PersistentVolume
	for _, pv := range volumes {
		fits := node == nil || FitsNode(pv, node)
		if BoundToClaim(pv, pvc) {
			if !fits {
				return nil
			}
			return pv
		}
		if pv.Spec.ClaimRef != nil || (node == nil && delay) || pv.Status.Phase != corev1.VolumeAvailable || !fits || !MatchesClaim(pv, pvc) {
			continue
		}
		if best == nil || smaller(pv, best) {
			best = pv
		}
	}
	return best
}

func smaller(a, b *corev1.PersistentVolume) bool {
	qa, qb := a.Spec.Capacity[corev1.ResourceStorage], b.Spec.Capacity[corev1.ResourceStorage]
	return qa.Cmp(qb) < 0
}
