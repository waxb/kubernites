package livescenes

import (
	"fmt"
	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/view"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

const EngineerFixture = "pkg/live/testdata/kind-snapshot.json"

var engineerContexts = []hud.LiveContext{
	{Name: "kind-kubernites", Cluster: "kind-kubernites", Server: "https://127.0.0.1:41231", User: "kind-kubernites"},
	{Name: "lab-west", Cluster: "lab-west", Server: "https://lab-west.example.internal:6443", User: "reader"},
	{Name: "minikube", Cluster: "minikube", Server: "https://192.168.49.2:8443", User: "minikube"},
	{Name: "staging-eu", Cluster: "staging-eu", Server: "https://staging-eu.example.internal", User: "staging-admin"},
}

func init() {
	app.CaptureScenes["engineer"] = setupEngineer
	app.CaptureScenes["engineer-isles"] = setupEngineerIsles
	app.CaptureScenes["engineer-worker"] = setupEngineerWorker
	app.CaptureScenes["engineer-keep"] = setupEngineerKeep
	app.CaptureScenes["engineer-hut"] = setupEngineerHut
	app.CaptureScenes["engineer-stone"] = engineerStone("kubectl get pods -n shop")
	app.CaptureScenes["engineer-stone-write"] = engineerStone("kubectl delete pod hermit -n batch")
	app.CaptureScenes["engineer-stone-describe"] = engineerStoneOn("hog-", "kubectl describe pod -n batch ")
	app.CaptureScenes["engineer-warning"] = engineerPicker(func(p *hud.LivePicker) {})
	app.CaptureScenes["engineer-contexts"] = engineerPicker(func(p *hud.LivePicker) { p.ShowContexts(engineerContexts, nil) })
	app.CaptureScenes["engineer-chosen"] = engineerPicker(func(p *hud.LivePicker) {
		p.ShowContexts(engineerContexts, nil)
		p.Chosen = 0
	})
	app.CaptureScenes["engineer-confirm"] = engineerPicker(func(p *hud.LivePicker) {
		p.ShowContexts(engineerContexts, nil)
		p.Chosen, p.Step = 0, hud.LiveStepConfirm
		p.Who = hud.LiveWho{Context: engineerContexts[0].Name, Asked: true}
	})
	app.CaptureScenes["engineer-confirm-who"] = engineerPicker(func(p *hud.LivePicker) {
		p.ShowContexts(engineerContexts, nil)
		p.Chosen, p.Step = 0, hud.LiveStepConfirm
		p.Who = hud.LiveWho{Context: engineerContexts[0].Name, Username: "kubernetes-admin", Groups: []string{"kubeadm:cluster-admins", "system:authenticated"}}
	})
	for i, at := range []hud.LiveProgress{
		{List: "nodes", Done: 1, Total: 22},
		{List: "pods", Count: 500, Done: 3, Total: 22, Nodes: 3, Namespaces: 9, Pods: 500},
		{List: "persistentvolumeclaims", Count: 14, Done: 13, Total: 22, Nodes: 3, Namespaces: 9, Pods: 58},
		{List: "events", Done: 22, Total: 22, Nodes: 3, Namespaces: 9, Pods: 58},
	} {
		app.CaptureScenes[fmt.Sprintf("engineer-loading-%d", i+1)] = engineerPicker(func(p *hud.LivePicker) {
			p.ShowContexts(engineerContexts, nil)
			p.Chosen, p.Step, p.Progress = 3, hud.LiveStepOpening, at
			p.Advance(engineerReading[i])
		})
	}
	for i, frames := range []int{14, 110} {
		app.CaptureScenes[fmt.Sprintf("engineer-drawing-%d", i+1)] = engineerPicker(func(p *hud.LivePicker) {
			p.ShowContexts(engineerContexts, nil)
			p.Chosen, p.Step, p.Progress = 3, hud.LiveStepOpening, hud.LiveProgress{List: "events", Done: 22, Total: 22, Nodes: 3, Namespaces: 9, Pods: 58}
			p.Advance(engineerReading[3])
			p.Progress.Step = hud.LiveDrawing
			p.Advance(frames)
		})
	}
	app.CaptureScenes["engineer-failed"] = engineerPicker(func(p *hud.LivePicker) {
		p.Failed(live.NoAnswer(labWestLookup()))
	})
}

var engineerReading = []int{90, 300, 720, 1500}

func labWestLookup() error {
	host := "lab-west.example.internal"
	return &url.Error{Op: "Get", URL: "https://" + host + ":6443/version", Err: &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}}}
}

func ModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		if filepath.Dir(d) == d {
			return dir
		}
	}
}

func FixturePath(rel string) string {
	root := ModuleRoot()
	if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
		return filepath.Join(root, rel)
	}
	nested, _ := filepath.Glob(filepath.Join(root, "*", "go.mod"))
	for _, mod := range nested {
		if _, err := os.Stat(filepath.Join(filepath.Dir(mod), rel)); err == nil {
			return filepath.Join(filepath.Dir(mod), rel)
		}
	}
	return filepath.Join(root, rel)
}

func RecordedLiveRealm() *live.Realm {
	f, err := os.Open(FixturePath(EngineerFixture))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	snap, err := live.ReadSnapshot(f)
	if err != nil {
		panic(err)
	}
	snap.Context = live.ContextInfo{Name: "kind-kubernites", Cluster: "kind-kubernites", Server: "https://127.0.0.1:41231", User: "kind-kubernites"}
	r, err := live.OpenRecorded(snap)
	if err != nil {
		panic(err)
	}
	return r
}

func setupEngineer(g *app.Game) hud.Input {
	g.EnterLive(RecordedLiveRealm())
	return hud.Input{MX: -10, MY: -10}
}

func engineerPicker(set func(*hud.LivePicker)) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		g.Scene = app.SceneTitle
		g.Picker.Open([]string{"/home/player/.kube/config"})
		set(&g.Picker)
		return hud.Input{MX: -10, MY: -10}
	}
}

func engineerSession(g *app.Game) *app.Session {
	g.EnterLive(RecordedLiveRealm())
	DismissIntros(g.Sess)
	return g.Sess
}

func setupEngineerIsles(g *app.Game) hud.Input {
	s := engineerSession(g)
	s.CenterOn(float64(s.Site.Width)/2, float64(s.Site.Height)/2)
	s.Cam = s.Cam.ZoomAt(render.ZoomLevels[0], hud.Width/2, float64(hud.TopH+hud.ConsoleY)/2)
	return hud.Input{MX: -10, MY: -10}
}

func workerKeep(s *app.Session) view.Structure {
	for _, st := range s.Site.Structures {
		if st.Kind == view.StructKeep && st.Variant != app.TowerVariant {
			return st
		}
	}
	return view.Structure{}
}

func setupEngineerWorker(g *app.Game) hud.Input {
	s := engineerSession(g)
	k := workerKeep(s)
	s.CenterOn(float64(k.X)+1, float64(k.Y)+1)
	return hud.Input{MX: -10, MY: -10}
}

func setupEngineerKeep(g *app.Game) hud.Input {
	s := engineerSession(g)
	k := workerKeep(s)
	s.CenterOn(float64(k.X)+1, float64(k.Y)+1)
	SelectStructure(s, k)
	WarmSession(s, 1)
	return hud.Input{MX: -10, MY: -10}
}

func setupEngineerHut(g *app.Game) hud.Input {
	s := engineerSession(g)
	SelectPod(s, func(p model.PodView) bool { return strings.HasPrefix(p.Name, "crasher-") })
	if k, ok := keepOfNode(s.Site, s.Selection.Pod.Node); ok {
		s.CenterOn(float64(k.X)+1, float64(k.Y)+1)
	}
	WarmSession(s, 1)
	return hud.Input{MX: -10, MY: -10}
}

func keepOfNode(site view.Site, node string) (view.Structure, bool) {
	for _, st := range site.Structures {
		if st.Kind == view.StructKeep && st.Object.Name == node {
			return st, true
		}
	}
	return view.Structure{}, false
}

func engineerStone(command string) func(*app.Game) hud.Input {
	return engineerStoneOn("", command)
}

func engineerStoneOn(podPrefix, command string) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		s := engineerSession(g)
		for _, p := range s.Pods {
			if podPrefix != "" && strings.HasPrefix(p.Name, podPrefix) {
				command += p.Name
				break
			}
		}
		s.Hud.OpenStone(command)
		s.Update(hud.Input{MX: -10, MY: -10, Keys: []ebiten.Key{ebiten.KeyEnter}}, 0, &app.Pointer{}, true)
		WarmSession(s, .5)
		return hud.Input{MX: -10, MY: -10}
	}
}
