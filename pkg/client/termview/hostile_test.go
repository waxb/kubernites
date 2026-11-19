package termview

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestOneRepeatSequenceNeverPinsTheView(t *testing.T) {
	w := New(154, 27, func([]byte) {})
	wrote := make(chan struct{})
	go func() {
		w.Write([]byte("x\x1b[2147483646b"))
		close(wrote)
	}()
	time.Sleep(200 * time.Millisecond)
	done := make(chan struct{})
	go func() {
		w.Row(0)
		w.Key(Key{Rune: 'p', Ctrl: true})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Row (the Stone's draw) and Key (ctrl-p) waited behind one REP sequence")
	}
	select {
	case <-wrote:
	case <-time.After(5 * time.Second):
		t.Fatal("one REP sequence held the emulator for more than 5 s")
	}
	w.Close()
	v := New(154, 27, func([]byte) {})
	defer v.Close()
	start := time.Now()
	v.Write([]byte(fmt.Sprintf("x\x1b[%db", 2000000)))
	if took := time.Since(start); took > 2*time.Second {
		t.Fatalf("REP 2000000 took %v", took)
	}
}

func TestMalformedSequencesNeverPanicOrHang(t *testing.T) {
	inputs := []string{
		"\x1b", "\x1b[", "\x1b[999999999999999999999999H", "\x1b[;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;m",
		"\x1b]8;;", "\x1b]0;", "\x1bP", "\x1bP$q", "\x1b_", "\x1b^", "\x1bX", "\x1b]52;c;SGVsbG8=\x07",
		"\x1b[?69h\x1b[5;2s", "\x1b[?69h\x1b[100;200s", "\x1b[?6h\x1b[5;3r\x1b[2147483646L", "\x1b[5;3r\x1b[2147483646M",
		"\x1b[2147483646@", "\x1b[2147483646P", "\x1b[2147483646X", "\x1b[2147483646S", "\x1b[2147483646T",
		"\x1b[2147483646I", "\x1b[2147483646Z", "\x1b[2147483646G", "\x1b[2147483646d", "\x1b[2147483646e", "\x1b[2147483646a",
		"\xff\xfe\xfd", "́́́", "\U0001F600\x1b[1;154H\U0001F600\U0001F600", "\x1b[?1049h\x1b[?1049h\x1b[?1049l\x1b[?1049l",
		"\x1bc\x1bc", "\x1b[?2048h", "\x1b[0 q\x1b[99 q", "\x1b[38;2;999;999;999m", "\x1b[38;5;9999m", "\x1b[48:2::1:2:3m",
		"\x1b]10;?\x07\x1b]11;?\x07\x1b]12;?\x07", "\x1b]10;rgb:zz/zz/zz\x07", "\x1b]4;300;rgb:1/2/3\x07", "\x1b[6n\x1b[?6n\x1b[5n\x1b[c\x1b[>c",
		"\x1b[$p\x1b[?$p\x1b[?99999$p", "\x1b(0lqk\x1b(B", "\x1bN\x1bO", "\x1b#8", "\x1b[3J\x1b[2J", "\x1b[0;0r\x1b[0;0H",
		"\x1b[?6h\x1b[?69h\x1b[3;4s\x1b[2;3r\x1b[Hxxxxxxxxxxxxxxxx\x1b[2147483646b",
	}
	for _, in := range inputs {
		if strings.HasSuffix(in, "2147483646b") {
			in = strings.TrimSuffix(in, "2147483646b") + "50b"
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("%q panicked: %v", in, p)
				}
			}()
			v := New(154, 27, func([]byte) {})
			defer v.Close()
			v.Write([]byte(in))
			v.Resize(1, 1)
			v.Write([]byte(in))
			v.Resize(154, 27)
			v.Text()
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Errorf("%q still running after 5s", in)
		}
	}
}

func TestCombiningMarksKeepTheViewsMemoryBounded(t *testing.T) {
	v := New(154, 27, func([]byte) {})
	defer v.Close()
	chunk := "a" + strings.Repeat("́", 16000) + "\r\n"
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range 3000 {
		v.Write([]byte(chunk))
	}
	v.Text()
	runtime.GC()
	runtime.ReadMemStats(&after)
	grew := (int64(after.HeapInuse) - int64(before.HeapInuse)) >> 20
	if grew > 16 {
		t.Fatalf("sent %d MB, the view's heap grew by %d MB", len(chunk)*3000>>20, grew)
	}
}

func TestAClusterOfMarksIsCappedAndTheViewKeepsNoScrollback(t *testing.T) {
	v := New(80, 24, func([]byte) {})
	defer v.Close()
	v.Write([]byte("a" + strings.Repeat("́", 5000) + strings.Repeat("‍", 5000) + "\r\n"))
	if cell := v.Row(0)[0].Text; len(cell) > 1+2*maxMarks+3*maxMarks {
		t.Fatalf("one cell holds %d bytes", len(cell))
	}
	for range 2000 {
		v.Write([]byte("line\r\n"))
	}
	v.mu.Lock()
	kept := v.emu.ScrollbackLen()
	v.mu.Unlock()
	if kept > 1 {
		t.Fatalf("the view keeps %d lines of scrollback", kept)
	}
}
