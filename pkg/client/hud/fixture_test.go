package hud

import (
	"encoding/json"
	"image/color"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var allReveals = []string{"serve", "gold", "build", "stone", "tidings", "summon", "wisdom", "berths", "renown",
	"chronicles", "grimoire", "soulvault", "musters", "slo", "ageup", "speed", "scriptorium", "loom", "reforge", "freight", "losses",
	"depot", "augury", "codex-bloat", "hands", "lore"}

type fakeRunner struct {
	commands []string
	applied  []string
}

func (f *fakeRunner) backend() Backend {
	return Backend{Kubectl: f.run, Inspect: f.run, Apply: f.apply}
}

func (f *fakeRunner) apply(yaml string) (string, bool) {
	f.applied = append(f.applied, yaml)
	if strings.Contains(yaml, "broken") {
		return "error: error parsing manifest: yaml: line 3: mapping values are not allowed in this context\n", false
	}
	return "replicaset.apps/porters configured\n", true
}

func (f *fakeRunner) run(cmd string) view.CommandResult {
	f.commands = append(f.commands, cmd)
	if strings.Contains(cmd, "fail") {
		return view.CommandResult{Output: "The connection to the server 127.0.0.1:6443 was refused"}
	}
	return view.CommandResult{Output: "NAME\tSTATUS\nlone-tower\tReady\n", OK: true}
}

var fixtureSite = sync.OnceValue(func() view.Site {
	data, err := os.ReadFile("testdata/fixture-site.json")
	if err != nil {
		panic(err)
	}
	var site view.Site
	if err := json.Unmarshal(data, &site); err != nil {
		panic(err)
	}
	return site
})

func testSite(t *testing.T) view.Site {
	t.Helper()
	s := fixtureSite()
	s.Structures = slices.Clip(s.Structures)
	s.Islands = slices.Clip(s.Islands)
	s.Havens = slices.Clip(s.Havens)
	s.Spots = slices.Clip(s.Spots)
	return s
}

func fullFrame(t *testing.T) Frame {
	t.Helper()
	return Frame{
		State: fixtureState(), Pods: fixturePods(), Site: testSite(t), Clock: 10,
		Events: []Tiding{{Kind: "event", Text: "Started pod"}, {Kind: "calamity", Text: "Tremor!", Warning: true}},
		Metrics: view.MetricsSnapshot{
			Pods:  map[string]view.PodMetrics{"default/porter-a": {CPUMilli: 30, MemoryBytes: 40 << 20, ServedRate: .3}},
			Nodes: map[string]view.NodeMetrics{"lone-tower": {CPUAllocatableMilli: 1900, CPURequestedMilli: 1300, MemoryAllocatableBytes: 3 << 30, MemoryRequestedBytes: 1 << 30}},
		},
		Crews:      []model.CrewChain{{Namespace: "default", Name: "porters", Image: "porter:1.0", Desired: 2, Ready: 1}},
		Vocabulary: testVocabulary,
		Horns:      []model.Horn{{Name: "porters", Target: "porters", Min: 2, Max: 10, Current: 3, Desired: 5, Utilization: 84, Goal: 70, Sounding: true, Measured: true}},
		KeepsReady: 2,
		View:       [4][2]float64{{1, 1}, {10, 1}, {10, 10}, {1, 10}},
	}
}

func fixtureState() view.GameState {
	st := view.GameState{}
	st.Reveals = allReveals
	st.Gold, st.Wisdom, st.Renown, st.GoldPerSec, st.Speed = 1500, 40, 3, 1.5, 1
	st.Age = "dark"
	st.Buildables = []view.Buildable{
		{Kind: "keep", Name: "Keep", K8s: "Node", Description: "A machine", Cost: 10, W: 3, H: 3, Available: true},
		{Kind: "codex", Name: "The Codex", K8s: "etcd", Cost: 15, W: 2, H: 2, Available: false, Reason: "Needs a Keep"},
	}
	st.Soulstones = []view.SoulstoneEntry{
		{Image: "porter:1.0", Name: "Porters, class of 1.0", Description: "They light the way", Cost: 0, Owned: true, Visible: true},
		{Image: "porter:1.1", Name: "Porters, class of 1.1", Description: "Brighter", Cost: 50, Visible: true, Cursed: true},
	}
	st.Grimoire = []view.GrimoirePage{
		{ID: "muster", Name: "Replicas", K8s: "ReplicaSet", Description: "Keep huts running", State: "available", MaxRank: 2,
			Cost: view.Cost{Gold: 60, Wisdom: 10}, Ranks: []view.RankView{{Title: "Novice", Runes: []string{"kubectl get rs"}, Learned: true}, {Title: "Adept"}}},
		{ID: "pledges", Name: "Pledges", K8s: "requests", State: "locked", Requires: []string{"muster"}, MaxRank: 1, Rank: 0},
		{ID: "secret", Name: "Secret", State: "hidden"},
	}
	st.Chronicles = []view.ChronicleView{{ID: "c1", Calamity: "Tremor", K8s: "pod deleted", Question: "Why?", Options: []view.ChronicleOption{{ID: "a", Text: "Chaos"}, {ID: "b", Text: "Nothing"}}}}
	st.AgeUp = view.AgeUpView{NextAge: "feudal", Requirements: []view.Requirement{{Text: "Three hearths", Met: true}, {Text: "A crew"}}}
	st.Berths = view.BerthView{Open: 1, NextCost: 100, Multiplier: 1.25, Effect: "+25% ships for 100 MiB (x1.25 now, x1.56 after)"}
	st.Objectives = []view.ObjectiveView{{ID: "o1", Text: "Serve 5 requests", Hint: "Press serve", Done: false}}
	st.Unlocks = map[string]bool{"pledges": true}
	st.Reforge = view.ReforgeView{Available: true, Earn: 10, Weave: .8, Nines: 2, Threads: 4, Patterns: []view.PatternView{
		{ID: "royal-treasury", Name: "Spare Pool", Description: "+50 MiB", Rank: 1, MaxRank: 3, Cost: 2, Affordable: true},
		{ID: "swift-builders", Name: "Swift Builders", Description: "faster", Rank: 0, MaxRank: 3, Cost: 9},
		{ID: "heraldic-fame", Name: "Heraldic Fame", Description: "more requests", Rank: 3, MaxRank: 3, Cost: 12},
		{ID: "deep-ledger", Name: "Deep Ledger", Description: "The Archive's ledger holds 50% more records per rank and then some more words", Tier: 2, MaxRank: 3, Cost: 4, Locked: true, OpensAt: 12},
		{ID: "ancestral-renown", Name: "Rolling Restart", Description: "+10% value", Tier: 3, Rank: 7, Repeatable: true, Cost: 20, Affordable: false},
	}, RunThreads: 3, NextThreadAt: "2 Keeps", Spent: 7}
	for i := range st.Reforge.Patterns[:3] {
		st.Reforge.Patterns[i].Tier = 1
	}
	st.Freight = fixtureFreight()
	st.Bottleneck, st.Advice = "crew", "Scale the porters: parcels spoil faster than they are carried"
	st.SLO = view.SLOView{Current: .95, Target: .99}
	st.Unlocks["war-horns"] = true
	st.Siege = view.SiegeView{Available: true, Crowd: 2, Hands: 2.4, Spare: 1.9, Target: .99, Reward: 5, Served: .97, SealsGoal: 8, Seals: 1}
	st.Mastery = view.MasteryView{Verbs: []string{"get", "describe", "logs", "top", "explain", "run", "delete", "scale", "label", "apply"},
		Incidents: []view.IncidentView{{Calamity: "A Pod was killed", K8s: "Pod deleted"}}, Declared: 3, DeclaredKinds: []string{"ReplicaSet", "Service"}}
	st.Proclamations = []view.Proclamation{{T: 1, Kind: "reveal", Text: "The Gate opens"}}
	st.Pilgrims = view.PilgrimView{ArrivalRate: .5, ServedRate: .4, Queue: 3, DayPhase: .5, GateOpen: true}
	return st
}

func fixturePods() []model.PodView {
	return []model.PodView{
		{Namespace: "default", Name: "porter-a", Node: "lone-tower", Image: "porter:1.0", Profession: "Porter", Sprite: "porter", Tier: 1, State: model.HearthReady, Restarts: 2},
		{Namespace: "default", Name: "porter-b", Image: "porter:1.0", Profession: "Porter", Sprite: "porter", Tier: 1, State: model.HearthGhost, Reason: "ImagePullBackOff"},
		{Namespace: "kube-system", Name: "etcd-lone-tower", System: true, State: model.HearthReady},
	}
}

func clickOn(t *testing.T, h *HUD, f Frame, id string) []Intent {
	t.Helper()
	for _, b := range h.Buttons(f) {
		if b.ID == id {
			return h.Update(Input{MX: b.Rect.X + 1, MY: b.Rect.Y + 1, Click: true}, f)
		}
	}
	t.Fatalf("button %s not found", id)
	return nil
}

func hasIntent(intents []Intent, kind string) (Intent, bool) {
	for _, it := range intents {
		if it.Kind == kind {
			return it, true
		}
	}
	return Intent{}, false
}

type gameSiege = view.SiegeView

type gamePattern = view.PatternView

func greenToken() color.RGBA { return ui.Palette.Green }

func redTextToken() color.RGBA { return ui.Palette.RedText }

func staged(f Frame) Frame {
	f.State.Age = view.AgeFeudal
	f.State.Reveals = append(append([]string(nil), f.State.Reveals...), allReveals...)
	return f
}
