package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func freshLabels() {
	storyLabel = lexicon.StoryLabeler(labelMemory)
}

func drawnText(runs []ui.TextRun) string {
	out := []string{}
	for _, r := range runs {
		out = append(out, r.Text)
	}
	return strings.Join(out, " | ")
}

func TestStoryLabelsGlossATermOnlyWhereItIsFirstIntroduced(t *testing.T) {
	defer ui.SetNamer(nil)
	freshLabels()
	ui.SetNamer(nil)
	first, later := "Builders begin The Codex (etcd).", "The Codex (etcd) swells past half its quota."
	if got := ToastLabel(Toast{Text: first}); got != first {
		t.Fatalf("the first introduction lost its gloss: %q", got)
	}
	if got := ToastLabel(Toast{Text: later, Count: 2}); got != "The Codex swells past half its quota. (x2)" {
		t.Fatalf("a later label kept the gloss: %q", got)
	}
	if got := GlossTip("The Codex", "etcd", "The one book of truth."); !strings.Contains(got, "(etcd)") {
		t.Fatalf("a tooltip lost its gloss: %q", got)
	}
	ui.SetNamer(names.Kubernetes(64))
	if got := Label(later); got != later {
		t.Fatalf("KUBERNETES mode label policy touched %q: %q", later, got)
	}
}

func TestTidingRowsDropRepeatedGlossesButTheirCardKeepsThem(t *testing.T) {
	defer ui.SetNamer(nil)
	freshLabels()
	ui.SetNamer(nil)
	Label("Your porters wait Unhoused (Pending).")
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.Events = []Tiding{{Kind: "lesson", Text: "Clerks wait Unhoused (Pending): no Keep has room.", Title: "Unhoused", Detail: "Clerks wait Unhoused (Pending): no Keep has room."}}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	drawn := drawnText(recordRuns(func() { h.DrawTidings(dst, f, WellBody()) }))
	if strings.Contains(drawn, "(Pending)") || !strings.Contains(drawn, "Clerks wait Unhoused") {
		t.Fatalf("tiding row = %s", drawn)
	}
	h.PressTiding("0", f)
	if !strings.Contains(h.LessonCard.Text, "(Pending)") {
		t.Fatalf("the tiding's card lost its gloss: %+v", h.LessonCard)
	}
}
