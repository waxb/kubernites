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

func evictionsSeen(rec *livetest.Record) []string {
	var out []string
	for _, l := range rec.Full() {
		if strings.HasPrefix(l, "POST ") && strings.Contains(l, "/eviction") {
			out = append(out, l)
		}
	}
	return out
}

func TestLeavingConnectingAnotherOrClosingTheWindowStopsARunningDrainAtOnce(t *testing.T) {
	for _, how := range []string{"leaving the realm", "connecting another context", "closing the window"} {
		t.Run(how, func(t *testing.T) {
			rig := connectAt(t, hud.LevelSeamless, livetest.Options{Answer: func(r *http.Request, _ []byte) (int, string, bool) {
				if strings.HasSuffix(r.URL.Path, "/eviction") {
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
			sent := evictionsSeen(rig.rec)
			time.Sleep(300 * time.Millisecond)
			if after := evictionsSeen(rig.rec); len(after) > len(sent) {
				t.Errorf("after %s, %d more evictions reached the server", how, len(after)-len(sent))
			}
			if dr := left.Writer.drain; dr == nil || !dr.over {
				t.Errorf("after %s the left session's drain still runs", how)
			}
			data, err := os.ReadFile(left.Writer.ledger.Path())
			if err != nil {
				t.Fatal(err)
			}
			pods := map[string]bool{}
			for _, l := range sent {
				path := strings.TrimPrefix(l, "POST ")
				path, _, _ = strings.Cut(path, "?")
				parts := strings.Split(path, "/")
				pods[parts[len(parts)-2]] = true
			}
			for pod := range pods {
				if !strings.Contains(string(data), `"order":"evict"`) || !strings.Contains(string(data), "/"+pod+`"`) || !strings.Contains(string(data), `"object":"pod `) {
					t.Errorf("after %s the ledger has no line for the eviction of %s that reached the server:\n%s", how, pod, data)
				}
			}
		})
	}
}
