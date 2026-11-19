package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	IntentLive       = "live"
	IntentLiveList   = "live:list"
	IntentLiveOpen   = "live:open"
	IntentLiveCancel = "live:cancel"
	IntentLiveWho    = "live:who"
	IntentLiveForget = "live:forget"

	LiveStepWarn    = "warn"
	LiveStepList    = "list"
	LiveStepConfirm = "confirm"
	LiveStepOpening = "opening"
	LiveStepFailed  = "failed"
	LiveStepWrites  = "writes"
	LiveStepSeam    = "seamless"

	LevelReadOnly = "READ ONLY"
	LevelWrites   = "WRITES"
	LevelSeamless = "DANGEROUSLY SEAMLESS"
	ReadOnlyTip   = "Ticked by default: Kubernites only reads this cluster. Untick to allow changes. Set before connecting."
	forcedTip     = "kubernites --readonly: this run only reads, whatever was chosen before."

	liveRows    = 8
	liveRowH    = 22
	liveBodyW   = 480
	liveConfirm = "live:confirm"
	liveBack    = "live:back"
	liveChoose  = "live:choose"
	liveRow     = "live:row:"
	LiveBoxID   = "live:readonly"
	liveLevel   = "live:level"
	liveAllow   = "live:allow"
	liveSeam    = "live:seam"
)

var LiveBox = ui.Rect{X: 220, Y: 110, W: 520, H: 300}

type LiveContext struct {
	Name      string
	Cluster   string
	Server    string
	User      string
	Namespace string
}

type LivePicker struct {
	Step       string
	Files      []string
	Contexts   []LiveContext
	Chosen     int
	Err        string
	Who        LiveWho
	Progress   LiveProgress
	Lib        *gfx.Library
	ReadOnly   bool
	Level      string
	Forced     bool
	Remembered map[string]LiveMemory
	Changed    string
	warned     map[string]string
	survey     liveSurvey
	scroll     int
	hover      Button
	hovered    bool
	held       int
}

type LiveMemory struct {
	Server  string
	User    string
	Level   string
	Warned  string
	Classes []string
}

type LiveWho struct {
	Context  string
	Username string
	Groups   []string
	Err      string
	Asked    bool
}

func (p *LivePicker) Answered(context, username string, groups []string, err error) {
	if !p.Who.Asked || p.Who.Context != context {
		return
	}
	p.Who.Username, p.Who.Groups, p.Who.Asked = username, groups, false
	if err != nil {
		p.Who.Err = err.Error()
	}
}

func (p *LivePicker) Open(files []string) {
	*p = LivePicker{Step: LiveStepWarn, Files: files, Chosen: -1, Lib: p.Lib, ReadOnly: true, Forced: p.Forced, Remembered: p.Remembered}
}

func (p *LivePicker) Close() {
	*p = LivePicker{Lib: p.Lib, Forced: p.Forced, Remembered: p.Remembered}
}

func (p *LivePicker) ChosenLevel() string {
	if p.ReadOnly || p.Forced || p.Level == "" {
		return LevelReadOnly
	}
	return p.Level
}

func levelRank(l string) int {
	switch l {
	case LevelWrites:
		return 1
	case LevelSeamless:
		return 2
	}
	return 0
}

func (p *LivePicker) passed(c LiveContext, level string) bool {
	if levelRank(p.warned[c.Name]) >= levelRank(level) {
		return true
	}
	m, ok := p.Remembered[c.Name]
	return ok && m.Server == c.Server && m.User == c.User && levelRank(m.Warned) >= levelRank(level)
}

