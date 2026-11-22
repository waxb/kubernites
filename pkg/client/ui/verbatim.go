package ui

import "sync"

const verbatimMemory = 1024

type verbatimSet struct {
	mu   sync.Mutex
	seen map[string]bool
	ring []string
	next int
}

var verbatims = &verbatimSet{seen: make(map[string]bool, verbatimMemory), ring: make([]string, verbatimMemory)}

func Verbatim(text string) string {
	if text == "" {
		return text
	}
	v := verbatims
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.seen[text] {
		return text
	}
	if old := v.ring[v.next]; old != "" {
		delete(v.seen, old)
	}
	v.ring[v.next], v.seen[text] = text, true
	v.next = (v.next + 1) % verbatimMemory
	return text
}

func isVerbatim(text string) bool {
	v := verbatims
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.seen[text]
}
