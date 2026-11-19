package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func keysOf(actions []CardAction) []string {
	out := []string{}
	for _, a := range actions {
		out = append(out, a.Key)
	}
	return out
}

func stoneKeep() view.Structure {
	return view.Structure{ID: "keep-2", Kind: view.StructKeep, Variant: "stone", State: view.StateReady, Label: "Stone Keep 2",
		Object: view.ObjectRef{Kind: "Node", Name: "stone-keep-2"}}
}

func selectStructure(f Frame, s view.Structure) Frame {
	f.Selection = Selection{Kind: "structure", ID: s.ID, Structure: s}
	return f
}

func TestKeepCardOffersEveryNodeVerb(t *testing.T) {
	f := selectStructure(staged(fullFrame(t)), stoneKeep())
	acts := SelectionActions(f)
	if !slices.Equal(keysOf(acts), []string{KeyDescribe, KeyCordon, KeyDrain, KeyTaint, KeyLabel, KeyTop}) {
		t.Fatalf("keep actions = %v", keysOf(acts))
	}
	if acts[0].Command != "describe node stone-keep-2" || acts[1].Command != "cordon stone-keep-2" || !acts[2].Disabled || acts[5].Command != "top node stone-keep-2" {
		t.Fatalf("keep commands = %+v", acts)
	}
	if !acts[5].Disabled || acts[5].Tip != NoMetricsTip {
		t.Errorf("TOP waits for the Omens: %+v", acts[5])
	}
	f.Metrics.Augury = true
	f.Nodes = map[string]model.NodeView{"stone-keep-2": {Unschedulable: true}}
	f.State.Grimoire = append(f.State.Grimoire, view.GrimoirePage{ID: "masonry", Ranks: []view.RankView{{Learned: true, Runes: []string{"drain"}}}})
	acts = SelectionActions(f)
	if acts[1].Key != KeyUncordon || acts[1].Command != "uncordon stone-keep-2" || acts[2].Disabled || acts[5].Disabled {
		t.Fatalf("cordoned keep with the drain rune = %+v", acts)
	}
	if CardSource(f) != FromKeepCard {
		t.Errorf("source = %q", CardSource(f))
	}
}

func TestTaintChooserSetsAndLiftsWards(t *testing.T) {
	n := model.NodeView{TaintSpecs: []string{"ward=true:NoSchedule", "node-role.kubernetes.io/control-plane:NoSchedule", "node.kubernetes.io/unreachable:NoExecute"}}
	got := TaintChoices("lone-tower", n)
	cmds := []string{}
	for _, c := range got {
		cmds = append(cmds, c.Command)
	}
	want := []string{
		"taint node lone-tower ward=true:NoSchedule", "taint node lone-tower ward=true:PreferNoSchedule", "taint node lone-tower ward=true:NoExecute",
		"taint node lone-tower ward:NoSchedule-", "taint node lone-tower node-role.kubernetes.io/control-plane:NoSchedule-",
	}
	if !slices.Equal(cmds, want) {
		t.Fatalf("taint choices = %v", cmds)
	}
	if !got[0].Disabled || got[1].Disabled {
		t.Errorf("a Ward that stands cannot be set twice: %+v", got[:2])
	}
	if !strings.Contains(got[4].Label, "Royal Ward") || !strings.Contains(got[3].Label, "the Ward") {
		t.Errorf("lift labels = %q %q", got[3].Label, got[4].Label)
	}
}

func TestLabelChoosersForKeepsAndHearths(t *testing.T) {
	keep := NodeLabelChoices("stone-keep-2", model.NodeView{Labels: map[string]string{"kubernetes.io/hostname": "stone-keep-2", "tier": "front"}})
	if len(keep) != 2 || keep[0].Command != "label node stone-keep-2 disktype=ssd" || keep[1].Command != "label node stone-keep-2 tier-" {
		t.Fatalf("keep label choices = %+v", keep)
	}
	if got := NodeLabelChoices("n", model.NodeView{Labels: map[string]string{"disktype": "ssd"}}); len(got) != 1 || got[0].Command != "label node n disktype-" {
		t.Fatalf("an ssd keep = %+v", got)
	}
	crewed := PodLabelChoices(model.PodView{Name: "porters-a", Image: "porter:1.0", Labels: map[string]string{"app": "porters"}}, "")
	if len(crewed) != 2 || crewed[0].Command != "label pod porters-a app-" || crewed[1].Command != "label pod porters-a debug=true" {
		t.Fatalf("crewed hearth choices = %+v", crewed)
	}
	loose := PodLabelChoices(model.PodView{Name: "porters-a", Image: "porter:1.0", Labels: map[string]string{"debug": "true"}}, " -n side")
	if len(loose) != 2 || loose[0].Command != "label pod porters-a -n side app=porters" || loose[1].Command != "label pod porters-a -n side debug-" {
		t.Fatalf("a quarantined hearth = %+v", loose)
	}
	if got := PodLabelChoices(model.PodView{Name: "x", Image: "mystery:9"}, ""); len(got) != 1 {
		t.Fatalf("an unknown trade = %+v", got)
	}
}

