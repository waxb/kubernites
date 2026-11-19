package app

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	AwayIdle   = 120.0
	awayMinSec = 30.0
	awayTitle  = "WHILE YOU WERE AWAY"
	awayMore   = 3
)

type awaySpan struct {
	on      bool
	Since   float64 `json:"-"`
	Gold    float64 `json:"-"`
	wisdom  float64
	renown  float64
	Served  float64 `json:"-"`
	ProcsAt float64 `json:"-"`
}

func (s *Session) StartAway() awaySpan {
	st := s.State
	return awaySpan{on: true, Since: st.RealmTime, Gold: st.Gold, wisdom: st.Wisdom, renown: st.Renown, Served: st.Pilgrims.TotalServed, ProcsAt: latestProclamation(st.Proclamations, 0)}
}

func acted(in hud.Input) bool {
	return in.Click || in.DoubleClick || in.RightClick || in.Wheel != 0 || in.WheelX != 0 || len(in.Keys) > 0 || len(in.Chars) > 0
}

func (s *Session) NoteAct(in hud.Input) {
	if !acted(in) {
		return
	}
	if s.Away.on && s.Clock-s.LastActAt >= AwayIdle {
		s.awayDue = true
	}
	s.LastActAt = s.Clock
	if !s.awayDue {
		s.Away = s.StartAway()
	}
}

func (s *Session) StepAway() {
	if !s.Away.on {
		s.Away, s.LastActAt = s.StartAway(), s.Clock
		return
	}
	if s.awayDue && !s.Paused && !s.Auto.Held {
		s.awayDue = false
		s.endAway()
	}
}

func (s *Session) endAway() {
	span := s.Away
	s.Away = s.StartAway()
	if !span.on || s.State.RealmTime-span.Since < awayMinSec {
		return
	}
	if text := s.awayText(span); text != "" {
		s.Hud.Result(awayTitle+" ("+hud.Duration(s.State.RealmTime-span.Since)+")", text)
	}
}

func (s *Session) awayText(span awaySpan) string {
	st := s.State
	lines := []string{}
	if d := st.Gold - span.Gold; d >= 1 || d <= -1 {
		lines = append(lines, "Gold "+signedAmount(d)+".")
	}
	if n := int(st.Pilgrims.TotalServed - span.Served); n > 0 {
		lines = append(lines, fmt.Sprintf("%d boxes carried.", n))
	}
	ends, troubles := s.awayEvents(span.ProcsAt)
	if len(ends) > 0 {
		lines = append(lines, "Steps done: "+listMore(ends, awayMore)+".")
	}
	if len(troubles) > 0 {
		lines = append(lines, "Troubles: "+listMore(troubles, awayMore)+".")
	}
	if d := st.Wisdom - span.wisdom; d >= 1 {
		lines = append(lines, fmt.Sprintf("Wisdom +%.0f.", d))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, " ")
}

func (s *Session) awayEvents(from float64) ([]string, []string) {
	var ends, troubles []string
	for _, p := range s.State.Proclamations {
		if p.T <= from {
			continue
		}
		if end, ok := stepEndOf(p); ok {
			ends = append(ends, end.title)
			continue
		}
		if p.Kind == "calamity" && !strings.HasPrefix(p.Text, view.CitadelText) {
			troubles = append(troubles, Headline(strings.TrimSpace(p.Text)))
		}
	}
	return ends, troubles
}

func listMore(items []string, limit int) string {
	if len(items) <= limit {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:limit], ", ") + fmt.Sprintf(" and %d more", len(items)-limit)
}

func signedAmount(v float64) string {
	if v < 0 {
		return "-" + hud.Compact(-v)
	}
	return "+" + hud.Compact(v)
}