func (p *LivePicker) untick() []Intent {
	c, ok := p.chosen()
	click := Intent{Kind: IntentSound, Command: "click"}
	if !ok {
		return nil
	}
	if p.Forced {
		return []Intent{{Kind: IntentSound, Command: "error"}}
	}
	if m, ok := p.Remembered[c.Name]; ok {
		if m.Server == c.Server && m.User == c.User {
			p.ReadOnly, p.Level = false, m.Level
			return []Intent{click}
		}
		var what []string
		if m.Server != c.Server {
			what = append(what, "its API server was "+m.Server)
		}
		if m.User != c.User {
			what = append(what, "its user entry was "+m.User)
		}
		p.Changed = "Changed since " + m.Level + " was chosen: " + strings.Join(what, " and ") + ". The remembered choice is dropped."
		delete(p.Remembered, c.Name)
		p.Step = LiveStepWrites
		return []Intent{click, {Kind: IntentLiveForget, Target: c.Name}}
	}
	if p.passed(c, LevelWrites) {
		p.ReadOnly, p.Level = false, LevelWrites
		return []Intent{click}
	}
	p.Step = LiveStepWrites
	return []Intent{click}
}

func (p *LivePicker) Active() bool {
	return p.Step != ""
}

func (p *LivePicker) ShowContexts(list []LiveContext, err error) {
	p.Contexts, p.Chosen, p.scroll = list, -1, 0
	if err != nil {
		p.Step, p.Err = LiveStepFailed, err.Error()
		return
	}
	p.Step = LiveStepList
}

func (p *LivePicker) Failed(err error) {
	p.Step, p.Err = LiveStepFailed, err.Error()
}

func (p *LivePicker) chosen() (LiveContext, bool) {
	if p.Chosen < 0 || p.Chosen >= len(p.Contexts) {
		return LiveContext{}, false
	}
	return p.Contexts[p.Chosen], true
}

func (p *LivePicker) buttonRow() (ui.Rect, ui.Rect) {
	y := LiveBox.Y + LiveBox.H - 30
	back := ui.Rect{X: LiveBox.X + 16, Y: y, W: 110, H: 20}
	ok := ui.Rect{X: LiveBox.X + LiveBox.W - 16 - 200, Y: y, W: 200, H: 20}
	return back, ok
}

func (p *LivePicker) Buttons() []Button {
	backR, okR := p.buttonRow()
	back := Button{ID: liveBack, Rect: backR, Label: "BACK", Tip: "Back out [Esc]"}
	switch p.Step {
	case LiveStepWarn:
		return []Button{back, {ID: liveConfirm, Rect: okR, Label: "I UNDERSTAND: LIST [Y]", Tip: "Read the kubeconfig and list its contexts"}}
	case LiveStepList:
		out := []Button{back}
		for i := p.scroll; i < len(p.Contexts) && i < p.scroll+liveRows; i++ {
			r := ui.Rect{X: LiveBox.X + 16, Y: p.rowY(i), W: LiveBox.W - 32, H: liveRowH - 2}
			out = append(out, Button{ID: fmt.Sprintf("%s%d", liveRow, i), Rect: r, Tip: p.Contexts[i].Server, Selected: i == p.Chosen})
		}
		choose := Button{ID: liveChoose, Rect: okR, Label: "CHOOSE [ENTER]", Tip: "Choose the context you picked (Up and Down pick one)"}
		if _, ok := p.chosen(); !ok {
			choose.Disabled, choose.Tip = true, "Pick a context first: none is chosen for you"
		}
		return append(out, choose)
	case LiveStepConfirm:
		return append([]Button{back, {ID: liveConfirm, Rect: okR, Label: "CONNECT [Y]", Tip: p.connectTip()}}, p.levelButtons()...)
	case LiveStepWrites:
		seam := Button{ID: liveSeam, Rect: ui.Rect{X: backR.X + backR.W + 12, Y: backR.Y, W: 160, H: backR.H}, Label: LevelSeamless + "...", Tip: "A second, stronger warning: no previews and no CONFIRM"}
		return []Button{back, seam, {ID: liveAllow, Rect: okR, Label: "ALLOW WRITES [Y]", Tip: "Untick READ ONLY for this context: every change shows its preview and waits for CONFIRM"}}
	case LiveStepSeam:
		return []Button{back, {ID: liveAllow, Rect: okR, Label: "ALLOW " + LevelSeamless + " [Y]", Tip: "Orders and commands go to the cluster without previews or CONFIRM; a card order asks once"}}
	case LiveStepOpening, LiveStepFailed:
		return []Button{back}
	}
	return nil
}

