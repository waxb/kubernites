package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestTheLiveWarningsSayReachingIntoAWorkloadWarnsOnceAndWhatAConnectionShows(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, mode := range []string{"story", "names"} {
		if mode == "story" {
			ui.SetStoryNamer(func(s string) string { return lexicon.StoryText(lexicon.StoryNames(s)) })
		} else {
			ui.SetNamer(names.Kubernetes(64))
		}
		for _, step := range []string{LiveStepWrites, LiveStepSeam} {
			p := confirmPicker(nil, false)
			p.Changed = "Changed since WRITES was chosen: its API server was https://127.0.0.1:38817. The remembered choice is dropped."
			p.Step = step
			var runs []ui.TextRun
			stop := ui.RecordText(func(r ui.TextRun) { runs = append(runs, r) })
			dst := ebiten.NewImage(Width, Height)
			p.Draw(dst)
			stop()
			dst.Deallocate()
			top := Height
			for _, b := range p.Buttons() {
				top = min(top, b.Rect.Y)
			}
			var words []string
			for _, r := range runs {
				if r.Rect.Y >= top {
					continue
				}
				words = append(words, r.Text)
				if r.Rect.X < LiveBox.X || r.Rect.X+r.Rect.W > LiveBox.X+LiveBox.W-8 || r.Rect.Y+r.Rect.H > top {
					t.Errorf("[%s %s] %q at %+v leaves the box %+v above the buttons at y %d", mode, step, r.Text, r.Rect, LiveBox, top)
				}
			}
			told, opens := false, false
			text := strings.Join(words, " ")
			if strings.Contains(text, "not part of Kubernites yet") || strings.Contains(text, "later round") {
				t.Errorf("[%s %s] the warning still says reaching into a workload is not part of Kubernites: %q", mode, step, text)
			}
			for _, sentence := range strings.SplitAfter(text, ". ") {
				if strings.Contains(strings.ToLower(sentence), "workload") && strings.Contains(sentence, "warn") {
					for _, door := range []string{"exec", "attach", "port-forward", "cp", "debug", "proxy"} {
						if !strings.Contains(sentence, door) {
							t.Errorf("[%s %s] the workloads warning does not name %s: %q", mode, step, door, sentence)
						}
					}
					told = true
				}
				opens = opens || strings.Contains(sentence, "CONFIRM shows what it opens")
			}
			if !told {
				t.Errorf("[%s %s] the warning does not say reaching into a workload warns once: %q", mode, step, words)
			}
			if step == LiveStepWrites && !opens {
				t.Errorf("[%s %s] WRITES does not say a connection's CONFIRM shows what it opens: %q", mode, step, words)
			}
		}
	}
}
