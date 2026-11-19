package names

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var namedForTest = Kubernetes(1 << 18)

func TestCorpusDump(t *testing.T) {
	path := os.Getenv("CORPUS_DUMP")
	if path == "" {
		t.Skip()
	}
	var b strings.Builder
	lits := shownLiterals(t)
	keys := make([]string, 0, len(lits))
	for s := range lits {
		keys = append(keys, s)
	}
	sort.Strings(keys)
	for _, s := range keys {
		if out := namedForTest(s); out != s {
			b.WriteString(strings.ReplaceAll(s, "\n", "\\n") + "\n=> " + strings.ReplaceAll(out, "\n", "\\n") + "\n")
		}
	}
	_ = os.WriteFile(path, []byte(b.String()), 0o644)
}

var grammarDefects = map[string]*regexp.Regexp{
	"a verb turned into Node":     regexp.MustCompile(`\bNodes (the|a|an|your|its|their|every|each|all|one|no|them|it|what|three|two)\b`),
	"a unit without a number":     regexp.MustCompile(`(^|[^0-9])\bGi\b`),
	"a plural verb after SLI/SLO": regexp.MustCompile(`\b(SLI|SLO) (are|have|now ask|hold|ask|fall|say|show|count)\b`),
	"a service name renamed":      regexp.MustCompile(`\b(service|svc|deployment)/(web|database|customer|batch)\b`),
	"a Service named by cargo":    regexp.MustCompile(`\b(web requests|database requests|uploads|batch jobs) Service\b`),
	"an old tier name":            regexp.MustCompile(`\bImperial\b`),
	"a fused tier name":           regexp.MustCompile(`\bfeudal (Environment|Tier)\b`),
}

func doubledWord(text string) string {
	words := strings.Fields(text)
	for i := 1; i < len(words); i++ {
		if len(words[i]) > 2 && strings.EqualFold(words[i], words[i-1]) && !strings.ContainsAny(words[i], "0123456789.,;:") {
			return words[i]
		}
	}
	return ""
}

func TestKubernetesNamesKeepTheirGrammar(t *testing.T) {
	for lit, at := range shownLiterals(t) {
		got := namedForTest(lit)
		if w := doubledWord(got); w != "" && doubledWord(lit) == "" {
			t.Errorf("%s: doubled %q: %q reads %q", at, w, lit, got)
		}
		for what, pat := range grammarDefects {
			if pat.MatchString(got) && !pat.MatchString(lit) {
				t.Errorf("%s: %s: %q reads %q", at, what, lit, got)
			}
		}
	}
}
