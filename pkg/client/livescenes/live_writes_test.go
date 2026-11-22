package livescenes

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
)

var writesScenes = []string{"live-level-writes", "live-level-seamless", "live-crew-orders", "live-crew-more", "live-hut-orders", "live-keep-orders", "live-order-scale",
	"live-preview", "live-preview-rollback", "live-delete-seamless", "live-restart-seamless", "live-class-destroy", "live-class-secrets", "live-class-workloads", "live-class-unpreviewed", "live-hut-more", "live-forward",
	"live-refused", "live-ledger", "live-drain", "live-stale-orders", "live-hints-off", "live-hints-on", "live-history", "live-menu", "live-choices"}

func TestStoryModeLiveWritesSpeakNoKubernetes(t *testing.T) {
	defer app.ApplyNames(false)
	snap := writesSnapshot()
	skip := []string{snap.Context.Name, snap.Context.Server, snap.Identity.Username, ledgerShown, refusalReason, k8sWebhookRefusal()}
	for _, p := range snap.Pods {
		skip = append(skip, p.Name)
	}
	for _, name := range writesScenes {
		t.Run(name, func(t *testing.T) {
			app.ApplyNames(false)
			g := app.NewGame(nil, app.Options{LiveOnly: true}, nil, nil)
			g.Settings.LiveNames = storyNames()
			in := app.CaptureScenes[name](g)
			s := g.Sess
			for range 40 {
				s.Update(in, app.TickSeconds, &app.Pointer{}, true)
			}
			for _, leak := range drawnInStory(t, s, skip) {
				t.Errorf("%s: %s", name, leak)
			}
		})
	}
}

func TestEveryWritesSceneShowsWhatItNames(t *testing.T) {
	defer app.ApplyNames(false)
	checks := map[string]func(*app.Session) string{
		"live-level-writes": func(s *app.Session) string {
			return expect(s.Hud.LiveCard().Level == hud.LevelWrites, "level %q", s.Hud.LiveCard().Level)
		},
		"live-level-seamless": func(s *app.Session) string {
			return expect(s.Hud.LiveCard().Level == hud.LevelSeamless, "level %q", s.Hud.LiveCard().Level)
		},
		"live-restart-seamless": func(s *app.Session) string {
			f, ok := s.Hud.OrderOpen()
			return expect(ok && f.Key == "restart" && f.Go == "RESTART [Y]" && len(f.Lines) == 1 && len(s.Hud.LiveCard().Writes) == 0, "form %+v writes %v", f, s.Hud.LiveCard().Writes)
		},
		"live-order-scale": func(s *app.Session) string {
			f, ok := s.Hud.OrderOpen()
			return expect(ok && f.Key == "scale" && len(f.Lines) == 1, "form %+v", f)
		},
		"live-preview": func(s *app.Session) string {
			f, ok := s.Hud.OrderOpen()
			return expect(ok && strings.HasPrefix(f.Go, "CONFIRM") && len(f.Changes) == 1 && f.Changes[0].After == "4", "form %+v", f)
		},
		"live-class-destroy": func(s *app.Session) string {
			f, ok := s.Hud.OrderOpen()
			return expect(ok && f.Warning, "form %+v", f)
		},
		"live-drain": func(s *app.Session) string {
			f, ok := s.Hud.OrderOpen()
			return expect(ok && len(f.Progress) >= 2 && strings.Contains(f.Progress[0], " 21 OF 28 ") && strings.HasPrefix(f.Progress[1], "WAITING: "), "form %+v", f)
		},
		"live-class-unpreviewed": func(s *app.Session) string {
			f, ok := s.Hud.OrderOpen()
			return expect(ok && f.Warning && strings.Contains(f.Note, "switch NAMES to read why") && !strings.Contains(f.Note, "dry run"), "form %+v", f)
		},
		"live-refused": func(s *app.Session) string {
			for _, o := range s.Hud.LiveOrders() {
				if o.Key == "scale" {
					return expect(o.Disabled && strings.Contains(o.Tip, "they may not scale it") && !strings.Contains(o.Tip, refusalReason), "scale %+v", o)
				}
			}
			return "no scale order"
		},
		"live-ledger": func(s *app.Session) string {
			c := s.Hud.LiveCard()
			return expect(len(c.Writes) == 1 && c.Ledger == ledgerShown, "card %+v", c)
		},
		"live-stale-orders": func(s *app.Session) string {
			for _, o := range s.Hud.LiveOrders() {
				if !o.Disabled || o.Why != "STALE" {
					return "not stale: " + o.Key
				}
			}
			return ""
		},
		"live-history": func(s *app.Session) string {
			ch := s.State.Chronicles
			return expect(len(ch) == 1 && ch[0].Plain && len(ch[0].Options) == 0, "chronicles %+v", ch)
		},
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			app.ApplyNames(false)
			g := app.NewGame(nil, app.Options{LiveOnly: true}, nil, nil)
			g.Settings.LiveNames = storyNames()
			in := app.CaptureScenes[name](g)
			for range 10 {
				g.Sess.Update(in, app.TickSeconds, &app.Pointer{}, true)
			}
			if msg := check(g.Sess); msg != "" {
				t.Error(msg)
			}
		})
	}
}

