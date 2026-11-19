package podutil

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func annotated(annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default", Annotations: annotations}}
}

func TestStaticAndMirrorPodsAreToldApart(t *testing.T) {
	cases := []struct {
		name           string
		annotations    map[string]string
		static, mirror bool
	}{
		{"api pod", nil, false, false},
		{"explicit api source", map[string]string{ConfigSourceKey: "api"}, false, false},
		{"static pod on the kubelet", map[string]string{ConfigSourceKey: FileSource, ConfigHashKey: "abc"}, true, false},
		{"mirror pod in the API", map[string]string{ConfigSourceKey: FileSource, ConfigHashKey: "abc", corev1.MirrorPodAnnotationKey: "abc"}, true, true},
	}
	for _, c := range cases {
		p := annotated(c.annotations)
		if IsStaticPod(p) != c.static || IsMirrorPod(p) != c.mirror {
			t.Errorf("%s: static=%v mirror=%v, want %v %v", c.name, IsStaticPod(p), IsMirrorPod(p), c.static, c.mirror)
		}
	}
}

func withPorts(ports ...corev1.ContainerPort) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "lantern", Image: "lantern:1.0", Ports: ports}}},
	}
}

func TestHostPortsListOnlyPortsBoundOnTheNode(t *testing.T) {
	p := withPorts(
		corev1.ContainerPort{Name: "http", ContainerPort: 8080, HostPort: 80},
		corev1.ContainerPort{Name: "metrics", ContainerPort: 9090},
		corev1.ContainerPort{Name: "dns", ContainerPort: 53, HostPort: 53, Protocol: corev1.ProtocolUDP, HostIP: "10.0.0.1"},
	)
	got := HostPorts(p)
	want := []HostPort{{IP: "0.0.0.0", Protocol: corev1.ProtocolTCP, Port: 80}, {IP: "10.0.0.1", Protocol: corev1.ProtocolUDP, Port: 53}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("host ports = %+v", got)
	}
	if !HoldsHostPort(p, 80) || HoldsHostPort(p, 8080) || HoldsHostPort(withPorts(), 80) {
		t.Fatal("HoldsHostPort confused container and host ports")
	}
}

func TestHostPortConflictsFollowNodePorts(t *testing.T) {
	held := []*corev1.Pod{withPorts(corev1.ContainerPort{ContainerPort: 8080, HostPort: 80})}
	cases := []struct {
		name string
		want *corev1.Pod
		fits bool
	}{
		{"same port", withPorts(corev1.ContainerPort{ContainerPort: 80, HostPort: 80}), false},
		{"same port on a named address", withPorts(corev1.ContainerPort{ContainerPort: 80, HostPort: 80, HostIP: "172.18.0.2"}), false},
		{"other protocol", withPorts(corev1.ContainerPort{ContainerPort: 80, HostPort: 80, Protocol: corev1.ProtocolUDP}), true},
		{"other port", withPorts(corev1.ContainerPort{ContainerPort: 80, HostPort: 81}), true},
		{"no host port", withPorts(corev1.ContainerPort{ContainerPort: 80}), true},
	}
	for _, c := range cases {
		if got := FitsHostPorts(c.want, held); got != c.fits {
			t.Errorf("%s: fits = %v, want %v", c.name, got, c.fits)
		}
	}
	named := []*corev1.Pod{withPorts(corev1.ContainerPort{ContainerPort: 80, HostPort: 80, HostIP: "10.0.0.1"})}
	if !FitsHostPorts(withPorts(corev1.ContainerPort{ContainerPort: 80, HostPort: 80, HostIP: "10.0.0.2"}), named) {
		t.Fatal("distinct host addresses should not conflict")
	}
	if FitsHostPorts(withPorts(corev1.ContainerPort{ContainerPort: 80, HostPort: 80}), named) {
		t.Fatal("the wildcard address conflicts with every address")
	}
}

func named(namespace, name string, annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Annotations: annotations}}
}

func TestWorldSeesStaticPodsInsteadOfTheirMirrors(t *testing.T) {
	mirror := named("default", "lantern-lone-tower", map[string]string{ConfigSourceKey: FileSource, corev1.MirrorPodAnnotationKey: "h"})
	api := []*corev1.Pod{named("default", "b", nil), mirror, named("kube-system", "etcd-lone-tower", map[string]string{ConfigSourceKey: FileSource})}
	static := []*corev1.Pod{named("default", "lantern-lone-tower", map[string]string{ConfigSourceKey: FileSource}), named("other", "x-lone-tower", nil)}
	got := World(api, static, "default")
	if len(got) != 2 || got[0].Name != "b" || got[1] != static[0] {
		t.Fatalf("default world = %v", names(got))
	}
	all := World(api, static, "")
	if want := "b,lantern-lone-tower,etcd-lone-tower,x-lone-tower"; names(all) != want {
		t.Fatalf("world = %s, want %s", names(all), want)
	}
	if len(api) != 3 || api[1] != mirror {
		t.Fatal("World changed its input")
	}
}

func names(pods []*corev1.Pod) string {
	out := ""
	for i, p := range pods {
		if i > 0 {
			out += ","
		}
		out += p.Name
	}
	return out
}
