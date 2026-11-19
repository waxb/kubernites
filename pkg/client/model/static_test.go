package model

import (
	"testing"
)

func TestStaticPodsAreSeenOutsideKubeSystem(t *testing.T) {
	door := pod("porter-lone-tower", "lone-tower", "porter:1.0")
	door.Annotations = map[string]string{StaticSourceKey: StaticSourceFile}
	etcd := pod("etcd-lone-tower", "lone-tower", "etcd:3.5")
	etcd.Namespace = "kube-system"
	etcd.Annotations = map[string]string{StaticSourceKey: StaticSourceFile}
	api := pod("porter-api", "lone-tower", "porter:1.0")
	api.Annotations = map[string]string{StaticSourceKey: "api"}
	cases := []struct {
		name string
		view PodView
		want bool
	}{
		{"door hearth", ViewOf(door), true},
		{"control plane", ViewOf(etcd), false},
		{"declared hearth", ViewOf(api), false},
		{"bare pod", ViewOf(pod("x", "n1", "porter:1.0")), false},
	}
	for _, c := range cases {
		if c.view.Static != c.want {
			t.Errorf("%s: static = %v, want %v", c.name, c.view.Static, c.want)
		}
	}
}
