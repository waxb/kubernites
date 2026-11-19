package model

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func sidecar(name, image string, always bool) corev1.Container {
	c := corev1.Container{Name: name, Image: image}
	if always {
		policy := corev1.ContainerRestartPolicyAlways
		c.RestartPolicy = &policy
	}
	return c
}

func TestFamiliarsComeFromAlwaysRestartingInitContainers(t *testing.T) {
	p := pod("p", "n1", "porter:1.0")
	p.Spec.InitContainers = []corev1.Container{
		sidecar("raven", "raven:1.0", true),
		sidecar("setup", "busybox:1.36", false),
		sidecar("hound", "academy.kubernites.dev/hound:1.0", true),
		sidecar("digest", "registry:5000/scout@sha256:abc", true),
	}
	got := ViewOf(p).Familiars
	want := []string{FamiliarRaven, FamiliarHound, "scout"}
	if len(got) != len(want) {
		t.Fatalf("familiars = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("familiars = %v, want %v", got, want)
		}
	}
	if len(ViewOf(pod("plain", "n1", "porter:1.0")).Familiars) != 0 {
		t.Error("a pod without sidecars has no familiars")
	}
}

func TestFamiliarKind(t *testing.T) {
	cases := map[string]string{"raven:1.0": "raven", "hound": "hound", "a/b/hound:2.0": "hound", "": ""}
	for image, want := range cases {
		if got := FamiliarKind(image); got != want {
			t.Errorf("FamiliarKind(%q) = %q, want %q", image, got, want)
		}
	}
}

func TestLastExitAndOOM(t *testing.T) {
	p := waiting(pod("p", "n1", "porter:1.0"), "CrashLoopBackOff")
	p.Status.ContainerStatuses[0].LastTerminationState = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137}}
	v := ViewOf(p)
	if v.LastExit != "OOMKilled" || !v.OOM() {
		t.Fatalf("last exit = %q oom = %v", v.LastExit, v.OOM())
	}
	crashed := ViewOf(waiting(pod("q", "n1", "porter:1.0"), "CrashLoopBackOff"))
	if crashed.LastExit != "" || crashed.OOM() {
		t.Fatalf("plain crash loop is not OOM: %+v", crashed)
	}
	if !(PodView{Reason: "OOMKilled"}).OOM() {
		t.Error("a pod terminated by the OOM killer is OOM")
	}
}
