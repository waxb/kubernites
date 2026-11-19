package hud

import (
	"strings"
	"testing"
)

var testVocabulary = Vocabulary{Names: map[string][]string{
	"pods":  {"porter-7f2c9", "porter-8ab01", "etcd-lone-tower"},
	"nodes": {"lone-tower", "stone-keep-1"},
}}

func TestCompleteVerbsAndTypes(t *testing.T) {
	cases := []struct {
		line, ghost string
		n           int
	}{
		{"desc", "ribe", 1},
		{"get po", "d", 2},
		{"get pods por", "ter-", 2},
		{"describe node lo", "ne-tower", 1},
		{"logs etc", "d-lone-tower", 1},
		{"kubectl get no", "de", 2},
		{"top no", "de", 2},
		{"cordon st", "one-keep-1", 1},
		{"get pods -n kube-system por", "ter-", 2},
	}
	for _, c := range cases {
		got := Complete(c.line, testVocabulary)
		if got.Ghost != c.ghost || len(got.Candidates) != c.n {
			t.Errorf("Complete(%q) = %+v, want ghost %q and %d candidates", c.line, got, c.ghost, c.n)
		}
	}
}

func TestCompleteNeverGuesses(t *testing.T) {
	for _, line := range []string{"", "get ", "frobnicate x", "get pods -z", "get pods -o q", "get pods -l a", "run x", "get widgets a"} {
		if got := Complete(line, testVocabulary); got.Ghost != "" || len(got.Candidates) != 0 {
			t.Errorf("Complete(%q) should offer nothing, got %+v", line, got)
		}
	}
	if CommonPrefix(nil) != "" {
		t.Error("empty prefix")
	}
}

func TestStoneTabAcceptsThenCycles(t *testing.T) {
	s := NewStone()
	for _, r := range "get pods por" {
		s.Insert(r)
	}
	s.Tab(testVocabulary)
	if string(s.Input) != "get pods porter-" {
		t.Fatalf("first tab = %q", string(s.Input))
	}
	s.Tab(testVocabulary)
	first := string(s.Input)
	s.Tab(testVocabulary)
	second := string(s.Input)
	if !strings.HasPrefix(first, "get pods porter-") || first == second {
		t.Fatalf("cycling = %q then %q", first, second)
	}
	s.Tab(testVocabulary)
	if string(s.Input) != first {
		t.Fatalf("cycle wraps: %q", string(s.Input))
	}
	s.SetInput("desc")
	s.Tab(testVocabulary)
	if string(s.Input) != "describe " {
		t.Fatalf("unique completion adds a space: %q", string(s.Input))
	}
	s.SetInput("desc")
	s.Move(-3)
	s.Tab(testVocabulary)
	if string(s.Input) != "desc" {
		t.Fatal("no completion unless the cursor is at the end")
	}
	s.SetInput("")
	s.Tab(testVocabulary)
	if len(s.Input) != 0 {
		t.Fatal("empty line: tab inserts nothing")
	}
}
