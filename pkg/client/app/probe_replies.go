package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

type replyMark struct {
	stone, echoes int
}

func (s *Session) replyMark() replyMark {
	return replyMark{stone: len(s.Hud.StoneLines()), echoes: len(s.Echoes)}
}

func (s *Session) shownSince(m replyMark) []string {
	out := []string{}
	lines := s.Hud.StoneLines()
	for _, l := range lines[min(m.stone, len(lines)):] {
		out = append(out, ui.Name(hud.ConsoleText(l)))
	}
	for _, e := range s.Echoes[min(m.echoes, len(s.Echoes)):] {
		out = append(out, ui.Name(e.Text))
	}
	for _, t := range s.Hud.Toasts() {
		out = append(out, ui.Name(hud.ToastLabel(t)))
	}
	return out
}

func (p *GuideProbe) Acted(a view.Action, done view.Performed, err error) []string {
	var out []string
	onScreenThread(func() {
		m := p.S.replyMark()
		p.S.EchoSpoken(done.Commands, hud.FromButton, done.Message, err)
		p.S.ReplyToasts(a, done.Message, len(done.Commands) > 0, err)
		out = p.S.shownSince(m)
	})
	return out
}

func (p *GuideProbe) Spoke(command string, res view.CommandResult) []string {
	var out []string
	onScreenThread(func() {
		m := p.S.replyMark()
		for _, it := range p.S.Hud.EchoFrom(command, hud.FromObjectiveCard, res.Output, res.OK) {
			p.S.Handle(it)
		}
		out = p.S.shownSince(m)
	})
	return out
}
