package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func liveNodes() map[string]view.NodeMetrics {
	return map[string]view.NodeMetrics{
		"node-a": {CPURequestedMilli: 1850, CPUAllocatableMilli: 3920, MemoryRequestedBytes: 2 << 30, MemoryAllocatableBytes: 14 << 30, Pods: 21, PodsAllocatable: 110},
		"node-b": {CPURequestedMilli: 650, CPUAllocatableMilli: 3920, MemoryRequestedBytes: 1 << 30, MemoryAllocatableBytes: 14 << 30, Pods: 9, PodsAllocatable: 110},
	}
}

func liveTopHUD(c LiveCard) *HUD {
	h := New(nil, Backend{})
	h.SetLive(c)
	return h
}

func liveStatTexts(h *HUD, f Frame) string {
	out := []string{}
	for _, st := range h.TopStats(f) {
		out = append(out, ui.Name(st.Text))
	}
	return strings.Join(out, " | ")
}

func TestTheLiveTopBarShowsTheClustersNumbersInBothModes(t *testing.T) {
	defer ui.SetNamer(nil)
	f := Frame{Metrics: view.MetricsSnapshot{Nodes: liveNodes()}}
	h := liveTopHUD(LiveCard{Context: "kind-kind", Server: "https://127.0.0.1:6443", Version: "v1.37.0", Seen: "14:05 UTC"})
	story := liveStatTexts(h, f)
	for _, want := range []string{"2.5K/7.8K", "3.1K/29K", "HUTS 30/220"} {
		if !strings.Contains(story, want) {
			t.Errorf("story mode top bar %q lacks %q", story, want)
		}
	}
	icons := map[string]bool{}
	for _, st := range h.TopStats(f) {
		icons[st.icon] = true
	}
	if !icons["food"] || !icons["housing"] {
		t.Errorf("story mode shows the icons %v, want the campaign's food and housing", icons)
	}
	ui.SetNamer(names.Kubernetes(64))
	kube := liveStatTexts(h, f)
	for _, want := range []string{"CPU 2.5/7.84", "MEMORY 3/28Gi", "PODS 30/220"} {
		if !strings.Contains(kube, want) {
			t.Errorf("NAMES mode top bar %q lacks %q", kube, want)
		}
	}
	for _, st := range h.TopStats(f) {
		if st.icon != "" {
			t.Errorf("NAMES mode draws the icon %q beside %q: its words name the resource", st.icon, st.Text)
		}
	}
}

func TestTheLiveTopBarLeavesOutWhatItDoesNotKnow(t *testing.T) {
	defer ui.SetNamer(nil)
	h := liveTopHUD(LiveCard{Context: "kind-kind", Version: "v1.37.0", Seen: "14:05 UTC"})
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		if got := len(h.TopStats(Frame{})); got != 2 {
			t.Errorf("names %v: with no node read the bar shows %d items: %q", ui.Naming(), got, liveStatTexts(h, Frame{}))
		}
		nodes := map[string]view.NodeMetrics{"node-a": {CPURequestedMilli: 100, CPUAllocatableMilli: 2000}}
		text := liveStatTexts(h, Frame{Metrics: view.MetricsSnapshot{Nodes: nodes}})
		for _, absent := range []string{"N/A", NoValue, "HUTS", "PODS", "MEMORY", "/0"} {
			if strings.Contains(text, absent) {
				t.Errorf("names %v: the bar %q shows %q for a number it does not know", ui.Naming(), text, absent)
			}
		}
	}
}

