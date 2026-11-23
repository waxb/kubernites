package livescenes

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	usedCPUShare    = 0.35
	usedMemoryShare = 0.55
	idleCPUMilli    = 3
	idleMemoryBytes = 20 << 20
	nodeCPUMilli    = 180
	nodeMemoryBytes = 700 << 20
)

func init() {
	scenes := map[string]func(*app.Game) hud.Input{
		"live-numbers":             measuredScene(ManagedSnapshot, false, hoverLive("info:live:cpu")),
		"live-numbers-metrics":     measuredScene(ManagedSnapshot, true, hoverLive("info:live:cpu")),
		"live-node-worker":         measuredScene(ManagedSnapshot, true, selectNode(false)),
		"live-node-tower":          measuredScene(KindSnapshot, false, selectNode(true)),
		"live-dashboard":           measuredScene(KindSnapshot, false, openDashboard(hud.TabQuartermaster)),
		"live-dashboard-metrics":   measuredScene(ManagedSnapshot, true, openDashboard(hud.TabQuartermaster)),
		"live-dashboard-stewards":  measuredScene(KindSnapshot, false, openDashboard(hud.TabStewards)),
		"live-dashboard-wayfinder": measuredScene(ManagedSnapshot, true, openDashboard(hud.TabWayfinder)),
		"live-dashboard-augur":     measuredScene(ManagedSnapshot, true, openDashboard(hud.TabAugur)),
		"live-dashboard-blind":     measuredScene(KindSnapshot, false, openDashboard(hud.TabAugur)),
	}
	for name, setup := range scenes {
		app.CaptureScenes[name] = setup
		app.CaptureScenes[name+"-small"] = setup
		app.WindowScenes[name+"-small"] = app.WindowSizes["-small"]
		if strings.HasPrefix(name, "live-numbers") {
			app.LiveScenes[name], app.LiveScenes[name+"-small"] = true, true
		}
	}
}

func SceneUsage(s live.Snapshot) live.Usage {
	u := live.Usage{Available: true, Pods: map[string]live.UsageOf{}, Nodes: map[string]live.UsageOf{}}
	for _, n := range s.Nodes {
		u.Nodes[n.Name] = live.UsageOf{CPUMilli: nodeCPUMilli, MemBytes: nodeMemoryBytes}
	}
	for _, p := range s.Pods {
		if p.Status.Phase != corev1.PodRunning {
			continue
		}
		var used live.UsageOf
		for _, c := range p.Spec.Containers {
			used.CPUMilli += idleCPUMilli + float64(c.Resources.Requests.Cpu().MilliValue())*usedCPUShare
			used.MemBytes += idleMemoryBytes + float64(c.Resources.Requests.Memory().Value())*usedMemoryShare
		}
		u.Pods[p.Namespace+"/"+p.Name] = used
		if node, ok := u.Nodes[p.Spec.NodeName]; ok {
			node.CPUMilli, node.MemBytes = node.CPUMilli+used.CPUMilli, node.MemBytes+used.MemBytes
			u.Nodes[p.Spec.NodeName] = node
		}
	}
	return u
}

func measuredScene(snapshot func() live.Snapshot, measured bool, set func(*app.Session) hud.Input) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		snap := snapshot()
		usage := live.Usage{}
		if measured {
			usage = SceneUsage(snap)
		}
		r, err := live.Open(snap, nil, usage)
		if err != nil {
			panic(err)
		}
		g.EnterLive(r)
		s := g.Sess
		DismissIntros(s)
		WarmSession(s, 1)
		in := set(s)
		WarmSession(s, 1)
		return in
	}
}

func hoverLive(id string) func(*app.Session) hud.Input {
	return func(s *app.Session) hud.Input {
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		return hoverButton(s, id)
	}
}

func selectNode(tower bool) func(*app.Session) hud.Input {
	return func(s *app.Session) hud.Input {
		for _, st := range s.Site.Structures {
			if st.Kind != view.StructKeep || (st.Variant == app.TowerVariant) != tower {
				continue
			}
			s.CenterOn(float64(st.X)+1, float64(st.Y)+1)
			SelectStructure(s, st)
			break
		}
		return idleInput()
	}
}

func openDashboard(tab string) func(*app.Session) hud.Input {
	return func(s *app.Session) hud.Input {
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		s.Hud.OpenCouncil(tab)
		return idleInput()
	}
}
