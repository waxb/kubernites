package model

import (
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestAPodViewNamesTheDeedsItMounts(t *testing.T) {
	p := pod("archive-0", "keep-1", "archivist:1.0")
	p.Spec.Volumes = []corev1.Volume{
		{Name: "scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "books", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "books-archive-0"}}},
	}
	if got := ViewOf(p).Claims; !slices.Equal(got, []string{"books-archive-0"}) {
		t.Fatalf("claims = %v", got)
	}
	if ViewOf(pod("plain", "", "porter:1.0")).Claims != nil {
		t.Fatal("a pod with no claim names none")
	}
}
