package livescenes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"

	"github.com/waxb/kubernites/pkg/client"
	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/write"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	writesCrewNS  = "market"
	writesCrew    = "storefront"
	ledgerShown   = "/home/player/.config/kubernites/live-ledger-staging-north-5f2c19a07d3e.jsonl"
	refusalReason = `RBAC: User "viewer@example.com" cannot patch resource "deployments/scale" in API group "apps" in the namespace "market"`
)

func k8sWebhookRefusal() string {
	return `admission webhook "audit.policy.example.com" does not support dry run`
}

var writesNow = time.Date(2026, 11, 15, 14, 9, 30, 0, time.UTC)

type fakeCluster struct {
	deny      bool
	webhook   bool
	budget    bool
	answering func(*http.Request)
	seen      *seenRequests
}

type seenRequests struct {
	mu    sync.Mutex
	lines []string
}

func (s *seenRequests) add(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lines = append(s.lines, line)
}

func (s *seenRequests) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.lines...)
}

func (f fakeCluster) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
	}
	if f.seen != nil {
		f.seen.add(req.Method + " " + req.URL.Path + "?" + req.URL.RawQuery)
	}
	if f.answering != nil {
		f.answering(req)
	}
	code, out := f.answer(req, body)
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(out)), Request: req}, nil
}

func status(code int, reason, message string) string {
	data, _ := json.Marshal(map[string]any{"kind": "Status", "apiVersion": "v1", "status": "Failure", "code": code, "reason": reason, "message": message})
	return string(data)
}

func (f fakeCluster) answer(req *http.Request, body []byte) (int, string) {
	path := req.URL.Path
	dry := req.URL.Query().Get("dryRun") == "All"
	switch {
	case strings.HasSuffix(path, "/selfsubjectaccessreviews"):
		if f.deny {
			return http.StatusCreated, `{"status":{"allowed":false,"reason":` + quote(refusalReason) + `}}`
		}
		return http.StatusCreated, `{"status":{"allowed":true}}`
	case f.webhook && dry:
		return http.StatusBadRequest, status(400, "BadRequest", k8sWebhookRefusal())
	case strings.HasSuffix(path, "/eviction") && f.budget && strings.Contains(path, "/"+writesCrew+"-"):
		data, _ := json.Marshal(map[string]any{"kind": "Status", "code": 429, "reason": "TooManyRequests", "message": "Cannot evict pod as it would violate the pod's disruption budget.",
			"details": map[string]any{"causes": []any{map[string]any{"reason": "DisruptionBudget", "message": "The disruption budget storefront needs 2 healthy pods and has 2 currently"}}}})
		return http.StatusTooManyRequests, string(data)
	case req.Method == http.MethodDelete:
		return http.StatusOK, `{"metadata":{"deletionTimestamp":"2026-11-15T14:09:30Z","deletionGracePeriodSeconds":30}}`
	case req.Method == http.MethodPatch && req.Header.Get("Content-Type") == write.JSONPatch:
		var ops []map[string]any
		_ = json.Unmarshal(body, &ops)
		obj := map[string]any{"metadata": map[string]any{}, "spec": map[string]any{}}
		for _, op := range ops {
			switch op["path"] {
			case "/spec/template":
				obj["spec"].(map[string]any)["template"] = op["value"]
			case "/metadata/annotations":
				obj["metadata"].(map[string]any)["annotations"] = op["value"]
			}
		}
		data, _ := json.Marshal(obj)
		return http.StatusOK, string(data)
	case req.Method == http.MethodPatch || req.Method == http.MethodPost:
		if len(bytes.TrimSpace(body)) == 0 {
			return http.StatusCreated, `{}`
		}
		return http.StatusOK, string(body)
	}
	return http.StatusOK, `{}`
}

func quote(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

func writesSnapshot() live.Snapshot {
	snap := ManagedSnapshot()
	min, util := int32(2), int32(70)
	snap.HPAs = append(snap.HPAs, autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Namespace: writesCrewNS, Name: writesCrew, UID: "hpa-storefront", ResourceVersion: "88123"},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: writesCrew},
			MinReplicas: &min, MaxReplicas: 6, Metrics: []autoscalingv2.MetricSpec{{Type: autoscalingv2.ResourceMetricSourceType, Resource: &autoscalingv2.ResourceMetricSource{Name: "cpu", Target: autoscalingv2.MetricTarget{Type: autoscalingv2.UtilizationMetricType, AverageUtilization: &util}}}}},
	})
	two := intstr.FromInt32(2)
	snap.PDBs = append(snap.PDBs, policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Namespace: writesCrewNS, Name: writesCrew, UID: "pdb-storefront", ResourceVersion: "77001"},
		Spec:       policyv1.PodDisruptionBudgetSpec{MinAvailable: &two, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": writesCrew}}},
		Status:     policyv1.PodDisruptionBudgetStatus{CurrentHealthy: 2, DesiredHealthy: 2, DisruptionsAllowed: 0, ExpectedPods: 2},
	})
	return snap
}

