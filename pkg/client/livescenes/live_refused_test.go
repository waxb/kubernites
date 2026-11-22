package livescenes

import (
	"strings"
	"testing"

	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func refusedScaleTip(t *testing.T, kube bool) string {
	t.Helper()
	app.ApplyNames(kube)
	g := app.NewGame(nil, app.Options{LiveOnly: true}, nil, nil)
	g.Settings.LiveNames = &kube
	in := app.CaptureScenes["live-refused"](g)
	for range 10 {
		g.Sess.Update(in, app.TickSeconds, &app.Pointer{}, true)
	}
	for _, o := range g.Sess.Hud.LiveOrders() {
		if o.Key == "scale" && o.Disabled {
			return ui.Name(o.Tip)
		}
	}
	t.Fatalf("kubernetes names %v: no refused scale order", kube)
	return ""
}

func TestARefusalShowsTheServersExactWordsOnlyUnderKubernetesNames(t *testing.T) {
	defer app.ApplyNames(false)
	if tip := refusedScaleTip(t, true); !strings.Contains(tip, refusalReason) {
		t.Errorf("with kubernetes names the tip changes the server's words: %q", tip)
	}
	tip := refusedScaleTip(t, false)
	for _, word := range []string{"RBAC", "API group", "/scale", "viewer@example.com"} {
		if strings.Contains(tip, word) {
			t.Errorf("the story tip carries %q: %q", word, tip)
		}
	}
	if !strings.Contains(tip, "they may not scale it") {
		t.Errorf("the story tip does not say what is refused: %q", tip)
	}
}

func TestANoPreviewNoteCarriesTheServersWordsOnlyUnderKubernetesNames(t *testing.T) {
	defer app.ApplyNames(false)
	for _, kube := range []bool{true, false} {
		app.ApplyNames(kube)
		g := app.NewGame(nil, app.Options{LiveOnly: true}, nil, nil)
		g.Settings.LiveNames = &kube
		in := app.CaptureScenes["live-class-unpreviewed"](g)
		for range 10 {
			g.Sess.Update(in, app.TickSeconds, &app.Pointer{}, true)
		}
		f, ok := g.Sess.Hud.OrderOpen()
		if !ok {
			t.Fatalf("kubernetes names %v: no order open", kube)
		}
		note := ui.Name(f.Note)
		if has := strings.Contains(note, k8sWebhookRefusal()); has != kube {
			t.Errorf("kubernetes names %v: the note carries the server's words %v: %q", kube, has, note)
		}
	}
}