func TestTheLiveTopBarFitsBesideItsItemsAtTheLongestNames(t *testing.T) {
	defer ui.SetNamer(nil)
	cards := []LiveCard{
		{Context: "kind-kind", Version: "v1.37.0", Seen: "14:05 UTC"},
		{Context: strings.Repeat("gke_project-name_europe-west1_", 2), Version: "v1.37.0-gke.1100000", Watching: true, Seen: "14:05 UTC", Level: LevelSeamless},
		{Context: "arn:aws:eks:eu-west-1:123456789012:cluster/staging", Version: "v1.37.2-eks-a1b2c3d", Stale: "14:09 UTC", Level: LevelWrites},
	}
	nodes := map[string]view.NodeMetrics{"n": {CPURequestedMilli: 123456, CPUAllocatableMilli: 999000, MemoryRequestedBytes: 9999 << 30,
		MemoryAllocatableBytes: 99999 << 30, Pods: 9999, PodsAllocatable: 99999}}
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		for _, c := range cards {
			h := liveTopHUD(c)
			f := Frame{Metrics: view.MetricsSnapshot{Nodes: nodes}}
			var runs []ui.TextRun
			stop := ui.RecordText(func(r ui.TextRun) {
				if r.Rect.Y < TopH {
					runs = append(runs, r)
				}
			})
			h.DrawTopBar(ebiten.NewImage(Width, Height), f)
			stop()
			buttons := h.TopButtons(f)
			for i, a := range runs {
				if a.Rect.X < 0 || a.Rect.X+a.Rect.W > Width {
					t.Errorf("names %v, %s: %q at %+v leaves the window", ui.Naming(), c.Context, a.Text, a.Rect)
				}
				for _, b := range runs[i+1:] {
					if Overlaps(a.Rect, b.Rect) {
						t.Errorf("names %v, %s: %q at %+v runs into %q at %+v", ui.Naming(), c.Context, a.Text, a.Rect, b.Text, b.Rect)
					}
				}
				for _, b := range buttons {
					if !strings.Contains(b.Label, a.Text) && Overlaps(a.Rect, b.Rect) {
						t.Errorf("names %v, %s: %q at %+v runs under the button %q", ui.Naming(), c.Context, a.Text, a.Rect, b.Label)
					}
				}
			}
			if len(h.TopStats(f)) < 3 {
				t.Errorf("names %v, %s: no number fits beside the longest names: %q", ui.Naming(), c.Context, liveStatTexts(h, f))
			}
		}
	}
}

func TestALiveNumberTellsWhatItCountsWhenHovered(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	nodes := liveNodes()
	a := nodes["node-a"]
	a.Measured, a.CPUMilli, a.MemoryBytes = true, 420, 3<<30
	nodes["node-a"] = a
	f := Frame{Metrics: view.MetricsSnapshot{Nodes: nodes, Augury: true}}
	h := liveTopHUD(LiveCard{Context: "kind-kind", Version: "v1.37.0", Seen: "14:05 UTC"})
	tips := map[string]string{}
	for _, b := range h.StatButtons(f) {
		tips[b.ID] = h.liveGate(b).Tip
	}
	for id, want := range map[string]string{liveCPUID: "CPU requests 2500m of 7840m allocatable. metrics-server: 420m in use.",
		liveMemoryID: "Memory requests 3Gi of 28Gi allocatable. metrics-server: 3Gi in use.", livePodsID: "30 Pods on the nodes, of 220 the nodes allow"} {
		if !strings.Contains(tips[id], want) {
			t.Errorf("%s tells %q, want %q", id, tips[id], want)
		}
	}
}

func TestALiveNumbersTipSpeaksStoryWordsInStoryMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(nil)
	nodes := liveNodes()
	a := nodes["node-a"]
	a.Measured, a.CPUMilli, a.MemoryBytes = true, 420, 3<<30
	nodes["node-a"] = a
	f := Frame{Metrics: view.MetricsSnapshot{Nodes: nodes, Augury: true}}
	h := liveTopHUD(LiveCard{Context: "kind-kind", Version: "v1.37.0", Seen: "14:05 UTC"})
	text := ""
	for _, st := range LiveNumbers(f) {
		text += st.Tip + "|"
	}
	for _, kube := range []string{"CPU", "Memory", "memory", "allocatable", "Succeeded", "Failed", "Pod", "node", "scheduler", "metrics-server", "requests"} {
		if strings.Contains(text, kube) {
			t.Errorf("the story-mode tips say the Kubernetes word %q: %q", kube, text)
		}
	}
	for _, want := range []string{"2,500 of 7,840 rations pledged", "pledged", "Augur", "huts", "places"} {
		if !strings.Contains(text, want) {
			t.Errorf("the story-mode tips %q lack %q", text, want)
		}
	}
	if len(h.TopStats(f)) != 5 {
		t.Errorf("the bar shows %q", liveStatTexts(h, f))
	}
}