type writesSetup struct {
	level   write.Level
	cluster fakeCluster
	hints   bool
	classes []write.Class
}

func openWrites(g *app.Game, w writesSetup) (*app.Session, *live.Realm) {
	snap := writesSnapshot()
	var realm *live.Realm
	g.Opts.Live = app.LiveSource{
		Files: []string{"/home/player/.kube/config"},
		OpenGated: func(_ context.Context, _ []string, _ string, gate *write.Gate, _ func(live.Progress)) (client.Realm, error) {
			if _, err := gate.Client(&rest.Config{Host: snap.Context.Server, Transport: w.cluster}); err != nil {
				return nil, err
			}
			r, err := live.OpenRecordedGated(snap, gate)
			realm = r
			return r, err
		},
	}
	g.Settings.LiveHints = w.hints
	g.Picker.Open(g.Opts.Live.Files)
	g.Picker.ShowContexts([]hud.LiveContext{{Name: snap.Context.Name, Cluster: snap.Context.Cluster, Server: snap.Context.Server, User: snap.Context.User}}, nil)
	g.Picker.Chosen, g.Picker.Step = 0, hud.LiveStepConfirm
	g.Picker.ReadOnly, g.Picker.Level = false, w.level.String()
	if err := g.TitleIntent(hud.Intent{Kind: hud.IntentLiveOpen, Target: snap.Context.Name, Command: w.level.String()}); err != nil {
		panic(err)
	}
	for deadline := time.Now().Add(30 * time.Second); g.Scene != app.SceneRealm; time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			panic("the recorded realm never opened: " + g.Picker.Err)
		}
		g.StepLive()
	}
	s := g.Sess
	for _, c := range w.classes {
		s.PassForCapture(c)
	}
	DismissIntros(s)
	s.PrepareLiveCapture(writesNow, ledgerShown)
	return s, realm
}

func writesScene(w writesSetup, set func(*app.Session, *live.Realm) hud.Input) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		s, r := openWrites(g, w)
		Settle(s, .5)
		in := set(s, r)
		return in
	}
}

func selectCrew(s *app.Session) {
	s.Selection = hud.Selection{Kind: hud.SelectCrew, ID: writesCrewNS + "/" + writesCrew}
	Settle(s, .3)
}

func selectHut(s *app.Session) {
	SelectPod(s, func(p model.PodView) bool {
		return p.Namespace == writesCrewNS && strings.HasPrefix(p.Name, writesCrew+"-")
	})
	if k, ok := keepOfNode(s.Site, s.Selection.Pod.Node); ok {
		s.CenterOn(float64(k.X)+1, float64(k.Y)+1)
	}
	Settle(s, .3)
}

func selectKeep(s *app.Session) view.Structure {
	k := workerKeep(s)
	SelectStructure(s, k)
	s.CenterOn(float64(k.X)+1, float64(k.Y)+1)
	Settle(s, .3)
	return k
}

func order(s *app.Session, key string) {
	s.Handle(hud.Intent{Kind: hud.IntentLiveOrder, Target: key})
	Settle(s, .3)
}

func scroll(s *app.Session, cmd string) {
	s.Handle(hud.Intent{Kind: hud.IntentLiveScroll, Command: cmd, Target: s.Hud.OrderValues()})
	Settle(s, .3)
}

func setInput(s *app.Session, i int, value string) {
	form, ok := s.Hud.OrderOpen()
	if !ok || i >= len(form.Inputs) {
		return
	}
	form.Inputs[i].Value = value
	s.Hud.OpenOrder(form)
}

var (
	writesAt   = writesSetup{level: write.Writes}
	seamlessAt = writesSetup{level: write.Seamless}
)

func drainAt(cluster fakeCluster) writesSetup {
	return writesSetup{level: write.Seamless, cluster: cluster}
}

func drainScene(s *app.Session, _ *live.Realm) hud.Input {
	selectHut(s)
	if k, ok := keepOfNode(s.Site, s.Selection.Pod.Node); ok {
		SelectStructure(s, k)
		Settle(s, .3)
	}
	s.CaptureAsync()
	s.Handle(hud.Intent{Kind: hud.IntentLiveOrder, Target: "drain"})
	s.Handle(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollGo, Target: s.Hud.OrderValues()})
	if !s.CaptureDrainWaiting(30 * time.Second) {
		panic("live-drain: the drain never came to wait on the disruption budget")
	}
	Settle(s, .3)
	for range 20 {
		Settle(s, .05)
	}
	return idleInput()
}

