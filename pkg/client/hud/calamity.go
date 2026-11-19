package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	calamityPrefix = "calamity"
	CalamityClose  = calamityPrefix + ":close"
	CalamityNext   = calamityPrefix + ":next"
	calamityShelve = calamityPrefix + ":shelve"
	calamityBack   = "back:"
	shelveTilePre  = "shelved:"
	CalamityShow   = calamityPrefix + ":show"
	calamityFix    = calamityPrefix + ":fix"
	calamityFixRow = calamityOKH + 6
	CalamityOK     = calamityPrefix + ":ok"
	calamityOKW    = 48
	calamityOKH    = 16
	calamityOKRow  = calamityOKH + 6
	dismissTip     = "Dismiss: it returns only if it strikes again"
	causePrefix    = "cause"
	CalamitySettle = 5.0
	DismissMemory  = 30.0
	CalamityGap    = 45.0
	RepeatWait     = 60.0
	calamityW      = 340
	CalamityPad    = 6
	CalamityHeadH  = 20
	maxRemedies    = 2
	causeLines     = 6
	whyLines       = 8
	calamityShowW  = 38
	calamityNextW  = 18
	calamityBtnH   = 14
	remedyRowH     = ui.LineHeight + ChipH + lineGap
	RemedyMaxLines = 2
	remedyW        = calamityW - 2*CalamityPad
)

type CalamityState struct {
	since     map[string]float64
	Dismissed map[string]float64 `json:"-"`
	opened    map[string]float64
	now       float64
	index     int
}

type CauseTip struct {
	Rect ui.Rect
	Text string
}

type CalamityGate struct {
	repeat func(view.CalamityView) bool
	hidden func(view.CalamityView) bool
}

func (g CalamityGate) wait(v view.CalamityView) float64 {
	if g.repeat != nil && g.repeat(v) {
		return CalamitySettle + RepeatWait
	}
	return CalamitySettle
}

func (g CalamityGate) drawable(v view.CalamityView) bool {
	return g.hidden == nil || !g.hidden(v)
}

func (c CalamityState) Step(list []view.CalamityView, now float64, gate CalamityGate) CalamityState {
	next := CalamityState{since: map[string]float64{}, Dismissed: map[string]float64{}, opened: map[string]float64{}, now: now, index: c.index}
	present := map[string]bool{}
	for _, v := range list {
		present[v.ID] = true
		at, ok := c.since[v.ID]
		if !ok {
			at = now
		}
		next.since[v.ID] = at
		if t, ok := c.opened[v.ID]; ok {
			next.opened[v.ID] = t
			continue
		}
		if v.Mended == "" && !v.Passing && !v.Quiet && now-at >= gate.wait(v) && gate.drawable(v) {
			next.opened[v.ID] = now
		}
	}
	for id, seen := range c.Dismissed {
		switch {
		case present[id]:
			next.Dismissed[id] = now
		case now-seen < DismissMemory:
			next.Dismissed[id] = seen
		}
	}
	return next
}

func (c CalamityState) Dismiss(id string) CalamityState {
	next := CalamityState{since: c.since, Dismissed: map[string]float64{id: c.now}, opened: c.opened, now: c.now, index: c.index}
	for k, v := range c.Dismissed {
		if k != id {
			next.Dismissed[k] = v
		}
	}
	return next
}

func (c CalamityState) Open(id string, now float64) CalamityState {
	next := CalamityState{since: c.since, Dismissed: map[string]float64{}, opened: map[string]float64{id: now}, now: c.now, index: c.index}
	for k, v := range c.Dismissed {
		if k != id {
			next.Dismissed[k] = v
		}
	}
	for k, v := range c.opened {
		if k != id {
			next.opened[k] = v
		}
	}
	return next
}

func (c CalamityState) Ready(list []view.CalamityView, now float64) []view.CalamityView {
	out := []view.CalamityView{}
	for _, v := range list {
		_, dismissed := c.Dismissed[v.ID]
		if _, ok := c.opened[v.ID]; ok && !dismissed {
			out = append(out, v)
		}
	}
	return out
}

