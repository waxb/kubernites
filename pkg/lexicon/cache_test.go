package lexicon

import "testing"

func TestTrueNamerMatchesTrueNamesAndStaysBounded(t *testing.T) {
	name := TrueNamer(4)
	for i := 0; i < 3; i++ {
		for _, s := range []string{"a hut", "the Keep", "Musters", "plain", "Guildhall", "Codex"} {
			if got, want := name(s), TrueNames(s); got != want {
				t.Fatalf("namer(%q) = %q, want %q", s, got, want)
			}
		}
	}
}
