package ui

import (
	"sync"
	"sync/atomic"
)

const namedMemory = 8192

type namerState struct {
	mu    sync.Mutex
	f     func(string) string
	named map[string]string
	kube  bool
}

var current atomic.Pointer[namerState]

func SetNamer(f func(string) string) {
	if f == nil {
		current.Store(nil)
		return
	}
	current.Store(&namerState{f: f, named: make(map[string]string, namedMemory), kube: true})
}

func SetStoryNamer(f func(string) string) {
	if f == nil {
		current.Store(nil)
		return
	}
	current.Store(&namerState{f: f, named: make(map[string]string, namedMemory)})
}

func Naming() bool {
	s := current.Load()
	return s != nil && s.kube
}

func StoryNaming() bool {
	s := current.Load()
	return s != nil && !s.kube
}

func Name(text string) string {
	if s := current.Load(); s != nil {
		return s.name(text)
	}
	return text
}

func Settle(text string) string {
	if s := current.Load(); s != nil {
		s.settle(text)
	}
	return text
}

func Bridge(story, k8s string) string {
	if !Naming() {
		return story
	}
	return Settle(k8s)
}

func (s *namerState) name(text string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if out, ok := s.named[text]; ok {
		return out
	}
	out := s.f(text)
	s.remember(text, out)
	s.remember(out, out)
	return out
}

func (s *namerState) settle(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remember(text, text)
}

func (s *namerState) remember(text, out string) {
	if _, ok := s.named[text]; !ok && len(s.named) >= namedMemory {
		s.named = make(map[string]string, namedMemory)
	}
	s.named[text] = out
}

func (s *namerState) size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.named)
}

func (f Font) named(text string) string {
	if f.Literal {
		return text
	}
	return Decorate(Name(text))
}

func (f Font) settled(text string) string {
	if f.Literal {
		return text
	}
	return Settle(text)
}
