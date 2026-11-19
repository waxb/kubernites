package hud

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

type steadyPanel struct {
	key     string
	From    float64 `json:"-"`
	before  []Line
	status  []Line
	after   []Line
	rows    int
	tick    float64
	chip    bool
	cmd     string
	offer   bridgeOffer
	offered bool
}

func (h *HUD) steadyKey(f Frame) string {
	o, ok := f.ShownObjective()
	switch {
	case !ok:
		return ""
	case o.Done:
		return actDone + o.ID
	}
	return fmt.Sprintf("%s%s %v %v %s", ActStep, o.ID, h.folded, h.HintOpen == o.ID, ui.Name(namesProbe))
}

func splitStatus(lines []Line) ([]Line, []Line, []Line) {
	first := slices.IndexFunc(lines, func(l Line) bool { return l.status })
	if first < 0 {
		return lines, nil, nil
	}
	last := first
	for i := first; i < len(lines); i++ {
		if lines[i].status {
			last = i
		}
	}
	return lines[:first], lines[first : last+1], lines[last+1:]
}

func (p steadyPanel) Read() []Line { return append(slices.Clone(p.before), p.after...) }

func (p steadyPanel) lines() []Line {
	status := slices.Clone(p.status)
	for len(status) < p.rows {
		status = append(status, Line{status: true})
	}
	return append(append(slices.Clone(p.before), status[:min(len(status), max(p.rows, len(p.status)))]...), p.after...)
}

func (h *HUD) noteSteady(f Frame) {
	h.noteSteadyBoard(f.RawFrame())
	f = h.Stage(f)
	h.noteSteadyTrouble(f)
	h.noteSteadyTop(f)
	key := h.steadyKey(f)
	before, status, after := splitStatus(h.rawObjectiveLines(f))
	p := &h.Steady
	wantChip := f.SpeakCommand() != "" || f.chipAhead()
	if key != p.key {
		*p = steadyPanel{key: key, From: h.Now, before: before, status: status, after: after, rows: len(status), tick: h.Now, chip: wantChip, cmd: f.SpeakCommand()}
		p.offer, p.offered = h.stepOffer(f)
		return
	}
	if h.Now-h.attn.pressed <= attnAnswerWindow {
		p.offer, p.offered = h.stepOffer(f)
	}
	reading := h.Now < p.From+ReadTime(SurfaceText(p.Read()))
	cmd := f.SpeakCommand()
	flip := !p.chip && cmd != "" && !reading
	changed := flip || !sameLines(status, p.status) || cmd != p.cmd || !reading && (!sameLines(before, p.before) || !sameLines(after, p.after))
	if !changed || h.Now-p.tick < CounterTick {
		return
	}
	if reading && len(status) > p.rows {
		short, fits := h.shortStatus(f, p.rows)
		if !fits || sameLines(short, p.status) {
			return
		}
		status = short
	}
	p.status, p.cmd, p.tick, p.chip = status, cmd, h.Now, p.chip || flip
	if reading {
		return
	}
	p.before, p.after, p.rows = before, after, max(p.rows, len(p.status))
}

func widened(text string) string {
	return digitsRun.ReplaceAllStringFunc(text, func(n string) string { return "0" + n })
}

func (h *HUD) statusLines(text string, w, most int, clr color.RGBA) []Line {
	out := []Line{}
	for _, row := range h.Font.WrapLines(text, w, 1, most) {
		out = append(out, Line{Text: row, Clr: clr, status: true})
	}
	for len(out) < len(h.Font.WrapLines(widened(text), w, 1, most)) {
		out = append(out, Line{status: true})
	}
	return out
}

func sameLines(a, b []Line) bool {
	return slices.EqualFunc(a, b, func(x, y Line) bool { return x.Text == y.Text })
}

func (h *HUD) chipRoom(f Frame) bool {
	if p := h.Steady; p.key != "" && p.key == h.steadyKey(f) {
		return p.chip
	}
	return f.SpeakCommand() != "" || f.chipAhead()
}

func (h *HUD) heldStepOffer(f Frame) (bridgeOffer, bool) {
	if p := h.Steady; p.key != "" && p.key == h.steadyKey(f) && strings.HasPrefix(p.key, actDone) {
		return p.offer, p.offered
	}
	return h.stepOffer(f)
}

func (h *HUD) shownCommand(f Frame) string {
	if p := h.Steady; p.key != "" && p.key == h.steadyKey(f) {
		return p.cmd
	}
	return f.SpeakCommand()
}

