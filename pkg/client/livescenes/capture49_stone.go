package livescenes

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"k8s.io/client-go/rest"

	"github.com/waxb/kubernites/pkg/client"
	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

const (
	StoneSnapshotFixture = "pkg/live/testdata/stone-kind-snapshot.json"
	StoneReplayFixture   = "pkg/live/testdata/stone-kind-replay.json"
	stoneWait            = 20 * time.Second
)

func init() {
	app.CaptureScenes["live-stone-kubectl"] = stoneScene(func(g *app.Game, s *app.Session) {
		speak(s, "get pods -A")
	})
	app.CaptureScenes["live-stone-confirm"] = stoneSceneAt(hud.StoneModeKube, write.Writes, func(g *app.Game, s *app.Session) {
		s.Hud.OpenStone("")
		ask := StoneReplay().Ask
		s.Hud.EchoTyped(strings.TrimPrefix(ask.Command, "kubectl "))
		s.Hud.StartStoneJob(&heldJob{ask: app.RenderAsk(recordedAsk(ask))})
	})
	app.CaptureScenes["live-stone-class"] = stoneSceneAt(hud.StoneModeKube, write.Writes, func(g *app.Game, s *app.Session) {
		s.Hud.OpenStone("")
		s.Hud.EchoTyped("get secrets -n parity")
		s.Hud.StartStoneJob(&heldJob{ask: app.RenderAsk(live.StoneAsk{Why: write.NeedsClass, Class: write.Secrets, Level: write.Writes, Command: "kubectl get secrets -n parity"})})
	})
	app.CaptureScenes["live-scriptorium-edit"] = stoneScene(func(g *app.Game, s *app.Session) {
		speak(s, "edit deployment web-b -n parity")
	})
	app.CaptureScenes["live-scriptorium-conflict"] = stoneSceneAt(hud.StoneModeKube, write.Writes, func(g *app.Game, s *app.Session) {
		conflicted(s)
	})
	app.CaptureScenes["live-scriptorium-force"] = stoneSceneAt(hud.StoneModeKube, write.Writes, func(g *app.Game, s *app.Session) {
		conflicted(s)
		s.Hud.PressLiveEdit("force")
	})
	app.CaptureScenes["live-details-edit"] = stoneScene(func(g *app.Game, s *app.Session) {
		SelectPod(s, func(p model.PodView) bool { return strings.HasPrefix(p.Name, "web-b-") })
		WarmSession(s, .5)
		s.Hud.OpenDetails(s.Selection, hud.DetailYAML)
		WarmSession(s, .5)
	})
	app.CaptureScenes["live-menu-stone"] = stoneSceneAt(hud.StoneModeStory, write.ReadOnly, func(g *app.Game, s *app.Session) {
		s.Hud.SetStoneMode("")
		s.Hud.Open(hud.OverlayMenu)
	})
}

func StoneReplay() livetest.Recording {
	rec, err := livetest.ReadRecording(FixturePath(StoneReplayFixture))
	if err != nil {
		panic(err)
	}
	return rec
}

func StoneSnapshot() live.Snapshot {
	return readFixture(StoneSnapshotFixture)
}

func stoneScene(set func(*app.Game, *app.Session)) func(*app.Game) hud.Input {
	return stoneSceneAt(hud.StoneModeKube, write.ReadOnly, set)
}

func stoneSceneAt(mode string, level write.Level, set func(*app.Game, *app.Session)) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		rec := StoneReplay()
		srv := livetest.ServeRecording(rec)
		snap := StoneSnapshot()
		kubeconfig := replayKubeconfigK8s(srv.URL, rec.Context)
		if level == write.ReadOnly {
			r, err := live.OpenRecordedTree(snap, kubeconfig, rec.Context, nil)
			if err != nil {
				panic(err)
			}
			g.EnterLive(r)
		} else {
			openStoneAt(g, snap, kubeconfig, rec.Context, srv.URL, level)
		}
		s := g.Sess
		DismissIntros(s)
		if gr, ok := s.Realm.(interface{ Gate() *write.Gate }); ok && gr.Gate() != nil {
			s.PrepareLiveCapture(writesNow, "/home/player/.config/kubernites/"+filepath.Base(write.LedgerFile("/", gr.Gate().Context())))
		}
		s.Hud.SetStoneMode(mode)
		set(g, s)
		WarmSession(s, 1)
		return idleInput()
	}
}

