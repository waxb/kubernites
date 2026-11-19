package app

import (
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const MaxAlerts = 40

type AlertSources struct {
	Site      view.Site
	Pods      []model.PodView
	State     view.GameState
	Nodes     map[string]corev1.ConditionStatus
	Home      func(pod string) (world.Vec, bool)
	Omen      *world.Omen
	Horns     []model.Horn
	Advancing map[string]bool
}

func (s *Session) Alerts() []hud.Alert {
	derived := DeriveAlerts(AlertSources{Site: s.Site, Pods: s.Pods, State: s.State, Nodes: NodeReadiness(s.Cache), Home: s.Renderer.Home, Omen: s.Omen, Horns: s.Horns, Advancing: s.advancingOwners()})
	return s.Rail.Alarms.Grade(derived)
}

func (s *Session) advancingOwners() map[string]bool {
	now := s.State.FoundedAt.Add(time.Duration(s.State.RealmTime * float64(time.Second)))
	return s.Cache.AdvancingOwners(now, time.Duration(SustainFor[hud.AlertPending]*float64(time.Second)))
}

func NodeReadiness(c *model.Cache) map[string]corev1.ConditionStatus {
	out := map[string]corev1.ConditionStatus{}
	for _, n := range c.Nodes() {
		out[n.Name] = ReadyOf(n)
	}
	return out
}

func ReadyOf(n *corev1.Node) corev1.ConditionStatus {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status
		}
	}
	return corev1.ConditionFalse
}

func DeriveAlerts(src AlertSources) []hud.Alert {
	if src.Site.Width == 0 {
		return nil
	}
	out := hearthAlerts(src)
	out = append(out, keepAlerts(src)...)
	out = append(out, realmAlerts(src)...)
	sort.SliceStable(out, func(i, j int) bool { return hud.AlertLevel(out[i].Kind) > hud.AlertLevel(out[j].Kind) })
	return out[:min(len(out), MaxAlerts)]
}

func centre(s view.Structure) [2]float64 {
	return [2]float64{float64(s.X) + float64(s.W)/2, float64(s.Y) + float64(s.H)/2}
}

func hearthSpot(src AlertSources, p model.PodView, pending int) [2]float64 {
	if src.Home != nil {
		if v, ok := src.Home(p.Name); ok {
			return [2]float64{v.X + .5, v.Y + .5}
		}
	}
	if k, ok := world.KeepForNode(src.Site, p.Node); ok && p.Node != "" {
		return centre(k)
	}
	v := world.PendingSpot(src.Site, pending)
	return [2]float64{v.X, v.Y}
}

func hearthAlert(p model.PodView) (string, string, bool) {
	switch {
	case p.System || p.Terminating:
		return "", "", false
	case p.State == model.HearthGuttering && p.Restarts == 0 && p.Reason != "CrashLoopBackOff":
		return hud.AlertFire, fmt.Sprintf("%s's worker stopped: %s", p.Name, orUnknown(p.Reason)), true
	case p.State == model.HearthGuttering:
		return hud.AlertFire, fmt.Sprintf("%s is guttering: %s", p.Name, orUnknown(p.Reason)), true
	case p.State == model.HearthGhost:
		return hud.AlertGhost, fmt.Sprintf("%s: the graduate never arrived (%s)", p.Name, orUnknown(p.Reason)), true
	case p.State == model.HearthBanished:
		return hud.AlertFallen, fmt.Sprintf("%s is turned out: %s", p.Name, orUnknown(p.Reason)), true
	case p.State == model.HearthPending:
		return hud.AlertPending, p.Name + " waits Unhoused (Pending)", true
	}
	return "", "", false
}

func orUnknown(reason string) string {
	if reason == "" {
		return "Failed"
	}
	return reason
}

