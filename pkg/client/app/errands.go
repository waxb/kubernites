package app

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/view"
)

func (s *Session) Errands() []world.Errand { return s.Town.Active }

func (s *Session) trackTown() {
	t := &s.Town
	version := s.Cache.Version()
	events := s.Cache.Events()
	if !t.Primed {
		_, t.seen = freshEvents(events, nil)
		t.WatchPods(s.Pods, s.Renderer.Home, s.Clocks.World, false)
		t.writes, t.Primed = version, true
		return
	}
	t.Active = LiveErrands(t.Active, s.Clocks.World)
	s.watchWards()
	if version == t.writes {
		return
	}
	writes := version - t.writes
	t.writes = version
	var fresh []*corev1.Event
	fresh, t.seen = freshEvents(events, t.seen)
	for _, e := range fresh {
		if cue, ok := CueOf(e); ok {
			s.dispatch(cue)
		}
	}
	t.WatchPods(s.Pods, s.Renderer.Home, s.Clocks.World, true)
	s.Scribble(writes)
	s.decree(writes)
}

func (s *Session) dispatch(c Cue) {
	e, ok := s.ErrandFor(c)
	if ok {
		s.Town.send(e, s.Clocks.World)
	}
}

func (s *Session) ErrandFor(c Cue) (world.Errand, bool) {
	e := world.Errand{Kind: c.Kind, Role: c.Role, Pod: c.Pod}
	switch c.Kind {
	case world.ErrandSchedule:
		from, okFrom := doorFor(s.Site, view.StructQuartermaster)
		keep, okKeep := world.KeepForNode(s.Site, c.Node)
		e.From, e.To = from, s.scheduleTarget(c.Pod, keep)
		return e, okFrom && okKeep
	case world.ErrandRaise:
		from, ok := doorFor(s.Site, view.StructStewards)
		e.From, e.To = from, world.PendingSpot(s.Site, 0)
		return e, ok
	}
	keep, okKeep := world.KeepForNode(s.Site, c.Node)
	home, okHome := s.Renderer.Home(c.Pod)
	e.From, e.To = world.Porch(keep), world.Vec{X: home.X + .55, Y: home.Y + .95}
	return e, okKeep && okHome && s.HearthFire(c)
}

func (s *Session) HearthFire(c Cue) bool {
	obj, ok := s.Cache.Get(store.KindPod, c.Namespace, c.Pod)
	pod, typed := obj.(*corev1.Pod)
	if !ok || !typed || len(pod.Spec.Containers) == 0 {
		return c.Container != ""
	}
	return c.Container == pod.Spec.Containers[0].Name
}

func (s *Session) Scribble(writes uint64) {
	at, ok := doorFor(s.Site, view.StructCodex)
	if !ok {
		return
	}
	for i := uint64(0); i < min(writes, maxScribbles); i++ {
		s.Town.send(world.Errand{Kind: world.ErrandScribble, Role: "clerk", From: at, To: at}, s.Clocks.World)
	}
}
