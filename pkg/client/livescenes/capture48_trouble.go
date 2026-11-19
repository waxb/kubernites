package livescenes

import (
	"time"

	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
)

const (
	staleSince = 3 * time.Minute
	staleWhy   = "dial tcp 203.0.113.10:443: connect: connection refused"
)

func init() {
	for _, tr := range Troubles {
		app.CaptureScenes["live-"+tr.Name] = troubleScene(tr, false, nil)
	}
	app.CaptureScenes["live-mended"] = troubleScene(TroubleNamed("oomkilled"), true, nil)
	app.CaptureScenes["live-chronicle"] = troubleScene(TroubleNamed("oomkilled"), true, openChronicles)
	app.CaptureScenes["live-quiet"] = liveScene(KindSnapshot, func(s *app.Session) hud.Input {
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		return hoverButton(s, hud.LiveQuietID)
	})
	app.CaptureScenes["live-stale"] = func(g *app.Game) hud.Input {
		snap := ManagedSnapshot()
		r := openLive(g, snap)
		if err := r.ReplayStale(snap.TakenAt.Add(staleSince), staleWhy); err != nil {
			panic(err)
		}
		s := g.Sess
		Settle(s, 1)
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		return idleInput()
	}
	for _, name := range []string{"live-quiet"} {
		app.LiveScenes[name] = true
	}
}

func openLive(g *app.Game, snap live.Snapshot) *live.Realm {
	r, err := live.OpenRecorded(snap)
	if err != nil {
		panic(err)
	}
	g.EnterLive(r)
	DismissIntros(g.Sess)
	return r
}

func Settle(s *app.Session, seconds float64) {
	for i := 0; float64(i)*app.TickSeconds < seconds; i++ {
		s.Advance(app.TickSeconds)
		s.Hud.Update(idleInput(), s.Frame(true))
	}
}

func troubleScene(tr Trouble, mend bool, after func(*app.Session)) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		snap := ManagedSnapshot()
		r := openLive(g, snap)
		s := g.Sess
		if err := Play(r, tr.Break(snap)); err != nil {
			panic(err)
		}
		Settle(s, hud.CalamitySettle+1)
		if v, ok := s.Hud.ShownCalamity(s.Frame(true)); ok {
			if at, found := s.Places()[v.Object]; found {
				s.CenterOn(at[0], at[1])
			}
		}
		if mend {
			if err := Play(r, tr.Mend(snap)); err != nil {
				panic(err)
			}
			Settle(s, live.MendSettle+1)
		}
		if after != nil {
			after(s)
		}
		return idleInput()
	}
}

func openChronicles(s *app.Session) {
	PressButton(s, "cmd:chronicles")
	Settle(s, .5)
}

func hoverButton(s *app.Session, id string) hud.Input {
	for _, b := range s.Hud.Buttons(s.Frame(true)) {
		if b.ID == id {
			return hud.Input{MX: b.Rect.X + b.Rect.W/2, MY: b.Rect.Y + b.Rect.H/2}
		}
	}
	return idleInput()
}
