package app

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	DecisionBeat = 20.0

	NewsLife    = 45.0
	HeadlineMax = 44
	openedNear  = .5
	listMax     = 3
)

var newsIcons = map[string]string{"info": "icon/chronicle", "calamity": "icon/alert"}

const decreeIcon = "icon/decree"

var railSounds = map[string]sound.SFX{hud.NoticeDanger: sound.SFXCalamity, hud.NoticeDecision: sound.SFXReveal}

type RailInput struct {
	Now   float64
	Frame hud.Frame
	Quiet map[string]bool
}

type Rail struct {
	Primed    bool `json:"-"`
	after     float64
	news      []hud.Notice
	since     map[string]float64
	dismissed map[string]bool
	Alarms    Alarms `json:"-"`
	decided   bool
	decidedAt float64
	Notices   []hud.Notice
	Arrived   []string
}

func latestProclamation(procs []view.Proclamation, floor float64) float64 {
	out := floor
	for _, p := range procs {
		out = max(out, p.T)
	}
	return out
}

func (r Rail) Step(in RailInput) Rail {
	next := Rail{Primed: true, after: r.after, since: map[string]float64{}, dismissed: map[string]bool{},
		Alarms: r.Alarms.Step(in.Frame.State.RealmTime, in.Now, in.Frame.Alerts)}
	procs := in.Frame.State.Proclamations
	if !r.Primed {
		next.after = latestProclamation(procs, 0)
	}
	calm := in.Frame.State.Calm > 0
	next.news = MergeNews(append(freshNews(r.news, in.Now), PlaceNews(NewsNotices(calmNews(procs, calm), next.after, in.Now, in.Frame.Site), in.Frame.Places)...))
	next.after = latestProclamation(procs, next.after)
	next.news = slices.DeleteFunc(next.news, func(n hud.Notice) bool { return gildedNews(n) && !gildedDocked(in.Frame.State, n.Told) })
	for i, n := range next.news {
		if in.Quiet[n.ID] && !n.Quiet {
			next.news[i].Quiet, next.news[i].Since = true, n.Since-hud.RailPulseFor
		}
		if gildedNews(n) {
			next.news[i].Timely, next.news[i].Expires = true, max(n.Expires, in.Now+NewsLife)
		}
	}
	live := append(DangerNotices(next.Alarms.Dangers()), DecisionNotices(in.Frame)...)
	if calm {
		live = nil
	}
	live, next.decided, next.decidedAt = r.paced(live, in.Now)
	visible := make([]hud.Notice, 0, len(live)+len(next.news))
	for _, n := range live {
		since, seen := r.since[n.ID]
		if !seen {
			since = in.Now
		}
		next.since[n.ID] = since
		if r.dismissed[n.ID] {
			next.dismissed[n.ID] = true
			continue
		}
		if !seen && r.Primed {
			var loud bool
			if next.Alarms, loud = next.Alarms.arrive(n, in.Now); loud {
				next.Arrived = append(next.Arrived, n.Kind)
			}
		}
		n.Since = since
		visible = append(visible, n)
	}
	for _, n := range next.news {
		if gildedNews(n) && !r.dismissed[n.ID] {
			visible = append(visible, n)
		}
	}
	next.Notices = hud.SortNotices(visible)
	return next
}

func (r Rail) paced(live []hud.Notice, now float64) ([]hud.Notice, bool, float64) {
	out, decided, at := make([]hud.Notice, 0, len(live)), r.decided, r.decidedAt
	for _, n := range live {
		_, seen := r.since[n.ID]
		if !r.Primed || seen || n.Kind != hud.NoticeDecision {
			out = append(out, n)
			continue
		}
		if decided && now < at+DecisionBeat {
			continue
		}
		decided, at = true, now
		out = append(out, n)
	}
	return out, decided, at
}

func calmNews(procs []view.Proclamation, calm bool) []view.Proclamation {
	if !calm {
		return procs
	}
	out := []view.Proclamation{}
	for _, p := range procs {
		if p.Kind == "age" {
			out = append(out, p)
		}
	}
	return out
}

func gildedNews(n hud.Notice) bool { return strings.HasPrefix(n.Detail, GildedText) }

