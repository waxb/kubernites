package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
)

func (s *Session) EchoSpoken(commands []string, from, msg string, err error) {
	if len(commands) == 0 || s.Live && !s.hints {
		return
	}
	out, ok := msg, err == nil
	if err != nil {
		out = err.Error()
	}
	for i, cmd := range commands {
		s.Hud.Echo(cmd, from)
		answer := ""
		if i == len(commands)-1 {
			answer = out
			s.Hud.EchoAnswer(out, ok)
		}
		s.Echoes = Echo(s.Echoes, hud.EchoIntent(cmd, answer, ok || i < len(commands)-1), s.echoClock())
	}
}

func (s *Session) echoClock() float64 {
	return float64(s.State.FoundedAt.UnixNano())/1e9 + s.State.RealmTime
}