func TestHearthCardOffersEveryPodVerb(t *testing.T) {
	p := model.PodView{Namespace: "default", Name: "porters-a", Image: "porter:1.0", Restarts: 2, Labels: map[string]string{"app": "porters"}}
	acts := HearthActions(p)
	if !slices.Equal(keysOf(acts), []string{KeyDescribe, KeyLogs, KeyDelete, KeyLabel, KeyPrevious, KeyYAML}) {
		t.Fatalf("hearth actions = %v", keysOf(acts))
	}
	if acts[4].Disabled || acts[4].Command != "logs porters-a --previous" || acts[5].Command != "get pod porters-a -o yaml" {
		t.Fatalf("restarted hearth = %+v", acts)
	}
	p.Restarts = 0
	if acts = HearthActions(p); !acts[4].Disabled {
		t.Error("PREVIOUS waits for a restart")
	}
	door := HearthActions(model.PodView{Namespace: "default", Name: "porter-lone-tower", Static: true})
	if door[2].Disabled || door[2].Tip != StaticDelete {
		t.Errorf("the door's DELETE explains the static Pod: %+v", door[2])
	}
	sys := HearthActions(model.PodView{Namespace: "kube-system", Name: "etcd-lone-tower", System: true})
	if !sys[2].Disabled || !sys[3].Disabled || sys[0].Command != "describe pod etcd-lone-tower -n kube-system" {
		t.Errorf("citadel hearth = %+v", sys)
	}
}

func TestDepotCodexChanceryAndObservatoryCards(t *testing.T) {
	f := fullFrame(t)
	hall := view.Structure{ID: "guildhall-1", Kind: view.StructGuildhall, State: view.StateReady, Object: view.ObjectRef{Kind: "Service", Namespace: "default", Name: "parcels"}}
	f = selectStructure(f, hall)
	acts := SelectionActions(f)
	if len(acts) != 3 || acts[1].Command != "get endpointslices -l kubernetes.io/service-name=parcels" || acts[2].Choices[0].Command != "delete service parcels" || CardSource(f) != FromDepotCard {
		t.Fatalf("depot actions = %+v", acts)
	}
	f.State.Codex = view.CodexView{Revision: 900, CompactRevision: 400}
	f = selectStructure(f, view.Structure{ID: "codex-1", Kind: view.StructCodex, State: view.StateReady})
	acts = SelectionActions(f)
	if !slices.Equal(keysOf(acts), []string{KeyStatus, KeyAlarms, KeyCompact, KeyDefrag, KeyDisarm, KeyEtcdVer}) || acts[2].Command != "etcdctl compact 900" || acts[2].Disabled || !acts[4].Disabled {
		t.Fatalf("codex actions = %+v", acts)
	}
	f.State.Codex = view.CodexView{Revision: 900, CompactRevision: 900, Alarm: true}
	acts = SelectionActions(f)
	if !slices.Equal(keysOf(acts)[:2], []string{KeyDefrag, KeyDisarm}) || acts[1].Disabled || !acts[4].Disabled || CardSource(f) != FromCodexCard {
		t.Fatalf("an alarmed codex puts DEFRAG and DISARM first: %+v", acts)
	}
	f = selectStructure(f, view.Structure{ID: "chancery-1", Kind: view.StructChancery, State: view.StateReady})
	if acts = SelectionActions(f); len(acts) != 6 || acts[0].Command != "get pods" || acts[5].Command != "api-resources" || CardSource(f) != FromChanceryCard {
		t.Fatalf("chancery actions = %+v", acts)
	}
	gets := acts[1].Choices
	if len(gets) != len(ListedKinds) || gets[0].Command != "get nodes" || gets[len(gets)-1].Command != "get pods -A" || gets[5].Details == "" {
		t.Fatalf("get choices = %+v", gets)
	}
	f = selectStructure(f, view.Structure{ID: "observatory-1", Kind: view.StructObservatory, State: view.StateReady})
	if acts = SelectionActions(f); len(acts) != 2 || !acts[0].Disabled || CardSource(f) != FromObservatoryCard {
		t.Fatalf("silent observatory = %+v", acts)
	}
	f.Metrics.Augury = true
	if acts = SelectionActions(f); acts[0].Disabled || acts[1].Command != "top nodes" {
		t.Fatalf("observatory = %+v", acts)
	}
	f = selectStructure(f, view.Structure{ID: "treasury-1", Kind: view.StructTreasury, State: view.StateReady})
	if len(SelectionActions(f)) != 0 {
		t.Error("this structure speaks no kubectl")
	}
	f = selectStructure(f, view.Structure{ID: "codex-2", Kind: view.StructCodex, State: view.StateConstruction})
	if len(SelectionActions(f)) != 0 {
		t.Error("a Codex under construction answers nothing")
	}
}

