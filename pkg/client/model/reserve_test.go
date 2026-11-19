package model

import (
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func requesting(p *corev1.Pod, ns, cpu, mem string) *corev1.Pod {
	next := p.DeepCopy()
	next.Namespace = ns
	next.Spec.Containers[0].Resources.Requests = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu), corev1.ResourceMemory: resource.MustParse(mem)}
	return next
}

func TestNodeViewsCountTheSystemShare(t *testing.T) {
	done := requesting(pod("job", "tower", "x"), "kube-system", "1", "1Gi")
	done.Status.Phase = corev1.PodSucceeded
	c := NewCache()
	c.Reset([]store.Object{
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "tower"}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "keep"}},
		requesting(pod("etcd-tower", "tower", "etcd"), "kube-system", "100m", "100Mi"),
		requesting(pod("kube-apiserver-tower", "tower", "api"), "kube-system", "250m", "0"),
		requesting(pod("coredns-a", "keep", "dns"), "kube-system", "100m", "70Mi"),
		requesting(pod("porter-a", "tower", "porter:1.0"), "default", "100m", "64Mi"),
		requesting(pod("pending", "", "x"), "kube-system", "500m", "1Gi"),
		done,
	})
	views := c.NodeViews()
	tower, keep := views["tower"], views["keep"]
	if tower.SystemCPUMilli != 350 || tower.SystemMemBytes != 100<<20 {
		t.Errorf("tower share = %dm %d", tower.SystemCPUMilli, tower.SystemMemBytes)
	}
	if keep.SystemCPUMilli != 100 || keep.SystemMemBytes != 70<<20 {
		t.Errorf("keep share = %dm %d", keep.SystemCPUMilli, keep.SystemMemBytes)
	}
}

func TestSpacedNamesGroupByNamespace(t *testing.T) {
	c := NewCache()
	c.Reset([]store.Object{
		requesting(pod("etcd-tower", "tower", "etcd"), "kube-system", "100m", "0"),
		pod("porter-a", "tower", "porter:1.0"),
		pod("porter-b", "tower", "porter:1.0"),
	})
	got := c.SpacedNames(store.KindPod)
	if !slices.Equal(got["kube-system"], []string{"etcd-tower"}) || !slices.Equal(got["default"], []string{"porter-a", "porter-b"}) {
		t.Errorf("spaced = %v", got)
	}
	if len(c.SpacedNames(store.KindService)) != 0 {
		t.Error("no services, no spaces")
	}
}
