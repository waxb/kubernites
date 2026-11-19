package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/termview"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

type stoneScreen struct {
	view *termview.View
	term *kubectltree.Terminal
}

func newStoneScreen(cols, rows int) *stoneScreen {
	s := &stoneScreen{}
	s.term = kubectltree.NewTerminal(cols, rows, s)
	s.view = termview.New(cols, rows, s.term.Type)
	return s
}

func (s *stoneScreen) Write(p []byte) (int, error) {
	return s.view.Write(p)
}

func (s *stoneScreen) Live() bool {
	return s.term.Live()
}

func (s *stoneScreen) Resize(cols, rows int) {
	s.view.Resize(cols, rows)
	s.term.Resize(cols, rows)
}

func (s *stoneScreen) Row(y int) []termview.Cell {
	return s.view.Row(y)
}

func (s *stoneScreen) Cursor() (int, int, bool) {
	return s.view.Cursor()
}

func (s *stoneScreen) Key(k termview.Key) {
	s.view.Key(k)
}

func (s *stoneScreen) Type(text string) {
	s.view.Type(text)
}

func (s *stoneScreen) Text() []string {
	return s.view.Text()
}

func (s *stoneScreen) close() {
	s.term.Close()
	s.view.Close()
}

func stoneSession(s kubectltree.Session) hud.StoneSession {
	kind := "Pod"
	return hud.StoneSession{Verb: s.Verb, Kind: kind, Namespace: s.Namespace, Name: s.Pod, Container: s.Container, Command: s.Command, TTY: s.TTY, Opened: s.Opened, Ended: string(s.Ended), Code: s.Code, Err: s.Err}
}
