package lexicon

import (
	"regexp"
	"strings"
	"sync"
)

var storyParen = regexp.MustCompile(` \(([^()]+)\)`)

type storyGloss struct {
	start, end int
	term       string
}

var kubernetesTerms = termsOf(trueValues)

func termsOf(values map[string]bool) map[string]bool {
	out := make(map[string]bool, len(values))
	for v := range values {
		out[v] = true
	}
	if lx, err := Load(); err == nil {
		for _, e := range lx.Entries {
			out[strings.ToLower(e.K8s)] = true
		}
	}
	return out
}

func kubernetesTerm(inner string) bool {
	head, _, _ := strings.Cut(strings.ToLower(inner), ": ")
	return kubernetesTerms[head] || kubernetesTerms[strings.TrimSuffix(head, "s")]
}

func storyGlosses(text string) []storyGloss {
	named := TrueNames(text)
	out := []storyGloss{}
	for _, m := range storyParen.FindAllStringSubmatchIndex(text, -1) {
		inner := text[m[2]:m[3]]
		if kubernetesTerm(inner) || TrueNames(text[:m[0]]+text[m[1]:]) == named {
			out = append(out, storyGloss{start: m[0], end: m[1], term: inner})
		}
	}
	return out
}

type storyLabels struct {
	mu         sync.Mutex
	limit      int
	seen       map[string]string
	introduced map[string]bool
}

func newStoryLabels(limit int) *storyLabels {
	return &storyLabels{limit: limit, seen: make(map[string]string, limit), introduced: map[string]bool{}}
}

func (s *storyLabels) label(text string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if out, ok := s.seen[text]; ok {
		return out
	}
	if len(s.seen) >= s.limit {
		s.seen = make(map[string]string, s.limit)
	}
	out := s.strip(text)
	s.seen[text] = out
	return out
}

func (s *storyLabels) strip(text string) string {
	var b strings.Builder
	last := 0
	for _, g := range storyGlosses(text) {
		key := strings.ToLower(g.term)
		if !s.introduced[key] {
			s.introduced[key] = true
			continue
		}
		b.WriteString(text[last:g.start])
		last = g.end
	}
	b.WriteString(text[last:])
	return b.String()
}

func StoryLabeler(limit int) func(string) string {
	return newStoryLabels(limit).label
}
