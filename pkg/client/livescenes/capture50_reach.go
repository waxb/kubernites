package livescenes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"k8s.io/client-go/rest"

	"github.com/waxb/kubernites/pkg/client"
	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

const (
	ReachSnapshotFixture = "pkg/live/testdata/reach-kind-snapshot.json"
	ReachReplayFixture   = "pkg/live/testdata/reach-kind-replay.json"
	ReachHome            = "/tmp/kubernites-scene-home"
)

var ReachScenes = []string{"live-terminal-shell", "live-terminal-top", "live-connect-confirm", "live-hut-more", "live-forward", "live-cp", "live-stone-exec-readonly"}

func init() {
	app.CaptureScenes["live-terminal-shell"] = reachScene(write.Seamless, func(s *app.Session, pod string) {
		terminal(s, "exec -it "+pod+" -n reach -- sh", "the hut answers 42")
	})
	app.CaptureScenes["live-terminal-top"] = reachScene(write.Seamless, func(s *app.Session, pod string) {
		s.Hud.Stone.Tall = true
		terminal(s, "exec -it "+pod+" -n reach -- top", "PID")
	})
	app.CaptureScenes["live-connect-confirm"] = reachScene(write.Writes, func(s *app.Session, pod string) {
		s.Hud.OpenStone("")
		cmd := "exec -it " + pod + " -n reach -- sh"
		s.Hud.EchoTyped(cmd)
		ask := live.StoneAsk{Why: write.NeedsConfirm, Level: write.Writes, Connect: true, Command: "kubectl " + cmd,
			Opens: write.Opening{Channel: "exec", Kind: "Pod", Namespace: "reach", Name: pod, Container: "main", Command: []string{"sh"}, Stdin: true, TTY: true}}
		s.Hud.StartStoneJob(&heldJob{ask: app.RenderAsk(ask)})
	})
	app.CaptureScenes["live-hut-more"] = writesScene(seamlessAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectHut(s)
		s.Hud.ShowLiveOrderPage(1)
		Settle(s, .2)
		return idleInput()
	})
	app.CaptureScenes["live-forward"] = reachScene(write.Seamless, func(s *app.Session, pod string) {
		s.CaptureForwards([]kubectltree.Forward{{ID: 1, Target: pod, Namespace: "reach", Pod: pod, Ports: []string{"8080"}, Listening: []string{"127.0.0.1:8080 -> 8080"}, Connections: 1}})
		Settle(s, .2)
	})
	app.CaptureScenes["live-stone-exec-readonly"] = reachScene(write.ReadOnly, func(s *app.Session, pod string) {
		speak(s, "exec -it "+pod+" -n reach -- sh")
	})
	app.CaptureScenes["live-cp"] = reachScene(write.Seamless, func(s *app.Session, pod string) {
		os.RemoveAll(ReachHome)
		speak(s, "cp "+pod+":/data/greeting.txt greeting.txt -n reach")
	})
	app.CaptureScenes["live-class-workloads"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectHut(s)
		s.CaptureClassWarning(write.Workloads)
		Settle(s, .2)
		return idleInput()
	})
	for _, name := range append([]string{"live-class-workloads"}, ReachScenes...) {
		app.LiveScenes[name] = true
	}
}

func ReachReplay() livetest.Recording {
	rec, err := livetest.ReadRecording(FixturePath(ReachReplayFixture))
	if err != nil {
		panic(err)
	}
	return rec
}

func reachPod(rec livetest.Recording) string {
	for _, e := range rec.Exchanges {
		if _, name, ok := strings.Cut(e.Path, "/namespaces/reach/pods/"); ok {
			return name
		}
	}
	panic("the reach recording names no Pod")
}

func reachFront(rec livetest.Recording) *httptest.Server {
	replay := livetest.ServeRecording(rec)
	u, _ := url.Parse(replay.URL)
	back := httputil.NewSingleHostReverseProxy(u)
	shell := &livetest.Shell{Replay: map[string]string{"sh": rec.Sessions["sh"], "top": rec.Sessions["top"]}, Tar: []byte(rec.Sessions["tar"])}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch p := r.URL.Path; {
		case strings.HasSuffix(p, "/exec") || strings.HasSuffix(p, "/attach"):
			shell.Serve(w, r)
		case strings.HasSuffix(p, "/portforward"):
			shell.ServeForward(w, r)
		default:
			back.ServeHTTP(w, r)
		}
	}))
}

func reachScene(level write.Level, set func(*app.Session, string)) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		rec := ReachReplay()
		srv := reachFront(rec)
		snap := readFixture(ReachSnapshotFixture)
		kubeconfig := replayKubeconfigK8s(srv.URL, rec.Context)
		os.MkdirAll(ReachHome, 0o755)
		if level == write.ReadOnly {
			r, err := live.OpenRecordedTreeIn(snap, kubeconfig, rec.Context, nil, ReachHome)
			if err != nil {
				panic(err)
			}
			g.EnterLive(r)
		} else {
			openReachAt(g, snap, kubeconfig, rec.Context, srv.URL, level)
		}
		s := g.Sess
		DismissIntros(s)
		if gr, ok := s.Realm.(interface{ Gate() *write.Gate }); ok && gr.Gate() != nil {
			s.PrepareLiveCapture(writesNow, "/home/player/.config/kubernites/live-ledger-kind-kubernites.jsonl")
			if level == write.Seamless {
				s.PassForCapture(write.Workloads)
			}
		}
		s.Hud.SetStoneMode(hud.StoneModeKube)
		set(s, reachPod(rec))
		WarmSession(s, 1)
		return idleInput()
	}
}

func openReachAt(g *app.Game, snap live.Snapshot, kubeconfig, name, server string, level write.Level) {
	g.Opts.Live = app.LiveSource{
		Files: []string{"/home/player/.kube/config"},
		OpenGated: func(_ context.Context, _ []string, _ string, gate *write.Gate, _ func(live.Progress)) (client.Realm, error) {
			if _, err := gate.Client(&rest.Config{Host: server}); err != nil {
				return nil, err
			}
			return live.OpenRecordedTreeIn(snap, kubeconfig, name, gate, ReachHome)
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

func terminal(s *app.Session, cmd, marker string) {
	s.Hud.OpenStone(cmd)
	s.Update(hud.Input{MX: -10, MY: -10, Keys: []ebiten.Key{ebiten.KeyEnter}}, 0, &app.Pointer{}, true)
	deadline := time.Now().Add(stoneWait)
	for time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		s.Update(hud.Input{MX: -10, MY: -10}, 0, &app.Pointer{}, true)
		if s.Hud.StoneTerminal() && strings.Contains(strings.Join(s.Hud.StoneScreenText(), "\n"), marker) {
			return
		}
	}
	panic("the Stone's terminal never showed " + marker + ": " + strings.Join(s.Hud.StoneOutput(), "\n"))
}