func (p *LivePicker) connectTip() string {
	switch p.ChosenLevel() {
	case LevelWrites:
		return "Connect at WRITES: every change shows the server's dry-run preview and waits for CONFIRM"
	case LevelSeamless:
		return "Connect at " + LevelSeamless + ": orders and commands go to the cluster without previews or CONFIRM"
	}
	return "Read this cluster and draw it as a realm"
}

func (p *LivePicker) boxRect() ui.Rect {
	_, okR := p.buttonRow()
	return ui.Rect{X: okR.X - 12 - 120, Y: okR.Y, W: 120, H: okR.H}
}

func (p *LivePicker) levelButtons() []Button {
	box := Button{ID: LiveBoxID, Rect: p.boxRect(), Label: "[x] READ ONLY", Tip: ReadOnlyTip}
	switch {
	case p.Forced:
		box.Disabled, box.Tip = true, forcedTip
	case !p.ReadOnly:
		box.Label = "[ ] READ ONLY"
	}
	out := []Button{box}
	if p.ChosenLevel() != LevelReadOnly {
		r := ui.Rect{X: LiveBox.X + 20 + 9*ui.Advance - 3, Y: LiveBox.Y + 48 + 4*ui.LineHeight - 3, W: max(100, len(LevelSeamless)*ui.Advance+12), H: 14}
		next := LevelSeamless
		if p.Level == LevelSeamless {
			next = LevelWrites
		}
		out = append(out, Button{ID: liveLevel, Rect: r, Label: p.Level, Tip: "The level for this context; press for " + next})
	}
	return out
}

func (p *LivePicker) rowY(i int) int {
	return LiveBox.Y + 52 + (i-p.scroll)*liveRowH
}

func (p *LivePicker) Update(in Input) []Intent {
	if !p.Active() {
		return nil
	}
	if p.Step == LiveStepOpening {
		p.survey.tick(p.Progress)
	}
	buttons := p.Buttons()
	hover, hovered := HitButton(buttons, in.MX, in.MY)
	switch {
	case !hovered:
		p.held = 0
	case p.hovered && hover.ID == p.hover.ID:
		p.held++
	default:
		p.held = 1
	}
	p.hover, p.hovered = hover, hovered
	if in.Pressed(ebiten.KeyEscape) || in.RightClick {
		return p.back()
	}
	if in.Wheel != 0 && p.Step == LiveStepList {
		p.scroll = min(max(p.scroll-int(in.Wheel), 0), max(len(p.Contexts)-liveRows, 0))
	}
	if in.Pressed(ebiten.KeyY) && !in.Control {
		if p.Step == LiveStepWrites || p.Step == LiveStepSeam {
			return p.press(liveAllow)
		}
		return p.press(liveConfirm)
	}
	if intents, ok := p.listKeys(in); ok {
		return intents
	}
	if !in.Click || !p.hovered {
		return nil
	}
	if p.hover.Disabled {
		return []Intent{{Kind: IntentSound, Command: "error"}}
	}
	return p.press(p.hover.ID)
}

func (p *LivePicker) listKeys(in Input) ([]Intent, bool) {
	if p.Step != LiveStepList || len(p.Contexts) == 0 {
		return nil, false
	}
	switch {
	case in.Pressed(ebiten.KeyEnter):
		return p.press(liveChoose), true
	case in.Pressed(ebiten.KeyDown):
		p.Chosen = min(p.Chosen+1, len(p.Contexts)-1)
	case in.Pressed(ebiten.KeyUp):
		p.Chosen = max(p.Chosen-1, 0)
	default:
		return nil, false
	}
	p.scroll = min(max(p.scroll, p.Chosen-liveRows+1), p.Chosen)
	return []Intent{{Kind: IntentSound, Command: "click"}}, true
}

