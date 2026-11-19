package kubectltree

import (
	"io"
	"sync"

	"k8s.io/client-go/tools/remotecommand"
)

const LeaveKeys = "ctrl-p,ctrl-q"

type Terminal struct {
	mu      sync.Mutex
	wake    *sync.Cond
	keys    []byte
	size    remotecommand.TerminalSize
	resized bool
	sent    bool
	live    bool
	active  bool
	nudge   bool
	tty     bool
	closed  bool
	out     io.Writer
}

func NewTerminal(cols, rows int, out io.Writer) *Terminal {
	t := &Terminal{size: remotecommand.TerminalSize{Width: uint16(max(cols, 1)), Height: uint16(max(rows, 1))}, out: out}
	t.wake = sync.NewCond(&t.mu)
	return t
}

func (t *Terminal) Type(b []byte) {
	if len(b) == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.keys = append(t.keys, b...)
	t.wake.Broadcast()
}

func (t *Terminal) Resize(cols, rows int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	size := remotecommand.TerminalSize{Width: uint16(max(cols, 1)), Height: uint16(max(rows, 1))}
	if size == t.size {
		return
	}
	t.size, t.resized = size, true
	t.wake.Broadcast()
}

func (t *Terminal) Size() (int, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return int(t.size.Width), int(t.size.Height)
}

func (t *Terminal) Live() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.live && !t.closed
}

func (t *Terminal) TTY() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tty
}

func (t *Terminal) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed, t.active, t.live = true, false, false
	t.wake.Broadcast()
}

func (t *Terminal) open(tty, nudge bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tty, t.nudge = tty, nudge
	t.keys = nil
	t.sent = false
	t.active = !t.closed
}

func (t *Terminal) end() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.live, t.active = false, false
	t.wake.Broadcast()
}

func (t *Terminal) Read(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.live = t.active
	for len(t.keys) == 0 && t.active {
		t.wake.Wait()
	}
	if len(t.keys) == 0 || !t.active {
		return 0, io.EOF
	}
	n := copy(p, t.keys)
	t.keys = t.keys[n:]
	return n, nil
}

func (t *Terminal) Next() *remotecommand.TerminalSize {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.live = t.active
	for t.sent && !t.resized && t.active {
		t.wake.Wait()
	}
	if !t.active {
		return nil
	}
	size := t.size
	if t.nudge && !t.sent {
		size.Width++
		size.Height++
		t.resized = true
	} else {
		t.resized = false
	}
	t.sent = true
	return &size
}

func (t *Terminal) Write(p []byte) (int, error) {
	t.mu.Lock()
	out, tty := t.out, t.tty
	t.mu.Unlock()
	if out == nil {
		return len(p), nil
	}
	if tty {
		return out.Write(p)
	}
	if _, err := out.Write(crlf(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

func crlf(p []byte) []byte {
	out := make([]byte, 0, len(p)+8)
	for i, b := range p {
		if b == '\n' && (i == 0 || p[i-1] != '\r') {
			out = append(out, '\r')
		}
		out = append(out, b)
	}
	return out
}
