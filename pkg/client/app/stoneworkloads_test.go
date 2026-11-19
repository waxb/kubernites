package app

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/write"
)

const stoneProxy = "get --raw /api/v1/namespaces/parity/services/web:80/proxy/"

func TestTheStoneAsksTheWorkloadsWarningOnceThenShowsWhatAConnectionOpens(t *testing.T) {
	rig, seen := connectStone(t, hud.LevelWrites)
	s := rig.s
	a := askedOnTheStone(t, s, stoneProxy)
	label, _ := a.GoLabel()
	if !strings.Contains(a.Title, "REACHING INTO WORKLOADS") || label != "I UNDERSTAND [Y]" {
		t.Fatalf("the Stone asked %+v", a)
	}
	stoneFrame(s, ebiten.KeyY)
	var confirm hud.StoneAsk
	for i := 0; i < 2000; i++ {
		stoneFrame(s)
		if c, ok := s.Hud.StoneAsking(); ok && c.Title != a.Title {
			confirm = c
			break
		}
	}
	text := strings.Join(confirm.Lines, "\n")
	if !strings.Contains(confirm.Title, "CONFIRM") || !strings.Contains(text, "Opens: the proxy of service parity/web:80, path /") || !strings.Contains(text, "No dry run") {
		t.Fatalf("the CONFIRM shows %q: %q", confirm.Title, text)
	}
	seenLong(s)
	stoneFrame(s, ebiten.KeyY)
	stoneEnds(t, s)
	if n := seen.count("GET /api/v1/namespaces/parity/services/web:80/proxy/"); n != 1 {
		t.Fatalf("the cluster saw %d proxies", n)
	}
	for _, r := range seen.all() {
		if strings.Contains(r, "/proxy") && strings.Contains(r, "dryRun") {
			t.Fatalf("a connection was previewed: %s", r)
		}
	}
	rig.frame()
	mem, err := write.LoadMemory(rig.dir)
	if err != nil {
		t.Fatal(err)
	}
	if r, _, ok := mem.Recall(rig.gate.Context()); !ok || !slices.Contains(r.Classes, write.Workloads) {
		t.Fatalf("remembered %+v %v", r, ok)
	}
	again := askedOnTheStone(t, s, stoneProxy)
	if strings.Contains(again.Title, "REACHING") || !strings.Contains(again.Title, "CONFIRM") {
		t.Fatalf("the warning was asked twice: %+v", again)
	}
	stoneFrame(s, ebiten.KeyEscape)
	stoneEnds(t, s)
	rig.g.forgetChoice(rig.gate.Context().Name)
	mem, err = write.LoadMemory(rig.dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := mem.Recall(rig.gate.Context()); ok {
		t.Fatal("RESET LIVE CHOICES kept the context and its warnings")
	}
}

func TestAConnectionsConfirmSaysWhatItOpensInStoryAndNames(t *testing.T) {
	asks := []live.StoneAsk{
		{Why: write.NeedsConfirm, Connect: true, Command: "kubectl exec -it web-1 -n shop -c app -- sh", Opens: write.Opening{Channel: "exec", Kind: "Pod", Namespace: "shop", Name: "web-1", Container: "app", Command: []string{"sh"}, Stdin: true, TTY: true}},
		{Why: write.NeedsConfirm, Connect: true, Command: "kubectl attach -it web-1 -n shop", Opens: write.Opening{Channel: "attach", Kind: "Pod", Namespace: "shop", Name: "web-1", Container: "app", Stdin: true, TTY: true}},
		{Why: write.NeedsConfirm, Connect: true, Command: "kubectl port-forward web-1 -n shop 8080:80", Opens: write.Opening{Channel: "portforward", Kind: "Pod", Namespace: "shop", Name: "web-1"}},
		{Why: write.NeedsConfirm, Connect: true, Command: "kubectl get --raw /api/v1/nodes/kind-worker/proxy/configz", Opens: write.Opening{Channel: "proxy", Kind: "Node", Name: "kind-worker", Path: "/configz"}},
	}
	want := [][2]string{
		{"Opens: a shell in hut shop/web-1, worker app", "Opens: exec in pod shop/web-1, container app"},
		{"Opens: attach to hut shop/web-1, worker app", "Opens: attach to pod shop/web-1, container app"},
		{"Opens: a port-forward to hut shop/web-1, ports 8080:80", "Opens: port-forward to pod shop/web-1, ports 8080:80"},
		{"Opens: the proxy of Keep kind-worker, path /configz", "Opens: the proxy of node kind-worker, path /configz"},
	}
	defer ui.SetNamer(nil)
	for i, a := range asks {
		for mode := range 2 {
			if mode == 0 {
				ui.SetStoryNamer(lexicon.StoryText)
			} else {
				ui.SetNamer(names.Kubernetes(64))
			}
			r := RenderAsk(a)
			text := strings.Join(r.Lines, "\n")
			if !r.Confirm || !strings.Contains(text, want[i][mode]) {
				t.Errorf("%s (mode %d): %q, want %q", a.Command, mode, text, want[i][mode])
			}
			if strings.Contains(text, "Dry run (dryRun=All)") || strings.Contains(text, "preview answered") {
				t.Errorf("%s: a connection shows a preview: %q", a.Command, text)
			}
		}
	}
}

func TestResetLiveChoicesForgetsTheWarningsPassedOnTheOpenConnection(t *testing.T) {
	for _, it := range []hud.Intent{{Kind: hud.IntentLiveChoices, Command: "forget-all"}, {Kind: hud.IntentLiveChoices, Command: "forget"}} {
		rig, _ := connectStone(t, hud.LevelWrites)
		s := rig.s
		if it.Command == "forget" {
			it.Target = rig.gate.Context().Name
		}
		a := askedOnTheStone(t, s, stoneProxy)
		if !strings.Contains(a.Title, "REACHING") {
			t.Fatalf("the Stone asked %+v", a)
		}
		stoneFrame(s, ebiten.KeyY)
		for i := 0; i < 2000; i++ {
			stoneFrame(s)
			if c, ok := s.Hud.StoneAsking(); ok && c.Title != a.Title {
				break
			}
		}
		stoneFrame(s, ebiten.KeyEscape)
		stoneEnds(t, s)
		if !rig.gate.Passed(write.Workloads) {
			t.Fatal("the warning was not passed")
		}
		if err := rig.g.RealmIntent(it); err != nil {
			t.Fatal(err)
		}
		if got := rig.gate.Classes(); len(got) != 0 {
			t.Fatalf("%s: the open connection still holds the warnings %v", it.Command, got)
		}
		again := askedOnTheStone(t, s, stoneProxy)
		if !strings.Contains(again.Title, "REACHING") {
			t.Fatalf("%s: after RESET the Stone asked %+v", it.Command, again)
		}
		stoneFrame(s, ebiten.KeyEscape)
		stoneEnds(t, s)
	}
}
