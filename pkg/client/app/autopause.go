package app

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	pauseAskLife  = 30.0
	DevPauseSpeed = 5.0
	pauseBannerOf = "PAUSED: "
	pauseResume   = " Press Space to go on."
	troubleResume = "Close or fix its card, or press Space, to go on."
	choiceResume  = "Answer it on the rail, or press Space, to go on."
	PauseGap      = 20.0
)

var pauseChoiceIDs = map[string]bool{"decision:cmd:chronicles": true, "decision:cmd:siege": true, "decision:" + hud.ContractsOpen: true}

type pendingAsk struct {
	ask hud.PauseAsk
	at  float64
}

type AutoPause struct {
	Primed  bool            `json:"-"`
	Done    map[string]bool `json:"-"`
	Choices map[string]bool `json:"-"`
	Pending []pendingAsk    `json:"-"`
	Held    bool            `json:"-"`
	freedAt float64
	cause   []hud.PauseAsk
	seen    map[string]bool
}

func (s *Session) pauseTrouble(f hud.Frame) bool {
	return !s.Paused && (s.Auto.freedAt == 0 || s.Clock-s.Auto.freedAt >= PauseGap) && !hud.Holds(s.Hud.Overlay()) && !s.ScrollOpen() && !s.IsUnguided() && !s.Hud.PauseBlocked(f)
}

func (s *Session) StepAutoPause(f hud.Frame) {
	asks := s.Hud.TakePauseAsks()
	if !s.AutoPauseOn && s.pauseTap == nil {
		return
	}
	if s.Auto.Held && s.Paused && s.pauseAnswered() {
		s.Paused = false
	}
	if s.Auto.Held && !s.Paused {
		s.Auto.Held, s.Auto.freedAt, s.Auto.cause = false, s.Clock, nil
		s.Hud.PinPause("", "")
	}
	if !s.Auto.Primed {
		s.Auto.Primed = true
		s.Auto.Done = map[string]bool{}
		for _, c := range s.State.Calamities {
			s.Auto.Done["trouble:"+c.ID] = true
		}
		s.Auto.Choices = map[string]bool{}
		for _, n := range s.Rail.Notices {
			s.Auto.Choices[n.ID] = true
		}
		return
	}
	asks = append(asks, s.newChoices(f.Notices)...)
	s.Auto.Pending = s.keepPending(asks)
	if len(s.Auto.Pending) == 0 || !s.pauseTrouble(f) {
		return
	}
	ready := s.readyAsks()
	s.Auto.Pending = nil
	switch {
	case len(ready) == 0:
	case !s.AutoPauseOn:
		s.pauseTap(ready)
		s.Auto.freedAt = max(s.Clock, 1e-9)
	default:
		s.autoPause(ready)
	}
}

func (s *Session) newChoices(shown []hud.Notice) []hud.PauseAsk {
	seen := map[string]bool{}
	var out []hud.PauseAsk
	for _, n := range shown {
		seen[n.ID] = true
		if n.Kind != hud.NoticeDecision || !pauseChoiceIDs[n.ID] || s.Auto.Choices[n.ID] {
			continue
		}
		out = append(out, hud.PauseAsk{Kind: hud.AskChoice, Key: n.ID, Title: n.Title, Why: n.Detail})
	}
	s.Auto.Choices = seen
	return out
}

func (s *Session) keepPending(fresh []hud.PauseAsk) []pendingAsk {
	live := map[string]bool{}
	for _, c := range s.State.Calamities {
		live["trouble:"+c.ID] = c.Mended == ""
	}
	var out []pendingAsk
	for _, p := range s.Auto.Pending {
		if s.Clock-p.at > pauseAskLife || p.ask.Kind == hud.AskTrouble && !live[p.ask.Key] {
			continue
		}
		out = append(out, p)
	}
	for _, a := range fresh {
		if s.Auto.Done[a.Key] {
			continue
		}
		s.Auto.Done[a.Key] = true
		out = append(out, pendingAsk{ask: a, at: s.Clock})
	}
	return out
}

func (s *Session) readyAsks() []hud.PauseAsk {
	if s.State.Age == view.AgeDark && s.State.Speed < DevPauseSpeed {
		return nil
	}
	var out []hud.PauseAsk
	for _, p := range s.Auto.Pending {
		if hud.PauseCovers(s.Hud.PauseFor(), p.ask.Kind) {
			out = append(out, p.ask)
		}
	}
	return out
}

func (s *Session) pauseAnswered() bool {
	if s.Auto.seen == nil {
		s.Auto.seen = map[string]bool{}
	}
	for _, a := range s.Auto.cause {
		open := s.askOpen(a)
		if open {
			s.Auto.seen[a.Key] = true
		}
		if !open && s.Auto.seen[a.Key] {
			return true
		}
	}
	return false
}

func (s *Session) askOpen(a hud.PauseAsk) bool {
	if a.Kind == hud.AskTrouble {
		id := strings.TrimPrefix(a.Key, "trouble:")
		return s.Hud.TroubleOpen(id) || s.troubleLive(id) && s.Hud.TroubleSetAside(id)
	}
	for _, n := range s.Rail.Notices {
		if n.ID == a.Key {
			return true
		}
	}
	return false
}

func (s *Session) autoPause(asks []hud.PauseAsk) {
	s.Paused, s.Auto.Held = true, true
	s.Auto.cause, s.Auto.seen = asks, map[string]bool{}
	title, _ := PauseReason(asks)
	hint := troubleResume
	if asks[0].Kind == hud.AskChoice {
		hint = choiceResume
	}
	s.Hud.PinPause(title, hint)
	for _, a := range asks {
		if a.At != nil {
			s.GlideTo(a.At[0], a.At[1])
			break
		}
	}
	if s.Record != nil {
		s.Record("autopause", map[string]any{"why": title, "count": len(asks)})
	}
}

func PauseReason(asks []hud.PauseAsk) (string, string) {
	if len(asks) == 1 {
		why := strings.TrimSpace(asks[0].Why)
		if why != "" && !strings.HasSuffix(why, ".") && !strings.HasSuffix(why, "?") && !strings.HasSuffix(why, "!") {
			why += "."
		}
		return pauseBannerOf + strings.ToUpper(asks[0].Title), strings.TrimSpace(why + pauseResume)
	}
	title, cause := PauseReason(asks[:1])
	return fmt.Sprintf("%s (+%d MORE)", title, len(asks)-1), cause
}

func (g *Game) cyclePausePolicy() {
	g.Settings.PauseFor = hud.NextPauseMode(g.Settings.PauseFor)
	if g.Sess != nil {
		g.Sess.Hud.SetPauseFor(g.Settings.PauseFor)
	}
	if err := SaveSettings(g.SettingsPath(), g.Settings); err != nil {
		g.Log.Warn("could not save settings", "err", err)
	}
}

func (s *Session) troubleLive(id string) bool {
	for _, c := range s.State.Calamities {
		if c.ID == id {
			return c.Mended == ""
		}
	}
	return false
}
