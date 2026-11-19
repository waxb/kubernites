package lexicon

import (
	"strings"
	"testing"
)

func TestEveryPlainWordIsShortAndFreeOfKubernetesWords(t *testing.T) {
	seen := map[string]bool{}
	words := PlainWords()
	if len(words) < 40 {
		t.Fatalf("the plain table holds %d words", len(words))
	}
	for _, w := range words {
		if seen[w.Term] {
			t.Errorf("%q appears twice", w.Term)
		}
		seen[w.Term] = true
		if n := len(strings.Fields(w.Means)); n > 7 {
			t.Errorf("%q means %q: %d words, the table keeps about six", w.Term, w.Means, n)
		}
		if leak := KubernetesWords.FindString(w.Means); leak != "" {
			t.Errorf("%q means %q, which keeps %q", w.Term, w.Means, leak)
		}
		if w.First != "" && w.First != "dev" && w.First != "staging" {
			t.Errorf("%q is first-seen at %q", w.Term, w.First)
		}
	}
}

func TestAPlainWordIsFoundWhereItsTermIsDrawn(t *testing.T) {
	got := PlainMatches("The Crowding Ward turns the Reeve's huts away from Stewards.")
	terms := []string{}
	for _, w := range got {
		terms = append(terms, w.Term)
	}
	want := []string{"Crowding Ward", "Stewards", "Reeve"}
	for _, term := range want {
		if !strings.Contains(strings.Join(terms, "|"), term) {
			t.Errorf("%q not found in %v", term, terms)
		}
	}
	for _, term := range terms {
		if term == "Ward" || term == "Crowding" {
			t.Errorf("%q shown beside the longer Crowding Ward: %v", term, terms)
		}
	}
	if len(PlainMatches("a badge on a hut keeps a ward")) != 0 {
		t.Error("common lowercase words must not draw a gloss")
	}
}
