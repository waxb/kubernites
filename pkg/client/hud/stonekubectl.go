package hud

import (
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	StoneMenuID    = "menu:stone"
	StoneModeStory = "story"
	StoneModeKube  = "kubectl"
	StoneConfirmID = "stone:confirm"
	StoneCancelID  = "stone:cancel"
	LineRaw        = 100
	LineRawError   = 101
	LineRawEcho    = 102
	stoneAskW      = 520
	editTitleRoom  = 7
)

type StoneChunk struct {
	Text string
	Err  bool
}

type StoneAsk struct {
	ID      int
	Title   string
	Lines   []string
	Confirm bool
	Go      string
	GoTip   string
}

func (a StoneAsk) GoLabel() (string, string) {
	if a.Go == "" {
		return "CONFIRM [Y]", "Send it to the cluster"
	}
	return a.Go, a.GoTip
}

func (a StoneAsk) buttonW(h *HUD) int {
	label, _ := a.GoLabel()
	return max(84, h.Font.Width(label, 1)+16)
}

const AskSeenFrames = 15

type StoneEnd struct {
	OK       bool
	Stopped  bool
	Edit     *LiveEdit
	Sent     StoneEditEnd
	Sessions []StoneSession
}

type StoneJob interface {
	Take() ([]StoneChunk, bool)
	Ask() (StoneAsk, bool)
	Answer(yes bool)
	Stop()
	End() StoneEnd
}

type stoneRun struct {
	job       StoneJob
	partial   [2]string
	ask       *StoneAsk
	askFrames int
	command   string
}

func (r *stoneRun) readAsk() {
	a, ok := r.job.Ask()
	if !ok {
		r.ask, r.askFrames = nil, 0
		return
	}
	if r.ask == nil || r.ask.ID != a.ID || r.ask.Title != a.Title {
		r.askFrames = 0
	}
	r.ask = &a
	r.askFrames++
}

func (r *stoneRun) answer(yes bool) {
	r.job.Answer(yes)
	r.ask, r.askFrames = nil, 0
}

func (r *stoneRun) yes(in Input) bool {
	return r.ask != nil && r.ask.Confirm && r.askFrames >= AskSeenFrames && in.Pressed(ebiten.KeyY) && !in.Control
}

func (h *HUD) SetStoneMode(mode string) {
	h.stoneMode = mode
}

func (h *HUD) StoneMode() string {
	return h.stoneMode
}

func (h *HUD) StoneKubectl() bool {
	return h.stoneMode == StoneModeKube || h.stoneMode == "" && ui.Naming()
}

func (h *HUD) StartStoneJob(job StoneJob) {
	h.kube = &stoneRun{job: job}
}

func (h *HUD) PressLiveEdit(rest string) []Intent {
	return h.pressLiveEdit(rest)
}

func (h *HUD) StoneRunning() bool {
	return h.kube != nil
}

func (h *HUD) StoneAsking() (StoneAsk, bool) {
	if h.kube == nil || h.kube.ask == nil {
		return StoneAsk{}, false
	}
	return *h.kube.ask, true
}

func (h *HUD) stoneLabel() [3]string {
	label := "STONE: STORY"
	if h.StoneKubectl() {
		label = stoneOnLabelK8s()
	}
	return [3]string{StoneMenuID, ui.Settle(label), ui.Bridge("The Speaking Stone keeps Kubernites' own read words, or speaks the realm's true tongue word for word", "The Speaking Stone keeps Kubernites' read verbs (STORY), or runs kubectl itself on this cluster (KUBECTL)")}
}

func stoneOnLabelK8s() string {
	return "STONE: KUBECTL"
}

func NextStoneMode(kubectl bool) string {
	if kubectl {
		return StoneModeStory
	}
	return StoneModeKube
}

func (h *HUD) startKubectl(cmd string) bool {
	if h.kube != nil || h.backend.Start == nil || !h.StoneKubectl() {
		return false
	}
	job, ok := h.backend.Start(cmd)
	if !ok {
		return false
	}
	h.kube = &stoneRun{job: job}
	return true
}

func (h *HUD) pumpStone() []Intent {
	r := h.kube
	if r == nil {
		return nil
	}
	chunks, done := r.job.Take()
	for _, c := range chunks {
		h.appendRaw(r, c)
	}
	r.readAsk()
	if !done {
		return nil
	}
	for i, rest := range r.partial {
		if rest != "" {
			h.Stone.Append(rawKind(i == 1), rawLine(rest))
		}
	}
	h.kube = nil
	end := r.job.End()
	for _, sess := range end.Sessions {
		if line, ok := SessionEndLine(sess); ok {
			h.Stone.Append(LineHint, line)
		}
	}
	sound := "click"
	switch {
	case end.Stopped && len(end.Sessions) == 0:
		h.Stone.Append(LineHint, "Stopped.")
	case !end.OK:
		sound = "error"
	}
	if end.Edit != nil && end.OK {
		h.OpenLiveEdit(*end.Edit)
	}
	return []Intent{{Kind: IntentSound, Command: sound}}
}

