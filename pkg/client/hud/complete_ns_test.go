package hud

import (
	"slices"
	"testing"
)

var spacedVocabulary = Vocabulary{
	Names: map[string][]string{
		"pods":       {"porter-7f2c9", "etcd-lone-tower", "kube-apiserver-lone-tower"},
		"nodes":      {"lone-tower"},
		"namespaces": {"default", "kube-node-lease", "kube-public", "kube-system"},
	},
	Spaced: map[string]map[string][]string{
		"pods": {"default": {"porter-7f2c9"}, "kube-system": {"etcd-lone-tower", "kube-apiserver-lone-tower"}},
	},
}

func TestCompleteFlagValues(t *testing.T) {
	cases := []struct {
		line, ghost string
		n           int
	}{
		{"get pods -n kube-s", "ystem", 1},
		{"get pods --namespace kube-", "", 3},
		{"get pods --namespace=kube-s", "ystem", 1},
		{"get pods -n=kube-p", "ublic", 1},
		{"get pods -o w", "ide", 1},
		{"get pods --output=y", "aml", 1},
		{"-n kube-s", "ystem", 1},
		{"kubectl -n kube-s", "ystem", 1},
	}
	for _, c := range cases {
		got := Complete(c.line, spacedVocabulary)
		if got.Ghost != c.ghost || len(got.Candidates) != c.n {
			t.Errorf("Complete(%q) = %+v, want ghost %q and %d candidates", c.line, got, c.ghost, c.n)
		}
	}
}

func TestCompleteOffersEveryNamespaceAfterTheFlag(t *testing.T) {
	got := Complete("get pods -n ", spacedVocabulary)
	if len(got.Candidates) != 4 || got.Token != "" {
		t.Fatalf("after -n = %+v", got)
	}
	if got := Complete("get pods ", spacedVocabulary); len(got.Candidates) != 0 {
		t.Errorf("a plain space still offers nothing: %+v", got)
	}
}

func TestCompleteFlagNames(t *testing.T) {
	if got := Complete("get pods --all-n", spacedVocabulary); got.Ghost != "amespaces" {
		t.Errorf("--all-n = %+v", got)
	}
	if got := Complete("get pods --names", spacedVocabulary); got.Ghost != "pace" {
		t.Errorf("--names = %+v", got)
	}
	got := Complete("get pods -", spacedVocabulary)
	if !slices.Contains(got.Candidates, "-n") || !slices.Contains(got.Candidates, "-A") || got.Ghost != "" {
		t.Errorf("get pods - = %+v", got)
	}
}

func TestCompleteNamesFollowTheNamespace(t *testing.T) {
	cases := []struct {
		line, ghost string
		n           int
	}{
		{"describe pod etc", "", 0},
		{"describe pod por", "ter-7f2c9", 1},
		{"describe pod -n kube-system etc", "d-lone-tower", 1},
		{"describe pod --namespace=kube-system etc", "d-lone-tower", 1},
		{"logs -n kube-system kube", "-apiserver-lone-tower", 1},
		{"logs etc", "", 0},
		{"get pods -A etc", "d-lone-tower", 1},
		{"get pods --all-namespaces kube", "-apiserver-lone-tower", 1},
		{"-n kube-system describe pod etc", "d-lone-tower", 1},
		{"describe node lo", "ne-tower", 1},
	}
	for _, c := range cases {
		got := Complete(c.line, spacedVocabulary)
		if got.Ghost != c.ghost || len(got.Candidates) != c.n {
			t.Errorf("Complete(%q) = %+v, want ghost %q and %d candidates", c.line, got, c.ghost, c.n)
		}
	}
}

func TestStoneTabCyclesNamespaces(t *testing.T) {
	s := NewStone()
	s.SetInput("get pods -n ")
	s.Tab(spacedVocabulary)
	if string(s.Input) != "get pods -n default" {
		t.Fatalf("first tab = %q", string(s.Input))
	}
	s.Tab(spacedVocabulary)
	if string(s.Input) != "get pods -n kube-node-lease" {
		t.Fatalf("second tab = %q", string(s.Input))
	}
	s.SetInput("get pods --namespace=kube-s")
	s.Tab(spacedVocabulary)
	if string(s.Input) != "get pods --namespace=kube-system " {
		t.Fatalf("attached value = %q", string(s.Input))
	}
}
