package app

import (
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
)

func hutOf(t *testing.T, rig *liveRig) (hud.Selection, map[string]hud.LiveOrder) {
	t.Helper()
	for _, sel := range rig.selections() {
		if sel.Kind != hud.SelectHearth {
			continue
		}
		orders := map[string]hud.LiveOrder{}
		for _, o := range rig.select_(sel) {
			orders[o.Key] = o
		}
		return sel, orders
	}
	t.Fatal("no hut to select")
	return hud.Selection{}, nil
}

func TestTheHutCardsReachOrdersRunTheirCommandsInTheStone(t *testing.T) {
	addr, shell, _ := shellFront(t)
	rig := connectStoneTo(t, hud.LevelSeamless, addr)
	s := rig.s
	sel, orders := hutOf(t, rig)
	want := map[string][4]string{
		"logs": {"LOGS", "V", "4", "0"}, "shell": {"SHELL", "Y", "3", "0"}, "more": {"MORE", "", "7", "0"},
		"attach": {"ATTACH", "J", "0", "1"}, "port-forward": {"FORWARD", "F", "1", "1"},
	}
	for key, w := range want {
		o, ok := orders[key]
		if !ok || o.Label != w[0] || o.Chord != w[1] || itoa(o.Cell) != w[2] || itoa(o.Page) != w[3] {
			t.Errorf("the hut card's %s is %+v, want %v", key, o, w)
		}
	}
	cells := map[string]int{}
	for _, b := range s.Hud.Buttons(s.Frame(true)) {
		for cell := range hud.GridCells {
			if hud.CellRect(cell) == b.Rect {
				cells[b.ID] = cell
			}
		}
	}
	if cells[hud.GridHome] != hud.BackCell || cells[hud.LiveOrderPrefix+":more"] != 7 {
		t.Errorf("the hut card's first page lost BACK in cell 9 or MORE is not beside it: %v", cells)
	}
	pod := sel.Pod
	rig.key(hud.Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyY}})
	frames(t, s, "the workloads warning", func() bool { a, ok := s.Hud.StoneAsking(); return ok && strings.Contains(a.Title, "REACHING") })
	seenLong(s)
	stoneFrame(s, ebiten.KeyY)
	frames(t, s, "the shell is live", s.Hud.StoneTerminal)
	out := strings.Join(s.Hud.StoneOutput(), "\n")
	if !strings.Contains(out, "exec -it "+pod.Name+" -n "+pod.Namespace+" -- sh -c") {
		t.Fatalf("CTRL-Y did not open the Stone with the shell's command:\n%s", out)
	}
	s.Update(hud.Input{MX: -10, MY: -10, Chars: []rune("exit")}, 0, &Pointer{}, true)
	stoneFrame(s, ebiten.KeyEnter)
	stoneEnds(t, s)
	_, urls, _, _ := shell.Seen()
	if len(urls) != 1 || !strings.Contains(urls[0], "/pods/"+pod.Name+"/exec?") || !strings.Contains(urls[0], "command=sh&command=-c&command=command+-v+bash") {
		t.Fatalf("the cluster saw %v", urls)
	}

	rig.select_(sel)
	rig.key(hud.Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyJ}})
	if s.Hud.StoneRunning() {
		t.Fatal("CTRL-J fired while the Stone held the keyboard")
	}
	closeStone(t, rig)
	rig.key(hud.Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyJ}})
	frames(t, s, "the attach is live", s.Hud.StoneTerminal)
	s.Update(hud.Input{MX: -10, MY: -10, Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyP}}, 0, &Pointer{}, true)
	s.Update(hud.Input{MX: -10, MY: -10, Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyQ}}, 0, &Pointer{}, true)
	stoneEnds(t, s)
	if out := strings.Join(s.Hud.StoneOutput(), "\n"); !strings.Contains(out, "attach -it "+pod.Name+" -n "+pod.Namespace) || !strings.Contains(out, "with ctrl-p ctrl-q") {
		t.Fatalf("CTRL-J did not attach:\n%s", out)
	}

	closeStone(t, rig)
	rig.select_(sel)
	rig.key(hud.Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyV}})
	frames(t, s, "the logs follow", s.Hud.StoneRunning)
	if out := strings.Join(s.Hud.StoneOutput(), "\n"); !strings.Contains(out, "logs -f "+pod.Name+" -n "+pod.Namespace) {
		t.Fatalf("CTRL-V did not follow the logs:\n%s", out)
	}
	stoneFrame(s, ebiten.KeyEscape)
	stoneEnds(t, s)

	closeStone(t, rig)
	rig.select_(sel)
	rig.key(hud.Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyF}})
	if form, open := s.Hud.OrderOpen(); !open || form.Go != "FORWARD [Y]" || s.Hud.StoneRunning() {
		t.Fatalf("CTRL-F at DANGEROUSLY SEAMLESS should ask once: %+v open %v", form, open)
	}
	for range hud.AskSeenFrames {
		rig.key(hud.Input{MX: -10, MY: -10})
	}
	rig.key(hud.Input{MX: -10, MY: -10, Keys: []ebiten.Key{ebiten.KeyY}})
	if spec, _ := s.spec("port-forward"); spec.stoneRun {
		stoneEnds(t, s)
		frames(t, s, "the forward is listed", func() bool { return len(s.Hud.LiveCard().Forwards) == 1 })
	} else if s.Hud.StoneRunning() || !strings.HasPrefix(string(s.Hud.Stone.Input), "port-forward "+pod.Name+" -n "+pod.Namespace) {
		t.Fatalf("a hut with no port should wait for one: running %v, input %q", s.Hud.StoneRunning(), string(s.Hud.Stone.Input))
	}
}