func repeatCalamity(f Frame, v view.CalamityView) bool {
	inc, ok := incidentOf(f, v.Incident)
	return ok && !inc.Guided
}

func (h *HUD) calamityGate(f Frame) CalamityGate {
	return CalamityGate{
		repeat: func(v view.CalamityView) bool { return repeatCalamity(f, v) },
		hidden: func(v view.CalamityView) bool {
			return h.OverlayID != OverlayNone || !calamityVisible(v, f) || EchoesObjective(v, f)
		},
	}
}

func CalamitiesShown(f Frame) bool {
	return f.Shown(SurfaceRail)
}

func calamityVisible(v view.CalamityView, f Frame) bool {
	return CalamitiesShown(f) || v.First
}

func commandKey(cmd string) string {
	return strings.Join(commandWords(cmd), " ")
}

func EchoesObjective(v view.CalamityView, f Frame) bool {
	l, ok := f.ObjectiveLine()
	if !ok || l.Command == "" {
		return false
	}
	for _, r := range v.Remedies {
		if r.Command != "" && commandKey(r.Command) == commandKey(l.Command) {
			return true
		}
	}
	return false
}

func (h *HUD) readyCalamities(f Frame) []view.CalamityView {
	out := []view.CalamityView{}
	for _, v := range h.Ops.Calamity.Ready(f.State.Calamities, f.Clock) {
		if !EchoesObjective(v, f) {
			out = append(out, v)
		}
	}
	return out
}

func (h *HUD) ShownCalamity(f Frame) (view.CalamityView, bool) {
	v, _, _, ok := h.CurrentCalamity(f)
	return v, ok
}

func (h *HUD) visibleCalamities(f Frame) []view.CalamityView {
	list := []view.CalamityView{}
	for _, v := range h.readyCalamities(f) {
		if calamityVisible(v, f) {
			list = append(list, v)
		}
	}
	return list
}

func (h *HUD) CurrentCalamity(f Frame) (view.CalamityView, int, int, bool) {
	if h.OverlayID != OverlayNone {
		return view.CalamityView{}, 0, 0, false
	}
	list := h.visibleCalamities(f)
	if len(list) == 0 {
		return view.CalamityView{}, 0, 0, false
	}
	if !h.Director.Stage.Primed {
		i := h.Ops.Calamity.index % len(list)
		return list[i], i, len(list), true
	}
	focus := h.Director.Stage.focus
	for i, v := range list {
		if v.ID == focus && h.Director.Stage.Has(ActTrouble+focus) {
			return v, i, len(list), true
		}
	}
	return view.CalamityView{}, 0, 0, false
}

func (h *HUD) troubleActed() {
	s := &h.Director.Stage
	if s.Primed && s.focus != "" {
		s.acted = withKey(s.acted, s.focus)
	}
}

func (h *HUD) TroubleOpen(id string) bool {
	s := h.Director.Stage
	key := ActTrouble + id
	return s.Primed && s.Has(key) && s.Waits[key] && !s.acted[id]
}

func (h *HUD) TroubleSetAside(id string) bool {
	s := h.Director.Stage
	_, closed := h.Ops.Calamity.Dismissed[id]
	return s.Primed && s.read[id] && !s.acted[id] && !closed && !s.Has(ActTrouble+id)
}

func (h *HUD) focusTrouble(id string) {
	s := &h.Director.Stage
	if !s.Primed || s.focus == id {
		return
	}
	if s.focus != "" {
		delete(s.On, ActTrouble+s.focus)
		delete(s.Waits, ActTrouble+s.focus)
	}
	s.focus = id
	s.read = withKey(s.read, id)
	s.On[ActTrouble+id], s.Waits[ActTrouble+id] = h.Now, true
	if kind, _, _ := strings.Cut(id, ":"); !urgentTrouble(kind) {
		h.soaksWhenGone(ActTrouble + id)
	}
}

func RemedyLine(r view.RemedyView) CommandLine {
	if r.Label == bridgeLabel {
		bridgeLabelText(r.Label)
	}
	l := CommandLine{Label: r.Label, Command: ShownCommand(r.Command), Action: r.Action, Blocked: r.Blocked, Unwired: r.Unwired}
	if r.Order != "" {
		l.Order = RemedyOrder(r)
	}
	return l
}

