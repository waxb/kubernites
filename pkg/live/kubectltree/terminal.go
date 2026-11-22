package kubectltree

import (
	"io"
	"sync"
	"time"

	"k8s.io/client-go/tools/remotecommand"
)

const LeaveKeys = "ctrl-p,ctrl-q"

const (
	MaxQueued  = 64 << 10
	LeaveGrace = 500 * time.Millisecond
	leaveFirst = 0x10
	leaveLast  = 0x11
)

const (
	typedGround = iota
	typedEscape
	typedControl
	typedString
	typedStringEnd
	typedSS3
)

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
	failed  error
	stop    func()
	left    bool
	armed   bool
	typed   int
	seq     []byte
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
	if t.closed || t.left {
		t.mu.Unlock()
		return
	}
	for _, c := range b {
		if t.typedByte(c) {
			t.left, t.armed, t.seq = true, false, nil
			stop := t.stop
			drained := len(t.keys) == 0
			t.wake.Broadcast()
			t.mu.Unlock()
			if stop != nil {
				if drained {
					stop()
				} else {
					time.AfterFunc(LeaveGrace, stop)
				}
			}
			return
		}
	}
	if t.typed != typedGround {
		if t.armed {
			t.disarm()
		}
		t.typed, t.seq = typedGround, t.seq[:0]
	}
	t.wake.Broadcast()
	t.mu.Unlock()
}

func (t *Terminal) queue(b ...byte) {
	room := MaxQueued - len(t.keys)
	if room <= 0 {
		return
	}
	if len(b) > room {
		b = b[:room]
	}
	t.keys = append(t.keys, b...)
}

func (t *Terminal) disarm() {
	if t.armed {
		t.armed = false
		t.queue(leaveFirst)
		t.queue(t.seq...)
	}
	t.seq = t.seq[:0]
}

func (t *Terminal) inSequence(c byte) {
	if t.armed {
		t.seq = append(t.seq, c)
	} else {
		t.queue(c)
	}
}

func (t *Terminal) sequenceEnds(reply bool) {
	t.typed = typedGround
	if !t.armed {
		return
	}
	if !reply {
		t.disarm()
		return
	}
	t.queue(t.seq...)
	t.seq = t.seq[:0]
}

func (t *Terminal) typedByte(c byte) bool {
	switch t.typed {
	case typedEscape:
		t.inSequence(c)
		switch c {
		case '[':
			t.typed = typedControl
		case ']', 'P', '_', '^', 'X':
			t.typed = typedString
		case 'O':
			t.typed = typedSS3
		default:
			t.sequenceEnds(false)
		}
		return false
	case typedSS3:
		t.inSequence(c)
		t.sequenceEnds(false)
		return false
	case typedControl:
		t.inSequence(c)
		switch {
		case c >= 0x40 && c <= 0x7e:
			t.sequenceEnds(replyFinal(c))
		case len(t.seq) > maxReply:
			t.sequenceEnds(false)
		}
		return false
	case typedString, typedStringEnd:
		t.inSequence(c)
		switch {
		case c == 0x07, t.typed == typedStringEnd && c == '\\':
			t.sequenceEnds(true)
		case c == 0x1b:
			t.typed = typedStringEnd
		case len(t.seq) > maxReply:
			t.sequenceEnds(false)
		default:
			t.typed = typedString
		}
		return false
	}
	switch {
	case c == 0x1b:
		t.typed = typedEscape
		t.inSequence(c)
	case c == leaveLast && t.armed:
		t.armed = false
		t.seq = t.seq[:0]
		return true
	case c == leaveFirst:
		t.disarm()
		t.armed = true
	default:
		t.disarm()
		t.queue(c)
	}
	return false
}

const maxReply = 256

func replyFinal(c byte) bool {
	switch c {
	case 'R', 'c', 'n', 'y', 't', 'I', 'O':
		return true
	}
	return false
}

func (t *Terminal) Left() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.left
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
	t.failed, t.stop = nil, nil
	t.left, t.armed, t.typed, t.seq = false, false, typedGround, nil
	t.active = !t.closed
}

func (t *Terminal) stopWith(stop func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stop = stop
}

func (t *Terminal) Failed() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failed
}

func (t *Terminal) fail(err error) {
	t.mu.Lock()
	if t.failed == nil {
		t.failed = err
	}
	stop := t.stop
	t.active = false
	t.wake.Broadcast()
	t.mu.Unlock()
	if stop != nil {
		stop()
	}
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
	for len(t.keys) == 0 && t.active && !t.left {
		t.wake.Wait()
	}
	if len(t.keys) == 0 || !t.active {
		if t.left && t.stop != nil {
			go t.stop()
		}
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
	if !tty {
		if _, err := out.Write(crlf(p)); err != nil {
			t.fail(err)
			return 0, err
		}
		return len(p), nil
	}
	n, err := out.Write(p)
	if err != nil {
		t.fail(err)
	}
	return n, err
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