func TestAtReadOnlyTheHutCardOffersOnlyItsLogs(t *testing.T) {
	addr, shell, _ := shellFront(t)
	rig := connectStoneTo(t, hud.LevelReadOnly, addr)
	_, orders := hutOf(t, rig)
	for _, key := range []string{"shell", "attach", "port-forward"} {
		if o := orders[key]; !o.Disabled || o.Why != readOnlyWhy {
			t.Errorf("%s at READ ONLY: %+v", key, o)
		}
	}
	if orders["logs"].Disabled {
		t.Fatalf("LOGS at READ ONLY: %+v", orders["logs"])
	}
	for _, k := range []ebiten.Key{ebiten.KeyY, ebiten.KeyJ, ebiten.KeyF} {
		rig.key(hud.Input{Ctrl: true, Control: true, Keys: []ebiten.Key{k}})
	}
	if rig.s.Hud.StoneRunning() {
		t.Fatal("a greyed reach order ran")
	}
	if _, urls, _, _ := shell.Seen(); len(urls) != 0 {
		t.Fatalf("READ ONLY reached into a workload: %v", urls)
	}
}

func closeStone(t *testing.T, rig *liveRig) {
	t.Helper()
	rig.key(hud.Input{Keys: []ebiten.Key{ebiten.KeyEscape}})
	if rig.s.Hud.Overlay() == hud.OverlayStone {
		t.Fatalf("Esc left the Stone open")
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func TestAWorkersShellAttachAndLogsReachThatWorker(t *testing.T) {
	addr, _, _ := shellFront(t)
	rig := connectStoneTo(t, hud.LevelSeamless, addr)
	sel, _ := hutOf(t, rig)
	if len(sel.Pod.People) == 0 {
		t.Fatal("the hut has no worker")
	}
	worker := sel.Pod.People[len(sel.Pod.People)-1].Name
	sel.Kind, sel.Pod = "walker", sel.Pod.Focused(worker)
	rig.s.Selection = sel
	seen := map[string]string{}
	for _, sp := range rig.s.selectedSpecs() {
		seen[sp.key] = sp.stone
	}
	for _, key := range []string{"shell", "attach", "logs"} {
		if !strings.Contains(seen[key], " -c "+worker) {
			t.Errorf("the worker's %s runs %q, not in worker %s", key, seen[key], worker)
		}
	}
	sel.Kind, sel.Pod = hud.SelectHearth, sel.Pod.Focused("")
	rig.s.Selection = sel
	for _, sp := range rig.s.selectedSpecs() {
		if before, _, _ := strings.Cut(sp.stone, " -- "); (sp.key == "shell" || sp.key == "attach" || sp.key == "logs") && strings.Contains(before, " -c ") {
			t.Errorf("the hut's %s names a worker: %q", sp.key, sp.stone)
		}
	}
}