func expect(ok bool, format string, args ...any) string {
	if ok {
		return ""
	}
	return strings.TrimSpace(sprintf(format, args...))
}

func sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

const heldEviction = 12

type heldFrames struct {
	*live.Realm
	s        *app.Session
	asked    chan struct{}
	answered chan struct{}
	waited   bool
	told     float64
	release  sync.Once
}

func (h *heldFrames) Pump(dt float64) {
	if _, open := h.s.Hud.OrderOpen(); open && !h.waited {
		select {
		case <-h.asked:
		case <-time.After(2 * time.Second):
		}
		h.waited, h.told = true, h.s.State.RealmTime
	}
	if h.waited && h.s.State.RealmTime != h.told {
		h.release.Do(func() { close(h.answered) })
	}
	h.Realm.Pump(dt)
}

func TestTheDrainSceneShowsTheSameTidingsHoweverTheClusterAnswers(t *testing.T) {
	defer app.ApplyNames(false)
	shown := func(cluster fakeCluster, opened func(*app.Session, *live.Realm)) (tidings []string, times []float64) {
		app.ApplyNames(false)
		g := app.NewGame(nil, app.Options{LiveOnly: true}, nil, nil)
		g.Settings.LiveNames = storyNames()
		writesScene(drainAt(cluster), func(s *app.Session, r *live.Realm) hud.Input {
			opened(s, r)
			return drainScene(s, r)
		})(g)
		for _, td := range g.Sess.Tidings {
			tidings = append(tidings, td.Text)
		}
		for _, e := range g.Sess.Echoes {
			if strings.HasPrefix(e.Text, "WRITE ") {
				times = append(times, e.Time)
			}
		}
		return tidings, times
	}
	atOnce, _ := shown(fakeCluster{budget: true}, func(*app.Session, *live.Realm) {})
	asked, answered := make(chan struct{}), make(chan struct{})
	var evictions atomic.Int32
	late := fakeCluster{budget: true, answering: func(req *http.Request) {
		if !strings.HasSuffix(req.URL.Path, "/eviction") || evictions.Add(1) != heldEviction {
			return
		}
		close(asked)
		select {
		case <-answered:
		case <-time.After(time.Second):
		}
	}}
	held, times := shown(late, func(s *app.Session, r *live.Realm) {
		s.Realm = &heldFrames{Realm: r, s: s, asked: asked, answered: answered}
	})
	if len(times) == 0 {
		t.Fatal("the drain wrote nothing")
	}
	for i, at := range times {
		if at != times[0] {
			t.Errorf("write %d of the drain is told at %.3f s, the first at %.3f s: the realm's clock moved while the drain ran", i, at, times[0])
		}
	}
	if !slices.Equal(atOnce, held) {
		t.Errorf("the Tidings depend on when the cluster answers:\nat once %q\nheld    %q", atOnce, held)
	}
}
