package kubectltree_test

import (
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/client/termview"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func TestTheLeaveChordIsSeenAfterEscapeAndAnyKey(t *testing.T) {
	for _, typed := range [][]string{
		{"\x1b", "P", "i", ":wq\r"}, {"\x1b", "]"}, {"\x1b", "X"}, {"\x1b", "^"}, {"\x1b", "_"}, {"\x1b"}, {"\x1b", "O"},
	} {
		term := kubectltree.NewTerminal(80, 24, &screen{})
		for _, b := range append(typed, "\x10", "\x11") {
			term.Type([]byte(b))
		}
		if !term.Left() {
			t.Errorf("%q typed as keys (as in vi), then ctrl-p ctrl-q: the terminal never saw the chord and queued %q for the container", typed, kubectltree.QueuedKeys(term))
		}
	}
}

func TestCtrlPCtrlQLeavesAfterViPasteWhileTheContainerReadsNoInput(t *testing.T) {
	tree, s := stallTree(t)
	var term *kubectltree.Terminal
	view := termview.New(154, 10, func(b []byte) { term.Type(b) })
	term = kubectltree.NewTerminal(154, 10, view)
	defer view.Close()
	job := tree.StartIn("exec -it web-1 -n shop -- sh", nil, term)
	defer func() {
		s.free()
		job.Stop()
		for deadline := time.Now().Add(15 * time.Second); !job.Done() && time.Now().Before(deadline); {
			time.Sleep(20 * time.Millisecond)
		}
	}()
	until(t, "the terminal is live", term.Live)
	view.Key(termview.Key{Name: "escape"})
	view.Type("P")
	time.Sleep(3 * time.Second)
	view.Key(termview.Key{Rune: 'p', Ctrl: true})
	view.Key(termview.Key{Rune: 'q', Ctrl: true})
	deadline := time.Now().Add(15 * time.Second)
	for !job.Done() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !job.Done() {
		t.Fatalf("after Esc P (vi's paste), ctrl-p ctrl-q did not leave within 15 s while the container reads no input")
	}
}
