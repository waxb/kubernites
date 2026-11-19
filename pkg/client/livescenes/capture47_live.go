package livescenes

import (
	app "github.com/waxb/kubernites/pkg/client/app"
	"os"
	"strings"

	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"

	discoveryv1 "k8s.io/api/discovery/v1"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/view"

	networkingv1 "k8s.io/api/networking/v1"
)

const (
	ManagedFixture = "pkg/live/testdata/managed-snapshot.json"
	hoverLift      = 10
)

func init() {
	app.CaptureScenes["engineer-managed"] = liveScene(ManagedSnapshot, func(s *app.Session) hud.Input {
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		return idleInput()
	})
	app.CaptureScenes["engineer-managed-isle"] = liveScene(ManagedSnapshot, zoomedOut)
	app.CaptureScenes["engineer-heraldry"] = liveScene(ManagedSnapshot, func(s *app.Session) hud.Input {
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		s.Hud.SetLens(hud.LensHeraldry)
		return idleInput()
	})
	app.CaptureScenes["engineer-heraldry-keep"] = liveScene(ManagedSnapshot, func(s *app.Session) hud.Input {
		k := workerKeep(s)
		s.CenterOn(float64(k.X)+1, float64(k.Y)+1)
		s.Hud.SetLens(hud.LensHeraldry)
		return idleInput()
	})
	app.CaptureScenes["engineer-heraldry-kind"] = liveScene(KindSnapshot, func(s *app.Session) hud.Input {
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		s.Hud.SetLens(hud.LensHeraldry)
		return idleInput()
	})
	app.CaptureScenes["engineer-sea"] = liveScene(ManagedSnapshot, hoverKind(live.KindSeaChancery))
	app.CaptureScenes["engineer-gates"] = liveScene(ManagedSnapshot, hoverKind(live.KindSignpost))
	app.CaptureScenes["engineer-gate"] = liveScene(ManagedSnapshot, hoverKind(live.KindHarborGate))
	app.CaptureScenes["engineer-shrines"] = liveScene(ManagedSnapshot, hoverKind(live.KindShrine))
	app.CaptureScenes["engineer-vault"] = liveScene(ManagedSnapshot, hoverKind(view.StructVault))
	app.CaptureScenes["engineer-foreign"] = liveScene(ManagedSnapshot, hoverKind(live.KindForeign))
	app.CaptureScenes["engineer-depot"] = liveScene(ManagedSnapshot, hoverFirst(func(st view.Structure) bool { return strings.HasPrefix(st.Kind, "depot-") }))
	app.CaptureScenes["engineer-noharbor"] = liveScene(NoEntrySnapshot, func(s *app.Session) hud.Input {
		s.CenterOn(float64(s.Site.Gate.X), float64(s.Site.Gate.Y))
		return idleInput()
	})
	app.CaptureScenes["engineer-kind-vault"] = liveScene(KindSnapshot, hoverKind(view.StructVault))
	app.CaptureScenes["engineer-restricted"] = liveScene(restrictedSnapshot, func(s *app.Session) hud.Input {
		s.CenterOn(float64(s.Site.Plaza.X), float64(s.Site.Plaza.Y))
		return idleInput()
	})
	app.CaptureScenes["engineer-crew"] = liveScene(ManagedSnapshot, LiveCrew(""))
	app.CaptureScenes["engineer-crew-res"] = liveScene(ManagedSnapshot, LiveCrew("crew:mode:resources"))
	app.CaptureScenes["engineer-crew-kind"] = liveScene(KindSnapshot, LiveCrew(""))
	app.CaptureScenes["engineer-depot-card"] = liveScene(ManagedSnapshot, func(s *app.Session) hud.Input {
		for _, st := range s.Site.Structures {
			if strings.HasPrefix(st.Kind, "depot-") && st.Object.Namespace == "market" {
				c := world.FootprintCentre(st)
				s.CenterOn(c.X, c.Y)
				SelectStructure(s, st)
				break
			}
		}
		return idleInput()
	})
	app.CaptureScenes["engineer-hut-card"] = liveScene(ManagedSnapshot, func(s *app.Session) hud.Input {
		SelectPod(s, func(p model.PodView) bool { return p.Namespace == "market" && p.Owner != "" })
		if k, ok := keepOfNode(s.Site, s.Selection.Pod.Node); ok {
			s.CenterOn(float64(k.X)+1, float64(k.Y)+1)
		}
		return idleInput()
	})
	for _, name := range []string{"engineer-sea", "engineer-gates", "engineer-gate", "engineer-shrines", "engineer-vault", "engineer-foreign", "engineer-depot", "engineer-kind-vault"} {
		app.LiveScenes[name] = true
	}
}

func idleInput() hud.Input { return hud.Input{MX: -10, MY: -10} }

func readFixture(path string) live.Snapshot {
	f, err := os.Open(FixturePath(path))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	snap, err := live.ReadSnapshot(f)
	if err != nil {
		panic(err)
	}
	return snap
}

