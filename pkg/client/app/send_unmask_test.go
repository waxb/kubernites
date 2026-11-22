package app

import (
	"context"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func TestASendWithHiddenValuesNeverWaitsOnTheGameThread(t *testing.T) {
	addr, _, _ := shellFront(t)
	rig := connectStoneTo(t, hud.LevelWrites, addr)
	tr, ok := liveTreeOf(rig.s.Realm)
	if !ok {
		t.Fatal("no tree")
	}
	held := make(chan struct{})
	busy := tr.Tree().Start("delete pod shell-1 -n parity --wait=false", func(ctx context.Context, h kubectltree.Held) bool {
		select {
		case <-held:
		default:
			close(held)
		}
		<-ctx.Done()
		return false
	})
	defer busy.Stop()
	select {
	case <-held:
	case <-time.After(20 * time.Second):
		t.Fatal("the first send never reached its CONFIRM")
	}
	b := rig.s.treeBackend(hud.Backend{})
	text := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: s\n  namespace: parity\ndata:\n  password: " + hud.MaskedValue + "\n"
	sent := make(chan hud.StoneJob, 1)
	go func() {
		job, _ := b.Send(hud.LiveEdit{Kind: "Secret", Namespace: "parity", Name: "s", Format: "yaml", Text: text, Hidden: []string{"password"}}, text, false)
		sent <- job
	}()
	var job hud.StoneJob
	select {
	case job = <-sent:
	case <-time.After(time.Second):
		t.Fatal("SEND of a Secret with a hidden value waited on the game thread behind another Stone command")
	}
	revealed := make(chan hud.Revealed, 1)
	go func() {
		revealed <- <-b.Reveal(hud.LiveEdit{Kind: "Secret", Namespace: "parity", Name: "s"}, "password")
	}()
	if _, done := job.Take(); done {
		t.Fatal("the send ended before its hidden values were read again")
	}
	job.Stop()
	busy.Stop()
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		if _, done := job.Take(); done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("STOP did not end a send waiting to read its hidden values")
		}
	}
	select {
	case <-revealed:
	case <-time.After(10 * time.Second):
		t.Fatal("a REVEAL never answered")
	}
}
