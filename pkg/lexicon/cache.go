package lexicon

import "sync"

func TrueNamer(limit int) func(string) string {
	var mu sync.Mutex
	seen := make(map[string]string, limit)
	return func(text string) string {
		mu.Lock()
		defer mu.Unlock()
		if out, ok := seen[text]; ok {
			return out
		}
		if len(seen) >= limit {
			seen = make(map[string]string, limit)
		}
		out := TrueNames(text)
		seen[text] = out
		return out
	}
}
