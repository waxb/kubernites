package podutil

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func floor(req, lim string) corev1.ResourceRequirements {
	out := corev1.ResourceRequirements{Requests: corev1.ResourceList{}, Limits: corev1.ResourceList{}}
	if req != "" {
		out.Requests[corev1.ResourceEphemeralStorage] = resource.MustParse(req)
	}
	if lim != "" {
		out.Limits[corev1.ResourceEphemeralStorage] = resource.MustParse(lim)
	}
	return out
}

func sidecarPod(app, raven corev1.ResourceRequirements) *corev1.Pod {
	always := corev1.ContainerRestartPolicyAlways
	p := podWith(nil, nil)
	p.Spec.Containers[0].Resources = app
	p.Spec.InitContainers = []corev1.Container{{Name: "raven", Image: "raven:1.0", RestartPolicy: &always, Resources: raven}}
	return p
}

func TestEphemeralOnlyPodsAreBestEffort(t *testing.T) {
	p := podWith(nil, nil)
	p.Spec.Containers[0].Resources = floor("256Mi", "1Gi")
	if got := QoSClass(p); got != corev1.PodQOSBestEffort {
		t.Fatalf("QoS of an ephemeral-storage-only pod = %s, want BestEffort (QoS reads only cpu and memory)", got)
	}
	guaranteed := podWith(rl("100m", "64Mi"), rl("100m", "64Mi"))
	guaranteed.Spec.Containers[0].Resources.Requests[corev1.ResourceEphemeralStorage] = resource.MustParse("1Gi")
	if got := QoSClass(guaranteed); got != corev1.PodQOSGuaranteed {
		t.Fatalf("QoS = %s, want Guaranteed", got)
	}
}

func TestEphemeralRequestsSumContainersAndSidecars(t *testing.T) {
	p := sidecarPod(floor("256Mi", "1Gi"), floor("64Mi", ""))
	if got := EphemeralRequest(p); got != (256+64)*1024*1024 {
		t.Fatalf("request = %d", got)
	}
	if got := ContainerEphemeralRequest(p.Spec.Containers[0]); got != 256*1024*1024 {
		t.Fatalf("container request = %d", got)
	}
	limit, ok := EphemeralLimit(p)
	if !ok || limit != 1024*1024*1024 {
		t.Fatalf("pod limit = %d %v", limit, ok)
	}
}

func TestEphemeralRequestDefaultsToLimit(t *testing.T) {
	p := podWith(nil, nil)
	p.Spec.Containers[0].Resources = floor("", "512Mi")
	if got := EphemeralRequest(p); got != 512*1024*1024 {
		t.Fatalf("request = %d, want the limit", got)
	}
}

func TestEphemeralLimitAbsentWithoutAnyLimit(t *testing.T) {
	if _, ok := EphemeralLimit(podWith(nil, nil)); ok {
		t.Fatal("no container binds the floor")
	}
	zero := podWith(nil, nil)
	zero.Spec.Containers[0].Resources = floor("", "0")
	if _, ok := EphemeralLimit(zero); ok {
		t.Fatal("a zero limit binds nothing")
	}
	if got := EphemeralRequest(podWith(nil, nil)); got != 0 {
		t.Fatalf("request = %d", got)
	}
}