func pickerConfirm(level string, hover bool, set func(*hud.LivePicker)) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		g.Scene = app.SceneTitle
		g.Picker.Open([]string{"/home/player/.kube/config"})
		g.Picker.ShowContexts(engineerContexts, nil)
		g.Picker.Chosen, g.Picker.Step = 0, hud.LiveStepConfirm
		g.Picker.Who = hud.LiveWho{Context: engineerContexts[0].Name, Username: "kubernetes-admin"}
		if level != hud.LevelReadOnly {
			g.Picker.ReadOnly, g.Picker.Level = false, level
		}
		if set != nil {
			set(&g.Picker)
		}
		in := hud.Input{MX: -10, MY: -10}
		if hover {
			for _, b := range g.Picker.Buttons() {
				if b.ID == hud.LiveBoxID {
					in = hud.Input{MX: b.Rect.X + b.Rect.W/2, MY: b.Rect.Y + b.Rect.H/2}
				}
			}
			for range 40 {
				g.Picker.Update(in)
			}
		}
		return in
	}
}

func titleCaption(liveOnly bool) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		g.Scene = app.SceneTitle
		g.Title.LiveOnly = liveOnly
		in := hud.Input{MX: -10, MY: -10}
		for _, b := range g.Title.Buttons() {
			if b.ID == "title:live" {
				in = hud.Input{MX: b.Rect.X + b.Rect.W/2, MY: b.Rect.Y + b.Rect.H/2}
			}
		}
		for range hud.TitleTipTicks + 5 {
			g.Title.Update(in)
		}
		return in
	}
}