func hearthAlerts(src AlertSources) []hud.Alert {
	out := []hud.Alert{}
	pending := 0
	routine := routineOf(src)
	for _, p := range src.Pods {
		kind, text, ok := hearthAlert(p)
		if !ok {
			continue
		}
		out = append(out, hud.Alert{Kind: kind, At: hearthSpot(src, p, pending), Text: text, Key: "pod:" + p.Namespace + "/" + p.Name, Routine: routine.hearth(kind, p)})
		if kind == hud.AlertPending {
			pending++
		}
	}
	return out
}

func stormIsle(src AlertSources) (view.Island, bool) {
	storm := src.State.Storm
	if storm.Island == "" {
		return view.Island{}, false
	}
	for _, is := range src.Site.Islands {
		if is.ID == storm.Island {
			return is, true
		}
	}
	return view.Island{}, false
}

func keepAlerts(src AlertSources) []hud.Alert {
	storm, stormy := stormIsle(src)
	routine := routineOf(src)
	out := []hud.Alert{}
	for _, s := range src.Site.Structures {
		if stormy && onIsle(src.Site, s, storm.ID) {
			continue
		}
		if a, ok := structureAlert(s, src.Nodes); ok {
			a.Key, a.Routine = "structure:"+s.ID, routine.structure(s)
			out = append(out, a)
		}
	}
	return out
}

func onIsle(site view.Site, s view.Structure, isle string) bool {
	c := centre(s)
	is, ok := site.IslandAt(int(c[0]), int(c[1]))
	return s.Kind == view.StructKeep && ok && is.ID == isle
}

func darkText(s view.Structure) string {
	if s.Kind == view.StructKeep {
		return ui.Bridge(" has gone dark", " is NotReady")
	}
	return " has gone dark"
}

func structureAlert(s view.Structure, nodes map[string]corev1.ConditionStatus) (hud.Alert, bool) {
	c := centre(s)
	switch {
	case s.State == view.StateDark:
		return hud.Alert{Kind: hud.AlertDark, At: c, Text: s.Label + darkText(s)}, true
	case s.State == view.StateRuined:
		return hud.Alert{Kind: hud.AlertDark, At: c, Text: s.Label + " lies in ruins"}, true
	case s.Kind != view.StructKeep || s.State != view.StateReady:
		return hud.Alert{}, false
	}
	text, bad := keepTrouble(s, nodes)
	return hud.Alert{Kind: hud.AlertKeep, At: c, Text: text}, bad
}

func keepTrouble(s view.Structure, nodes map[string]corev1.ConditionStatus) (string, bool) {
	switch nodes[s.Object.Name] {
	case corev1.ConditionFalse:
		return s.Label + " is NotReady", true
	case corev1.ConditionUnknown:
		return s.Label + " is unreachable (Ready Unknown)", true
	}
	return "", false
}

func realmAlerts(src AlertSources) []hud.Alert {
	out := []hud.Alert{}
	if is, ok := stormIsle(src); ok {
		text := fmt.Sprintf("A storm cuts %s off: its Keeps are unreachable for %s", is.Name, hud.Duration(src.State.Storm.TimeLeft))
		out = append(out, hud.Alert{Kind: hud.AlertStorm, At: [2]float64{float64(is.Center.X), float64(is.Center.Y)}, Text: text, Key: "storm:" + is.ID})
	}
	if o := src.Omen; o != nil {
		p := model.PodView{Name: o.Pod}
		for _, q := range src.Pods {
			if q.Name == o.Pod {
				p = q
			}
		}
		out = append(out, hud.Alert{Kind: hud.AlertImp, At: hearthSpot(src, p, 0), Text: "The Simian Imp is running at " + o.Pod, Key: "imp:" + o.Pod})
	}
	if phase := src.State.Siege.Phase; phase != "" {
		text := "The Rush Tide floods the quay"
		if phase == "countdown" {
			text = "The Rush Tide gathers at sea"
		}
		out = append(out, hud.Alert{Kind: hud.AlertSiege, At: TideSpot(src.Site), Text: text, Key: "siege"})
	}
	return out
}
