package hud

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

var (
	liveFont      = ui.Font{Literal: true}
	namedFont     = ui.Font{}
	liveCardChars = ui.MaxChars(CardBox.W-16, 1)
	liveLinesTop  = 6 + ui.LineHeight + 2
	liveCardLines = (cardFloor() - CardBox.Y - liveLinesTop) / ui.LineHeight
)

const (
	LiveTip         = "Read-only: you are watching a real cluster"
	LiveWritesTip   = "WRITES: the orders on the cards change this real cluster, each after its dry-run preview and CONFIRM"
	LiveSeamlessTip = "DANGEROUSLY SEAMLESS: the orders on the cards change this real cluster at once, without previews or CONFIRM"
	liveNoOrderTip  = "Not one of the live orders: the Speaking Stone's kubectl reaches what the cards do not."
	liveDrainID     = "info:live:drain"
	liveLedgerID    = "info:live:ledger"
	liveTitle       = "LIVE CLUSTER"
	liveStoryTitle  = "LIVE REALM"
	liveBadge       = "READ-ONLY"
	liveBadgeID     = "info:live"
)

type LiveCard struct {
	Context   string
	Server    string
	Version   string
	Who       string
	Keeps     int
	Huts      int
	Shrines   int
	Fiefs     int
	Isles     int
	Trouble   int
	Seen      string
	Denied    []string
	Refused   []string
	Namespace string
	Idle      []string
	Reads     func(command string) bool
	Quiet     []QuietTrouble
	Watching  bool
	Stale     string
	StaleWhy  string
	Level     string
	Ledger    string
	Writes    []string
	Draining  string
	DrainWhy  string
	Hints     bool
	Listed    []ListedRead
	Forwards  []LiveForward
}

func (c *LiveCard) badge() string {
	if c.Level == "" || c.Level == LevelReadOnly {
		return liveBadge
	}
	return c.Level
}

func (c *LiveCard) badgeColour() color.RGBA {
	return LevelColour(c.Level)
}

func (c *LiveCard) levelTip() string {
	switch c.Level {
	case LevelWrites:
		return LiveWritesTip
	case LevelSeamless:
		return LiveSeamlessTip
	}
	return LiveTip
}

type QuietTrouble struct {
	ID    string
	Kind  string
	Cause string
}

const (
	liveRefusedID   = "info:live:refused"
	liveIdleID      = "info:live:idle"
	liveIdleShown   = 6
	LiveQuietID     = liveQuietPrefix + ":open"
	liveQuietPrefix = "livetrouble"
	liveStaleID     = "info:live:stale"
	liveQuietTold   = 4
	liveQuietCause  = 90
)

type liveLine struct {
	text    string
	literal bool
	clr     color.RGBA
	id      string
	tip     string
	stop    bool
}

func (c *LiveCard) who() string {
	if c.Who == "" {
		return "you"
	}
	return c.Who
}

func (c *LiveCard) lines() []liveLine {
	out := []liveLine{
		{text: liveFont.Clip(c.Context, liveCardChars), literal: true, clr: ui.Palette.Text},
		{text: liveFont.Clip(c.Server, liveCardChars), literal: true, clr: ui.Palette.TextDim},
	}
	if c.Who != "" {
		out = append(out, liveLine{text: liveFont.Clip("AS "+c.Who, liveCardChars), literal: true, clr: ui.Palette.TextDim})
	}
	out = append(out,
		liveLine{text: plural(c.Keeps, "KEEP", "KEEPS") + " ON " + plural(c.Isles, "ISLE", "ISLES"), clr: ui.Palette.Text},
		liveLine{text: c.hutsLine(), clr: ui.Palette.Text})
	if c.Stale != "" {
		out = append(out, liveLine{text: "STALE SINCE " + c.Stale, literal: true, clr: ui.Palette.RedText, id: liveStaleID,
			tip: ui.Bridge("The watch on the realm broke and has not recovered. Nothing that changed since then is drawn.",
				"The watch on the cluster broke and has not recovered: "+c.StaleWhy+". Nothing that changed since then is drawn.")})
	}
	if line, ok := c.listedLine(); ok {
		out = append(out, line)
	}
	if c.Draining != "" {
		out = append(out, liveLine{text: liveFont.Clip(c.Draining, liveCardChars), literal: true, clr: ui.Palette.GoldBright, id: liveDrainID, tip: c.drainTip()})
	}
	if c.Trouble > 0 {
		out = append(out, liveLine{text: "! " + plural(c.Trouble, "HUT", "HUTS") + " IN TROUBLE", clr: ui.Palette.RedText})
	}
	if line, ok := c.ledgerLine(); ok {
		out = append(out, line)
	}
	out = append(out, c.forwardLines()...)
	if line, ok := c.quietLine(); ok {
		out = append(out, line)
	}
	if text, tip := c.refusal(); text != "" {
		out = append(out, liveLine{text: liveFont.Clip(text, liveCardChars), literal: true, clr: ui.Palette.RedText, id: liveRefusedID, tip: tip})
	}
	if n := len(c.Idle); n > 0 {
		shown := c.Idle[:min(n, liveIdleShown)]
		tip := ui.Bridge("Shrines with no Keep to stand at, so none is drawn: ", "DaemonSets that run no Pod, because no Node matches them: ") + strings.Join(shown, ", ")
		if n > len(shown) {
			tip += fmt.Sprintf(" and %d more", n-len(shown))
		}
		out = append(out, liveLine{text: ui.Bridge(plural(n, "SHRINE STANDS", "SHRINES STAND")+" NOWHERE", plural(n, "DAEMONSET RUNS", "DAEMONSETS RUN")+" NO POD"), clr: ui.Palette.TextDim, id: liveIdleID, tip: tip})
	}
	return out
}

