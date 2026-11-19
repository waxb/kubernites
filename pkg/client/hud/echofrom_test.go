package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestEchoTextMarksTheButton(t *testing.T) {
	cases := map[[2]string]string{
		{"cordon stone-keep-2", FromKeepCard}:             "> kubectl cordon stone-keep-2  (from the Keep card)",
		{"kubectl get pods", FromChanceryCard}:            "> kubectl get pods  (from the Chancery card)",
		{"etcdctl defrag", FromCodexCard}:                 NodePrompt + "etcdctl defrag  (from the Codex card)",
		{"describe node lone-tower", ""}:                  "> kubectl describe node lone-tower",
		{"etcdctl endpoint status --write-out=table", ""}: NodePrompt + "etcdctl endpoint status --write-out=table",
	}
	for in, want := range cases {
		if got := EchoText(in[0], in[1]); got != want {
			t.Errorf("EchoText(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

func lastStone(h *HUD, n int) []StoneLine {
	lines := h.StoneLines()
	return lines[max(len(lines)-n, 0):]
}

func TestRunFromEchoesCommandAndAnswer(t *testing.T) {
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	res, intents := h.RunFrom("kubectl get pods", FromChanceryCard)
	if !res.OK || runner.commands[0] != "get pods" {
		t.Fatalf("ran %v -> %+v", runner.commands, res)
	}
	lines := lastStone(h, 3)
	if lines[0].Kind != LineEcho || lines[0].Text != "> kubectl get pods  (from the Chancery card)" || lines[1].Kind != LineOutput {
		t.Fatalf("stone = %+v", lines)
	}
	echo, ok := hasIntent(intents, IntentEcho)
	if !ok || echo.Command != "get pods" || echo.Target != "NAME\tSTATUS" || echo.Tab != "ok" {
		t.Fatalf("echo intent = %+v", intents)
	}
	if _, ok := hasIntent(intents, IntentKubectl); !ok {
		t.Error("the realm is polled after a button command")
	}
	_, intents = h.RunFrom("delete pod fail-me", FromHearthCard)
	if echo, _ := hasIntent(intents, IntentEcho); echo.Tab != "error" || lastStone(h, 1)[0].Kind != LineError {
		t.Errorf("a refused command echoes as an error: %+v", intents)
	}
	before := len(h.StoneLines())
	h.EchoAnswer("  \n", true)
	if len(h.StoneLines()) != before {
		t.Error("an empty answer adds no line")
	}
}

func TestActFromCarriesItsSource(t *testing.T) {
	it := ActFrom(view.Action{Action: "openDepot", Kind: "parcel"}, FromCommandCard)
	if it.Kind != IntentAction || it.From != FromCommandCard || EchoSource(it) != FromCommandCard {
		t.Fatalf("intent = %+v", it)
	}
	if EchoSource(Intent{Kind: IntentAction}) != FromButton {
		t.Error("an action without a source comes from a button")
	}
	if !strings.HasPrefix(EchoIntent("kubectl get pods", "", true).Command, "get") {
		t.Error("the tidings echo strips kubectl")
	}
}