func (p *LivePicker) back() []Intent {
	switch p.Step {
	case LiveStepConfirm:
		p.Step, p.ReadOnly, p.Level, p.Changed = LiveStepList, true, "", ""
		return []Intent{{Kind: IntentSound, Command: "click"}}
	case LiveStepWrites:
		p.Step, p.ReadOnly, p.Changed = LiveStepConfirm, true, ""
		return []Intent{{Kind: IntentSound, Command: "click"}}
	case LiveStepSeam:
		p.Step = LiveStepWrites
		if p.Level == LevelWrites && !p.ReadOnly {
			p.Step = LiveStepConfirm
		}
		return []Intent{{Kind: IntentSound, Command: "click"}}
	case LiveStepOpening:
		p.Step = LiveStepList
		return []Intent{{Kind: IntentLiveCancel}}
	}
	p.Close()
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (p *LivePicker) press(id string) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	switch {
	case id == liveBack:
		return p.back()
	case id == liveConfirm && p.Step == LiveStepWarn:
		return []Intent{click, {Kind: IntentLiveList}}
	case id == liveChoose:
		if c, ok := p.chosen(); ok {
			p.Step, p.Who = LiveStepConfirm, LiveWho{Context: c.Name, Asked: true}
			p.ReadOnly, p.Level, p.Changed = true, "", ""
			return []Intent{click, {Kind: IntentLiveWho, Target: c.Name}}
		}
	case id == liveConfirm && p.Step == LiveStepConfirm:
		if c, ok := p.chosen(); ok {
			level := p.ChosenLevel()
			p.Step, p.Progress = LiveStepOpening, LiveProgress{}
			p.survey.reset()
			return []Intent{click, {Kind: IntentLiveOpen, Target: c.Name, Command: level}}
		}
	case id == LiveBoxID && p.Step == LiveStepConfirm:
		if p.ReadOnly {
			return p.untick()
		}
		p.ReadOnly = true
		return []Intent{click}
	case id == liveLevel && p.Step == LiveStepConfirm && !p.ReadOnly:
		c, _ := p.chosen()
		if p.Level == LevelSeamless {
			p.Level = LevelWrites
			return []Intent{click}
		}
		if p.passed(c, LevelSeamless) {
			p.Level = LevelSeamless
			return []Intent{click}
		}
		p.Step = LiveStepSeam
		return []Intent{click}
	case id == liveSeam && p.Step == LiveStepWrites:
		p.Step = LiveStepSeam
		return []Intent{click}
	case id == liveAllow && (p.Step == LiveStepWrites || p.Step == LiveStepSeam):
		c, _ := p.chosen()
		level := LevelWrites
		if p.Step == LiveStepSeam {
			level = LevelSeamless
		}
		if p.warned == nil {
			p.warned = map[string]string{}
		}
		if levelRank(level) > levelRank(p.warned[c.Name]) {
			p.warned[c.Name] = level
		}
		p.Step, p.ReadOnly, p.Level = LiveStepConfirm, false, level
		return []Intent{click}
	case len(id) > len(liveRow) && id[:len(liveRow)] == liveRow && p.Step == LiveStepList:
		var i int
		if _, err := fmt.Sscanf(id[len(liveRow):], "%d", &i); err == nil && i >= 0 && i < len(p.Contexts) {
			p.Chosen = i
			return []Intent{click}
		}
	}
	return nil
}