func openStoneAt(g *app.Game, snap live.Snapshot, kubeconfig, name, server string, level write.Level) {
	g.Opts.Live = app.LiveSource{
		Files: []string{"/home/player/.kube/config"},
		OpenGated: func(_ context.Context, _ []string, _ string, gate *write.Gate, _ func(live.Progress)) (client.Realm, error) {
			if _, err := gate.Client(&rest.Config{Host: server}); err != nil {
				return nil, err
			}
			return live.OpenRecordedTree(snap, kubeconfig, name, gate)
		},
	}
	g.Picker.Open(g.Opts.Live.Files)
	g.Picker.ShowContexts([]hud.LiveContext{{Name: snap.Context.Name, Cluster: snap.Context.Cluster, Server: snap.Context.Server, User: snap.Context.User}}, nil)
	g.Picker.Chosen, g.Picker.Step = 0, hud.LiveStepConfirm
	g.Picker.ReadOnly, g.Picker.Level = false, level.String()
	if err := g.TitleIntent(hud.Intent{Kind: hud.IntentLiveOpen, Target: snap.Context.Name, Command: level.String()}); err != nil {
		panic(err)
	}
	for deadline := time.Now().Add(30 * time.Second); g.Scene != app.SceneRealm; time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			panic("the recorded realm never opened: " + g.Picker.Err)
		}
		g.StepLive()
	}
}

func replayKubeconfigK8s(server, context string) string {
	dir, err := os.MkdirTemp("", "kubernites-stone-scene-")
	if err != nil {
		panic(err)
	}
	u, _ := url.Parse(server)
	path := filepath.Join(dir, "kubeconfig")
	cfg := fmt.Sprintf("apiVersion: v1\nkind: Config\nclusters:\n- name: replay\n  cluster:\n    server: %s\ncontexts:\n- name: %s\n  context:\n    cluster: replay\n    user: replay\nusers:\n- name: replay\n  user: {}\n", u.String(), context)
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		panic(err)
	}
	return path
}

func speak(s *app.Session, cmd string) {
	s.Hud.OpenStone(cmd)
	s.Update(hud.Input{MX: -10, MY: -10, Keys: []ebiten.Key{ebiten.KeyEnter}}, 0, &app.Pointer{}, true)
	settleStone(s)
}

func settleStone(s *app.Session) {
	deadline := time.Now().Add(stoneWait)
	for (s.Hud.StoneRunning() || editBusy(s)) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		s.Update(hud.Input{MX: -10, MY: -10}, 0, &app.Pointer{}, true)
	}
	if s.Hud.StoneRunning() || editBusy(s) {
		panic("the Stone's kubectl did not finish against the recording")
	}
}

func editBusy(s *app.Session) bool {
	_, editing := s.Hud.LiveEditing()
	return editing && s.Hud.Overlay() != hud.OverlayScriptorium
}

func conflicted(s *app.Session) {
	speak(s, "edit deployment web-b -n parity")
	text := s.Hud.Script.Editor.Text()
	s.Hud.Script.Editor = hud.NewEditor(strings.Replace(text, "replicas: 3", "replicas: 5", 1))
	s.Hud.Script.Editor.Dirty = true
	var cs []hud.EditConflict
	for _, c := range StoneReplay().Conflicts {
		manager := c.Manager
		if c.Subresource != "" {
			manager += " (" + c.Subresource + ")"
		}
		cs = append(cs, hud.EditConflict{Manager: manager, Fields: c.Fields})
	}
	s.Hud.ShowSent(hud.StoneEditEnd{Conflicts: cs})
}

func recordedAsk(a *livetest.Ask) live.StoneAsk {
	q, _ := url.ParseQuery(a.Query)
	return live.StoneAsk{Why: write.NeedsConfirm, Level: write.Writes, Command: a.Command,
		Order:   write.Order{Request: write.Request{Method: a.Method, Path: a.Path, Query: q, Body: []byte(a.Body)}, ContentType: a.ContentType},
		Preview: write.Answer{Code: a.Code}}
}

type heldJob struct {
	ask hud.StoneAsk
}

func (j *heldJob) Take() ([]hud.StoneChunk, bool) { return nil, false }
func (j *heldJob) Ask() (hud.StoneAsk, bool)      { return j.ask, true }
func (j *heldJob) Answer(bool)                    {}
func (j *heldJob) Stop()                          {}
func (j *heldJob) End() hud.StoneEnd              { return hud.StoneEnd{} }
