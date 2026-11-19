package live

import (
	"context"
	"time"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func SetWatchTiming(r *Realm, now func() time.Time, sleep func(context.Context, time.Duration) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wire != nil {
		r.wire.now, r.wire.sleep = now, sleep
	}
}

func MirrorHas(r *Realm, key store.Key) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.mirror.Peek(key)
	return ok
}
