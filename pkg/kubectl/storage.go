package kubectl

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/sim/volumeutil"
)

func init() {
	resourceTypes = append(resourceTypes,
		ResourceType{Kind: store.KindPersistentVolume, Plural: "persistentvolumes", Singular: "persistentvolume", short: "pv", apiVersion: "v1"},
		ResourceType{Kind: store.KindPersistentVolumeClaim, Plural: "persistentvolumeclaims", Singular: "persistentvolumeclaim", short: "pvc", apiVersion: "v1", Namespaced: true},
		ResourceType{Kind: store.KindStorageClass, Plural: "storageclasses", Singular: "storageclass", short: "sc", apiVersion: "storage.k8s.io/v1", group: "storage.k8s.io"},
		ResourceType{Kind: store.KindStatefulSet, Plural: "statefulsets", Singular: "statefulset", short: "sts", apiVersion: "apps/v1", group: "apps", Namespaced: true},
	)
}

func storagePrinter(kind string) (printer, bool) {
	switch kind {
	case store.KindPersistentVolume:
		return printer{pvHeaders, pvRow}, true
	case store.KindPersistentVolumeClaim:
		return printer{pvcHeaders, pvcRow}, true
	case store.KindStorageClass:
		return printer{func(bool) []string {
			return []string{"NAME", "PROVISIONER", "RECLAIMPOLICY", "VOLUMEBINDINGMODE", "ALLOWVOLUMEEXPANSION", "AGE"}
		}, scRow}, true
	case store.KindStatefulSet:
		return printer{stsHeaders, stsRow}, true
	}
	return printer{}, false
}

func accessModes(modes []corev1.PersistentVolumeAccessMode) string {
	short := map[corev1.PersistentVolumeAccessMode]string{corev1.ReadWriteOnce: "RWO", corev1.ReadOnlyMany: "ROX", corev1.ReadWriteMany: "RWX", corev1.ReadWriteOncePod: "RWOP"}
	out := []string{}
	for _, m := range []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce, corev1.ReadOnlyMany, corev1.ReadWriteMany, corev1.ReadWriteOncePod} {
		if volumeutil.HasMode(modes, m) {
			out = append(out, short[m])
		}
	}
	return strings.Join(out, ",")
}

func volumeModeOf(mode *corev1.PersistentVolumeMode) string {
	if mode == nil {
		return "<unset>"
	}
	return string(*mode)
}

func pvHeaders(wide bool) []string {
	h := []string{"NAME", "CAPACITY", "ACCESS MODES", "RECLAIM POLICY", "STATUS", "CLAIM", "STORAGECLASS", "VOLUMEATTRIBUTESCLASS", "REASON", "AGE"}
	if wide {
		h = append(h, "VOLUMEMODE")
	}
	return h
}

func pvRow(c Cluster, o store.Object, wide bool) []string {
	pv := o.(*corev1.PersistentVolume)
	claim, phase := "", string(pv.Status.Phase)
	if ref := pv.Spec.ClaimRef; ref != nil {
		claim = ref.Namespace + "/" + ref.Name
	}
	if pv.DeletionTimestamp != nil {
		phase = "Terminating"
	}
	size := pv.Spec.Capacity[corev1.ResourceStorage]
	row := []string{pv.Name, size.String(), accessModes(pv.Spec.AccessModes), string(pv.Spec.PersistentVolumeReclaimPolicy), phase, claim, pv.Spec.StorageClassName, "<unset>", pv.Status.Reason, age(c, pv.CreationTimestamp)}
	if wide {
		row = append(row, volumeModeOf(pv.Spec.VolumeMode))
	}
	return row
}

func pvcHeaders(wide bool) []string {
	h := []string{"NAME", "STATUS", "VOLUME", "CAPACITY", "ACCESS MODES", "STORAGECLASS", "VOLUMEATTRIBUTESCLASS", "AGE"}
	if wide {
		h = append(h, "VOLUMEMODE")
	}
	return h
}

func pvcRow(c Cluster, o store.Object, wide bool) []string {
	pvc := o.(*corev1.PersistentVolumeClaim)
	phase, capacity, modes := string(pvc.Status.Phase), "", ""
	if pvc.DeletionTimestamp != nil {
		phase = "Terminating"
	}
	if pvc.Spec.VolumeName != "" {
		size := pvc.Status.Capacity[corev1.ResourceStorage]
		capacity, modes = size.String(), accessModes(pvc.Status.AccessModes)
	}
	row := []string{pvc.Name, phase, pvc.Spec.VolumeName, capacity, modes, volumeutil.ClassOf(pvc), "<unset>", age(c, pvc.CreationTimestamp)}
	if wide {
		row = append(row, volumeModeOf(pvc.Spec.VolumeMode))
	}
	return row
}

func scRow(c Cluster, o store.Object, _ bool) []string {
	sc := o.(*storagev1.StorageClass)
	name := sc.Name
	if def := volumeutil.DefaultClass(store.View[*storagev1.StorageClass](c.Store(), store.KindStorageClass, "")); def != nil && def.Name == sc.Name {
		name += " (default)"
	}
	reclaim, mode, expand := string(corev1.PersistentVolumeReclaimDelete), string(storagev1.VolumeBindingImmediate), false
	if sc.ReclaimPolicy != nil {
		reclaim = string(*sc.ReclaimPolicy)
	}
	if sc.VolumeBindingMode != nil {
		mode = string(*sc.VolumeBindingMode)
	}
	if sc.AllowVolumeExpansion != nil {
		expand = *sc.AllowVolumeExpansion
	}
	return []string{name, sc.Provisioner, reclaim, mode, fmt.Sprint(expand), age(c, sc.CreationTimestamp)}
}

func stsHeaders(wide bool) []string {
	if wide {
		return []string{"NAME", "READY", "AGE", "CONTAINERS", "IMAGES"}
	}
	return []string{"NAME", "READY", "AGE"}
}

func stsRow(c Cluster, o store.Object, wide bool) []string {
	s := o.(*appsv1.StatefulSet)
	desired := int32(1)
	if s.Spec.Replicas != nil {
		desired = *s.Spec.Replicas
	}
	row := []string{s.Name, fmt.Sprintf("%d/%d", s.Status.ReadyReplicas, desired), age(c, s.CreationTimestamp)}
	if !wide {
		return row
	}
	names, images := []string{}, []string{}
	for _, ct := range s.Spec.Template.Spec.Containers {
		names, images = append(names, ct.Name), append(images, ct.Image)
	}
	return append(row, strings.Join(names, ","), strings.Join(images, ","))
}
