package volumeutil

import (
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

const (
	AnnSelectedNode           = "volume.kubernetes.io/selected-node"
	AnnBindCompleted          = "pv.kubernetes.io/bind-completed"
	AnnBoundByController      = "pv.kubernetes.io/bound-by-controller"
	AnnDynamicallyProvisioned = "pv.kubernetes.io/provisioned-by"
	VaultDriver               = "vaults.kubernites.io"
	PVCProtectionFinalizer    = "kubernetes.io/pvc-protection"
)

func ZoneAffinity(zone string) *corev1.VolumeNodeAffinity {
	return &corev1.VolumeNodeAffinity{Required: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{
		MatchExpressions: []corev1.NodeSelectorRequirement{{Key: corev1.LabelTopologyZone, Operator: corev1.NodeSelectorOpIn, Values: []string{zone}}},
	}}}}
}

func ZoneOf(pv *corev1.PersistentVolume) string {
	if pv.Spec.NodeAffinity == nil || pv.Spec.NodeAffinity.Required == nil {
		return ""
	}
	for _, term := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
		for _, req := range term.MatchExpressions {
			if req.Key == corev1.LabelTopologyZone && req.Operator == corev1.NodeSelectorOpIn && len(req.Values) > 0 {
				return req.Values[0]
			}
		}
	}
	return ""
}

func NodeZone(node *corev1.Node) string {
	return node.Labels[corev1.LabelTopologyZone]
}

func FitsNode(pv *corev1.PersistentVolume, node *corev1.Node) bool {
	if pv.Spec.NodeAffinity == nil || pv.Spec.NodeAffinity.Required == nil {
		return true
	}
	for _, term := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
		if termMatches(term, node) {
			return true
		}
	}
	return false
}

func termMatches(term corev1.NodeSelectorTerm, node *corev1.Node) bool {
	if len(term.MatchExpressions) == 0 && len(term.MatchFields) == 0 {
		return false
	}
	if !requirementsMatch(term.MatchExpressions, labels.Set(node.Labels)) {
		return false
	}
	return requirementsMatch(term.MatchFields, labels.Set{"metadata.name": node.Name})
}

func requirementsMatch(reqs []corev1.NodeSelectorRequirement, set labels.Set) bool {
	ops := map[corev1.NodeSelectorOperator]selection.Operator{
		corev1.NodeSelectorOpIn: selection.In, corev1.NodeSelectorOpNotIn: selection.NotIn,
		corev1.NodeSelectorOpExists: selection.Exists, corev1.NodeSelectorOpDoesNotExist: selection.DoesNotExist,
		corev1.NodeSelectorOpGt: selection.GreaterThan, corev1.NodeSelectorOpLt: selection.LessThan,
	}
	for _, req := range reqs {
		op, ok := ops[req.Operator]
		if !ok {
			return false
		}
		r, err := labels.NewRequirement(req.Key, op, req.Values)
		if err != nil || !r.Matches(set) {
			return false
		}
	}
	return true
}

func ClaimNames(pod *corev1.Pod) []string {
	out := []string{}
	for _, v := range pod.Spec.Volumes {
		if v.PersistentVolumeClaim != nil {
			out = append(out, v.PersistentVolumeClaim.ClaimName)
		}
	}
	return out
}

func ClassOf(pvc *corev1.PersistentVolumeClaim) string {
	if pvc.Spec.StorageClassName != nil {
		return *pvc.Spec.StorageClassName
	}
	return pvc.Annotations[corev1.BetaStorageClassAnnotation]
}

func DelaysBinding(sc *storagev1.StorageClass) bool {
	return sc != nil && sc.VolumeBindingMode != nil && *sc.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer
}

func HasMode(modes []corev1.PersistentVolumeAccessMode, want corev1.PersistentVolumeAccessMode) bool {
	for _, m := range modes {
		if m == want {
			return true
		}
	}
	return false
}

func ModesCover(have, want []corev1.PersistentVolumeAccessMode) bool {
	for _, m := range want {
		if !HasMode(have, m) {
			return false
		}
	}
	return true
}

const (
	AnnDefaultClass     = "storageclass.kubernetes.io/is-default-class"
	annBetaDefaultClass = "storageclass.beta.kubernetes.io/is-default-class"
)

func IsDefaultClass(sc *storagev1.StorageClass) bool {
	return sc.Annotations[AnnDefaultClass] == "true" || sc.Annotations[annBetaDefaultClass] == "true"
}

func DefaultClass(classes []*storagev1.StorageClass) *storagev1.StorageClass {
	var best *storagev1.StorageClass
	for _, sc := range classes {
		if !IsDefaultClass(sc) {
			continue
		}
		if best == nil || best.CreationTimestamp.Before(&sc.CreationTimestamp) || (best.CreationTimestamp.Equal(&sc.CreationTimestamp) && sc.Name < best.Name) {
			best = sc
		}
	}
	return best
}

func HasClass(pvc *corev1.PersistentVolumeClaim) bool {
	_, beta := pvc.Annotations[corev1.BetaStorageClassAnnotation]
	return pvc.Spec.StorageClassName != nil || beta
}
