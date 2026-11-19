package soulstone

import "testing"

func TestMigrateRenamesRetiredTradesOnly(t *testing.T) {
	cases := map[string]string{
		"lantern:1.0":                          "porter:1.0",
		"lantern:1.1":                          "porter:1.1",
		"lantern:2.0":                          "porter:2.0",
		"lantern:2.1":                          "porter:2.1",
		"lantern":                              "porter",
		"lantern:latest":                       "porter:latest",
		"lantern@sha256:ab":                    "porter@sha256:ab",
		RegistryHost + "/lantern:1.1@sha256:c": RegistryHost + "/porter:1.1@sha256:c",
		"scribe:1.0":                           "clerk:1.0",
		RegistryHost + "/scribe:1.0":           RegistryHost + "/clerk:1.0",
		"porter:1.0":                           "porter:1.0",
		"raven:1.0":                            "raven:1.0",
		"lanternfish:1.0":                      "lanternfish:1.0",
		"lanturn:1.0":                          "lanturn:1.0",
		"registry.k8s.io/etcd:3.6.0-0":         "registry.k8s.io/etcd:3.6.0-0",
		"":                                     "",
	}
	for in, want := range cases {
		if got := Migrate(in); got != want {
			t.Errorf("Migrate(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMigratedImagesResolve(t *testing.T) {
	for _, old := range []string{"lantern:1.0", "lantern:1.1", "lantern:2.0", "lantern:2.1", "lantern", "scribe:1.0"} {
		if _, ok := Lookup(old); ok {
			t.Fatalf("retired image %s still resolves", old)
		}
		if _, ok := Lookup(Migrate(old)); !ok {
			t.Fatalf("migrated %s does not resolve", Migrate(old))
		}
	}
}

func TestIsRetired(t *testing.T) {
	if !IsRetired("lantern:1.0") || !IsRetired(RegistryHost+"/scribe:1.0") || IsRetired("porter:1.0") || IsRetired("") {
		t.Fatal("retired detection is wrong")
	}
}
