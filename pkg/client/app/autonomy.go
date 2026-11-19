package app

import (
	"maps"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	burstWindow    = 1.5
	MaxCallouts    = 2
	CalloutsTaught = 3
	CalloutQuiet   = 90.0
)

var actorPortraits = map[string]string{
	"steward": "steward", "stewards": "steward", "replicaset-controller": "steward", "controller-manager": "steward",
	"node-controller": "steward", "quartermaster": "quartermaster", "scheduler": "quartermaster", "default-scheduler": "quartermaster",
	"reeve": "reeve", "kubelet": "reeve", "horn": "warhorn", "war-horn": "warhorn", "horizontal-pod-autoscaler": "warhorn",
}

type Burst struct {
	Kind, Actor, text string       `json:"-"`
	Object            model.ObjRef `json:"-"`
	Start             float64      `json:"-"`
	Count             int          `json:"-"`
}

type Autonomy struct {
	Primed  bool `json:"-"`
	seen    map[string]bool
	Bursts  []Burst `json:"-"`
	told    map[string]int
	burstAt map[string]float64
}

func ActorPortrait(actor string) string {
	if p, ok := actorPortraits[strings.ToLower(actor)]; ok {
		return p
	}
	return "steward"
}

func (a Autonomy) Step(acts []model.Autonomy, now float64) Autonomy {
	next := Autonomy{Primed: true, seen: make(map[string]bool, len(acts)), told: maps.Clone(a.told), burstAt: maps.Clone(a.burstAt)}
	if next.told == nil {
		next.told, next.burstAt = map[string]int{}, map[string]float64{}
	}
	for _, b := range a.Bursts {
		if now-b.Start < world.CalloutLife {
			next.Bursts = append(next.Bursts, b)
		}
	}
	for _, act := range acts {
		key := act.Key()
		next.seen[key] = true
		if a.seen[key] || !a.Primed {
			continue
		}
		switch {
		case next.bursting(act.Kind, now):
		case next.tellable(act.Kind, now):
			next.told[act.Kind]++
			next.burstAt[act.Kind] = now
		default:
			continue
		}
		next.Bursts = joinBurst(next.Bursts, act, now)
	}
	return next
}

func (a Autonomy) bursting(kind string, now float64) bool {
	at, ok := a.burstAt[kind]
	return ok && now-at < burstWindow
}

func (a Autonomy) tellable(kind string, now float64) bool {
	at, ok := a.burstAt[kind]
	return a.told[kind] < CalloutsTaught || !ok || now-at >= CalloutQuiet
}

func joinBurst(bursts []Burst, act model.Autonomy, now float64) []Burst {
	for i := len(bursts) - 1; i >= 0; i-- {
		b := bursts[i]
		if b.Kind == act.Kind && b.Actor == act.Actor && now-b.Start < burstWindow {
			out := append([]Burst(nil), bursts...)
			out[i] = Burst{Kind: b.Kind, Actor: b.Actor, text: act.Text, Object: act.Object, Start: now, Count: b.Count + 1}
			return out
		}
	}
	return append(append([]Burst(nil), bursts...), Burst{Kind: act.Kind, Actor: act.Actor, text: act.Text, Object: act.Object, Start: now, Count: 1})
}

func (a Autonomy) Callouts(now float64, locate func(model.ObjRef) (world.Vec, bool)) []world.Callout {
	out := []world.Callout{}
	for i := len(a.Bursts) - 1; i >= 0 && len(out) < MaxCallouts; i-- {
		b := a.Bursts[i]
		at, ok := locate(b.Object)
		if !ok {
			continue
		}
		out = append(out, world.Callout{Portrait: ActorPortrait(b.Actor), Text: b.text, At: at, Age: now - b.Start, Count: b.Count})
	}
	return out
}

func Locate(site view.Site, pods []string, home func(string) (world.Vec, bool)) func(model.ObjRef) (world.Vec, bool) {
	return func(o model.ObjRef) (world.Vec, bool) {
		switch o.Kind {
		case "Pod":
			if v, ok := home(o.Name); ok {
				return world.Vec{X: v.X + .5, Y: v.Y + .5}, true
			}
		case "Node":
			if k, ok := world.KeepForNode(site, o.Name); ok {
				return world.Anchor(k), true
			}
		case "ReplicaSet", "HorizontalPodAutoscaler":
			for _, p := range pods {
				if strings.HasPrefix(p, o.Name+"-") {
					if v, ok := home(p); ok {
						return world.Vec{X: v.X + .5, Y: v.Y + .5}, true
					}
				}
			}
		}
		if s, ok := world.FirstOfKind(site, view.StructStewards, true); ok {
			return world.Door(s), true
		}
		return world.Vec{}, false
	}
}

var objectKinds = map[string]string{
	"pod": "Pod", "node": "Node", "replicaset": "ReplicaSet", "rs": "ReplicaSet", "hpa": "HorizontalPodAutoscaler",
	"horizontalpodautoscaler": "HorizontalPodAutoscaler", "service": "Service", "svc": "Service",
}

func ObjectOf(ref string) model.ObjRef {
	parts := strings.Split(ref, "/")
	kind := objectKinds[strings.ToLower(parts[0])]
	if kind == "" {
		kind = parts[0]
	}
	switch len(parts) {
	case 2:
		return model.ObjRef{Kind: kind, Name: parts[1]}
	case 3:
		return model.ObjRef{Kind: kind, Namespace: parts[1], Name: parts[2]}
	}
	return model.ObjRef{Kind: kind}
}

func AutonomyOf(st view.GameState) ([]model.Autonomy, int) {
	out := make([]model.Autonomy, 0, len(st.Autonomy))
	for _, a := range st.Autonomy {
		out = append(out, model.Autonomy{Kind: a.Kind, Actor: a.Actor, Text: a.Text, At: a.At, Object: ObjectOf(a.Object)})
	}
	return out, st.AutonomyToday
}

func (s *Session) autonomyNow() ([]model.Autonomy, int) {
	if s.AutonomyFixture != nil {
		return s.AutonomyFixture(s.Clocks.World)
	}
	return AutonomyOf(s.State)
}

func (s *Session) trackAutonomy() {
	acts, today := s.autonomyNow()
	s.Autonomy = s.Autonomy.Step(acts, s.Clocks.World)
	s.autonomyToday = today
}

func (s *Session) callouts() []world.Callout {
	names := make([]string, 0, len(s.Pods))
	for _, p := range s.Pods {
		names = append(names, p.Name)
	}
	return s.Autonomy.WithoutForemen(s.Crews).Callouts(s.Clocks.World, Locate(s.Site, names, s.Renderer.Home))
}
