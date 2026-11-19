package soulstone

import "testing"

func TestLookupNormalizesRegistryDigestAndAliases(t *testing.T) {
	cases := map[string]string{
		"porter:1.0":                          "porter:1.0",
		RegistryHost + "/porter:1.1":          "porter:1.1",
		"porter:2.0@sha256:abcdef":            "porter:2.0",
		"porter":                              "porter:latest",
		"porter:latest":                       "porter:latest",
		"porter:2.1":                          "porter:2.1",
		" registry.k8s.io/kube-proxy:v1.37.0": "registry.k8s.io/kube-proxy:v1.37.0",
	}
	for in, want := range cases {
		p, ok := Lookup(in)
		if !ok || p.Image != want {
			t.Errorf("Lookup(%q) = %q, %v; want %q", in, p.Image, ok, want)
		}
	}
	if _, ok := Lookup("portr:1.0"); ok {
		t.Fatal("typo should not resolve")
	}
}

func TestParts(t *testing.T) {
	p, _ := Lookup("registry.k8s.io/kube-apiserver:v1.37.0")
	if p.Repository != "registry.k8s.io/kube-apiserver" || p.Tag != "v1.37.0" || !p.System {
		t.Fatalf("parts = %q %q %v", p.Repository, p.Tag, p.System)
	}
}

func TestIsLatestReference(t *testing.T) {
	cases := map[string]bool{
		"porter":                       true,
		"porter:latest":                true,
		"porter:1.0":                   false,
		"porter@sha256:aa":             false,
		"registry.k8s.io:5000/foo":     true,
		"registry.k8s.io:5000/foo:1.2": false,
	}
	for in, want := range cases {
		if got := IsLatestReference(in); got != want {
			t.Errorf("IsLatestReference(%q) = %v", in, got)
		}
	}
}

func TestPlayableExcludesSystemAndIsSortedByCost(t *testing.T) {
	list := Playable()
	if len(list) != 32 {
		t.Fatalf("len = %d", len(list))
	}
	for i := 1; i < len(list); i++ {
		if list[i].Cost < list[i-1].Cost || list[i].System {
			t.Fatalf("bad order or system entry: %+v", list[i])
		}
	}
	cursed := 0
	for _, p := range list {
		if p.Cursed {
			cursed++
		}
	}
	if cursed != 5 {
		t.Fatalf("crash-looping = %d", cursed)
	}
}

func TestEveryImageHasAProfession(t *testing.T) {
	for _, img := range []string{"porter:1.0", "porter:2.1", "clerk:1.0", "hauler:1.0", "courier:1.0", "sealwright:1.0", "registry.k8s.io/kube-apiserver:v1.37.0", "registry.k8s.io/etcd:3.6.0-0", "registry.k8s.io/kube-scheduler:v1.37.0", "registry.k8s.io/kube-controller-manager:v1.37.0", "registry.k8s.io/coredns/coredns:v1.12.0", "registry.k8s.io/kube-proxy:v1.37.0", "registry.k8s.io/metrics-server/metrics-server:v0.8.0"} {
		name, sprite := ProfessionOf(img)
		if name == "" || sprite == "" || sprite == "wanderer" {
			t.Errorf("%s has no profession", img)
		}
	}
	if _, sprite := ProfessionOf("portr:9"); sprite != "wanderer" {
		t.Fatal("unknown image should be a wanderer")
	}
}
