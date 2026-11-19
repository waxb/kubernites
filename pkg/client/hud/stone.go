package hud

import (
	"image/color"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	LineOutput = iota
	LineError
	LineEcho
	LineHint
	maxStoneLines = 600
	maxInput      = 1024
)

type StoneLine struct {
	Text string
	Kind int
}

func (l StoneLine) color() color.RGBA {
	switch l.Kind {
	case LineError:
		return ui.Palette.Red
	case LineEcho, LineRawEcho:
		return ui.Palette.GoldBright
	case LineHint:
		return ui.Palette.TextDim
	case LineRawError:
		return ui.Palette.Red
	}
	return ui.Palette.Text
}

type cycle struct {
	list  []string
	index int
}

type Stone struct {
	Input   []rune
	Cursor  int
	History []string
	hist    int
	Output  []StoneLine
	Written int
	Scroll  ui.Scroll
	Tall    bool
	away    bool
	cycle   cycle

	systemSeen bool
	Fresh      bool `json:"-"`
	glossed    map[string]bool
	folded     []string
}

const (
	stoneListens = "The Speaking Stone listens. Speak kubectl to the Chancery."
	stoneTab     = "Tab completes a word; Tab Tab cycles the choices."
)

func NewStone() Stone {
	return Stone{Output: []StoneLine{
		{stoneListens, LineHint},
		{"Try: get pods   get pods -A   get nodes   describe node lone-tower   get events   help", LineHint},
		{stoneTab, LineHint},
	}}
}

func LiveGreeting(nodes []string) []string {
	try := "Try: get pods   get pods -A   get nodes   get events   help"
	if len(nodes) > 0 {
		try = "Try: get pods   get pods -A   get nodes   describe node " + slices.Min(nodes) + "   get events   help"
	}
	return []string{stoneListens, try, stoneTab}
}

func ChapterGreetingK8s(number int) []string {
	lines := []string{"A terminal on " + view.TowerName + ": kubectl runs here, and so do the Node's ls, cat and sudo mv, at the prompt " + strings.TrimSpace(NodePrompt)}
	try := "Try: get nodes   get pods -A   describe node " + view.TowerName + "   get events   help"
	if number >= 2 {
		lines = append(lines, "kubectl reads $HOME/.kube/config: Borg copies kubeadm's /etc/kubernetes/admin.conf there once kubeadm init writes it, as kubeadm's guide says.")
	}
	if number >= 3 {
		try = "Try: get nodes   get pods -A   describe pod <pod>   logs <pod>   get events   help"
	}
	return append(lines, try, "Tab completes a word; Tab Tab cycles the choices.")
}

func (s *Stone) Greet(lines []string) {
	out := make([]StoneLine, 0, len(lines))
	for _, l := range lines {
		out = append(out, StoneLine{l, LineHint})
	}
	s.Output, s.Written, s.Scroll = out, 0, ui.Scroll{}
}

func (s *Stone) edited() {
	s.cycle = cycle{}
	s.Fresh = false
}

func (s *Stone) Insert(r rune) {
	if len(s.Input) >= maxInput || r < 32 || r == 127 {
		return
	}
	if s.Fresh {
		s.Input, s.Cursor = nil, 0
	}
	next := make([]rune, 0, len(s.Input)+1)
	next = append(append(append(next, s.Input[:s.Cursor]...), r), s.Input[s.Cursor:]...)
	s.Input, s.Cursor = next, s.Cursor+1
	s.edited()
}

func (s *Stone) Backspace() {
	if s.Cursor == 0 {
		return
	}
	s.Input = append(append([]rune(nil), s.Input[:s.Cursor-1]...), s.Input[s.Cursor:]...)
	s.Cursor--
	s.edited()
}

func (s *Stone) Delete() {
	if s.Cursor < len(s.Input) {
		s.Input = append(append([]rune(nil), s.Input[:s.Cursor]...), s.Input[s.Cursor+1:]...)
	}
	s.edited()
}

func (s *Stone) Move(delta int) {
	s.Cursor = min(max(s.Cursor+delta, 0), len(s.Input))
	s.edited()
}

func (s *Stone) SetInput(text string) {
	s.Input = []rune(text)
	s.Cursor = len(s.Input)
	s.edited()
}

func (s *Stone) Recall(delta int) {
	if len(s.History) == 0 {
		return
	}
	s.hist = min(max(s.hist+delta, 0), len(s.History))
	text := ""
	if s.hist < len(s.History) {
		text = s.History[s.hist]
	}
	s.SetInput(text)
}

func (s *Stone) Submit() string {
	cmd := strings.TrimSpace(string(s.Input))
	s.SetInput("")
	if cmd != "" && (len(s.History) == 0 || s.History[len(s.History)-1] != cmd) {
		s.History = append(s.History, cmd)
	}
	s.hist = len(s.History)
	return cmd
}

func (s *Stone) Completion(v Vocabulary) Completion {
	if s.Cursor != len(s.Input) {
		return Completion{}
	}
	return Complete(string(s.Input), v)
}

func (s *Stone) Tab(v Vocabulary) {
	if n := len(s.cycle.list); n > 0 {
		s.cycle.index = (s.cycle.index + 1) % n
		s.replaceToken(s.cycle.list[s.cycle.index])
		return
	}
	c := s.Completion(v)
	switch {
	case c.Ghost != "":
		text := string(s.Input) + c.Ghost
		if len(c.Candidates) == 1 {
			text += " "
		}
		s.SetInput(text)
	case len(c.Candidates) > 1:
		list := c.Candidates
		s.replaceToken(list[0])
		s.cycle = cycle{list: list}
	}
}

func (s *Stone) replaceToken(word string) {
	text := string(s.Input)
	cut := strings.LastIndex(text, " ") + 1
	s.Input = []rune(text[:cut] + word)
	s.Cursor = len(s.Input)
}

func (s *Stone) Append(kind int, lines ...string) {
	for _, l := range lines {
		s.Output = append(s.Output, StoneLine{l, kind})
	}
	s.Written += len(lines)
	if len(s.Output) > maxStoneLines {
		s.Output = s.Output[len(s.Output)-maxStoneLines:]
	}
	s.Scroll = ui.Scroll{}
}

func (h *HUD) StoneWritten() int { return h.Stone.Written }

func (h *HUD) StoneOutput() []string {
	out := make([]string, 0, len(h.Stone.Output))
	for _, l := range h.Stone.Output {
		out = append(out, ui.Plain(ConsoleText(l)))
	}
	return out
}
