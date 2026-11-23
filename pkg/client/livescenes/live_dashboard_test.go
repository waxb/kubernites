package livescenes

import (
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
)

func TestTheLiveDashboardShowsTheRealmsOwnFacts(t *testing.T) {
	for name, snap := range map[string]func() live.Snapshot{"kind": KindSnapshot, "managed": ManagedSnapshot} {
		s := struct{ name string }{name}
		g, r := liveGame(t, snap())
		play(g.Sess, 1)
		f := g.Sess.Frame(true)
		if !f.Live || !reflect.DeepEqual(f.LiveCouncil, r.Council()) {
			t.Errorf("%s: the frame carries %+v, the realm counts %+v", s.name, f.LiveCouncil, r.Council())
		}
		if f.LiveCouncil.Nodes == 0 || f.LiveCouncil.PodsActive == 0 {
			t.Errorf("%s: the Dashboard counts no node or Pod: %+v", s.name, f.LiveCouncil)
		}
		in := idleInput()
		in.Keys = []ebiten.Key{ebiten.KeyC}
		g.Sess.Hud.Update(in, f)
		if g.Sess.Hud.Overlay() != hud.OverlayCouncil {
			t.Errorf("%s: C opens %q in live, want the Dashboard", s.name, g.Sess.Hud.Overlay())
		}
	}
}