func RemedyOrder(r view.RemedyView) string {
	return r.Order + "|" + r.Target.Kind + "|" + r.Target.Namespace + "|" + r.Target.Name
}

func (h *HUD) calamityLines(f Frame) []CommandLine {
	v, _, _, ok := h.CurrentCalamity(f)
	if !ok {
		return nil
	}
	if t := h.steadyTrouble; t.key != "" && t.key == troubleKey(v) {
		return t.remedies
	}
	return h.rawCalamityLines(f, v)
}

func (h *HUD) rawCalamityLines(f Frame, v view.CalamityView) []CommandLine {
	out := []CommandLine{}
	for _, r := range v.Remedies[:min(len(v.Remedies), maxRemedies)] {
		if r.Command != "" || r.Action != nil {
			out = append(out, h.liveRemedy(withRemedyEffect(withFade(withPayoff(RemedyLine(r), f), f), f)))
		}
	}
	return out
}

func (h *HUD) CalamityText(f Frame, v view.CalamityView) ([]string, []string) {
	if t := h.steadyTrouble; t.key != "" && t.key == troubleKey(v) {
		return t.lines()
	}
	return h.rawCalamityText(f, v)
}

func (h *HUD) rawCalamityText(f Frame, v view.CalamityView) ([]string, []string) {
	w := calamityW - 2*CalamityPad
	if v.Mended != "" {
		return h.Font.WrapLines(v.Mended, w, 1, causeLines), h.Font.WrapLines(v.Cause, w, 1, whyLines)
	}
	cause, why := incidentTexts(f, v)
	related := relatedText(v)
	if why == "" && related == "" {
		return h.Font.WrapLines(cause, w, 1, causeLines), nil
	}
	lines := []string{}
	if why != "" {
		lines = h.Font.WrapLines(why, w, 1, whyLines)
	}
	if related != "" {
		lines = append(lines, h.Font.WrapLines(related, w, 1, relatedLines)...)
	}
	return h.Font.WrapLines(cause, w, 1, causeLines), lines
}

const relatedLines = 3

func relatedText(v view.CalamityView) string {
	if len(v.Related) == 0 {
		return ""
	}
	parts := []string{}
	for _, rel := range v.Related {
		parts = append(parts, model.CauseTitle(rel.Kind)+" ("+firstClause(rel.Cause)+")")
	}
	return "From the same cause: " + strings.Join(parts, "; ") + "."
}

func (h *HUD) remedyLabel(l CommandLine) []string {
	if l.Command == "" {
		return h.Font.WrapLines(strings.ToUpper(l.Label), remedyW-ButtonPad, 1, RemedyMaxLines)
	}
	return h.Font.WrapLines(l.Label, remedyW, 1, RemedyMaxLines)
}

func (h *HUD) remedyEffect(l CommandLine) []string {
	if l.Effect == "" {
		return nil
	}
	return h.Font.WrapLines(l.Effect, remedyW, 1, RemedyMaxLines)
}

func (h *HUD) remedyHeight(l CommandLine) int {
	extra := (len(h.remedyLabel(l))-1)*ui.LineHeight + len(h.remedyEffect(l))*ui.LineHeight
	if l.Command == "" {
		return ChipH + extra + lineGap
	}
	return remedyRowH + extra
}

func (h *HUD) remedyRow(card ui.Rect, y int, l CommandLine) ui.Rect {
	height := h.remedyHeight(l)
	if l.Command == "" {
		return ui.Rect{X: card.X + CalamityPad, Y: y, W: remedyW, H: height - lineGap - len(h.remedyEffect(l))*ui.LineHeight}
	}
	return ui.Rect{X: card.X + CalamityPad, Y: y + height - ChipH - lineGap, W: remedyW, H: ChipH}
}

func (h *HUD) steadyCalamityHeight(f Frame, v view.CalamityView) int {
	height := h.calamityHeight(f, v, h.calamityLines(f), h.calamityVig(f, v))
	if t := h.steadyTrouble; t.key != "" && t.key == troubleKey(v) {
		return max(height, t.height)
	}
	return height
}