func (p *LivePicker) Draw(dst *ebiten.Image) {
	if !p.Active() {
		return
	}
	ui.Fill(dst, ui.Rect{W: Width, H: Height}, ui.Tint(ui.Palette.Shadow, 170))
	r := LiveBox
	ui.DarkPanel(dst, r)
	ui.Stroke(dst, r, ui.Palette.Gold)
	switch p.Step {
	case LiveStepWarn:
		p.drawWarning(dst)
	case LiveStepList:
		p.drawList(dst)
	case LiveStepConfirm:
		p.drawConfirm(dst)
	case LiveStepWrites:
		p.drawWritesWarning(dst)
	case LiveStepSeam:
		p.drawSeamlessWarning(dst)
	case LiveStepOpening:
		p.drawOpening(dst)
	case LiveStepFailed:
		p.title(dst, p.Progress.FailTitle(), ui.Palette.RedText)
		p.lines(dst, LiveBox.Y+60, liveFont.WrapLines(p.Err, liveBodyW, 1, 12), ui.Palette.RedText)
	}
	for _, b := range p.Buttons() {
		if len(b.ID) > len(liveRow) && b.ID[:len(liveRow)] == liveRow {
			continue
		}
		state := ui.ButtonState{Hovered: p.hovered && p.hover.ID == b.ID, Disabled: b.Disabled}
		ui.Button(dst, liveFont, b.Rect, b.Label, state)
	}
	ui.InRegion("tooltip", func() { p.drawTip(dst) })
}

const pickerTipTicks = 30

func (p *LivePicker) drawTip(dst *ebiten.Image) {
	if !p.hovered || p.held < pickerTipTicks || p.hover.Tip == "" || p.Step == LiveStepList {
		return
	}
	lines := liveFont.WrapLines(p.hover.Tip, titleTipW-2*titleTipPad, 1, titleTipLines)
	b := p.hover.Rect
	h := len(lines)*ui.LineHeight + 2*titleTipPad - (ui.LineHeight - ui.GlyphHeight)
	r := ui.Rect{X: min(b.X, Width-titleTipW-8), Y: b.Y - h - 6, W: titleTipW, H: h}
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 235))
	ui.Stroke(dst, r, ui.Palette.Gold)
	for i, line := range lines {
		liveFont.Draw(dst, line, r.X+titleTipPad, r.Y+titleTipPad+i*ui.LineHeight, 1, ui.Palette.Cream)
	}
}

func (p *LivePicker) title(dst *ebiten.Image, text string, clr color.RGBA) {
	liveFont.DrawCentered(dst, text, ui.Rect{X: LiveBox.X, Y: LiveBox.Y + 12, W: LiveBox.W, H: 16}, 2, clr)
	ui.Fill(dst, ui.Rect{X: LiveBox.X + 16, Y: LiveBox.Y + 36, W: LiveBox.W - 32, H: 1}, ui.Palette.Gold)
}

func (p *LivePicker) lines(dst *ebiten.Image, y int, lines []string, clr color.RGBA) int {
	for _, l := range lines {
		liveFont.Draw(dst, l, LiveBox.X+20, y, 1, clr)
		y += ui.LineHeight
	}
	return y
}

func (p *LivePicker) drawWarning(dst *ebiten.Image) {
	p.title(dst, "! A REAL CLUSTER !", ui.Palette.RedText)
	y := p.lines(dst, LiveBox.Y+48, []string{
		"Engineer mode draws a real Kubernetes cluster as a realm.",
		"It only reads unless READ ONLY is unticked before connecting,",
		"and Secrets are never read at READ ONLY.",
		"",
		"The credentials are still real. Every read goes to that",
		"cluster's API server as the user your kubeconfig names, and",
		"its exec plugins run as they do for kubectl.",
		"",
		"Next, Kubernites reads your kubeconfig to list its contexts.",
		"Choosing one asks that cluster who you are, and nothing else.",
		"Nothing is chosen for you; READ ONLY is ticked at each connect.",
	}, ui.Palette.Text)
	files := "no kubeconfig file found"
	if len(p.Files) > 0 {
		files = p.Files[0]
		if len(p.Files) > 1 {
			files = fmt.Sprintf("%s and %d more", files, len(p.Files)-1)
		}
	}
	liveFont.Draw(dst, ui.Clip("KUBECONFIG: "+files, 78), LiveBox.X+20, y+4, 1, ui.Palette.TextDim)
}

