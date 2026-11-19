package app

import (
	"maps"
	"slices"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	errandLag      = 1.5
	MaxActive      = 24
	maxScribbles   = 2
	systemNS       = "kube-system"
	assignedPrefix = " to "
	createdPrefix  = "Created pod: "
)

var containerPrefixes = []string{"Started container ", "Stopping container ", "Container "}

var ErrandGaps = map[string]float64{
	world.ErrandSchedule: .35, world.ErrandRaise: .35, world.ErrandLight: .25, world.ErrandDouse: .25,
	world.ErrandScribble: .3, world.ErrandFallen: .1,
}

type Cue struct {
	Kind      string
	Role      string
	Namespace string
	Pod       string
	Node      string
	Container string
}

type podMark struct {
	name   string
	sprite string
	tier   int
	up     bool
}

type Town struct {
	Primed bool `json:"-"`
	seen   map[string]bool
	marks  map[string]podMark
	writes uint64
	next   map[string]float64
	Active []world.Errand `json:"-"`
	wards  map[string]bool
}

func CueOf(e *corev1.Event) (Cue, bool) {
	ns, pod, host, comp := e.InvolvedObject.Namespace, e.InvolvedObject.Name, e.Source.Host, e.Source.Component
	hearth := ns != systemNS
	switch {
	case e.Reason == "Scheduled" && comp == "default-scheduler":
		i := strings.LastIndex(e.Message, assignedPrefix)
		if i < 0 {
			return Cue{}, false
		}
		return Cue{Kind: world.ErrandSchedule, Role: "quartermaster", Namespace: ns, Pod: pod, Node: e.Message[i+len(assignedPrefix):]}, true
	case e.Reason == "SuccessfulCreate" && comp == "replicaset-controller":
		name, ok := strings.CutPrefix(e.Message, createdPrefix)
		return Cue{Kind: world.ErrandRaise, Role: "steward", Namespace: ns, Pod: name}, ok && hearth
	case e.Reason == "Started" && comp == "kubelet":
		return Cue{Kind: world.ErrandLight, Role: "reeve", Namespace: ns, Pod: pod, Node: host, Container: containerOfEvent(e)}, hearth
	case e.Reason == "Killing" && comp == "kubelet":
		return Cue{Kind: world.ErrandDouse, Role: "reeve", Namespace: ns, Pod: pod, Node: host, Container: containerOfEvent(e)}, hearth
	}
	return Cue{}, false
}

func containerOfEvent(e *corev1.Event) string {
	path := e.InvolvedObject.FieldPath
	if open := strings.Index(path, "{"); open >= 0 && strings.HasSuffix(path, "}") {
		return path[open+1 : len(path)-1]
	}
	return ContainerOf(e.Message)
}

func ContainerOf(message string) string {
	for _, prefix := range containerPrefixes {
		if rest, ok := strings.CutPrefix(message, prefix); ok {
			if fields := strings.Fields(rest); len(fields) > 0 {
				return strings.TrimSuffix(fields[0], ",")
			}
		}
	}
	return ""
}

func (t *Town) Slot(kind string, now float64) (float64, bool) {
	if t.next == nil {
		t.next = map[string]float64{}
	}
	start := max(now, t.next[kind])
	if start-now > errandLag || len(t.Active) >= MaxActive {
		return 0, false
	}
	t.next[kind] = start + ErrandGaps[kind]
	return start, true
}

func (t *Town) send(e world.Errand, now float64) {
	t.Active = LiveErrands(t.Active, now)
	if e.Pod != "" && BusyWith(t.Active, e.Kind, e.Pod) {
		return
	}
	start, ok := t.Slot(e.Kind, now)
	if !ok {
		return
	}
	e.Start = start
	if e.Life <= 0 {
		e.Life = world.ErrandLife(e.Kind, e.From, e.To)
	}
	t.Active = append(t.Active, e)
}

func LiveErrands(list []world.Errand, now float64) []world.Errand {
	out := make([]world.Errand, 0, len(list))
	for _, e := range list {
		if now <= e.Start+e.Life {
			out = append(out, e)
		}
	}
	return out
}

func BusyWith(list []world.Errand, kind, pod string) bool {
	for _, e := range list {
		if e.Kind == kind && e.Pod == pod {
			return true
		}
	}
	return false
}

func freshEvents(events []*corev1.Event, seen map[string]bool) ([]*corev1.Event, map[string]bool) {
	next := make(map[string]bool, len(events))
	fresh := []*corev1.Event{}
	for _, e := range events {
		key := string(e.UID) + "/" + e.ResourceVersion
		if !seen[key] {
			fresh = append(fresh, e)
		}
		next[key] = true
	}
	sort.SliceStable(fresh, func(i, j int) bool { return EventTime(fresh[i]) < EventTime(fresh[j]) })
	return fresh, next
}

func walkerUp(p model.PodView) bool {
	return !p.System && !p.Terminating && (p.State == model.HearthReady || p.State == model.HearthGuttering)
}

func (t *Town) WatchPods(pods []model.PodView, homes func(string) (world.Vec, bool), now float64, live bool) {
	next := make(map[string]podMark, len(pods))
	for _, p := range pods {
		next[p.UID] = podMark{name: p.Name, sprite: p.Sprite, tier: p.Tier, up: walkerUp(p)}
	}
	for _, uid := range slices.Sorted(maps.Keys(t.marks)) {
		before := t.marks[uid]
		if after, ok := next[uid]; live && before.up && (!ok || !after.up) {
			t.fall(before, homes, now)
		}
	}
	t.marks = next
}

func (t *Town) fall(m podMark, homes func(string) (world.Vec, bool), now float64) {
	home, ok := homes(m.name)
	if !ok {
		return
	}
	at := world.Vec{X: home.X + .45, Y: home.Y + .8}
	t.send(world.Errand{Kind: world.ErrandFallen, Role: m.sprite, From: at, To: at, Pod: m.name, Tier: m.tier}, now)
}

func doorFor(site view.Site, kind string) (world.Vec, bool) {
	s, ok := world.FirstOfKind(site, kind, true)
	if !ok {
		return world.Vec{}, false
	}
	return world.Porch(s), true
}