func (h *HUD) chipShown(f Frame) bool {
	if p := h.Steady; p.key != "" && p.key == h.steadyKey(f) {
		return p.chip && p.cmd != ""
	}
	return f.SpeakCommand() != ""
}

func (h *HUD) ObjectiveLines(f Frame) []Line {
	if p := h.Steady; p.key != "" && p.key == h.steadyKey(f) {
		return h.fitAboveWindow(f, p.lines())
	}
	return h.fitAboveWindow(f, h.rawObjectiveLines(f))
}

func (h *HUD) fitAboveWindow(f Frame, lines []Line) []Line {
	w, ok := h.windowRect()
	if !ok || w.X >= Objective.X+Objective.W || w.X+w.W <= Objective.X || h.teacherShown() || h.trialShown() {
		return lines
	}
	room := (w.Y - TipClear - Objective.Y - ObjectiveTop - 4 - h.objectiveExtras(f)) / ui.LineHeight
	before, status, after := splitStatus(lines)
	for len(before)+len(status)+len(after) > room && len(after) > 0 {
		after = after[:len(after)-1]
	}
	for len(before)+len(status) > room && len(before) > 1 {
		before = before[:len(before)-1]
	}
	return append(append(slices.Clone(before), status...), after...)
}

type steadyCard struct {
	key      string
	from     float64
	cause    []string
	why      []string
	rows     [2]int
	tick     float64
	remedies []CommandLine
	vig      int
	height   int
	aside    string
	bare     bool
}

func troubleKey(v view.CalamityView) string {
	return fmt.Sprintf("%s %v %s", v.ID, v.Mended != "", ui.Name(namesProbe))
}

const namesProbe = "Keep"

func (c steadyCard) text() string { return strings.Join(append(slices.Clone(c.cause), c.why...), " ") }

func (c steadyCard) lines() ([]string, []string) {
	return padded(c.cause, c.rows[0]), padded(c.why, c.rows[1])
}

func padded(rows []string, n int) []string {
	out := slices.Clone(rows)
	for len(out) < n {
		out = append(out, "")
	}
	return out
}

func (h *HUD) noteSteadyTrouble(f Frame) {
	c := &h.steadyTrouble
	v, _, _, ok := h.CurrentCalamity(f)
	if !ok {
		*c = steadyCard{}
		return
	}
	cause, why := h.rawCalamityText(f, v)
	remedies := h.rawCalamityLines(f, v)
	if key := troubleKey(v); key != c.key {
		_, written := incidentOf(f, v.Incident)
		*c = steadyCard{key: key, from: h.Now, cause: cause, why: why, rows: [2]int{len(cause), len(why)}, tick: h.Now, remedies: remedies, aside: h.liveAsideLine(), bare: !written}
		c.vig = h.rawCalamityVig(f, v, remedies)
		c.height = h.steadyCalamityHeight(f, v)
		return
	}
	defer func() { c.height = max(c.height, h.calamityHeight(f, v, c.remedies, c.vig)) }()
	next := strings.Join(append(slices.Clone(cause), why...), " ")
	same := next == c.text() && remedyLabels(remedies) == remedyLabels(c.remedies)
	if same && !slices.Equal(remedyBlocks(remedies), remedyBlocks(c.remedies)) {
		c.remedies = remedies
		return
	}
	if same && !slices.Equal(remedyCommands(remedies), remedyCommands(c.remedies)) && h.Now-c.tick >= CounterTick {
		c.remedies, c.tick = remedies, h.Now
		return
	}
	if same || h.Now-c.tick < CounterTick {
		return
	}
	reading := h.Now < c.from+ReadTime(strings.Join([]string{h.asideLine(), c.text(), remedyLabels(c.remedies)}, " "))
	if reading && (masked(next) != masked(c.text()) || masked(remedyLabels(remedies)) != masked(remedyLabels(c.remedies))) {
		return
	}
	c.cause, c.why, c.remedies, c.tick = cause, why, remedies, h.Now
	c.rows = [2]int{max(c.rows[0], len(cause)), max(c.rows[1], len(why))}
}

func remedyCommands(lines []CommandLine) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.Command)
	}
	return out
}

func remedyLabels(lines []CommandLine) string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.Label)
	}
	return strings.Join(out, " ")
}

func remedyBlocks(lines []CommandLine) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.Blocked)
	}
	return out
}
