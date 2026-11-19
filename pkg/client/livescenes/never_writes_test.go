package livescenes

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/write"
)

func writesOf(lines []string) []string {
	var out []string
	for _, l := range lines {
		if strings.HasPrefix(l, "GET ") || strings.Contains(l, "selfsubjectaccessreviews") || strings.Contains(l, "selfsubjectreviews") {
			continue
		}
		out = append(out, l)
	}
	return out
}

func TestTheGameNeverWritesOnItsOwn(t *testing.T) {
	defer app.ApplyNames(false)
	seen := &seenRequests{}
	g := app.NewGame(nil, app.Options{LiveOnly: true}, nil, nil)
	g.Settings.LiveNames = storyNames()
	s, r := openWrites(g, writesSetup{level: write.Seamless, classes: write.AllClasses, cluster: fakeCluster{seen: seen}})
	snap := r.Snapshot()
	idle := hud.Input{MX: -10, MY: -10}
	step := func(in hud.Input) {
		s.Update(in, app.TickSeconds, &app.Pointer{}, true)
	}
	for _, tr := range Troubles {
		if err := Play(r, tr.Break(snap)); err != nil {
			t.Fatal(err)
		}
		for range int((hud.CalamitySettle + 2) / app.TickSeconds) {
			step(idle)
		}
	}
	for range int(300 / app.TickSeconds) {
		step(idle)
	}
	for _, tr := range Troubles {
		if err := Play(r, tr.Mend(snap)); err != nil {
			t.Fatal(err)
		}
	}
	for range int((live.MendSettle + 2) / app.TickSeconds) {
		step(idle)
	}
	selectCrew(s)
	for k := ebiten.KeyA; k <= ebiten.KeyZ; k++ {
		step(hud.Input{MX: -10, MY: -10, Keys: []ebiten.Key{k}})
		step(hud.Input{MX: -10, MY: -10, Ctrl: true, Keys: []ebiten.Key{k}})
		step(hud.Input{MX: -10, MY: -10, Shift: true, Keys: []ebiten.Key{k}})
	}
	for k := ebiten.Key0; k <= ebiten.Key9; k++ {
		step(hud.Input{MX: -10, MY: -10, Keys: []ebiten.Key{k}})
	}
	PressButton(s, "top:menu")
	PressButton(s, "menu:hints")
	PressButton(s, "top:menu")
	PressButton(s, "menu:resume")
	s.Hud.OpenStone("")
	s.Hud.FocusStone()
	for _, ch := range "kubectl delete pod storefront --force" {
		step(hud.Input{MX: -10, MY: -10, Chars: []rune{ch}, Control: ch == 'k' || ch == 'd', Keys: []ebiten.Key{ebiten.KeyK, ebiten.KeyD}})
	}
	for range 60 {
		step(idle)
	}
	if w := writesOf(seen.all()); len(w) != 0 {
		t.Fatalf("Kubernites wrote on its own: %v", w)
	}
	if len(seen.all()) == 0 {
		t.Fatal("the fake cluster saw nothing at all: the test is not wired")
	}
}

func TestWithHintsOffNoCardShowsACommandAndNoOrderEchoesOne(t *testing.T) {
	defer app.ApplyNames(false)
	for _, hints := range []bool{false, true} {
		g := app.NewGame(nil, app.Options{LiveOnly: true}, nil, nil)
		g.Settings.LiveNames = storyNames()
		s, r := openWrites(g, writesSetup{level: write.Seamless, classes: write.AllClasses, hints: hints})
		snap := r.Snapshot()
		if err := Play(r, TroubleNamed("crashloop").Break(snap)); err != nil {
			t.Fatal(err)
		}
		Settle(s, hud.CalamitySettle+1)
		var chips []string
		for _, b := range s.Hud.Buttons(s.Frame(true)) {
			if strings.HasPrefix(b.ID, "line:cal:") || hud.IsLiveOrder(b.ID) && strings.Contains(b.ID, "|") {
				chips = append(chips, b.Label)
			}
		}
		commands := 0
		for _, c := range chips {
			if strings.Contains(c, "logs ") || strings.Contains(c, "rollout") || strings.Contains(c, "kubectl") {
				commands++
			}
		}
		if len(chips) != 2 || hints && commands != 2 || !hints && commands != 0 {
			t.Errorf("HINTS %v: the card's chips %q", hints, chips)
		}
		selectCrew(s)
		order(s, "restart")
		scroll(s, hud.ScrollGo)
		echoed := false
		for _, e := range s.Echoes {
			echoed = echoed || strings.HasPrefix(e.Text, "$ ")
		}
		if echoed != hints {
			t.Errorf("HINTS %v: the order echoed a command: %v", hints, echoed)
		}
	}
}