func (p *LivePicker) drawList(dst *ebiten.Image) {
	p.title(dst, "CHOOSE A CONTEXT", ui.Palette.Gold)
	if len(p.Contexts) == 0 {
		p.lines(dst, LiveBox.Y+60, []string{"This kubeconfig holds no contexts."}, ui.Palette.TextDim)
		return
	}
	for _, b := range p.Buttons() {
		if len(b.ID) <= len(liveRow) || b.ID[:len(liveRow)] != liveRow {
			continue
		}
		var i int
		fmt.Sscanf(b.ID[len(liveRow):], "%d", &i)
		c := p.Contexts[i]
		fill := ui.Palette.ConsoleEdge
		switch {
		case b.Selected:
			fill = ui.Palette.BlueDark
		case p.hovered && p.hover.ID == b.ID:
			fill = ui.Palette.Console
		}
		ui.Fill(dst, b.Rect, fill)
		mark := "[ ]"
		if b.Selected {
			mark = "[x]"
			ui.Stroke(dst, b.Rect, ui.Palette.Gold)
		}
		liveFont.Draw(dst, mark+" "+ui.Clip(c.Name, 34), b.Rect.X+6, b.Rect.Y+6, 1, ui.Palette.Text)
		liveFont.Draw(dst, ui.Clip(c.Server, 36), b.Rect.X+b.Rect.W/2+20, b.Rect.Y+6, 1, ui.Palette.TextDim)
	}
	if len(p.Contexts) > liveRows {
		liveFont.Draw(dst, fmt.Sprintf("%d-%d OF %d  WHEEL SCROLLS", p.scroll+1, min(p.scroll+liveRows, len(p.Contexts)), len(p.Contexts)), LiveBox.X+20, LiveBox.Y+LiveBox.H-46, 1, ui.Palette.TextDim)
	}
}

func (p *LivePicker) drawConfirm(dst *ebiten.Image) {
	c, _ := p.chosen()
	p.title(dst, "! OPEN THIS CLUSTER ? !", ui.Palette.RedText)
	user := c.User
	if user == "" {
		user = "(no user named)"
	}
	y := p.lines(dst, LiveBox.Y+48, []string{"CONTEXT  " + ui.Clip(c.Name, 60), "SERVER   " + ui.Clip(c.Server, 60), "USER     " + ui.Clip(user, 60)}, ui.Palette.GoldBright)
	seen, clr := p.Who.line()
	y = p.lines(dst, y, []string{"SEEN AS  " + ui.Clip(seen, 60)}, clr)
	if level := p.ChosenLevel(); level != LevelReadOnly {
		liveFont.Draw(dst, "LEVEL", LiveBox.X+20, y, 1, LevelColour(level))
		y += ui.LineHeight + 2
	}
	switch p.ChosenLevel() {
	case LevelWrites:
		p.lines(dst, y+8, liveWritesBody(), ui.Palette.Text)
	case LevelSeamless:
		p.lines(dst, y+8, liveSeamlessBody(), ui.Palette.Text)
	default:
		p.lines(dst, y+8, liveConfirmBody(), ui.Palette.Text)
	}
}

func LevelColour(level string) color.RGBA {
	switch level {
	case LevelWrites:
		return ui.Palette.GoldBright
	case LevelSeamless:
		return ui.Palette.RedText
	}
	return ui.Palette.Green
}

func liveWritesBody() []string {
	return []string{
		"Kubernites will read this cluster as at READ ONLY, and the orders",
		"on the cards and the write keys may change it. Each change",
		"first shows the server's dry-run preview and waits for CONFIRM.",
		"",
		"Nothing is ever changed on connect, and Kubernites never writes",
		"on its own. Every change is written to a ledger file.",
	}
}

