package gfxtest

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
	_ "unsafe"

	_ "github.com/hajimehoshi/ebiten/v2"
)

const reapEvery = 20 * time.Millisecond

//go:linkname deferred github.com/hajimehoshi/ebiten/v2/internal/atlas.deferred
var deferred []func()

//go:linkname deferredM github.com/hajimehoshi/ebiten/v2/internal/atlas.deferredM
var deferredM sync.Mutex

var holds atomic.Int64

func Pending() int {
	deferredM.Lock()
	defer deferredM.Unlock()
	return len(deferred)
}

func Drop() {
	deferredM.Lock()
	deferred = nil
	deferredM.Unlock()
}

func reap(every time.Duration) func() {
	quit, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		tick := time.NewTicker(every)
		defer tick.Stop()
		for {
			select {
			case <-quit:
				return
			case <-tick.C:
				if holds.Load() == 0 {
					Drop()
				}
			}
		}
	}()
	return func() {
		close(quit)
		<-done
	}
}

func Hold() func() {
	holds.Add(1)
	var once sync.Once
	return func() { once.Do(func() { holds.Add(-1) }) }
}

func Main(m *testing.M) int {
	stop := reap(reapEvery)
	defer stop()
	return m.Run()
}

func Reap(every time.Duration) func() {
	return reap(every)
}
