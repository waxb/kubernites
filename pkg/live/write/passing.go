package write

import (
	"context"
	"net/http"
	"sync"
)

type Passing struct {
	mu     sync.Mutex
	passed bool
	change bool
}

type passingKey struct{}

func WithPassing(ctx context.Context) (context.Context, *Passing) {
	v := &Passing{}
	return context.WithValue(ctx, passingKey{}, v), v
}

func (v *Passing) Passed() (passed, change bool) {
	if v == nil {
		return false, false
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.passed, v.change
}

func notePassing(req *http.Request, change bool) {
	v, ok := req.Context().Value(passingKey{}).(*Passing)
	if !ok {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.passed = true
	v.change = v.change || change
}