func liveSeamlessBody() []string {
	return []string{
		"Kubernites will read this cluster, and the orders on the cards",
		"and the write keys change it with no dry-run preview and no",
		"CONFIRM, as kubectl does; a card order asks once, as in k9s.",
		"",
		"Nothing is ever changed on connect, and Kubernites never writes",
		"on its own. Every change is written to a ledger file.",
	}
}

func (p *LivePicker) whom() []string {
	c, _ := p.chosen()
	user := c.User
	if user == "" {
		user = "(no user named)"
	}
	return []string{"CONTEXT  " + ui.Clip(c.Name, 60), "SERVER   " + ui.Clip(c.Server, 60), "USER     " + ui.Clip(user, 60)}
}

func (p *LivePicker) drawWritesWarning(dst *ebiten.Image) {
	p.title(dst, "! ALLOW CHANGES ? !", ui.Palette.RedText)
	y := p.lines(dst, LiveBox.Y+48, p.whom(), ui.Palette.GoldBright)
	if p.Changed != "" {
		y = p.lines(dst, y, liveFont.WrapLines(p.Changed, liveBodyW, 1, 2), ui.Palette.RedText)
	}
	p.lines(dst, y+6, []string{
		"Unticking READ ONLY lets Kubernites change the cluster above,",
		"with the real credentials of the user entry above.",
		"",
		"At WRITES every order and command first shows the server's",
		"dry-run preview, and nothing changes until CONFIRM; a shell,",
		"attach or port-forward has no dry run, so CONFIRM shows what it",
		"opens. Secrets, reaching into a workload (exec, attach, cp,",
		"port-forward, debug, proxy) and destroying more than one thing",
		"each warn once more on first use.",
		"",
		"The choice is remembered for this context. RESET LIVE CHOICES",
		"in the MENU forgets it; READ ONLY is still ticked at each connect.",
	}, ui.Palette.Text)
}

func (p *LivePicker) drawSeamlessWarning(dst *ebiten.Image) {
	p.title(dst, "! NO PREVIEWS, NO CONFIRM !", ui.Palette.RedText)
	y := p.lines(dst, LiveBox.Y+48, p.whom(), ui.Palette.GoldBright)
	p.lines(dst, y+6, SeamlessWarningLines(), ui.Palette.Text)
}

func SeamlessWarningLines() []string {
	return []string{
		"At DANGEROUSLY SEAMLESS orders, commands and write keys",
		"change the cluster as kubectl does: no dry-run preview and",
		"no CONFIRM. Each order on a card still asks once, Y or Enter",
		"on its form, as k9s asks before a restart, a rollback, a",
		"trigger, a cordon, a delete or a drain. One chord, no question:",
		"Ctrl-K kills a Pod, as k9s's kill does.",
		"",
		"Secrets, reaching into a workload (exec, attach, port-forward,",
		"cp, debug, proxy) and destroying more than one thing still warn",
		"once each. Port-forwards listen on 127.0.0.1 only.",
		"",
		"One key changes this cluster: use it only where a mistake",
		"can be repaired.",
	}
}

func liveConfirmBody() []string {
	return []string{
		"Kubernites will list this cluster's nodes, namespaces, workloads,",
		"pods, services, ingresses, gateways, volumes, autoscalers, events",
		"and the names of its custom resources, and read pod logs when you",
		ui.Bridge("ask the Speaking Stone for them.", "ask kubectl for them."),
		"",
		"It never writes and never reads Secrets. If these credentials",
		"can write, a read-only kubeconfig is the safer choice.",
	}
}

func (w LiveWho) line() (string, color.RGBA) {
	switch {
	case w.Asked:
		return "asking who you are...", ui.Palette.TextDim
	case w.Username != "":
		return w.Username, ui.Palette.GoldBright
	case w.Err != "":
		return "the cluster did not say: " + w.Err, ui.Palette.RedText
	}
	return "the cluster did not say", ui.Palette.TextDim
}
