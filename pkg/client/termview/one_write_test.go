package termview

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEveryKeyAndEveryReplyReachesTheTerminalAsOneWrite(t *testing.T) {
	var mu sync.Mutex
	var writes []string
	v := New(80, 24, func(b []byte) {
		mu.Lock()
		writes = append(writes, string(b))
		mu.Unlock()
	})
	defer v.Close()
	wait := func(n int) []string {
		t.Helper()
		for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(time.Millisecond) {
			mu.Lock()
			got := append([]string(nil), writes...)
			mu.Unlock()
			if len(got) >= n {
				return got
			}
			if time.Now().After(deadline) {
				t.Fatalf("only %d of %d writes arrived: %q", len(got), n, got)
			}
		}
	}
	keys := []Key{{Name: "escape"}, {Rune: 'P'}, {Name: "f3"}, {Name: "up"}, {Name: "f12"}, {Rune: 'p', Ctrl: true}, {Name: "tab", Shift: true}, {Name: "pgdown"}}
	for i, k := range keys {
		v.Key(k)
		got := wait(i + 1)
		if len(got) != i+1 {
			t.Fatalf("key %+v reached the terminal as %d writes: %q", k, len(got)-i, got[i:])
		}
	}
	v.Write([]byte("\x1b[6n\x1b[c\x1b[>c\x1b[5n\x1b[?6n"))
	got := wait(len(keys) + 5)
	for _, reply := range got[len(keys):] {
		if !strings.HasPrefix(reply, "\x1b[") || strings.Count(reply, "\x1b") != 1 {
			t.Errorf("an emulator reply reached the terminal split or joined: %q", reply)
		}
	}
}
