package store

import (
	"maps"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func admitDaemonSet(ds *appsv1.DaemonSet, old Object) {
	generation := int64(1)
	if prev, ok := old.(*appsv1.DaemonSet); ok {
		generation = templateGeneration(prev)
		if !equality.Semantic.DeepEqual(prev.Spec.Template, ds.Spec.Template) {
			generation++
		}
	}
	annotations := maps.Clone(ds.Annotations)
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[appsv1.DeprecatedTemplateGeneration] = strconv.FormatInt(generation, 10)
	ds.Annotations = annotations
}

func templateGeneration(ds *appsv1.DaemonSet) int64 {
	generation, err := strconv.ParseInt(ds.Annotations[appsv1.DeprecatedTemplateGeneration], 10, 64)
	if err != nil || generation < 1 {
		return 1
	}
	return generation
}