func rawKind(stderr bool) int {
	if stderr {
		return LineRawError
	}
	return LineRaw
}

func rawLine(s string) string {
	return strings.ReplaceAll(strings.TrimRight(s, "\r"), "\t", "    ")
}

func (h *HUD) appendRaw(r *stoneRun, c StoneChunk) {
	i := 0
	if c.Err {
		i = 1
	}
	text := r.partial[i] + c.Text
	lines := strings.Split(text, "\n")
	rest := lines[len(lines)-1]
	for _, l := range lines[:len(lines)-1] {
		for _, piece := range cutLine(rawLine(l)) {
			h.Stone.Append(rawKind(c.Err), piece)
		}
	}
	for len(rest) > MaxRawLine {
		cut := runeCut(rest, MaxRawLine)
		for _, piece := range cutLine(rawLine(rest[:cut])) {
			h.Stone.Append(rawKind(c.Err), piece)
		}
		rest = rest[cut:]
	}
	r.partial[i] = rest
}

func runeCut(s string, n int) int {
	for n > 0 && n < len(s) && !utf8.RuneStart(s[n]) {
		n--
	}
	if n == 0 {
		return min(len(s), MaxRawLine)
	}
	return n
}

func cutLine(s string) []string {
	if len(s) <= MaxRawLine {
		return []string{s}
	}
	var out []string
	for len(s) > MaxRawLine {
		cut := runeCut(s, MaxRawLine)
		out, s = append(out, s[:cut]), s[cut:]
	}
	return append(out, s)
}

var reachStory = map[string]string{
	"exec":         "Speaking with a hut's worker",
	"attach":       "Listening at a hut's worker",
	"debug":        "Sending a helper into a hut",
	"port-forward": "Opening a door from here to a hut",
	"cp":           "Carrying scrolls in and out of a hut",
	"run --stdin":  "Speaking with a new hut's worker",
	"run --tty":    "Speaking with a new hut's worker",
	"run --attach": "Listening at a new hut's worker",
}

func ReachAnswer(verb string) string {
	story, ok := reachStory[verb]
	if !ok {
		story = "Reaching into a hut"
	}
	return ui.Bridge(story+" takes the Speaking Stone's true tongue: turn its switch in the MENU, or use the hut card's SHELL, ATTACH, PORT-FORWARD or LOGS.",
		"kubectl "+verb+" runs with STONE: KUBECTL in the MENU; the hut card's SHELL, ATTACH, PORT-FORWARD and LOGS run with either.")
}

var globalValueFlags = map[string]bool{"-n": true, "--namespace": true, "--context": true, "--kubeconfig": true, "--cluster": true, "--user": true, "-s": true, "--server": true, "--as": true, "--request-timeout": true}

func ReachVerb(cmd string) string {
	fields := strings.Fields(cmd)
	if len(fields) > 0 && fields[0] == "kubectl" {
		fields = fields[1:]
	}
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		switch {
		case globalValueFlags[f]:
			i++
		case strings.HasPrefix(f, "-"):
		case f == "run":
			return reachRun(fields[i+1:])
		default:
			if _, ok := reachStory[f]; ok {
				return f
			}
			return ""
		}
	}
	return ""
}

func reachRun(args []string) string {
	for _, a := range args {
		if a == "--" {
			return ""
		}
		switch {
		case a == "-i" || a == "--stdin" || strings.HasPrefix(a, "--stdin="):
			return "run --stdin"
		case a == "-t" || a == "--tty" || strings.HasPrefix(a, "--tty="):
			return "run --tty"
		case a == "-it" || a == "-ti":
			return "run --stdin"
		case a == "--attach" || strings.HasPrefix(a, "--attach="):
			return "run --attach"
		}
	}
	return ""
}

func (h *HUD) answerReach(cmd string) bool {
	verb := ReachVerb(cmd)
	if h.live == nil || verb == "" {
		return false
	}
	h.Stone.Append(LineError, ReachAnswer(verb))
	return true
}

func (h *HUD) kubectlKeys(in Input) []Intent {
	r := h.kube
	if r.yes(in) {
		return h.answerStone(true)
	}
	for _, k := range in.Keys {
		switch {
		case k == ebiten.KeyEscape && r.ask != nil:
			r.answer(false)
		case k == ebiten.KeyEscape, k == ebiten.KeyC && in.Ctrl:
			r.job.Stop()
		}
	}
	return nil
}

