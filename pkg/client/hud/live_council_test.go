package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var liveReveals = []string{"stone", "tidings", "heraldry", CouncilReveal}

func liveCouncilFacts() view.LiveCouncil {
	return view.LiveCouncil{NodesReady: 2, Nodes: 3, PodsReady: 40, PodsActive: 44, WorkloadsFull: 25, Workloads: 27, ServicesServed: 30, Services: 31,
		Pending:  []view.LivePending{{Namespace: "shop", Name: "web-7c9-x2", Reason: "0/3 nodes are available: 3 Insufficient cpu."}, {Namespace: "shop", Name: "fresh"}},
		Short:    []view.LiveWorkload{{Kind: "StatefulSet", Namespace: "shop", Name: "db", Ready: 1, Desired: 3}, {Kind: "Deployment", Namespace: "shop", Name: "web", Ready: 2, Desired: 3}},
		Unserved: []view.LiveService{{Namespace: "shop", Name: "lonely"}},
		Busiest:  []view.LivePodUsage{{Namespace: "shop", Name: "api-5d-b2", CPUMilli: 250, MemoryBytes: 10 << 20}}}
}

func liveCouncilFrame(measured bool) (*HUD, Frame) {
	h := liveTopHUD(LiveCard{Context: "kind-kind", Version: "v1.37.0", Seen: "14:05 UTC", Council: liveCouncilFacts()})
	f := liveKeepFrame(measured)
	f.Live = false
	f.State.Reveals = slices.Clone(liveReveals)
	f.Site.Structures = []view.Structure{liveKeep(liveWorker, "e2-standard-4"), liveKeep(liveTower, "")}
	f.Site.Structures[1].ID = "keep-2"
	return h, f
}

func TestTheDashboardOpensInLiveFromTheWellAndItsKey(t *testing.T) {
	h, f := liveCouncilFrame(false)
	var well *Button
	for _, b := range h.Buttons(f) {
		if b.ID == "cmd:council" {
			well = &b
		}
	}
	if well == nil || well.Disabled {
		t.Fatalf("the live well offers no Dashboard tab: %+v", well)
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyC}}, f)
	if h.Overlay() != OverlayCouncil {
		t.Fatalf("C opened %q, want the Dashboard", h.Overlay())
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyC}}, f)
	if h.Overlay() == OverlayCouncil {
		t.Fatal("C did not close the Dashboard again")
	}
}

func TestTheLiveDashboardOffersNoActionAndOnlyTheLiveTabs(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		h, f := liveCouncilFrame(true)
		h.OpenCouncil("")
		tabs := []string{}
		for _, b := range h.Buttons(f) {
			if !strings.HasPrefix(b.ID, CouncilPrefix+":") && !strings.HasPrefix(b.ID, "info:dial:") && b.ID != "overlay:close" {
				continue
			}
			if b.Disabled {
				t.Errorf("names %v: the Dashboard's button %s is greyed out in live: %q", ui.Naming(), b.ID, b.Tip)
			}
			kind, rest, _ := strings.Cut(strings.TrimPrefix(b.ID, CouncilPrefix+":"), ":")
			switch {
			case b.ID == "overlay:close" || strings.HasPrefix(b.ID, "info:dial:"):
			case kind == "tab":
				tabs = append(tabs, rest)
			default:
				t.Errorf("names %v: the live Dashboard offers %s (%q)", ui.Naming(), b.ID, b.Label)
			}
			if strings.HasPrefix(b.Label, "DO IT") || strings.HasPrefix(b.ID, lensPrefix) {
				t.Errorf("names %v: the live Dashboard offers %q", ui.Naming(), b.Label)
			}
		}
		if want := []string{TabQuartermaster, TabStewards, TabWayfinder, TabAugur}; !slices.Equal(tabs, want) {
			t.Errorf("names %v: the live Dashboard's tabs are %v, want %v", ui.Naming(), tabs, want)
		}
		for _, tab := range tabs {
			for _, it := range h.Press(CouncilPrefix+":tab:"+tab, f) {
				if !LiveIntent(it) || it.Kind == IntentAction {
					t.Errorf("the tab %s asks %+v of a live cluster", tab, it)
				}
			}
		}
	}
}

func TestTheLiveDashboardsGaugesHaveLiveDefinitions(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	_, f := liveCouncilFrame(false)
	f.Live, f.LiveCouncil = true, liveCouncilFacts()
	got := map[string]string{}
	for _, d := range RatingDials(f) {
		got[d.Label] = DialText(d)
		if !d.Known || d.Tip == "" {
			t.Errorf("the gauge %s is shown without a value or a definition: %+v", d.Label, d)
		}
	}
	want := map[string]string{"NODES": "2/3", "PODS": "40/44", "WORKLOADS": "25/27", "SERVICES": "30/31"}
	if len(got) != len(want) {
		t.Errorf("the live gauges are %v, want %v", got, want)
	}
	for label, text := range want {
		if got[label] != text {
			t.Errorf("the %s gauge reads %q, want %q", label, got[label], text)
		}
	}
	f.LiveCouncil.Services, f.LiveCouncil.ServicesServed, f.LiveCouncil.Unread = 0, 0, []string{"services"}
	for _, d := range RatingDials(f) {
		if d.Label == "SERVICES" {
			t.Errorf("the Services gauge shows %q while the cluster refused the list of Services", DialText(d))
		}
	}
}

