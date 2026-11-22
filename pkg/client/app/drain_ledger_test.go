package app

import (
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
	"github.com/waxb/kubernites/pkg/view"
)

func TestADrainRecordsItsCordonAndALineWhenItEndsOrIsStopped(t *testing.T) {
	for _, how := range []string{"ending", "STOP", "leaving the realm", "connecting another context", "closing the window"} {
		t.Run(how, func(t *testing.T) {
			rig := connectAt(t, hud.LevelSeamless, livetest.Options{Answer: func(r *http.Request, _ []byte) (int, string, bool) {
				if strings.HasSuffix(r.URL.Path, "/eviction") {
					if how == "ending" {
						return http.StatusCreated, `{"kind":"Status","code":201}`, true
					}
					return http.StatusTooManyRequests, `{"kind":"Status","code":429,"message":"Cannot evict pod as it would violate the pod's disruption budget."}`, true
				}
				return 0, "", false
			}}, write.AllClasses...)
			w := rig.s.Writer
			w.sync = false
			w.every = 20 * time.Millisecond
			node := rig.snap.Nodes[len(rig.snap.Nodes)-1].Name
			i := slices.IndexFunc(rig.s.Site.Structures, func(st view.Structure) bool { return st.Kind == view.StructKeep && st.Object.Name == node })
			rig.select_(hud.Selection{Kind: "structure", ID: rig.s.Site.Structures[i].ID, Structure: rig.s.Site.Structures[i]})
			for i := 0; i < 400 && slices.ContainsFunc(rig.s.Hud.LiveOrders(), func(o hud.LiveOrder) bool { return o.Why == askingWhy }); i++ {
				time.Sleep(5 * time.Millisecond)
				rig.frame()
			}
			rig.goAll("drain", 1)
			for deadline := time.Now().Add(10 * time.Second); len(evictionsSeen(rig.rec)) == 0; time.Sleep(time.Millisecond) {
				if time.Now().After(deadline) {
					t.Fatal("the drain never sent an eviction")
				}
			}
			left := rig.s
			switch how {
			case "ending":
				for deadline := time.Now().Add(10 * time.Second); left.Writer.drain == nil || !left.Writer.drain.over; time.Sleep(5 * time.Millisecond) {
					if time.Now().After(deadline) {
						t.Fatal("the drain never ended")
					}
					rig.frame()
				}
			case "STOP":
				rig.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollStop})
				for deadline := time.Now().Add(10 * time.Second); !left.Writer.drain.over; time.Sleep(5 * time.Millisecond) {
					if time.Now().After(deadline) {
						t.Fatal("STOP never ended the drain")
					}
					rig.frame()
				}
			case "leaving the realm":
				if err := rig.g.RealmIntent(hud.Intent{Kind: hud.IntentTitle}); err != nil {
					t.Fatal(err)
				}
			case "connecting another context":
				other, err := live.Open(rig.snap, nil, live.Usage{})
				if err != nil {
					t.Fatal(err)
				}
				rig.g.EnterLive(other)
			case "closing the window":
				rig.g.Shutdown()
			}
			data, err := os.ReadFile(left.Writer.ledger.Path())
			if err != nil {
				t.Fatal(err)
			}
			var cordon bool
			for _, l := range rig.rec.Full() {
				cordon = cordon || strings.HasPrefix(l, "PATCH ") && strings.Contains(l, "/nodes/"+node)
			}
			if !cordon {
				t.Fatalf("the drain sent no cordon: %v", rig.rec.Full())
			}
			if !strings.Contains(string(data), `"order":"cordon","object":"node/`+node+`"`) {
				t.Errorf("the drain's cordon reached the server but not the ledger:\n%s", data)
			}
			want := map[string]string{"ending": "drain over", "STOP": "drain stopped by STOP", "leaving the realm": "drain stopped by leaving the realm", "connecting another context": "drain stopped by leaving the realm", "closing the window": "drain stopped by leaving the realm or closing the window"}[how]
			if !strings.Contains(string(data), `"order":"kubectl drain `+node) || !strings.Contains(string(data), want) {
				t.Errorf("after %s the ledger has no line saying how the drain ended (%q):\n%s", how, want, data)
			}
			if how == "ending" || how == "STOP" {
				card := strings.Join(left.Hud.LiveCard().Writes, "\n")
				if !strings.Contains(card, "DRAIN") || !strings.Contains(strings.ToLower(card), "cordon") {
					t.Errorf("the live card does not list the cordon and the drain's end:\n%s", card)
				}
			}
		})
	}
}