func (h *HUD) answerStone(yes bool) []Intent {
	if h.kube == nil || h.kube.ask == nil {
		return nil
	}
	h.kube.answer(yes)
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) stoneAskRect(a StoneAsk) ui.Rect {
	out := h.stoneOutput()
	rows := len(h.stoneAskRows(a))
	height := 26 + rows*ui.LineHeight + 22
	return ui.Rect{X: out.X + (out.W-stoneAskW)/2, Y: out.Y + out.H - height - 4, W: stoneAskW, H: height}
}

func (h *HUD) stoneShownRows() int {
	visible := h.StoneVisible()
	a, ok := h.StoneAsking()
	if !ok {
		return visible
	}
	out, box := h.stoneOutput(), h.stoneAskRect(a)
	return max(min(visible, (box.Y-2-ui.GlyphHeight-out.Y-3)/ui.LineHeight+1), 0)
}

func (h *HUD) stoneAskRows(a StoneAsk) []string {
	var rows []string
	for _, l := range a.Lines {
		rows = append(rows, h.Font.Wrap(l, stoneAskW-16, 1)...)
	}
	return rows
}

func (h *HUD) stoneAskButtons() []Button {
	a, ok := h.StoneAsking()
	if !ok {
		return nil
	}
	r := h.stoneAskRect(a)
	y := r.Y + r.H - 20
	if !a.Confirm {
		return []Button{{ID: StoneCancelID, Rect: ui.Rect{X: r.X + r.W - 88, Y: y, W: 80, H: 16}, Label: "CLOSE", Tip: "Close: nothing is sent [Esc]"}}
	}
	label, tip := a.GoLabel()
	w := a.buttonW(h)
	return []Button{
		{ID: StoneConfirmID, Rect: ui.Rect{X: r.X + r.W - 96 - w, Y: y, W: w, H: 16}, Label: label, Tip: tip},
		{ID: StoneCancelID, Rect: ui.Rect{X: r.X + r.W - 88, Y: y, W: 80, H: 16}, Label: "CANCEL", Tip: "Send nothing [Esc]"},
	}
}

func (h *HUD) drawStoneRun(dst *ebiten.Image) {
	if h.kube == nil {
		return
	}
	out := h.stoneOutput()
	hint := ui.Bridge("The Stone is speaking: Esc stops it", "kubectl is running: Esc or Ctrl+C stops it")
	h.Font.Draw(dst, hint, out.X+4, out.Y+out.H+4, 1, ui.Palette.TextDim)
	a, ok := h.StoneAsking()
	if !ok {
		return
	}
	r := h.stoneAskRect(a)
	ui.DarkPanel(dst, r)
	ui.Stroke(dst, r, ui.Palette.Gold)
	h.Font.Draw(dst, a.Title, r.X+8, r.Y+8, 1, ui.Palette.GoldBright)
	y := r.Y + 24
	for _, row := range h.stoneAskRows(a) {
		Literal.Draw(dst, row, r.X+8, y, 1, ui.Palette.Cream)
		y += ui.LineHeight
	}
}

func (h *HUD) pumpKubectl() []Intent {
	return append(append(h.pumpStone(), h.pumpEdit()...), h.pumpFetch()...)
}

func (h *HUD) editTabButton(d Details) []Button {
	t, ok := targetOf(d.Sel)
	if h.live == nil || h.backend.Start == nil || !ok || t.name == "" {
		return nil
	}
	x := Overlay.X + Overlay.W - 32 - 5*68
	return []Button{{ID: "det:" + EditTab, Rect: ui.Rect{X: x, Y: Overlay.Y + 6, W: 64, H: 14}, Label: "EDIT",
		Tip: ui.Bridge("Open this in the Scriptorium and send changes to the realm", "kubectl edit "+t.kind+" "+t.name+nsFlag(t.namespace)+": the Scriptorium opens the live YAML")}}
}

func (h *HUD) editSelected() []Intent {
	t, ok := targetOf(h.DetailsCard.Sel)
	if !ok || h.fetching != nil || h.backend.Start == nil {
		return nil
	}
	cmd := EditCommand(t.kind+"/"+t.name, t.namespace)
	job, started := h.backend.Start(cmd)
	if !started {
		return nil
	}
	h.fetching = &stoneRun{job: job, command: cmd}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) pumpFetch() []Intent {
	r := h.fetching
	if r == nil {
		return nil
	}
	_, done := r.job.Take()
	if a, ok := r.job.Ask(); ok {
		if !a.Confirm || h.kube != nil {
			r.job.Answer(false)
		} else {
			h.fetching, h.kube = nil, r
			h.OpenStone("")
			h.EchoTyped(r.command)
			r.readAsk()
			return nil
		}
	}
	if !done {
		return nil
	}
	h.fetching = nil
	end := r.job.End()
	if end.Edit != nil && end.OK {
		h.OpenLiveEdit(*end.Edit)
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	h.Toast(endOutput(end), true)
	return []Intent{{Kind: IntentSound, Command: "error"}}
}

const MaxRawLine = 8192
