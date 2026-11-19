package store

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/waxb/kubernites/pkg/sim/volumeutil"
)

const (
	KindPersistentVolume      = "PersistentVolume"
	KindPersistentVolumeClaim = "PersistentVolumeClaim"
	KindStorageClass          = "StorageClass"
	KindStatefulSet           = "StatefulSet"
)

func init() {
	gvks[KindPersistentVolume] = corev1.SchemeGroupVersion.WithKind(KindPersistentVolume)
	gvks[KindPersistentVolumeClaim] = corev1.SchemeGroupVersion.WithKind(KindPersistentVolumeClaim)
	gvks[KindStorageClass] = storagev1.SchemeGroupVersion.WithKind(KindStorageClass)
	gvks[KindStatefulSet] = appsv1.SchemeGroupVersion.WithKind(KindStatefulSet)
	clusterScoped[KindPersistentVolume] = true
	clusterScoped[KindStorageClass] = true
	untrackedGeneration[KindPersistentVolume] = true
	untrackedGeneration[KindPersistentVolumeClaim] = true
	untrackedGeneration[KindStorageClass] = true
}

func storageKindOf(obj runtime.Object) (string, bool) {
	switch obj.(type) {
	case *corev1.PersistentVolume:
		return KindPersistentVolume, true
	case *corev1.PersistentVolumeClaim:
		return KindPersistentVolumeClaim, true
	case *storagev1.StorageClass:
		return KindStorageClass, true
	case *appsv1.StatefulSet:
		return KindStatefulSet, true
	}
	return "", false
}

func newStorageObject(kind string) (Object, bool) {
	switch kind {
	case KindPersistentVolume:
		return &corev1.PersistentVolume{}, true
	case KindPersistentVolumeClaim:
		return &corev1.PersistentVolumeClaim{}, true
	case KindStorageClass:
		return &storagev1.StorageClass{}, true
	case KindStatefulSet:
		return &appsv1.StatefulSet{}, true
	}
	return nil, false
}

func storageSpecChanged(before, after runtime.Object) bool {
	switch b := before.(type) {
	case *corev1.PersistentVolume:
		return !equality.Semantic.DeepEqual(b.Spec, after.(*corev1.PersistentVolume).Spec)
	case *corev1.PersistentVolumeClaim:
		return !equality.Semantic.DeepEqual(b.Spec, after.(*corev1.PersistentVolumeClaim).Spec)
	case *appsv1.StatefulSet:
		return !equality.Semantic.DeepEqual(b.Spec, after.(*appsv1.StatefulSet).Spec)
	}
	return false
}

func (s *Store) admitClaim(pvc *corev1.PersistentVolumeClaim, old Object) {
	if old != nil || volumeutil.HasClass(pvc) {
		return
	}
	classes := []*storagev1.StorageClass{}
	for _, key := range s.keysOf(KindStorageClass) {
		classes = append(classes, s.objects[key].(*storagev1.StorageClass))
	}
	if def := volumeutil.DefaultClass(classes); def != nil {
		name := def.Name
		pvc.Spec.StorageClassName = &name
	}
}
