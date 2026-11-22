package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestAScriptoriumSendStoppedWhileItsHiddenValuesAreReadNeverApplies(t *testing.T) {
	for _, level := range []string{hud.LevelWrites, hud.LevelSeamless} {
		t.Run(level, func(t *testing.T) {
			addr, _, seen := shellFront(t)
			rig := connectStoneTo(t, level, addr)
			rig.gate.Pass(write.Secrets)
			read, release := make(chan struct{}), make(chan struct{})
			saved := unmaskSend
			unmaskSend = func(ctx context.Context, tr treeRealm, e hud.LiveEdit, text string) (string, error) {
				close(read)
				<-release
				return strings.Replace(text, hud.MaskedValue, "aHVudGVyMg==", 1), nil
			}
			t.Cleanup(func() { unmaskSend = saved })
			b := rig.s.treeBackend(hud.Backend{})
			text := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: s\n  namespace: parity\ndata:\n  password: " + hud.MaskedValue + "\n"
			job, _ := b.Send(hud.LiveEdit{Kind: "Secret", Namespace: "parity", Name: "s", Format: "yaml", Text: text, Hidden: []string{"password"}}, text, false)
			t.Cleanup(job.Stop)
			select {
			case <-read:
			case <-time.After(10 * time.Second):
				t.Fatal("the send never began to read its hidden values")
			}
			job.Stop()
			close(release)
			applied := func() []string {
				var out []string
				for _, s := range seen.all() {
					if !strings.HasPrefix(s, "GET ") && strings.Contains(s, "/secrets") {
						out = append(out, s)
					}
				}
				return out
			}
			for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(10 * time.Millisecond) {
				if sent := applied(); len(sent) > 0 {
					t.Fatalf("a send STOP ended while its hidden values were read went on to apply the Secret at %s: %v", level, sent)
				}
				if _, done := job.Take(); done {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("a stopped send never ended")
				}
			}
			end := job.End()
			if !end.Stopped {
				t.Errorf("a send stopped while its hidden values were read ends as %+v, not as stopped", end)
			}
		})
	}
}
