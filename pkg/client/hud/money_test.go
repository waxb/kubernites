package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestMoneyReadsAsDollarsWithKubernetesNames(t *testing.T) {
	defer ui.SetNamer(nil)
	cases := []struct {
		got        func() string
		story, k8s string
	}{
		{func() string { return Gold(150) }, ui.IconGold + "150", "$150"},
		{func() string { return Gold(1500) }, ui.IconGold + "1.5K", "$1.5K"},
		{func() string { return Money("-12") }, "-" + ui.IconGold + "12", "-$12"},
		{func() string { return GoldRate(0.3) }, "+0.3 GOLD/S", "+$0.3/S"},
		{func() string { return GoldWord(2000) }, "2K GOLD", "$2K"},
		{func() string { return BoxGold(12.4) }, ui.IconGold + "12", "$12"},
		{func() string { return BoxGold(1.25) }, ui.IconGold + "1.2", "$1.2"},
		{func() string { return BoxGold(2500) }, "2.5K", "$2.5K"},
	}
	for _, c := range cases {
		ui.SetNamer(nil)
		if got := ui.Name(c.got()); got != c.story {
			t.Errorf("story = %q, want %q", got, c.story)
		}
		ui.SetNamer(names.Kubernetes(64))
		if got := ui.Name(c.got()); got != c.k8s {
			t.Errorf("kubernetes = %q, want %q", got, c.k8s)
		}
	}
}

func TestHandsReadAsMillicoresWithKubernetesNames(t *testing.T) {
	defer ui.SetNamer(nil)
	if got := Hands(1300); got != "1.30H" {
		t.Errorf("story hands = %q", got)
	}
	ui.SetNamer(names.Kubernetes(64))
	if got := ui.Name(LaborLine(view.PodMetrics{CPUMilli: 130, MemoryBytes: 40 << 20})); got != "CPU 130m  MEMORY 40Mi  0.0/S" {
		t.Errorf("kubernetes labor line = %q", got)
	}
	card := PodCard(model.PodView{Name: "porter-a", Profession: "Porter", Node: "lone-tower", Image: "porter:1.0"}, view.PodMetrics{})
	for _, l := range card.Lines {
		if strings.Contains(ui.Name(l.Text), "KEEP") {
			t.Errorf("a story word survives: %q", ui.Name(l.Text))
		}
	}
}

func TestRegistryButtonsSpeakKubernetes(t *testing.T) {
	defer ui.SetNamer(nil)
	h := New(nil, Backend{})
	f := fullFrame(t)
	ui.SetNamer(names.Kubernetes(64))
	labels := map[string]bool{}
	for _, b := range h.AcademyButtons(f) {
		labels[ui.Name(b.Label)] = true
	}
	for _, want := range []string{"BOUGHT", "BUY"} {
		if !labels[want] {
			t.Errorf("no %q among %v", want, labels)
		}
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	if runs := recordRuns(func() { h.DrawAcademy(dst, f) }); !hasRun(runs, "$50") {
		t.Errorf("the Registry's amount column does not say $50: %+v", runs)
	}
	ui.SetNamer(nil)
	for _, b := range h.AcademyButtons(f) {
		if b.Label == "BOUGHT" || strings.Contains(b.Label, "$") {
			t.Errorf("story label %q", b.Label)
		}
	}
}