func councilText(p CouncilPage) string {
	out := []string{ui.Name(p.Line)}
	for _, s := range p.Sections {
		out = append(out, ui.Name(s.Title), ui.Name(s.Empty))
		for _, h := range s.Head {
			out = append(out, ui.Name(h))
		}
		for _, r := range s.Rows {
			for _, c := range r {
				out = append(out, ui.Name(c.Text))
			}
		}
	}
	return strings.Join(out, "|")
}

func TestTheLiveDashboardsPagesSayOnlyWhatALiveClusterKnows(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	_, f := liveCouncilFrame(true)
	f.Live, f.LiveCouncil = true, liveCouncilFacts()
	f.Metrics.Augury = true
	pages := map[string][]string{
		TabQuartermaster: {"2 Pods are Pending", "1.9 / 3.9", "0.42", "21/110", "shop/web-7c9-x2", "0/3 nodes are available: 3 Insufficient cpu.", "shop/fresh", "no reason from the scheduler yet"},
		TabStewards:      {"shop/db", "StatefulSet", "1 of 3 ready", "shop/web", "Deployment", "2 of 3 ready"},
		TabWayfinder:     {"shop/lonely", "no endpoint"},
		TabAugur:         {"shop/api-5d-b2", "250m", "10Mi", "EPHEMERAL STORAGE", liveWorker},
	}
	for tab, wants := range pages {
		text := councilText(CouncilPageFor(tab, f))
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Errorf("the live %s page %q lacks %q", tab, text, want)
			}
		}
		for _, game := range []string{"SERVED/S", "HARBOR", "boxes", "CODEX", "Codex", "RANK", "WAR HORN", "money", "Money", "upkeep", "Favor", "Goodwill"} {
			if strings.Contains(text, game) {
				t.Errorf("the live %s page %q shows the game's %q", tab, text, game)
			}
		}
	}
	quiet := f
	quiet.LiveCouncil = view.LiveCouncil{NodesReady: 3, Nodes: 3, PodsReady: 9, PodsActive: 9, WorkloadsFull: 2, Workloads: 2, ServicesServed: 1, Services: 1}
	for _, tab := range []string{TabQuartermaster, TabStewards, TabWayfinder} {
		p := CouncilPageFor(tab, quiet)
		for _, s := range p.Sections {
			if len(s.Rows) == 0 && s.Empty == "" {
				t.Errorf("the quiet %s page's %s section shows nothing and does not say so", tab, s.Title)
			}
		}
	}
	if line := Counsel(quiet); !strings.Contains(line, "Every node is Ready") {
		t.Errorf("a quiet cluster's summary reads %q", line)
	}
	if line := Counsel(f); !strings.Contains(line, "2 Pods are Pending") {
		t.Errorf("the summary of a cluster with Pending Pods reads %q", line)
	}
}

func TestTheLiveDashboardInStoryModeSpeaksTheLexicon(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(nil)
	_, f := liveCouncilFrame(true)
	f.Live, f.LiveCouncil = true, liveCouncilFacts()
	f.Metrics.Augury = true
	text := Counsel(f)
	for _, d := range RatingDials(f) {
		text += "|" + d.Label + "|" + d.Tip
	}
	for _, tab := range []string{TabQuartermaster, TabStewards, TabWayfinder, TabAugur} {
		text += "|" + councilText(CouncilPageFor(tab, f)) + "|" + liveAdvisorTip(tab)
	}
	for _, game := range []string{"harbor", "ship", "Codex", "Rank", "War Horn"} {
		if strings.Contains(text, game) {
			t.Errorf("the live Dashboard names the game's %q: %q", game, text)
		}
	}
	if !strings.Contains(text, "switch NAMES to read why") {
		t.Errorf("the story-mode Dashboard does not say where the scheduler's reason is: %q", text)
	}
	for _, kube := range []string{"Pending", "Deployment", "StatefulSet", "DaemonSet", "Service", "endpoint", "replicas", "Node", "PODS", "NODES", "metrics-server", "Insufficient cpu"} {
		if strings.Contains(text, kube) {
			t.Errorf("the story-mode Dashboard says the Kubernetes word %q: %q", kube, text)
		}
	}
}

func TestTheLiveNodesTableLeavesOutUsageMetricsServerDoesNotGive(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	_, f := liveCouncilFrame(false)
	f.Live, f.LiveCouncil = true, liveCouncilFacts()
	nodes := CouncilPageFor(TabQuartermaster, f).Sections[0]
	text := councilText(CouncilPage{Sections: []Section{nodes}})
	for _, absent := range []string{"USED", NoValue} {
		if strings.Contains(text, absent) {
			t.Errorf("without metrics-server the nodes table shows %q: %q", absent, text)
		}
	}
	for _, row := range nodes.Rows {
		if len(row) != len(nodes.Cols) || len(nodes.Head) != len(nodes.Cols) {
			t.Errorf("the table's columns %v, heads %v and cells %d disagree", nodes.Cols, nodes.Head, len(row))
		}
	}
}

func TestTheLiveBusiestHutsCountFoodInRationsInStoryMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(nil)
	_, f := liveCouncilFrame(true)
	f.Live, f.LiveCouncil = true, liveCouncilFacts()
	f.Metrics.Augury = true
	busiest := CouncilPageFor(TabAugur, f).Sections[0]
	if len(busiest.Rows) == 0 || busiest.Rows[0][1].Text != "250" {
		t.Errorf("the busiest hut's food reads %+v, want 250 rations as the top bar and the Keep card count them", busiest.Rows)
	}
}