func (c *LiveCard) drainTip() string {
	if c.DrainWhy == "" {
		return ui.Bridge("The drain sends the Keep's huts away one by one.", "The drain evicts the Node's Pods one by one.")
	}
	return "WAITING: " + c.DrainWhy
}

func (c *LiveCard) ledgerLine() (liveLine, bool) {
	if c.Level == "" || c.Level == LevelReadOnly {
		return liveLine{}, false
	}
	text := "LEDGER: NO WRITE YET"
	if n := len(c.Writes); n > 0 {
		text = "LEDGER: " + c.Writes[n-1]
	}
	file := c.Ledger
	if file == "" {
		file = "(no file: this run keeps no settings directory)"
	}
	tip := "Every write is listed here, in the Speaking Stone's history and in " + file
	if len(c.Writes) > 0 {
		tip += "\n" + strings.Join(c.Writes, "\n")
	}
	return liveLine{text: liveFont.Clip(text, liveCardChars), literal: true, clr: ui.Palette.TextDim, id: liveLedgerID, tip: tip}, true
}

func (c *LiveCard) quietLine() (liveLine, bool) {
	if len(c.Quiet) == 0 {
		return liveLine{}, false
	}
	var titles, told []string
	seen := map[string]bool{}
	for _, q := range c.Quiet {
		title := model.CauseTitle(q.Kind)
		if !seen[title] {
			seen[title] = true
			titles = append(titles, title)
		}
		if len(told) < liveQuietTold {
			told = append(told, title+": "+ui.ClipWords(q.Cause, liveQuietCause))
		}
	}
	tip := "Already in trouble when you opened the realm, so no card opened for it. Press to read the first.\n" + strings.Join(told, "\n")
	if more := len(c.Quiet) - len(told); more > 0 {
		tip += fmt.Sprintf("\nand %d more", more)
	}
	return liveLine{text: namedFont.Clip("QUIET: "+strings.Join(titles, ", "), liveCardChars), clr: ui.Palette.TextDim, id: LiveQuietID, tip: tip}, true
}

func (c *LiveCard) hutsLine() string {
	huts := plural(c.Huts, "HUT", "HUTS") + " IN " + plural(c.Fiefs, "FIEF", "FIEFS")
	if c.Shrines == 0 {
		return huts
	}
	return ui.Bridge(fmt.Sprintf("%s, %d AT SHRINES", huts, c.Shrines),
		fmt.Sprintf("%s IN %s, %d OF DAEMONSETS", plural(c.Huts, "POD", "PODS"), plural(c.Fiefs, "NAMESPACE", "NAMESPACES"), c.Shrines))
}

func (c *LiveCard) refusal() (string, string) {
	if len(c.Refused) == 0 && len(c.Denied) == 0 {
		return "", ""
	}
	all := append(append([]string(nil), c.Refused...), c.Denied...)
	story := func(lists []string) string {
		out := make([]string, len(lists))
		for i, l := range lists {
			out[i] = lexicon.StoryList(l)
		}
		return strings.Join(out, ", ")
	}
	var tellStory, tellNames []string
	if len(c.Refused) > 0 {
		where, whereNames := "of every fief", "across the cluster"
		if c.Namespace != "" {
			where += ", so only the fief " + c.Namespace + " was read for them"
			whereNames += ", so only namespace " + c.Namespace + " was read for them"
		}
		tellStory = append(tellStory, fmt.Sprintf("%s may not see the %s %s", c.who(), story(c.Refused), where))
		tellNames = append(tellNames, fmt.Sprintf("%s may not list %s %s", c.who(), strings.Join(c.Refused, ", "), whereNames))
	}
	if len(c.Denied) > 0 {
		tellStory = append(tellStory, fmt.Sprintf("%s may not see the %s at all, so they are not drawn", c.who(), story(c.Denied)))
		tellNames = append(tellNames, fmt.Sprintf("%s may not list %s at all, so they are not drawn", c.who(), strings.Join(c.Denied, ", ")))
	}
	text := ui.Bridge("NOT SEEN: "+story(all), "REFUSED: "+strings.Join(all, ", "))
	if c.Namespace != "" {
		text = ui.Bridge("ONLY THE FIEF "+c.Namespace+": "+story(all), "ONLY "+c.Namespace+": "+strings.Join(all, ", "))
	}
	return text, ui.Bridge(strings.Join(tellStory, ". ")+".", strings.Join(tellNames, ". ")+".")
}

