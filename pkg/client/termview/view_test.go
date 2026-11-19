package termview

import (
	"strings"
	"sync"
	"testing"
	"time"
)

type sent struct {
	mu  sync.Mutex
	out []byte
}

func (s *sent) add(b []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.out = append(s.out, b...)
}

func (s *sent) wait(t *testing.T, want string) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(time.Millisecond) {
		s.mu.Lock()
		got := string(s.out)
		s.mu.Unlock()
		if got == want {
			s.mu.Lock()
			s.out = nil
			s.mu.Unlock()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("sent %q, want %q", got, want)
		}
	}
}

func TestTheViewDrawsColoursTheCursorAndTheAlternateScreen(t *testing.T) {
	v := New(20, 5, nil)
	defer v.Close()
	v.Write([]byte("plain \x1b[1;31mred\x1b[0m\r\n\x1b[7mrev\x1b[0m"))
	row := v.Row(0)
	if row[6].Text != "r" || !row[6].HasFg || row[6].Fg.R < 100 || row[6].Fg.G > 100 || !row[6].Bold || row[0].HasFg {
		t.Fatalf("row 0: %+v", row[:9])
	}
	if c := v.Row(1)[0]; c.Text != "r" || !c.Reverse {
		t.Fatalf("row 1: %+v", c)
	}
	if x, y, shown := v.Cursor(); x != 3 || y != 1 || !shown {
		t.Fatalf("cursor %d %d %v", x, y, shown)
	}
	v.Write([]byte("\x1b[?1049h\x1b[H\x1b[2Jtop\x1b[?25l"))
	if !v.AltScreen() || !strings.HasPrefix(v.Text()[0], "top") {
		t.Fatalf("alternate screen %v: %q", v.AltScreen(), v.Text())
	}
	if _, _, shown := v.Cursor(); shown {
		t.Fatal("a hidden cursor is shown")
	}
	v.Write([]byte("\x1b[?1049l\x1b[?25h"))
	if v.AltScreen() || !strings.HasPrefix(v.Text()[0], "plain red") {
		t.Fatalf("back on the main screen %v: %q", v.AltScreen(), v.Text())
	}
	v.Resize(30, 8)
	if w, h := v.Size(); w != 30 || h != 8 {
		t.Fatalf("resized to %dx%d", w, h)
	}
}

func TestKeysAreEncodedAsTheProgramAsks(t *testing.T) {
	s := &sent{}
	v := New(20, 5, s.add)
	defer v.Close()
	v.Key(Key{Name: "up"})
	s.wait(t, "\x1b[A")
	v.Write([]byte("\x1b[?1h"))
	v.Key(Key{Name: "up"})
	s.wait(t, "\x1bOA")
	v.Key(Key{Rune: 'p', Ctrl: true})
	v.Key(Key{Rune: 'q', Ctrl: true})
	s.wait(t, "\x10\x11")
	v.Key(Key{Rune: 'c', Ctrl: true})
	v.Key(Key{Name: "enter"})
	v.Key(Key{Name: "backspace"})
	v.Key(Key{Name: "escape"})
	v.Key(Key{Name: "tab", Shift: true})
	s.wait(t, "\x03\r\x7f\x1b\x1b[Z")
	v.Key(Key{Rune: 'x', Alt: true})
	v.Type("héllo")
	s.wait(t, "\x1bxhéllo")
	v.Key(Key{Name: "f5"})
	v.Key(Key{Name: "pgdown"})
	s.wait(t, "\x1b[15~\x1b[6~")
	v.Write([]byte("\x1b[6n"))
	s.wait(t, "\x1b[1;1R")
}
