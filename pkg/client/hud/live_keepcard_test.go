package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	liveWorker = "gke-staging-pool-1-a1b2"
	liveTower  = "kind-control-plane"
)

func liveKeepFrame(measured bool) Frame {
	worker := view.NodeMetrics{CPURequestedMilli: 1850, CPUAllocatableMilli: 3920, MemoryRequestedBytes: 3 << 30, MemoryAllocatableBytes: 14 << 30,
		Pods: 21, PodsAllocatable: 110, EphemeralRequestedBytes: 1 << 30, EphemeralAllocatableBytes: 20 << 30}
	tower := view.NodeMetrics{CPURequestedMilli: 950, CPUAllocatableMilli: 4000, MemoryRequestedBytes: 290 << 20, MemoryAllocatableBytes: 15 << 30,
		Pods: 9, PodsAllocatable: 110, ControlPlaneCPUMilli: 650, ControlPlaneMemoryBytes: 100 << 20}
	if measured {
		worker.Measured, worker.CPUMilli, worker.MemoryBytes = true, 420, 3174<<20
		tower.Measured, tower.CPUMilli, tower.MemoryBytes = true, 310, 900<<20
	}
	return Frame{Live: true, Metrics: view.MetricsSnapshot{Augury: measured, Nodes: map[string]view.NodeMetrics{liveWorker: worker, liveTower: tower}},
		Nodes: map[string]model.NodeView{
			liveWorker: {Name: liveWorker, SystemCPUMilli: 300, Labels: map[string]string{"node.kubernetes.io/instance-type": "e2-standard-4", "topology.kubernetes.io/zone": "europe-west1-b"}},
			liveTower:  {Name: liveTower, SystemCPUMilli: 950, Labels: map[string]string{"node-role.kubernetes.io/control-plane": ""}},
		}}
}

func liveKeep(node, instance string) view.Structure {
	s := view.Structure{ID: "keep-1", Kind: view.StructKeep, Label: "Keep " + node, State: view.StateReady, Instance: instance, CPU: 4, MemoryGiB: 14, Territory: 9}
	s.Object.Kind, s.Object.Name = "Node", node
	return s
}

func liveCardTexts(c Card) []string {
	out := []string{}
	for _, l := range c.Lines {
		out = append(out, ui.Name(l.Text)+" "+l.Suffix)
	}
	return out
}

func liveCardRows(t *testing.T, c Card) []Line {
	t.Helper()
	h := New(nil, Backend{})
	w, room := liveCardRoom()
	shown := h.FitLines(c.Lines, w, room)
	if rows := h.LineRows(shown, w); rows > room {
		t.Errorf("names %v: the card needs %d rows, it has %d: %q", ui.Naming(), rows, room, liveCardTexts(c))
	}
	return shown
}

func liveCardRoom() (int, int) {
	box := PortraitBox()
	return CardBox.X + CardBox.W - box.X - box.W - 14, (cardFloor() - (CardBox.Y + 6 + 2*ui.LineHeight)) / ui.LineHeight
}

func TestALiveWorkerCardShowsRequestsOfAllocatableAndItsLabels(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	f := liveKeepFrame(false)
	c := StructureCard(liveKeep(liveWorker, "e2-standard-4"), f)
	bars := map[string]Line{}
	for _, l := range c.Lines {
		if l.Bar {
			bars[strings.TrimSpace(ui.Name(l.Text))] = l
		}
	}
	for label, want := range map[string]struct {
		suffix string
		value  float64
	}{"CPU": {"1.85/3.92", 1850.0 / 3920}, "MEMORY": {"3/14Gi", 3.0 / 14}, "PODS": {"21/110", 21.0 / 110}} {
		b, ok := bars[label]
		if !ok || b.Suffix != want.suffix || b.Value != want.value {
			t.Errorf("the %s bar reads %+v, want %s (%.3f)", label, b, want.suffix, want.value)
		}
	}
	text := strings.Join(liveCardTexts(c), "|")
	for _, want := range []string{"requests of allocatable", "e2-standard-4 in zone europe-west1-b"} {
		if !strings.Contains(text, want) {
			t.Errorf("the live worker card %q lacks %q", text, want)
		}
	}
	for _, absent := range []string{"in use", "reach", "land", "holds", "N/A", NoValue} {
		if strings.Contains(text, absent) {
			t.Errorf("the live worker card %q says %q, which is not true of this node", text, absent)
		}
	}
	if shown := strings.Join(liveCardTexts(Card{Lines: liveCardRows(t, c)}), "|"); !strings.Contains(shown, "europe-west1-b") {
		t.Errorf("the card has room for its zone, yet shows only %q", shown)
	}
}