func (c *LiveCard) shown() []liveLine {
	lines := c.lines()
	for len(lines) > liveCardLines {
		i := slices.IndexFunc(lines, func(l liveLine) bool { return l.id == liveIdleID })
		if i < 0 {
			i = len(lines) - 1
		}
		lines = slices.Delete(lines, i, i+1)
	}
	return lines
}

func (h *HUD) pressLiveTrouble(rest string, f Frame) []Intent {
	if h.live == nil || len(h.live.Quiet) == 0 {
		return nil
	}
	id := h.live.Quiet[0].ID
	h.Ops.Calamity = h.Ops.Calamity.Open(id, f.Clock)
	for i, v := range h.visibleCalamities(f) {
		if v.ID == id {
			next := h.Ops.Calamity
			next.index = i
			h.Ops.Calamity = next
		}
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) liveCardButtons() []Button {
	var out []Button
	y := CardBox.Y + liveLinesTop
	for _, l := range h.live.shown() {
		switch {
		case l.stop:
			stop := liveStopRect(y)
			out = append(out, Button{ID: l.id, Rect: ui.Rect{X: CardBox.X + 6, Y: y, W: stop.X - CardBox.X - 8, H: ui.LineHeight}, Tip: l.tip},
				Button{ID: strings.Replace(l.id, ":", ":stop:", 1), Rect: stop, Tip: ui.Bridge("Close this port-forward: it stops listening on 127.0.0.1", "Stop this port-forward: its listener on 127.0.0.1 closes")})
		case l.id != "":
			out = append(out, Button{ID: l.id, Rect: ui.Rect{X: CardBox.X + 6, Y: y, W: CardBox.W - 12, H: ui.LineHeight}, Tip: l.tip})
		}
		y += ui.LineHeight
	}
	return out
}

func (h *HUD) SetLive(c LiveCard) {
	h.live = &c
}

func (h *HUD) LiveCard() LiveCard {
	if h.live == nil {
		return LiveCard{}
	}
	return *h.live
}

func (h *HUD) Live() bool {
	return h.live != nil
}

var liveButtons = map[string]bool{
	"top:menu": true, "menu:resume": true, "menu:names": true, "menu:mute": true, "menu:reduce": true, "menu:edge": true,
	"menu:fullscreen": true, "menu:title": true, "menu:quit": true, "overlay:close": true, "overlay:body": true, liveBadgeID: true,
	"well:stone": true, GridHome: true, liveRefusedID: true, liveIdleID: true, LiveQuietID: true, liveStaleID: true, liveDrainID: true, liveLedgerID: true,
	CalamityNext: true, CalamityShow: true, calamityShelve: true, CalamityOK: true, "cmd:chronicles": true,
	menuHints: true, menuLiveChoices: true, CrewPrefix + ":describe": true, crewYAML: true, StoneMenuID: true,
}

var livePrefixes = []string{choicesPrefix + ":", volumePrefix, lensPrefix + ":", "tab:", "stone:", "sel:", "det:", "alert:", "blip:", "lesson:", "window:", IntentNotice + ":",
	calamityPrefix + ":" + calamityBack, "chron:", WoePrefix + ":", EditPrefix, liveForwardPrefix + ":"}

func LiveReadButton(id string) bool {
	if liveButtons[id] || strings.HasSuffix(id, ":close") {
		return true
	}
	for _, p := range livePrefixes {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	return false
}

func (h *HUD) LiveRead(id string) bool {
	return LiveReadButton(id) || h.liveReads[id]
}

func (h *HUD) liveGate(b Button) Button {
	if h.live == nil || IsLiveOrder(b.ID) || LiveReadButton(b.ID) || h.liveReads[b.ID] || b.ID == "" {
		return b
	}
	if h.readChip(b) {
		if h.liveReads != nil {
			h.liveReads[b.ID] = true
		}
		return b
	}
	b.Disabled, b.Tip, b.Why = true, LiveTip, liveBadge
	if h.live.Level != "" && h.live.Level != LevelReadOnly {
		b.Tip, b.Why = liveNoOrderTip, "NO ORDER"
	}
	return b
}

func (h *HUD) readChip(b Button) bool {
	command := b.Label
	if b.Speaks != "" {
		command = b.Speaks
	}
	return strings.HasPrefix(b.ID, linePrefix+":") && strings.HasSuffix(b.ID, ":"+SpeakOp) && h.live.Reads != nil && h.live.Reads(command)
}

func (h *HUD) noteLiveReads(f Frame) {
	h.liveReads = map[string]bool{}
	if h.live == nil || h.live.Reads == nil {
		return
	}
	for _, a := range SelectionActions(f) {
		if a.Act.Action == "" && len(a.Choices) == 0 && a.Command != "" && h.live.Reads(a.Command) {
			h.liveReads[ActionPrefix+":"+a.Key] = true
		}
	}
}

func (h *HUD) liveGated(buttons []Button, f Frame) []Button {
	if h.live == nil {
		return buttons
	}
	h.noteLiveReads(f)
	out := make([]Button, len(buttons))
	for i, b := range buttons {
		out[i] = h.liveGate(b)
	}
	return out
}

var liveIntents = map[string]bool{
	IntentKubectl: true, IntentDetails: true, IntentCancel: true, IntentJump: true, IntentMute: true, IntentTitle: true,
	IntentSelect: true, IntentSound: true, IntentReduce: true, IntentFocus: true, IntentEdge: true, IntentNames: true, IntentStone: true,
	IntentFullscreen: true, IntentVolume: true, IntentQuit: true, IntentPause: true, IntentGo: true, IntentChoose: true,
	IntentNotice: true, IntentEcho: true, IntentLiveOrder: true, IntentLiveScroll: true, IntentHints: true, IntentLiveChoices: true,
	IntentLiveForwardStop: true,
}

const LiveChronicle = "chronicle"

func LiveIntent(it Intent) bool {
	return liveIntents[it.Kind] || it.Kind == IntentAction && it.Action.Action == LiveChronicle
}

func (h *HUD) liveFiltered(intents []Intent) []Intent {
	if h.live == nil {
		return intents
	}
	out := intents[:0]
	for _, it := range intents {
		if LiveIntent(it) {
			out = append(out, it)
		}
	}
	return out
}

func (h *HUD) liveStats() []Stat {
	name := liveFont.Clip(h.live.Context, 40)
	return []Stat{
		{Text: h.live.badge(), slot: h.live.badge(), clr: h.live.badgeColour(), Tip: h.live.levelTip(), Id: liveBadgeID},
		{Text: name, slot: name, clr: ui.Palette.Text, Tip: h.live.Server, Id: liveBadgeID},
	}
}

func (h *HUD) liveClock() string {
	switch {
	case h.live.Stale != "":
		return fmt.Sprintf("%s  STALE SINCE %s", strings.ToUpper(h.live.Version), h.live.Stale)
	case h.live.Watching:
		return fmt.Sprintf("%s  WATCHING SINCE %s", strings.ToUpper(h.live.Version), h.live.Seen)
	}
	return fmt.Sprintf("%s  SEEN %s", strings.ToUpper(h.live.Version), h.live.Seen)
}

func (h *HUD) drawLiveCard(dst *ebiten.Image) {
	c := h.live
	x, y := CardBox.X+8, CardBox.Y+6
	h.Font.DrawShadow(dst, ui.Bridge(liveStoryTitle, liveTitle), x, y, 1, ui.Palette.Gold)
	liveFont.Draw(dst, c.badge(), CardBox.X+CardBox.W-8-liveFont.Width(c.badge(), 1), y, 1, c.badgeColour())
	y = CardBox.Y + liveLinesTop
	for _, l := range c.shown() {
		if l.literal {
			liveFont.Draw(dst, l.text, x, y, 1, l.clr)
		} else {
			h.Font.Draw(dst, l.text, x, y, 1, l.clr)
		}
		if l.stop {
			h.drawLiveStop(dst, y)
		}
		y += ui.LineHeight
	}
}

func (h *HUD) settleLive(f Frame) {
	if h.live == nil || !ui.Naming() {
		return
	}
	for _, v := range f.State.Calamities {
		for _, text := range []string{v.Cause, v.Why, v.Mended} {
			ui.Settle(text)
		}
		for _, r := range v.Remedies {
			ui.Settle(r.Label)
		}
	}
	for _, c := range f.State.Chronicles {
		for _, text := range []string{c.Question, c.Truth, c.Explanation, c.Subject, c.MendLine} {
			ui.Settle(text)
		}
		for _, o := range c.Options {
			ui.Settle(o.Text)
		}
	}
}