func (h *HUD) calamityHeight(f Frame, v view.CalamityView, lines []CommandLine, vig int) int {
	cause, why := h.CalamityText(f, v)
	height := CalamityPad + h.calamityHead() + (len(cause)+len(why))*ui.LineHeight + h.incidentHeight(f, v) + vig + CalamityPad
	for _, l := range lines {
		height += h.remedyHeight(l)
	}
	if deadEnd(lines, v) {
		height += calamityOKRow
	}
	if _, ok := fixLine(lines); ok {
		height += calamityFixRow
	}
	return height + InviteHeight(v.Invite) + h.calamityOfferRow(f)
}

func fixLine(lines []CommandLine) (CommandLine, bool) {
	for _, l := range lines {
		if l.Show && l.Action != nil {
			return l, true
		}
	}
	return CommandLine{}, false
}

func fixWords(a view.Action, label string) string {
	if a.Action == view.ActScaleCrew && a.Replicas > 0 {
		return fmt.Sprintf("scale %s to %d", fixCrew(a), a.Replicas)
	}
	return strings.ToLower(label)
}

func deadEnd(lines []CommandLine, v view.CalamityView) bool {
	return len(lines) == 0 && v.Invite == nil
}

func CalamitySpots(height int) []ui.Rect {
	right := Width - calamityW - railRight - railSize - cardMargin - 2
	bottom := ConsoleY - height - cardMargin
	return []ui.Rect{
		{X: right, Y: bottom, W: calamityW, H: height}, {X: cardMargin, Y: bottom, W: calamityW, H: height},
		{X: right, Y: TopH + cardMargin, W: calamityW, H: height}, {X: cardMargin, Y: TopH + cardMargin, W: calamityW, H: height},
		{X: Width/2 - calamityW/2, Y: bottom, W: calamityW, H: height},
	}
}

func (h *HUD) calamityBlocked(f Frame) []ui.Rect {
	return append(inflated(h.GuideTargets(f), targetClear), h.calamityPanels(f)...)
}

func (h *HUD) calamityPanels(f Frame) []ui.Rect {
	out := inflated(FreightRects(f), 2)
	if h.ObjectiveShown(f) {
		out = append(out, h.ObjectiveRectOf(f).Inset(-2))
	}
	if h.LessonShown(f) {
		out = append(out, h.LessonRect(f).Inset(-2))
	}
	if h.IntroShown(f) {
		out = append(out, h.IntroCard(f).Inset(-2))
	}
	if rail := h.RailRect(f); rail.W > 0 {
		out = append(out, rail.Inset(-2))
	}
	if SiegeActive(f) {
		out = append(out, SiegeStripRect(f))
	}
	if h.LegendShown(f) {
		out = append(out, LegendRect(h.lens))
	}
	return append(out, inflated(h.crewControls(f), 2)...)
}

func (h *HUD) crewControls(f Frame) []ui.Rect {
	out := h.CrewRects(f)
	if r, ok := crewsWindow(h, f); ok {
		out = append(out, r)
	}
	if r, ok := kindWindow(h, f); ok {
		out = append(out, r)
	}
	return out
}

func (h *HUD) CalamityRect(f Frame) (ui.Rect, bool) {
	v, _, _, ok := h.CurrentCalamity(f)
	if !ok {
		return ui.Rect{}, false
	}
	spots := CalamitySpots(h.steadyCalamityHeight(f, v))
	r := f.placed(PanelCalamity, FirstClear(offCrew(spots, h.crewControls(f)), h.calamityBlocked(f)))
	_, pinned := f.Panels[PanelCalamity]
	return r, pinned || blockedArea(r, h.calamityPanels(f)) == 0
}

func offCrew(spots, crew []ui.Rect) []ui.Rect {
	clear := []ui.Rect{}
	for _, s := range spots {
		if blockedArea(s, crew) == 0 {
			clear = append(clear, s)
		}
	}
	if len(clear) == 0 {
		return spots
	}
	return clear
}

