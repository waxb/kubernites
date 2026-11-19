package app

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/talk"
	"github.com/waxb/kubernites/pkg/client/world"
)

const (
	insetRate    = 15.0
	workerReread = 0.25
)

type workerCard struct {
	inset  *ebiten.Image
	key    world.InsetKey
	drawn  float64
	view   *hud.WorkerView
	readAt float64
	person string
}

var alarmMarks = map[string]bool{"GUTTERING": true, "CrashLoopBackOff": true}

func (s *Session) workerView() *hud.WorkerView {
	if s.Selection.Kind != world.HitWalker {
		s.worker.view, s.worker.person = nil, ""
		return nil
	}
	key := world.PersonKey(s.Selection.Pod)
	if s.worker.view != nil && s.worker.person == key && s.Clock-s.worker.readAt < workerReread && !s.Selection.Gone {
		return s.worker.view
	}
	s.worker.view, s.worker.person, s.worker.readAt = s.readWorker(), key, s.Clock
	return s.worker.view
}

func (s *Session) readWorker() *hud.WorkerView {
	p := s.Selection.Pod
	if p.Focus == "" {
		p = p.Person(0)
	}
	f := s.WorkerFacts(p)
	id := IdentityOf(p)
	v := &hud.WorkerView{
		Title: strings.ToUpper(p.Profession), Line: hud.KuberniteLine(p), Names: f.Names,
		Mood: talk.Mood(f), Thought: talk.Thinks(id, f), Gone: s.Selection.Gone,
	}
	if f.Names {
		v.Title = p.Image
	}
	if s.worker.key.Person == world.PersonKey(p) {
		v.View = s.worker.inset
	}
	v.Status = talk.Status(f)
	if s.Selection.Gone {
		v.Status = goneWords(f.Names)
	}
	if last := talk.LastState(f); last != "" && f.Names {
		v.Last = last
	}
	if !s.Selection.Gone {
		for _, n := range talk.Needs(f) {
			v.Needs = append(v.Needs, hud.WorkerNeed{Label: n.Label, Value: n.Value, Mark: n.Mark, Alarm: alarmMarks[n.Mark]})
		}
	}
	now := s.realmNow()
	for _, t := range talk.Thoughts(f, s.happenings(p, f), talk.MaxThoughts) {
		v.Recent = append(v.Recent, hud.WorkerThought{Ago: talk.Ago(now-t.At) + " ago", Text: t.Text})
	}
	return v
}

func goneWords(names bool) string {
	if names {
		return "Its Pod is gone. The view holds on its last spot."
	}
	return "Its hut is gone. The view holds where it stood."
}

func (s *Session) drawInset(sc world.Scene) {
	if s.Selection.Kind != world.HitWalker {
		return
	}
	if s.worker.inset == nil {
		s.worker.inset = ebiten.NewImage(world.InsetW, world.InsetH)
	}
	key, ok := s.Renderer.InsetKey(sc)
	if !ok || key == s.worker.key || s.Clock-s.worker.drawn < 1/insetRate {
		return
	}
	if s.Renderer.DrawInset(s.worker.inset, sc) {
		s.worker.key, s.worker.drawn = key, s.Clock
	}
}

func init() {
	windowLabels[hud.PanelWorker] = "Worker card"
}
