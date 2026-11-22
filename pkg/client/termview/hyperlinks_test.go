package termview

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestHyperlinksKeepTheViewsMemoryBounded(t *testing.T) {
	cols, rows := 40, 12
	v := New(cols, rows, func([]byte) {})
	defer v.Close()
	cell := []byte("\x1b]8;;https://example.invalid/" + strings.Repeat("a", 60000) + "\x07x")
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range cols * rows {
		if _, err := v.Write(cell); err != nil {
			t.Fatal(err)
		}
	}
	v.Text()
	runtime.GC()
	runtime.ReadMemStats(&after)
	grew := (int64(after.HeapInuse) - int64(before.HeapInuse)) >> 20
	t.Logf("sent %d MB over %d cells, the view's heap grew by %d MB", len(cell)*cols*rows>>20, cols*rows, grew)
	if grew > 8 {
		t.Fatalf("the view's heap grew by %d MB for a %dx%d screen: every cell keeps its own hyperlink", grew, cols, rows)
	}
}

func TestNoStringSequenceKeepsItsPayloadInTheView(t *testing.T) {
	cols, rows := 40, 12
	payload := strings.Repeat("a", 60000)
	for name, intro := range map[string][2]string{
		"hyperlink":         {"\x1b]8;;https://example.invalid/", "\x07"},
		"title":             {"\x1b]0;", "\x07"},
		"working directory": {"\x1b]7;file://host/", "\x1b\\"},
		"clipboard":         {"\x1b]52;c;", "\x07"},
		"private OSC":       {"\x1b]1337;File=", "\x07"},
		"DCS":               {"\x1bPq", "\x1b\\"},
		"APC":               {"\x1b_G", "\x1b\\"},
		"PM":                {"\x1b^", "\x1b\\"},
		"SOS":               {"\x1bX", "\x1b\\"},
		"8-bit OSC":         {"\x9d8;;", "\x9c"},
	} {
		v := New(cols, rows, func([]byte) {})
		cell := []byte(intro[0] + payload + intro[1] + "x")
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for range cols * rows {
			if _, err := v.Write(cell); err != nil {
				t.Fatal(err)
			}
		}
		text := strings.Join(v.Text(), "\n")
		runtime.GC()
		runtime.ReadMemStats(&after)
		v.Close()
		if grew := (int64(after.HeapInuse) - int64(before.HeapInuse)) >> 20; grew > 8 {
			t.Errorf("%s: the view's heap grew by %d MB over %d cells", name, grew, cols*rows)
		}
		if strings.Contains(text, "aaaa") || !strings.Contains(text, "xxxx") {
			t.Errorf("%s: the screen shows the payload or lost the text after it", name)
		}
	}
}

func TestTheStoneStillAnswersAColourQueryAfterDroppingStrings(t *testing.T) {
	replies := make(chan string, 16)
	v := New(20, 5, func(b []byte) { replies <- string(b) })
	defer v.Close()
	if _, err := v.Write([]byte("\x1b]8;;https://example.invalid/\x07\x1b]11;?\x07")); err != nil {
		t.Fatal(err)
	}
	v.Text()
	select {
	case r := <-replies:
		if !strings.Contains(r, "rgb:") {
			t.Errorf("the background colour query got %q", r)
		}
	case <-time.After(5 * time.Second):
		t.Error("the background colour query got no answer")
	}
}