func TestKeepCellsFitTheGridAndNameTheirCommand(t *testing.T) {
	f := selectStructure(staged(navFrame(t)), stoneKeep())
	h := New(nil, (&fakeRunner{}).backend())
	buttons := h.CommandButtons(f)
	if len(buttons) != 7 {
		t.Fatalf("buttons = %v", gridIDs(h, f))
	}
	for i, b := range buttons {
		if !inside(b.Rect, Command) {
			t.Errorf("%s %+v leaves the command frame", b.ID, b.Rect)
		}
		if strings.HasPrefix(b.ID, "act:") && !strings.Contains(b.Tip, "\n") {
			t.Errorf("button %+v", b)
		}
		if i > 0 && Overlaps(b.Rect, buttons[i-1].Rect) {
			t.Errorf("%s overlaps %s", b.ID, buttons[i-1].ID)
		}
		if b.ID == "sel:details" {
			t.Error("DESCRIBE replaces DETAILS")
		}
	}
}

func TestTheGridKeepsDetailsUntilTheStoneSpeaks(t *testing.T) {
	f := fullFrame(t)
	f.Selection = Selection{Kind: "hearth", ID: "default/porter-a", Pod: f.Pods[0]}
	f.State.Reveals = []string{"gold"}
	h := New(nil, (&fakeRunner{}).backend())
	wantCells(t, h, f, map[string]int{"sel:details": 0, "sel:logs": 4, GridHome: 8})
	f = selectStructure(fullFrame(t), view.Structure{ID: "codex-1", Kind: view.StructCodex, State: view.StateReady, Object: view.ObjectRef{Kind: "Pod", Namespace: "kube-system", Name: "etcd-lone-tower"}})
	wantCells(t, h, f, map[string]int{"sel:details": 0})
}

func TestCodexBarLabelIsVisible(t *testing.T) {
	lines := CodexLines(model.Codex{SizeBytes: 100, InUseBytes: 50, QuotaBytes: 1000})
	if len(lines) == 0 || !lines[0].Bar || lines[0].Clr.A == 0 {
		t.Fatalf("the CODEX label draws in a transparent colour: %+v", lines)
	}
}

func TestDrainSendsTheRealmsDrainAndShowsTheKeepsOwnFlags(t *testing.T) {
	f := selectStructure(staged(fullFrame(t)), stoneKeep())
	f.State.Grimoire = append(f.State.Grimoire, view.GrimoirePage{ID: "masonry", Ranks: []view.RankView{{Learned: true, Runes: []string{"drain"}}}})
	f.State.Keeps = nil
	drain, ok := ActionByKey(SelectionActions(f), KeyDrain)
	if !ok || drain.Act != (view.Action{Action: view.ActDrainKeep, ID: "stone-keep-2"}) || drain.Command != "drain stone-keep-2 --ignore-daemonsets" {
		t.Fatalf("DRAIN = %+v", drain)
	}
	forced := "kubectl drain stone-keep-2 --ignore-daemonsets --delete-emptydir-data --force"
	f.State.Keeps = []view.KeepView{{Name: "stone-keep-3", DrainCommand: "kubectl drain stone-keep-3 --ignore-daemonsets --force"}, {Name: "stone-keep-2", DrainCommand: forced}}
	drain, _ = ActionByKey(SelectionActions(f), KeyDrain)
	if drain.Command != strings.TrimPrefix(forced, "kubectl ") {
		t.Fatalf("the Keep's own flags are lost: %q", drain.Command)
	}
	if tip := ActionTip(drain); !strings.Contains(tip, "--delete-emptydir-data --force") {
		t.Fatalf("the tip hides the flags: %q", tip)
	}
	f.State.Keeps = f.State.Keeps[:1]
	if drain, _ = ActionByKey(SelectionActions(f), KeyDrain); strings.Contains(drain.Command, "--force") {
		t.Fatalf("another Keep lends its flags: %q", drain.Command)
	}
	intents := clickOn(t, New(nil, (&fakeRunner{}).backend()), f, "act:drain")
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != view.ActDrainKeep || it.Action.ID != "stone-keep-2" {
		t.Fatalf("DRAIN press = %+v", intents)
	}
	if _, ok := hasIntent(intents, IntentKubectl); ok {
		t.Error("the card speaks drain itself instead of asking the realm")
	}
}
