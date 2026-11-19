package store

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	KindPod           = "Pod"
	KindNode          = "Node"
	KindService       = "Service"
	KindEvent         = "Event"
	KindNamespace     = "Namespace"
	KindConfigMap     = "ConfigMap"
	KindReplicaSet    = "ReplicaSet"
	KindDeployment    = "Deployment"
	KindEndpointSlice = "EndpointSlice"

	KindHorizontalPodAutoscaler = "HorizontalPodAutoscaler"
	KindPriorityClass           = "PriorityClass"
	KindLimitRange              = "LimitRange"
	KindDaemonSet               = "DaemonSet"
	KindControllerRevision      = "ControllerRevision"
	KindServiceAccount          = "ServiceAccount"
)

var gvks = map[string]schema.GroupVersionKind{
	KindPod:           corev1.SchemeGroupVersion.WithKind(KindPod),
	KindNode:          corev1.SchemeGroupVersion.WithKind(KindNode),
	KindService:       corev1.SchemeGroupVersion.WithKind(KindService),
	KindEvent:         corev1.SchemeGroupVersion.WithKind(KindEvent),
	KindNamespace:     corev1.SchemeGroupVersion.WithKind(KindNamespace),
	KindConfigMap:     corev1.SchemeGroupVersion.WithKind(KindConfigMap),
	KindReplicaSet:    appsv1.SchemeGroupVersion.WithKind(KindReplicaSet),
	KindDeployment:    appsv1.SchemeGroupVersion.WithKind(KindDeployment),
	KindEndpointSlice: discoveryv1.SchemeGroupVersion.WithKind(KindEndpointSlice),

	KindHorizontalPodAutoscaler: autoscalingv2.SchemeGroupVersion.WithKind(KindHorizontalPodAutoscaler),
	KindPriorityClass:           schedulingv1.SchemeGroupVersion.WithKind(KindPriorityClass),
	KindLimitRange:              corev1.SchemeGroupVersion.WithKind(KindLimitRange),
	KindDaemonSet:               appsv1.SchemeGroupVersion.WithKind(KindDaemonSet),
	KindControllerRevision:      appsv1.SchemeGroupVersion.WithKind(KindControllerRevision),
	KindServiceAccount:          corev1.SchemeGroupVersion.WithKind(KindServiceAccount),
}

var clusterScoped = map[string]bool{KindNode: true, KindNamespace: true, KindPriorityClass: true}

var untrackedGeneration = map[string]bool{KindNode: true, KindService: true, KindEvent: true, KindNamespace: true, KindConfigMap: true, KindLimitRange: true, KindControllerRevision: true, KindServiceAccount: true}

func setGeneration(obj Object, kind string, generation int64) {
	if untrackedGeneration[kind] {
		generation = 0
	}
	obj.SetGeneration(generation)
}

func KindOf(obj runtime.Object) (string, error) {
	switch obj.(type) {
	case *corev1.Pod:
		return KindPod, nil
	case *corev1.Node:
		return KindNode, nil
	case *corev1.Service:
		return KindService, nil
	case *corev1.Event:
		return KindEvent, nil
	case *corev1.Namespace:
		return KindNamespace, nil
	case *corev1.ConfigMap:
		return KindConfigMap, nil
	case *appsv1.ReplicaSet:
		return KindReplicaSet, nil
	case *appsv1.Deployment:
		return KindDeployment, nil
	case *discoveryv1.EndpointSlice:
		return KindEndpointSlice, nil
	case *autoscalingv2.HorizontalPodAutoscaler:
		return KindHorizontalPodAutoscaler, nil
	case *schedulingv1.PriorityClass:
		return KindPriorityClass, nil
	case *corev1.LimitRange:
		return KindLimitRange, nil
	case *appsv1.DaemonSet:
		return KindDaemonSet, nil
	case *appsv1.ControllerRevision:
		return KindControllerRevision, nil
	case *corev1.ServiceAccount:
		return KindServiceAccount, nil
	}
	if kind, ok := storageKindOf(obj); ok {
		return kind, nil
	}
	return "", fmt.Errorf("unregistered type %T", obj)
}

func GVKFor(kind string) (schema.GroupVersionKind, bool) {
	gvk, ok := gvks[kind]
	return gvk, ok
}

func IsClusterScoped(kind string) bool {
	return clusterScoped[kind]
}

func NewObject(kind string) (Object, error) {
	switch kind {
	case KindPod:
		return &corev1.Pod{}, nil
	case KindNode:
		return &corev1.Node{}, nil
	case KindService:
		return &corev1.Service{}, nil
	case KindEvent:
		return &corev1.Event{}, nil
	case KindNamespace:
		return &corev1.Namespace{}, nil
	case KindConfigMap:
		return &corev1.ConfigMap{}, nil
	case KindReplicaSet:
		return &appsv1.ReplicaSet{}, nil
	case KindDeployment:
		return &appsv1.Deployment{}, nil
	case KindEndpointSlice:
		return &discoveryv1.EndpointSlice{}, nil
	case KindHorizontalPodAutoscaler:
		return &autoscalingv2.HorizontalPodAutoscaler{}, nil
	case KindPriorityClass:
		return &schedulingv1.PriorityClass{}, nil
	case KindLimitRange:
		return &corev1.LimitRange{}, nil
	case KindDaemonSet:
		return &appsv1.DaemonSet{}, nil
	case KindControllerRevision:
		return &appsv1.ControllerRevision{}, nil
	case KindServiceAccount:
		return &corev1.ServiceAccount{}, nil
	}
	if obj, ok := newStorageObject(kind); ok {
		return obj, nil
	}
	return nil, fmt.Errorf("unknown kind %q", kind)
}

func specChanged(before, after runtime.Object) bool {
	switch b := before.(type) {
	case *corev1.Pod:
		return !equality.Semantic.DeepEqual(b.Spec, after.(*corev1.Pod).Spec)
	case *corev1.Node:
		return !equality.Semantic.DeepEqual(b.Spec, after.(*corev1.Node).Spec)
	case *corev1.Service:
		return !equality.Semantic.DeepEqual(b.Spec, after.(*corev1.Service).Spec)
	case *appsv1.ReplicaSet:
		return !equality.Semantic.DeepEqual(b.Spec, after.(*appsv1.ReplicaSet).Spec)
	case *appsv1.Deployment:
		next := after.(*appsv1.Deployment)
		return !equality.Semantic.DeepEqual(b.Spec, next.Spec) || !equality.Semantic.DeepEqual(b.Annotations, next.Annotations)
	case *autoscalingv2.HorizontalPodAutoscaler:
		return !equality.Semantic.DeepEqual(b.Spec, after.(*autoscalingv2.HorizontalPodAutoscaler).Spec)
	case *corev1.LimitRange:
		return !equality.Semantic.DeepEqual(b.Spec, after.(*corev1.LimitRange).Spec)
	case *appsv1.DaemonSet:
		return !equality.Semantic.DeepEqual(b.Spec, after.(*appsv1.DaemonSet).Spec)
	}
	return storageSpecChanged(before, after)
}
