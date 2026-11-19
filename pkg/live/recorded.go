package live

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/waxb/kubernites/pkg/live/write"
)

func ReadSnapshot(r io.Reader) (Snapshot, error) {
	var s Snapshot
	err := json.NewDecoder(r).Decode(&s)
	return s, err
}

func OpenRecorded(s Snapshot) (*Realm, error) {
	return Open(s, nil, Usage{})
}

func OpenRecordedGated(s Snapshot, g *write.Gate) (*Realm, error) {
	if g == nil {
		return nil, errors.New("a recorded realm with writes needs the write gate")
	}
	r, err := Open(s, nil, Usage{})
	if err != nil {
		return nil, err
	}
	r.gate = g
	g.StaleWhen(r.staleReason)
	return r, nil
}

var errConnected = errors.New("a connected realm takes events only from its own cluster")

func (r *Realm) Replay(list string, stream io.Reader) error {
	r.mu.Lock()
	connected := r.wire != nil
	r.replayed = r.replayed || !connected
	r.mu.Unlock()
	if connected {
		return errConnected
	}
	k, ok := r.kindOf(list)
	if !ok {
		return fmt.Errorf("the realm does not read %s", list)
	}
	version := ""
	_, err := r.follow(k, stream, &version)
	return err
}

func (r *Realm) kindOf(list string) (kindRead, bool) {
	for _, k := range kindReads {
		if k.name == list {
			return k, true
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.snap.CRDs {
		if c.List() == list {
			return customKind(c), true
		}
	}
	return kindRead{}, false
}

func WatchEvent(typ string, obj any) []byte {
	data, err := json.Marshal(map[string]any{"type": typ, "object": obj})
	if err != nil {
		panic(err)
	}
	return append(data, '\n')
}

func (r *Realm) ReplayStale(since time.Time, why string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wire != nil {
		return errConnected
	}
	r.replayed, r.staleAt, r.staleWhy = true, since, why
	return nil
}

func (r *Realm) ReplayListed(list []Listed) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wire != nil {
		return errConnected
	}
	r.replayed = true
	r.reads.listing = map[string]Listed{}
	for _, l := range list {
		r.reads.listing[l.List] = l
	}
	return nil
}