func gildedDocked(st view.GameState, told float64) bool {
	ship := st.Golden.Ship
	return ship != nil && told >= ship.Docked-1
}

func freshNews(news []hud.Notice, now float64) []hud.Notice {
	out := make([]hud.Notice, 0, len(news))
	for _, n := range news {
		if now < n.Expires {
			out = append(out, n)
		}
	}
	return out
}

func (r Rail) Handle(id, command string) Rail {
	next := r
	next.dismissed = map[string]bool{}
	for k, v := range r.dismissed {
		next.dismissed[k] = v
	}
	news := strings.HasPrefix(id, hud.NoticeNews+":")
	switch {
	case news:
		next.news = without(r.news, id)
	case command == hud.NoticeDismiss:
		next.dismissed[id] = true
	default:
		return next
	}
	next.Notices = without(r.Notices, id)
	return next
}

func without(ns []hud.Notice, id string) []hud.Notice {
	out := make([]hud.Notice, 0, len(ns))
	for _, n := range ns {
		if n.ID != id {
			out = append(out, n)
		}
	}
	return out
}

func (s *Session) railFrame() hud.Frame {
	return hud.Frame{State: s.State, Pods: s.Pods, Site: s.Site, Alerts: s.Alerts(), Guide: s.Guide, Caesar: s.caesar}
}

func (s *Session) RailSounds() []sound.SFX {
	out := []sound.SFX{}
	played := map[string]bool{}
	for _, kind := range s.Rail.Arrived {
		if fx, ok := railSounds[kind]; ok && !played[kind] {
			played[kind] = true
			out = append(out, fx)
		}
	}
	s.Rail.Arrived = nil
	return out
}

func newsID(p view.Proclamation) string {
	return fmt.Sprintf("news:%.3f:%08x", p.T, model.Hash(p.Kind+strings.TrimSpace(p.Text)))
}

func NewsNotices(procs []view.Proclamation, after, now float64, site view.Site) []hud.Notice {
	out := []hud.Notice{}
	quiet := quietSteps(procs)
	for i, p := range procs {
		if quiet[i] || p.Routine {
			continue
		}
		icon, ok := newsIcons[p.Kind]
		if ContractNews(p) {
			icon, ok = decreeIcon, true
		}
		if OfferNews(p) {
			continue
		}
		if !ok || p.T <= after {
			continue
		}
		text := strings.TrimSpace(p.Text)
		title, detail := Headline(text), text
		if p.Kind == "lesson" && p.Title != "" {
			title, detail = LessonOf(p)
		}
		n := hud.Notice{ID: newsID(p), Kind: hud.NoticeNews, Icon: icon,
			Title: title, Detail: detail, Since: now, Expires: now + NewsLife, Told: p.T}
		if at, ok := newsPlace(site, p.T, text); ok {
			n.At = &at
		}
		out = append(out, n)
	}
	return out
}

var headlineGloss = regexp.MustCompile(` \([^()]*\)`)

func Headline(text string) string {
	text = headlineGloss.ReplaceAllString(text, "")
	cut := len(text)
	for _, sep := range []string{". ", ": ", "! "} {
		if i := strings.Index(text, sep); i > 0 && i < cut {
			cut = i
		}
	}
	head := strings.TrimSuffix(text[:cut], ".")
	if len(head) > HeadlineMax {
		return strings.TrimSpace(head[:HeadlineMax]) + "..."
	}
	return head
}

func newsPlace(site view.Site, t float64, text string) ([2]float64, bool) {
	lower := strings.ToLower(text)
	for _, h := range site.Havens {
		named := h.Name != "" && strings.Contains(lower, strings.ToLower(h.Name))
		opened := h.OpenedAt > 0 && t-h.OpenedAt < openedNear && h.OpenedAt-t < openedNear
		if named || opened {
			return [2]float64{float64(h.Quay.X), float64(h.Quay.Y)}, true
		}
	}
	best, found := view.Structure{}, false
	for _, st := range site.Structures {
		if st.Label != "" && strings.Contains(lower, strings.ToLower(st.Label)) && (!found || len(st.Label) > len(best.Label)) {
			best, found = st, true
		}
	}
	return centre(best), found
}