func TestALiveCardShowsUsageOnlyWhenMetricsServerAnswers(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		f := liveKeepFrame(true)
		for _, node := range []string{liveWorker, liveTower} {
			c := StructureCard(liveKeep(node, ""), f)
			text := strings.Join(liveCardTexts(c), "|")
			want := map[bool]string{false: "eaten", true: "in use"}[ui.Naming()]
			if !strings.Contains(text, want) {
				t.Errorf("names %v: %s's card %q does not say what metrics-server measures", ui.Naming(), node, text)
			}
			liveCardRows(t, c)
		}
	}
}

func TestALiveControlPlaneCardSaysWhatTheControlPlaneHolds(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	c := StructureCard(liveKeep(liveTower, ""), liveKeepFrame(false))
	text := strings.Join(liveCardTexts(c), "|")
	if !strings.Contains(text, "Control plane holds 650m, 100Mi") {
		t.Errorf("the control-plane card %q does not say what the control plane holds", text)
	}
	for _, l := range c.Lines {
		if l.Bar && strings.TrimSpace(ui.Name(l.Text)) == "CPU" && l.reserved != 650.0/4000 {
			t.Errorf("the CPU bar marks %.3f for the control plane, want 650m of 4000m", l.reserved)
		}
	}
	w := StructureCard(liveKeep(liveWorker, ""), liveKeepFrame(false))
	if text := strings.Join(liveCardTexts(w), "|"); strings.Contains(text, "holds") {
		t.Errorf("a worker running kube-proxy and the CNI says %q: they are not the control plane", text)
	}
}

func TestALiveCardInStoryModeSpeaksTheCampaignsWords(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(nil)
	c := StructureCard(liveKeep(liveWorker, "e2-standard-4"), liveKeepFrame(false))
	text := strings.Join(liveCardTexts(c), "|")
	for _, want := range []string{"FOOD", "HOUSING", "HUTS", "1,850/3,920", "pledged of available", "isle europe-west1-b"} {
		if !strings.Contains(text, want) {
			t.Errorf("the story card %q lacks %q", text, want)
		}
	}
	for _, kube := range []string{"CPU", "MEMORY", "PODS", "zone", "requests"} {
		if strings.Contains(text, kube) {
			t.Errorf("the story card %q says the Kubernetes word %q", text, kube)
		}
	}
	if shown := strings.Join(liveCardTexts(Card{Lines: liveCardRows(t, c)}), "|"); !strings.Contains(shown, "europe-west1-b") {
		t.Errorf("the card has room for its isle, yet shows only %q", shown)
	}
}

func TestTheCampaignKeepCardShowsBothBarsAndItsSizeOnOneRow(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		f, s := towerFrame(t)
		f.State.Reveals = []string{"hands"}
		s.Instance, s.CPU, s.MemoryGiB, s.Class, s.Territory = "general-large", 2, 4, "balanced", 9
		c := StructureCard(s, f)
		labels := []string{}
		for _, l := range c.Lines {
			if l.Bar {
				labels = append(labels, strings.TrimSpace(ui.Name(l.Text)))
			}
		}
		if len(labels) != 2 {
			t.Errorf("names %v: the Keep card shows the bars %v, want food and housing both, as its size line names both", ui.Naming(), labels)
		}
		w, _ := liveCardRoom()
		h := New(nil, Backend{})
		for _, l := range liveCardRows(t, c) {
			if !l.Bar && len(h.Font.WrapLines(l.Text, w, 1, 9)) > 1 {
				t.Errorf("names %v: %q wraps onto a second row", ui.Naming(), ui.Name(l.Text))
			}
		}
	}
}
