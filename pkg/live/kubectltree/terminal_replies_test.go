package kubectltree

import (
	"bytes"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/client/termview"
)

func TestAContainerThatReadsNoInputCannotGrowTheTerminalsQueue(t *testing.T) {
	var term *Terminal
	view := termview.New(154, 10, func(b []byte) { term.Type(b) })
	term = NewTerminal(154, 10, view)
	defer view.Close()
	term.open(true, false)
	queries := bytes.Repeat([]byte("\x1b[6n\x1b[c"), 1<<15)
	sent := 0
	for range 16 {
		term.Write(queries)
		sent += len(queries)
	}
	queued, last := 0, -1
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline) && queued != last; time.Sleep(50 * time.Millisecond) {
		last = queued
		term.mu.Lock()
		queued = len(term.keys)
		term.mu.Unlock()
	}
	if queued > 1<<20 {
		t.Fatalf("the container wrote %d bytes of terminal queries and read no input: %d bytes of answers wait in the app's memory, with no bound", sent, queued)
	}
}
