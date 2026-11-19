package kubectltree

import (
	"context"
	"sync"
)

type Chunk struct {
	Text   string
	Stderr bool
}

type Job struct {
	mu      sync.Mutex
	held    int
	fresh   []Chunk
	writes  []Write
	done    bool
	res     Result
	applied *Applied
	cancel  context.CancelFunc
	term    *Terminal
	session *Session
}

func (t *Tree) Start(line string, hold Hold) *Job {
	ctx, cancel := context.WithCancel(context.Background())
	j := &Job{cancel: cancel}
	go func() {
		res := t.run(ctx, line, hooks{sink: j.add, hold: hold, wrote: j.wrote})
		cancel()
		j.finish(res, nil)
	}()
	return j
}

func (t *Tree) StartIn(line string, hold Hold, term *Terminal) *Job {
	ctx, cancel := context.WithCancel(context.Background())
	j := &Job{cancel: cancel, term: term}
	go func() {
		res := t.run(ctx, line, hooks{sink: j.add, hold: hold, wrote: j.wrote, term: term, opened: j.opened})
		cancel()
		if term != nil {
			term.Close()
		}
		j.finish(res, nil)
	}()
	return j
}

func (j *Job) opened(s *Session) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.session = s
}

func (j *Job) Terminal() *Terminal {
	return j.term
}

func (t *Tree) StartApply(text string, force bool, hold Hold) *Job {
	ctx, cancel := context.WithCancel(context.Background())
	j := &Job{cancel: cancel}
	go func() {
		a := t.apply(ctx, text, force, hooks{hold: hold, wrote: j.wrote})
		cancel()
		j.finish(a.Result, &a)
	}()
	return j
}

func (j *Job) Applied() (Applied, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.applied == nil {
		return Applied{}, false
	}
	return *j.applied, true
}

func (j *Job) add(text string, stderr bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.held += len(text)
	if n := len(j.fresh); n > 0 && j.fresh[n-1].Stderr == stderr && len(j.fresh[n-1].Text) < MaxOutput/16 {
		j.fresh[n-1].Text += text
	} else {
		j.fresh = append(j.fresh, Chunk{Text: text, Stderr: stderr})
	}
	for j.held > MaxOutput && len(j.fresh) > 1 {
		j.held -= len(j.fresh[0].Text)
		j.fresh = j.fresh[1:]
	}
}

func (j *Job) wrote(w Write) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.writes = append(j.writes, w)
}

func (j *Job) TakeWrites() []Write {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := j.writes
	j.writes = nil
	return out
}

func (j *Job) finish(res Result, a *Applied) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.res, j.applied, j.done = res, a, true
}

func (j *Job) Take() ([]Chunk, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := j.fresh
	j.fresh, j.held = nil, 0
	return out, j.done
}

func (j *Job) Done() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.done
}

func (j *Job) Result() Result {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.res
}

func (j *Job) Stop() {
	j.cancel()
}
