package hud

import (
	"strings"
	"testing"
)

func TestARevealRunsOffTheGameThreadAndIsNotOfferedWhileASendRuns(t *testing.T) {
	yaml := "apiVersion: v1\nkind: Secret\ndata:\n  password: " + MaskedValue + "\n  tls.crt: " + MaskedValue + "\nmetadata:\n  name: s\n"
	h, f := liveHUD(t, true)
	answer := make(chan Revealed, 1)
	h.backend.Reveal = func(e LiveEdit, key string) <-chan Revealed { return answer }
	h.OpenLiveEdit(LiveEdit{Kind: "Secret", Name: "s", Namespace: "shop", Text: yaml, Hidden: []string{"password", "tls.crt"}})
	h.pressScript("live:reveal:tls.crt", Frame{})
	for _, b := range h.scriptButtons() {
		if strings.HasPrefix(b.ID, EditRevealID) && !b.Disabled {
			t.Errorf("while a reveal waits, %s is offered", b.ID)
		}
	}
	answer <- Revealed{Value: "Y2VydA==", OK: true}
	h.pumpKubectl()
	if !strings.Contains(h.Script.Editor.Text(), "tls.crt: Y2VydA==") {
		t.Fatalf("the answered reveal was not shown:\n%s", h.Script.Editor.Text())
	}
	h.sendLiveEdit(false)
	if len(f.sent) == 0 {
		t.Fatal("SEND did not start")
	}
	for _, b := range h.scriptButtons() {
		if strings.HasPrefix(b.ID, EditRevealID) {
			t.Errorf("while a send runs, %s is offered", b.ID)
		}
	}
	h.pressScript("live:reveal:password", Frame{})
	if strings.Contains(h.Script.Editor.Text(), "password: Y2VydA==") {
		t.Error("a reveal ran while a send runs")
	}
}