func init() {
	app.CaptureScenes["live-title-caption"] = titleCaption(false)
	app.CaptureScenes["live-title-caption-open"] = titleCaption(true)
	app.CaptureScenes["live-picker-box"] = pickerConfirm(hud.LevelReadOnly, true, nil)
	app.CaptureScenes["live-picker-writes"] = pickerConfirm(hud.LevelWrites, false, nil)
	app.CaptureScenes["live-picker-forced"] = pickerConfirm(hud.LevelReadOnly, true, func(p *hud.LivePicker) { p.Forced = true })
	app.CaptureScenes["live-warn-writes"] = pickerConfirm(hud.LevelReadOnly, false, func(p *hud.LivePicker) { p.Step = hud.LiveStepWrites })
	app.CaptureScenes["live-warn-seamless"] = pickerConfirm(hud.LevelReadOnly, false, func(p *hud.LivePicker) { p.Step = hud.LiveStepSeam })
	app.CaptureScenes["live-warn-changed"] = pickerConfirm(hud.LevelReadOnly, false, func(p *hud.LivePicker) {
		p.Step = hud.LiveStepWrites
		p.Changed = "Changed since WRITES was chosen: its API server was https://127.0.0.1:38817. The remembered choice is dropped."
	})
	app.CaptureScenes["live-level-writes"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		return idleInput()
	})
	app.CaptureScenes["live-level-seamless"] = writesScene(seamlessAt, func(s *app.Session, _ *live.Realm) hud.Input {
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		return idleInput()
	})
	app.CaptureScenes["live-crew-orders"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectCrew(s)
		return hoverButton(s, hud.LiveOrderPrefix+":scale")
	})
	app.CaptureScenes["live-crew-more"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectCrew(s)
		PressButton(s, hud.LiveOrderPrefix+":more")
		Settle(s, .2)
		return idleInput()
	})
	app.CaptureScenes["live-hut-orders"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectHut(s)
		return hoverButton(s, hud.LiveOrderPrefix+":kill")
	})
	app.CaptureScenes["live-keep-orders"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectKeep(s)
		return hoverButton(s, hud.LiveOrderPrefix+":drain")
	})
	app.CaptureScenes["live-order-scale"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectCrew(s)
		order(s, "scale")
		setInput(s, 0, "4")
		return idleInput()
	})
	app.CaptureScenes["live-preview"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectCrew(s)
		order(s, "scale")
		setInput(s, 0, "4")
		scroll(s, hud.ScrollGo)
		return idleInput()
	})
	app.CaptureScenes["live-preview-rollback"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectCrew(s)
		order(s, "restart")
		return idleInput()
	})
	app.CaptureScenes["live-delete-seamless"] = writesScene(seamlessAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectHut(s)
		order(s, "delete")
		return idleInput()
	})
	app.CaptureScenes["live-restart-seamless"] = writesScene(seamlessAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectCrew(s)
		order(s, "restart")
		return idleInput()
	})
	app.CaptureScenes["live-class-destroy"] = writesScene(seamlessAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectHut(s)
		order(s, "delete")
		setInput(s, 1, "on")
		scroll(s, hud.ScrollGo)
		return idleInput()
	})
	app.CaptureScenes["live-class-secrets"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectHut(s)
		s.CaptureClassWarning(write.Secrets)
		Settle(s, .2)
		return idleInput()
	})
	app.CaptureScenes["live-class-unpreviewed"] = writesScene(writesSetup{level: write.Writes, cluster: fakeCluster{webhook: true}}, func(s *app.Session, _ *live.Realm) hud.Input {
		selectCrew(s)
		order(s, "image")
		setInput(s, 1, "registry.example/storefront:1.5.0")
		scroll(s, hud.ScrollGo)
		return idleInput()
	})
	app.CaptureScenes["live-refused"] = writesScene(writesSetup{level: write.Writes, cluster: fakeCluster{deny: true}}, func(s *app.Session, _ *live.Realm) hud.Input {
		selectCrew(s)
		return hoverButton(s, hud.LiveOrderPrefix+":scale")
	})
	app.CaptureScenes["live-ledger"] = writesScene(seamlessAt, func(s *app.Session, _ *live.Realm) hud.Input {
		selectCrew(s)
		order(s, "restart")
		scroll(s, hud.ScrollGo)
		s.Selection = hud.Selection{}
		Settle(s, .3)
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		return hoverButton(s, "info:live:ledger")
	})
	app.CaptureScenes["live-drain"] = writesScene(drainAt(fakeCluster{budget: true}), drainScene)
	app.CaptureScenes["live-stale-orders"] = writesScene(writesAt, func(s *app.Session, r *live.Realm) hud.Input {
		if err := r.ReplayStale(writesNow.Add(-2*time.Minute), staleWhy); err != nil {
			panic(err)
		}
		selectKeep(s)
		return hoverButton(s, hud.LiveOrderPrefix+":cordon")
	})
	for _, hints := range []bool{false, true} {
		name := "live-hints-off"
		if hints {
			name = "live-hints-on"
		}
		app.CaptureScenes[name] = writesScene(writesSetup{level: write.Writes, hints: hints}, func(s *app.Session, r *live.Realm) hud.Input {
			if err := Play(r, TroubleNamed("crashloop").Break(r.Snapshot())); err != nil {
				panic(err)
			}
			Settle(s, hud.CalamitySettle+1)
			if v, ok := s.Hud.ShownCalamity(s.Frame(true)); ok {
				if at, found := s.Places()[v.Object]; found {
					s.CenterOn(at[0], at[1])
				}
			}
			return idleInput()
		})
	}
	app.CaptureScenes["live-history"] = writesScene(writesAt, func(s *app.Session, r *live.Realm) hud.Input {
		snap := r.Snapshot()
		tr := TroubleNamed("crashloop")
		if err := Play(r, tr.Break(snap)); err != nil {
			panic(err)
		}
		Settle(s, hud.CalamitySettle+1)
		if err := Play(r, tr.Mend(snap)); err != nil {
			panic(err)
		}
		Settle(s, live.MendSettle+1)
		openChronicles(s)
		return idleInput()
	})
	app.CaptureScenes["live-menu"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		PressButton(s, "top:menu")
		Settle(s, .2)
		return hoverButton(s, "menu:hints")
	})
	app.CaptureScenes["live-choices"] = writesScene(writesAt, func(s *app.Session, _ *live.Realm) hud.Input {
		s.Hud.SetLiveChoices([]hud.LiveChoice{
			{Name: "kind-kubernites", Server: "https://127.0.0.1:41231", Level: hud.LevelSeamless, Classes: []string{"destroy"}},
			{Name: "staging-north", Server: "https://203.0.113.10", Level: hud.LevelWrites},
		})
		PressButton(s, "top:menu")
		PressButton(s, "menu:livechoices")
		Settle(s, .2)
		return idleInput()
	})
	for _, name := range []string{"live-level-writes", "live-level-seamless", "live-crew-orders", "live-crew-more", "live-hut-orders", "live-keep-orders", "live-order-scale",
		"live-preview", "live-preview-rollback", "live-delete-seamless", "live-restart-seamless", "live-class-destroy", "live-class-secrets", "live-class-unpreviewed",
		"live-refused", "live-ledger", "live-drain", "live-stale-orders", "live-hints-off", "live-hints-on", "live-history", "live-menu", "live-choices"} {
		app.LiveScenes[name] = true
	}
}