func (h *HUD) calamityVig(f Frame, v view.CalamityView) int {
	if t := h.steadyTrouble; t.key != "" && t.key == troubleKey(v) {
		return t.vig
	}
	return h.rawCalamityVig(f, v, h.calamityLines(f))
}

func (h *HUD) rawCalamityVig(f Frame, v view.CalamityView, lines []CommandLine) int {
	blocked := h.calamityBlocked(f)
	full := CalamitySpots(h.calamityHeight(f, v, lines, calamityVigH))
	if blockedArea(FirstClear(full, blocked), blocked) == 0 {
		return calamityVigH
	}
	return 0
}

func (h *HUD) calamityHeader(f Frame, r ui.Rect, v view.CalamityView, count int) []Button {
	out := []Button{closeButtonAs(r, CalamityClose, dismissTip)}
	x := r.X + r.W - closeReach - 2
	x -= calamityNextW
	out = append(out, Button{ID: calamityShelve, Rect: ui.Rect{X: x, Y: r.Y + 5, W: calamityNextW, H: calamityBtnH}, Label: "_", Tip: "Fold this card away: it waits on the rail and reopens from there"})
	x -= 4
	if count > 1 {
		x -= calamityNextW
		out = append(out, Button{ID: CalamityNext, Rect: ui.Rect{X: x, Y: r.Y + 5, W: calamityNextW, H: calamityBtnH}, Label: ">", Tip: h.moreTip(f, v)})
		x -= 4
	}
	if _, ok := f.Places[v.Object]; ok {
		x -= calamityShowW
		out = append(out, Button{ID: CalamityShow, Rect: ui.Rect{X: x, Y: r.Y + 5, W: calamityShowW, H: calamityBtnH}, Label: "SHOW", Tip: "Glide to where it strikes", Selected: showLit(f, v)})
		x -= 4
	}
	return append(out, h.calamityAlmanacButton(f, r, x, v.Kind, v.Incident)...)
}

func (h *HUD) moreTip(f Frame, shown view.CalamityView) string {
	lines := []string{"More troubles, the most urgent first:"}
	for _, v := range h.visibleCalamities(f) {
		if v.ID != shown.ID {
			lines = append(lines, model.CauseTitle(v.Kind)+": "+firstClause(v.Cause))
		}
	}
	return strings.Join(append(lines, "Press > for the next."), "\n")
}

func firstClause(text string) string {
	if head, _, ok := strings.Cut(text, ":"); ok && len(head) > 12 {
		return head
	}
	return ui.Clip(text, 60)
}

func (h *HUD) FocusCalamity(kind string, f Frame) bool {
	for i, v := range h.visibleCalamities(f) {
		if v.Kind == kind && v.Mended == "" {
			next := h.Ops.Calamity
			next.index = i
			h.Ops.Calamity = next
			h.focusTrouble(v.ID)
			return true
		}
	}
	for _, v := range f.State.Calamities {
		if v.Kind != kind || v.Mended != "" {
			continue
		}
		h.Ops.Calamity = h.Ops.Calamity.Open(v.ID, f.Clock)
		for i, w := range h.visibleCalamities(f) {
			if w.ID == v.ID {
				next := h.Ops.Calamity
				next.index = i
				h.Ops.Calamity = next
				h.focusTrouble(w.ID)
			}
		}
		return true
	}
	return false
}

