package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/termview"
)

type screen struct {
	live  bool
	typed string
}

func (s *screen) Live() bool               { return s.live }
func (s *screen) Resize(int, int)          {}
func (s *screen) Row(int) []termview.Cell  { return nil }
func (s *screen) Cursor() (int, int, bool) { return 0, 0, false }
func (s *screen) Key(termview.Key)         {}
func (s *screen) Type(text string)         { s.typed += text }

type termJob struct {
	fakeJob
	screen *screen
}

func (j *termJob) Screen() StoneScreen { return j.screen }

func TestKeysTypedAfterAContainerEndsItsSessionNeverRunInTheStone(t *testing.T) {
	h, f := liveHUD(t, true)
	sc := &screen{live: true}
	job := &termJob{screen: sc}
	h.StartStoneJob(job)
	frame := func(in Input) {
		h.keys(in, Frame{})
		h.pumpKubectl()
	}
	for _, r := range "kubectl " {
		frame(Input{Chars: []rune{r}})
	}
	if sc.typed != "kubectl " {
		t.Fatalf("the container's terminal got %q, want what was typed", sc.typed)
	}
	sc.live = false
	job.finish(StoneEnd{OK: true, Sessions: []StoneSession{{Verb: "exec", Kind: "Pod", Namespace: "shop", Name: "web-1", Opened: true, TTY: true, Ended: SessionExited}}})
	frame(Input{})
	for _, r := range "delete namespace shop" {
		frame(Input{Chars: []rune{r}})
	}
	frame(Input{Keys: []ebiten.Key{ebiten.KeyEnter}})
	if len(f.started) > 0 {
		t.Errorf("the container ended its session while a line was being typed into it, and the rest of that line ran in the Stone as %q, with the connected user's credentials and no hand-back", f.started)
	}
}