func ManagedSnapshot() live.Snapshot {
	return readFixture(ManagedFixture)
}

func KindSnapshot() live.Snapshot {
	snap := readFixture(EngineerFixture)
	snap.Context = live.ContextInfo{Name: "kind-kubernites", Cluster: "kind-kubernites", Server: "https://127.0.0.1:41231", User: "kind-kubernites"}
	return snap
}

func NoEntrySnapshot() live.Snapshot {
	snap := KindSnapshot()
	services := snap.Services[:0]
	for _, svc := range snap.Services {
		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer && svc.Spec.Type != corev1.ServiceTypeNodePort {
			services = append(services, svc)
		}
	}
	snap.Services, snap.Ingresses, snap.Gateways = services, nil, nil
	return snap
}

func liveScene(snapshot func() live.Snapshot, set func(*app.Session) hud.Input) func(*app.Game) hud.Input {
	return func(g *app.Game) hud.Input {
		r, err := live.OpenRecorded(snapshot())
		if err != nil {
			panic(err)
		}
		g.EnterLive(r)
		s := g.Sess
		DismissIntros(s)
		in := set(s)
		WarmSession(s, 1)
		return in
	}
}

func zoomedOut(s *app.Session) hud.Input {
	s.CenterOn(float64(s.Site.Width)/2, float64(s.Site.Height)/2)
	s.Cam = s.Cam.ZoomAt(render.ZoomLevels[0], hud.Width/2, float64(hud.TopH+hud.ConsoleY)/2)
	return idleInput()
}

func hoverKind(kind string) func(*app.Session) hud.Input {
	return hoverFirst(func(st view.Structure) bool { return st.Kind == kind })
}

func hoverFirst(match func(view.Structure) bool) func(*app.Session) hud.Input {
	return func(s *app.Session) hud.Input {
		for _, st := range s.Site.Structures {
			if !match(st) {
				continue
			}
			c := world.FootprintCentre(st)
			s.CenterOn(c.X, c.Y)
			x, y := world.ScreenOf(s.Cam, s.Site, c)
			return hud.Input{MX: int(x), MY: int(y) - hoverLift*int(s.Cam.Scale())}
		}
		return idleInput()
	}
}

func LiveCrew(mode string) func(*app.Session) hud.Input {
	return func(s *app.Session) hud.Input {
		for _, c := range s.Crews {
			if c.Bare || c.Namespace == "kube-system" {
				continue
			}
			s.Selection = hud.Selection{Kind: hud.SelectCrew, ID: c.ID()}
			PressButton(s, hud.ChainButton)
			if mode != "" {
				PressButton(s, mode)
			}
			break
		}
		return idleInput()
	}
}

const restrictedNamespace = "shop"

func inRestricted[T any](items []T, ns func(T) string) []T {
	out := items[:0]
	for _, it := range items {
		if ns(it) == restrictedNamespace {
			out = append(out, it)
		}
	}
	return out
}

func restrictedSnapshot() live.Snapshot {
	s := KindSnapshot()
	s.Identity = live.Identity{Username: "reader@example.com", Groups: []string{"system:authenticated"}}
	s.Pods = inRestricted(s.Pods, func(p corev1.Pod) string { return p.Namespace })
	s.Services = inRestricted(s.Services, func(v corev1.Service) string { return v.Namespace })
	s.EndpointSlices = inRestricted(s.EndpointSlices, func(v discoveryv1.EndpointSlice) string { return v.Namespace })
	s.Deployments = inRestricted(s.Deployments, func(v appsv1.Deployment) string { return v.Namespace })
	s.ReplicaSets = inRestricted(s.ReplicaSets, func(v appsv1.ReplicaSet) string { return v.Namespace })
	s.DaemonSets = inRestricted(s.DaemonSets, func(v appsv1.DaemonSet) string { return v.Namespace })
	s.StatefulSets = inRestricted(s.StatefulSets, func(v appsv1.StatefulSet) string { return v.Namespace })
	s.Jobs, s.CronJobs = nil, nil
	s.PVCs = inRestricted(s.PVCs, func(v corev1.PersistentVolumeClaim) string { return v.Namespace })
	s.Ingresses = inRestricted(s.Ingresses, func(v networkingv1.Ingress) string { return v.Namespace })
	s.Events = inRestricted(s.Events, func(v corev1.Event) string { return v.Namespace })
	s.PVs, s.StorageClasses, s.CRDs, s.Custom = nil, nil, nil, nil
	s.Refused = nil
	for _, list := range []string{"pods", "services", "endpointslices", "deployments", "replicasets", "daemonsets", "statefulsets", "jobs", "cronjobs", "persistentvolumeclaims", "ingresses", "events"} {
		s.Refused = append(s.Refused, live.Refusal{List: list, Namespace: restrictedNamespace})
	}
	s.Denied = []string{"persistentvolumes", "storageclasses", "customresourcedefinitions"}
	return s
}