func (h *HUD) CalamityButtons(f Frame) []Button {
	v, _, count, ok := h.CurrentCalamity(f)
	if !ok {
		return nil
	}
	r, placed := h.CalamityRect(f)
	if !placed {
		return nil
	}
	out := h.calamityHeader(f, r, v, count)
	cause, why := h.CalamityText(f, v)
	y := r.Y + CalamityPad + h.calamityHead() + (len(cause)+len(why))*ui.LineHeight + h.incidentHeight(f, v) + h.calamityVig(f, v)
	lines := h.calamityLines(f)
	for i, l := range lines {
		out = append(out, h.withOrderState(LineButtons(fmt.Sprintf("%s:cal:%d", linePrefix, i), h.remedyRow(r, y, l), l))...)
		y += h.remedyHeight(l)
	}
	if l, ok := fixLine(lines); ok {
		label := ui.Clip("FIX: "+fixWords(*l.Action, l.Label), ui.MaxChars(remedyW-ButtonPad, 1))
		out = append(out, Button{ID: calamityFix, Rect: ui.Rect{X: r.X + CalamityPad, Y: y + 2, W: remedyW, H: calamityOKH}, Label: label,
			Tip: "Apply the fix now: " + l.Label + ". SHOW lets you do it by hand instead."})
		y += calamityFixRow
	}
	if deadEnd(lines, v) {
		out = append(out, Button{ID: CalamityOK, Rect: ui.Rect{X: r.X + r.W - CalamityPad - calamityOKW, Y: y + 2, W: calamityOKW, H: calamityOKH}, Label: "OK", Tip: dismissTip})
	}
	offerY := y
	if h.calamityOfferRow(f) > 0 {
		offerY = r.Y + r.H - calamityOKRow
	}
	out = append(out, h.calamityOfferButtons(f, r, offerY)...)
	return append(out, InviteButtons(v.Invite, r, y)...)
}

func (h *HUD) shelve(id string) {
	next := make(map[string]bool, len(h.Ops.shelved)+1)
	for k, v := range h.Ops.shelved {
		next[k] = v
	}
	next[id] = true
	h.Ops.shelved = next
}

func (h *HUD) unshelve(id string) {
	next := make(map[string]bool, len(h.Ops.shelved))
	for k, v := range h.Ops.shelved {
		if k != id {
			next[k] = v
		}
	}
	h.Ops.shelved = next
}

func (h *HUD) pruneShelved(f Frame) {
	for id := range h.Ops.shelved {
		live := false
		for _, v := range f.State.Calamities {
			live = live || v.ID == id && v.Mended == ""
		}
		if !live {
			h.unshelve(id)
		}
	}
}

func (h *HUD) shelvedNotices(f Frame) []Notice {
	out := []Notice{}
	for _, v := range f.State.Calamities {
		if !h.Ops.shelved[v.ID] || v.Mended != "" {
			continue
		}
		out = append(out, Notice{ID: shelveTilePre + v.ID, Kind: NoticeDanger, Icon: "icon/alert", Title: model.CauseTitle(v.Kind),
			Detail: "A folded trouble card: " + firstClause(v.Cause), Intent: calamityPrefix + ":" + calamityBack + v.ID, Since: f.Clock - 10})
	}
	return out
}

func (h *HUD) PressCalamity(rest string, f Frame) []Intent {
	if id, ok := strings.CutPrefix(rest, calamityBack); ok {
		h.unshelve(id)
		h.Ops.Calamity = h.Ops.Calamity.Open(id, f.Clock)
		h.focusTrouble(id)
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	v, _, _, ok := h.CurrentCalamity(f)
	if !ok {
		return nil
	}
	click := []Intent{{Kind: IntentSound, Command: "click"}}
	switch rest {
	case "almanac":
		p, _ := calamityPage(f, v.Kind, v.Incident)
		return h.OpenBook(p.ID, f)
	case "shelve":
		h.shelve(v.ID)
		h.Ops.Calamity = h.Ops.Calamity.Dismiss(v.ID)
		return click
	case "close", "ok":
		h.Ops.Calamity = h.Ops.Calamity.Dismiss(v.ID)
		return click
	case "next":
		next := h.Ops.Calamity
		next.index++
		h.Ops.Calamity = next
		if list := h.visibleCalamities(f); len(list) > 1 {
			_, i, _, _ := h.CurrentCalamity(f)
			h.focusTrouble(list[(i+1)%len(list)].ID)
		}
		return click
	case "show":
		if at, found := f.Places[v.Object]; found {
			return h.focus(at)
		}
	case "fix":
		if l, ok := fixLine(h.calamityLines(f)); ok {
			h.troubleActed()
			return h.doLine(l, FromCalamityCard)
		}
	}
	return nil
}

func (h *HUD) DrawCalamity(dst *ebiten.Image, f Frame) {
	v, _, count, ok := h.CurrentCalamity(f)
	if !ok {
		return
	}
	r, placed := h.CalamityRect(f)
	if !placed {
		return
	}
	ui.Fill(dst, r.Offset(2, 3), ui.Palette.Shadow)
	ui.Fill(dst, r, ui.Tint(ui.Palette.Console, 240))
	accent, text := ui.Palette.RedText, ui.Palette.Text
	if v.Mended != "" {
		accent, text = ui.Palette.Green, ui.Palette.Green
	}
	ui.Stroke(dst, r, accent)
	h.icon(dst, "icon/alert", r.X+CalamityPad, r.Y+CalamityPad-1)
	title := model.CauseTitle(v.Kind)
	if v.Mended != "" {
		title += " MENDED"
	}
	h.Font.DrawShadow(dst, title, r.X+CalamityPad+20, r.Y+CalamityPad+3, 1, accent)
	if count > 1 {
		tw, _ := h.Font.Measure(title, 1)
		h.Font.Draw(dst, fmt.Sprintf("+%d MORE", count-1), r.X+CalamityPad+20+tw+2*ui.Advance, r.Y+CalamityPad+3, 1, ui.Palette.TextDim)
	}
	y := r.Y + CalamityPad + CalamityHeadH
	if line := h.asideLine(); line != "" {
		h.Font.Draw(dst, line, r.X+CalamityPad, y, 1, ui.Palette.Gold)
		y += ui.LineHeight
	}
	cause, why := h.CalamityText(f, v)
	for _, row := range cause {
		h.Font.Draw(dst, row, r.X+CalamityPad, y, 1, text)
		y += ui.LineHeight
	}
	for _, row := range why {
		h.Font.Draw(dst, row, r.X+CalamityPad, y, 1, ui.Palette.TextDim)
		y += ui.LineHeight
	}
	h.drawIncident(dst, f, v, r.X+CalamityPad, y)
	y += h.incidentHeight(f, v)
	vig := h.calamityVig(f, v)
	if vig > 0 {
		h.drawCalamityVig(dst, f, v, r)
	}
	y += vig
	shown := h.calamityLines(f)
	for _, l := range shown {
		h.drawRemedyLabel(dst, l, r.X+CalamityPad, y+1)
		y += h.remedyHeight(l)
	}
	if _, ok := fixLine(shown); ok {
		y += calamityFixRow
	}
	h.DrawInvite(dst, v.Invite, r, y)
	for _, b := range h.CalamityButtons(f) {
		if !h.drawLineButton(dst, b) && !h.drawOrderChip(dst, b) {
			h.drawButton(dst, b)
		}
	}
}

func (h *HUD) drawRemedyLabel(dst *ebiten.Image, l CommandLine, x, y int) {
	at := y + ChipH + (len(h.remedyLabel(l))-1)*ui.LineHeight + 1
	if l.Command != "" {
		ink := ui.Palette.Cream
		if l.Unwired {
			ink = ui.Palette.TextDim
		}
		for i, row := range h.remedyLabel(l) {
			h.Font.Draw(dst, row, x, y+i*ui.LineHeight, 1, ink)
		}
		at = y + len(h.remedyLabel(l))*ui.LineHeight
	}
	ink := ui.Palette.Green
	if l.EffectBad {
		ink = ui.Palette.RedText
	}
	for i, row := range h.remedyEffect(l) {
		h.Font.Draw(dst, row, x, at+i*ui.LineHeight, 1, ink)
	}
}

func (h *HUD) CauseButtons(f Frame) []Button {
	if h.OverlayID != OverlayNone || !CalamitiesShown(f) {
		return nil
	}
	out := []Button{}
	for i, c := range f.Causes {
		if c.Rect.Y > TopH && c.Rect.Y+c.Rect.H < ConsoleY {
			out = append(out, Button{ID: fmt.Sprintf("%s:%d", causePrefix, i), Rect: c.Rect, Tip: c.Text})
		}
	}
	return out
}

func init() {
	prefixHandlers[calamityPrefix] = (*HUD).PressCalamity
	prefixHandlers[causePrefix] = func(*HUD, string, Frame) []Intent { return nil }
	prefixHandlers[linePrefix] = (*HUD).PressLine
}
