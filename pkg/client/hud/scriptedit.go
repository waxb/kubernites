package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	EditTab        = "edit"
	EditPrefix     = "scr:live:"
	EditSendID     = EditPrefix + "send"
	EditReloadID   = EditPrefix + "reload"
	EditForceID    = EditPrefix + "force"
	EditForceYesID = EditPrefix + "forceyes"
	EditForceNoID  = EditPrefix + "forceno"
	EditConfirmID  = EditPrefix + "confirm"
	EditCancelID   = EditPrefix + "cancel"
	EditRevealID   = EditPrefix + "reveal:"
)

type LiveEdit struct {
	Ref       string
	Kind      string
	Namespace string
	Name      string
	Format    string
	Text      string
	Hidden    []string
}

type EditConflict struct {
	Manager string
	Fields  []string
}

type StoneEditEnd struct {
	OK        bool
	Output    string
	Conflicts []EditConflict
	Stale     bool
	Refused   bool
}

type editState struct {
	edit      LiveEdit
	run       *stoneRun
	reloading bool
	conflicts []EditConflict
	forcing   bool
	result    []string
	ok        bool
	stale     bool
	refused   bool
	at        float64
	revealing string
	reveal    <-chan Revealed
}

func (h *HUD) LiveEditing() (LiveEdit, bool) {
	if h.liveEdit == nil {
		return LiveEdit{}, false
	}
	return h.liveEdit.edit, true
}

func (h *HUD) OpenLiveEdit(e LiveEdit) {
	h.liveEdit = &editState{edit: e}
	h.loadLiveEdit(e)
	h.OverlayID = OverlayScriptorium
}

func (h *HUD) loadLiveEdit(e LiveEdit) {
	s := &h.Script
	s.Editor = NewEditor(e.Text)
	s.Story, s.Opened, s.Raw = ui.StoryNaming(), true, true
	s.Confirm, s.Warn, s.Result, s.ErrLine, s.Pending, s.Approved = "", "", nil, 0, false, false
}

func (h *HUD) closeLiveEdit() {
	if h.liveEdit == nil {
		return
	}
	if r := h.liveEdit.run; r != nil {
		r.job.Stop()
	}
	h.liveEdit = nil
	h.Script = NewScript()
}

func (h *HUD) busyEdit() bool {
	return h.liveEdit != nil && h.liveEdit.run != nil
}

func (h *HUD) sendLiveEdit(force bool) []Intent {
	st := h.liveEdit
	if st == nil || st.run != nil || st.revealing != "" || h.backend.Send == nil || h.liveReadOnly() {
		return nil
	}
	job, ok := h.backend.Send(st.edit, h.Script.Editor.Text(), force)
	if !ok {
		return nil
	}
	st.run, st.reloading, st.forcing, st.result, st.conflicts, st.stale, st.refused = &stoneRun{job: job}, false, false, nil, nil, false, false
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) reloadLiveEdit() []Intent {
	st := h.liveEdit
	if st == nil || st.run != nil || h.backend.Start == nil {
		return nil
	}
	job, ok := h.backend.Start(EditCommand(st.edit.Ref, st.edit.Namespace))
	if !ok {
		return nil
	}
	st.run, st.reloading = &stoneRun{job: job}, true
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func EditCommand(ref, namespace string) string {
	cmd := "edit " + ref
	if namespace != "" {
		cmd += " -n " + namespace
	}
	return cmd
}

func (h *HUD) pumpEdit() []Intent {
	h.pumpReveal()
	st := h.liveEdit
	if st == nil || st.run == nil {
		return nil
	}
	r := st.run
	_, done := r.job.Take()
	r.readAsk()
	if !done {
		return nil
	}
	st.run = nil
	end := r.job.End()
	if st.reloading {
		st.reloading = false
		if end.Edit != nil && end.OK {
			st.edit = *end.Edit
			h.loadLiveEdit(st.edit)
			st.result, st.ok, st.at = []string{ui.Bridge("Read again from the realm.", "Read again from the cluster.")}, true, h.Now
			return []Intent{{Kind: IntentSound, Command: "click"}}
		}
		st.result, st.ok, st.at = SplitOutput(endOutput(end)), false, h.Now
		return []Intent{{Kind: IntentSound, Command: "error"}}
	}
	return h.settleSent(end.Sent)
}

func (h *HUD) ShowSent(sent StoneEditEnd) []Intent {
	if h.liveEdit == nil {
		return nil
	}
	return h.settleSent(sent)
}

func (h *HUD) settleSent(sent StoneEditEnd) []Intent {
	st := h.liveEdit
	st.at = h.Now
	switch {
	case sent.OK:
		st.result, st.ok = SplitOutput(sent.Output), true
		h.Script.Editor.Dirty = false
		return append(h.reloadLiveEdit(), Intent{Kind: IntentSound, Command: "rollout"})
	case len(sent.Conflicts) > 0:
		st.conflicts, st.ok = sent.Conflicts, false
		st.result = []string{ConflictLine(sent.Conflicts)}
	case sent.Stale:
		st.stale, st.ok = true, false
		st.result = []string{ui.Bridge("The decree changed in the realm since it was read: RELOAD and write it again.", "The object changed since it was read (409 Conflict): RELOAD and edit it again.")}
	case sent.Refused:
		st.refused, st.ok = true, false
		st.result = []string{ui.Bridge("This realm is read-only: it is only watched, and nothing was sent.", "Refused before it left Kubernites: this connection is READ ONLY. Nothing was sent.")}
	default:
		st.result, st.ok = SplitOutput(sent.Output), false
	}
	return []Intent{{Kind: IntentSound, Command: "error"}}
}

func endOutput(end StoneEnd) string {
	if end.Sent.Output != "" {
		return end.Sent.Output
	}
	return ui.Bridge("The decree could not be read from the realm.", "The object could not be read.")
}

func ConflictLine(cs []EditConflict) string {
	var parts []string
	for _, c := range cs {
		parts = append(parts, fmt.Sprintf("%s is managed by %s", strings.Join(c.Fields, ", "), c.Manager))
	}
	return ui.Bridge("CONFLICT: another hand keeps these lines: ", "CONFLICT: ") + strings.Join(parts, "; ")
}

func ForceWarning(cs []EditConflict) string {
	var managers []string
	for _, c := range cs {
		managers = append(managers, c.Manager)
	}
	return ui.Bridge("FORCE takes these lines from "+strings.Join(managers, ", ")+": Kubernites' hand alone keeps them, and if the other hand writes them again it changes them back.",
		"FORCE makes kubernites the only manager of these fields and removes them from "+strings.Join(managers, ", ")+"; if that manager writes them again, it changes them back.")
}

func (h *HUD) liveEditButtons() []Button {
	st := h.liveEdit
	rail, foot := scriptRail(), scriptFooter()
	out := []Button{}
	for i, key := range st.edit.Hidden {
		if st.run != nil {
			break
		}
		label := "REVEAL " + strings.ToUpper(ui.Clip(key, 14))
		if st.revealing == key {
			label = "REVEALING..."
		}
		out = append(out, Button{ID: EditRevealID + key, Rect: ui.Rect{X: rail.X + 4, Y: rail.Y + 96 + i*20, W: rail.W - 8, H: 16}, Label: label, Tip: ui.Bridge("Show this one hidden value", "Show this key's value; the others stay masked"), Disabled: st.revealing != ""})
	}
	if r := st.run; r != nil && r.ask != nil {
		label, tip := r.ask.GoLabel()
		w := r.ask.buttonW(h)
		if !r.ask.Confirm {
			label, tip, w = "CLOSE", "Nothing is sent", 84
		}
		out = append(out, Button{ID: EditConfirmID, Rect: ui.Rect{X: foot.X + foot.W - 92 - w, Y: foot.Y + 2, W: w, H: 18}, Label: label, Tip: tip})
		if r.ask.Confirm {
			out = append(out, Button{ID: EditCancelID, Rect: ui.Rect{X: foot.X + foot.W - 88, Y: foot.Y + 2, W: 84, H: 18}, Label: "CANCEL", Tip: "Send nothing [Esc]"})
		}
		return append(out, closeButton(Overlay))
	}
	switch {
	case st.forcing:
		out = append(out,
			Button{ID: EditForceYesID, Rect: ui.Rect{X: foot.X + foot.W - 176, Y: foot.Y + 2, W: 84, H: 18}, Label: "FORCE", Tip: ForceWarning(st.conflicts)},
			Button{ID: EditForceNoID, Rect: ui.Rect{X: foot.X + foot.W - 88, Y: foot.Y + 2, W: 84, H: 18}, Label: "CANCEL", Tip: "Keep the conflict; send nothing [Esc]"})
	case len(st.conflicts) > 0:
		out = append(out,
			Button{ID: EditForceID, Rect: ui.Rect{X: foot.X + foot.W - 212, Y: foot.Y + 2, W: 100, H: 18}, Label: "FORCE...", Tip: ui.Bridge("Take the lines from their keeper, after a warning", "Send again with --force-conflicts, after a warning")},
			Button{ID: EditReloadID, Rect: ui.Rect{X: foot.X + foot.W - 104, Y: foot.Y + 2, W: 100, H: 18}, Label: "RELOAD", Tip: "Read the live object again; your changes are lost"})
	default:
		label := "SEND ^ENTER"
		if st.run != nil {
			label = "..."
		}
		out = append(out,
			Button{ID: EditReloadID, Rect: ui.Rect{X: foot.X + foot.W - 212, Y: foot.Y + 2, W: 100, H: 18}, Label: "RELOAD", Tip: "Read the live object again; your changes are lost", Disabled: st.run != nil},
			h.sendButton(foot, label, st.run != nil || st.revealing != ""))
	}
	return append(out, closeButton(Overlay))
}

func (h *HUD) pressLiveEdit(rest string) []Intent {
	st := h.liveEdit
	if st == nil {
		return nil
	}
	click := []Intent{{Kind: IntentSound, Command: "click"}}
	switch {
	case rest == "send":
		return h.sendLiveEdit(false)
	case rest == "reload":
		return h.reloadLiveEdit()
	case rest == "force":
		st.forcing = true
	case rest == "forceyes":
		return h.sendLiveEdit(true)
	case rest == "forceno":
		st.forcing = false
	case rest == "confirm" && st.run != nil:
		st.run.answer(true)
	case rest == "cancel" && st.run != nil:
		st.run.answer(false)
	case strings.HasPrefix(rest, "reveal:"):
		h.revealKey(strings.TrimPrefix(rest, "reveal:"))
	}
	return click
}

func (h *HUD) revealKey(key string) {
	st := h.liveEdit
	if st == nil || h.backend.Reveal == nil || st.run != nil || st.revealing != "" || !slices.Contains(st.edit.Hidden, key) {
		return
	}
	st.revealing, st.reveal = key, h.backend.Reveal(st.edit, key)
}

func (h *HUD) pumpReveal() {
	st := h.liveEdit
	if st == nil || st.reveal == nil {
		return
	}
	var r Revealed
	select {
	case r = <-st.reveal:
	default:
		return
	}
	key := st.revealing
	st.revealing, st.reveal = "", nil
	if !r.OK {
		return
	}
	text, done := RevealLine(h.Script.Editor.Text(), key, r.Value)
	if !done {
		return
	}
	dirty := h.Script.Editor.Dirty
	h.Script.Editor = NewEditor(text)
	h.Script.Editor.Dirty = dirty
	st.edit.Hidden = without(st.edit.Hidden, key)
}

func without(list []string, s string) []string {
	out := []string{}
	for _, x := range list {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}

func RevealLine(text, key, value string) (string, bool) {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		trimmed := strings.TrimLeft(l, " ")
		if (strings.HasPrefix(trimmed, key+": ") || strings.HasPrefix(trimmed, `"`+key+`": `)) && strings.Contains(trimmed, MaskedValue) {
			lines[i] = strings.Replace(l, MaskedValue, value, 1)
			return strings.Join(lines, "\n"), true
		}
	}
	return text, false
}

const MaskedValue = "KioqKg=="

func (h *HUD) liveEditKeys(in Input) []Intent {
	st := h.liveEdit
	if r := st.run; r != nil {
		switch {
		case r.yes(in):
			r.answer(true)
			return []Intent{{Kind: IntentSound, Command: "click"}}
		case in.Pressed(ebiten.KeyEscape) && r.ask != nil:
			r.answer(false)
		}
		return nil
	}
	if st.forcing {
		if in.Pressed(ebiten.KeyEscape) {
			st.forcing = false
		}
		return nil
	}
	s := &h.Script
	for _, r := range in.Chars {
		s.Editor.Insert(r)
	}
	for _, k := range in.Keys {
		if edit, ok := editorKeys[k]; ok {
			edit(&s.Editor)
			continue
		}
		switch {
		case k == ebiten.KeyEscape:
			h.OverlayID = OverlayNone
		case (k == ebiten.KeyEnter || k == ebiten.KeyNumpadEnter) && in.Control:
			return h.sendLiveEdit(false)
		case k == ebiten.KeyEnter || k == ebiten.KeyNumpadEnter:
			s.Editor.Newline()
		}
	}
	if len(in.Chars) > 0 || len(in.Keys) > 0 {
		s.Editor.Follow(scriptVisible(), s.columns())
	}
	return nil
}

func (h *HUD) drawLiveEditRail(dst *ebiten.Image, rail ui.Rect) {
	st := h.liveEdit
	h.Font.Draw(dst, ui.Bridge("THE LIVING DECREE", "LIVE OBJECT"), rail.X+6, rail.Y+5, 1, ui.Palette.TextDim)
	y := rail.Y + 20
	width := ui.MaxChars(rail.W-12, 1)
	h.Font.Draw(dst, ui.Clip(st.edit.Kind, width), rail.X+6, y, 1, ui.Palette.Cream)
	y += ui.LineHeight
	for _, row := range []string{st.edit.Name, st.edit.Namespace} {
		if row == "" {
			continue
		}
		Literal.Draw(dst, Literal.Clip(row, width), rail.X+6, y, 1, ui.Palette.Cream)
		y += ui.LineHeight
	}
	y += 4
	for _, row := range h.Font.Wrap(ui.Bridge("Sent whole, as Kubernites' own hand. Lines kept by another hand stop it.", "Sent with server-side apply as kubernites. A field another manager owns is a conflict."), rail.W-12, 1) {
		h.Font.Draw(dst, row, rail.X+6, y, 1, ui.Palette.TextDim)
		y += ui.LineHeight
	}
	if len(st.edit.Hidden) > 0 {
		h.Font.Draw(dst, ui.Bridge("HIDDEN VALUES", "MASKED KEYS"), rail.X+6, rail.Y+84, 1, ui.Palette.TextDim)
	}
	if len(st.conflicts) == 0 {
		return
	}
	y = max(y, rail.Y+96+len(st.edit.Hidden)*20) + 6
	h.Font.Draw(dst, "CONFLICT", rail.X+6, y, 1, ui.Palette.RedText)
	y += ui.LineHeight
	for _, c := range st.conflicts {
		for _, row := range Literal.Wrap(c.Manager+": "+strings.Join(c.Fields, ", "), rail.W-12, 1) {
			Literal.Draw(dst, row, rail.X+6, y, 1, ui.Palette.Cream)
			y += ui.LineHeight
		}
	}
}

func (h *HUD) drawLiveEditFooter(dst *ebiten.Image, r ui.Rect) {
	st := h.liveEdit
	ui.Well(dst, r)
	width := liveEditFootChars()
	switch {
	case st.run != nil && st.run.ask != nil:
		h.Font.Draw(dst, ui.Clip(st.run.ask.Title, width), r.X+6, r.Y+7, 1, ui.Palette.GoldBright)
	case st.run != nil:
		h.Font.Draw(dst, ui.Bridge("Sending...", "Sending with server-side apply..."), r.X+6, r.Y+7, 1, ui.Palette.TextDim)
	case st.forcing:
		h.Font.Draw(dst, ui.Clip(ui.Bridge("FORCE has its own warning, above.", "FORCE has its own warning, above: read it first."), width), r.X+6, r.Y+7, 1, ui.Palette.GoldBright)
	case len(st.result) > 0 && !st.ok:
		Literal.Draw(dst, Literal.Clip(strings.Join(st.result, "  "), width), r.X+6, r.Y+7, 1, ui.Palette.RedText)
	case len(st.result) > 0 && h.Now-st.at < resultFade:
		Literal.Draw(dst, Literal.Clip(strings.Join(st.result, "  "), width), r.X+6, r.Y+7, 1, ui.Palette.Green)
	case h.liveReadOnly():
		h.Font.Draw(dst, ui.Clip(readOnlySend(), width), r.X+6, r.Y+7, 1, ui.Palette.TextDim)
	default:
		h.Font.Draw(dst, liveEditSendHint(), r.X+6, r.Y+7, 1, ui.Palette.TextDim)
	}
}

func (h *HUD) liveReadOnly() bool {
	return h.live != nil && h.live.Level == LevelReadOnly
}

func liveEditFootChars() int {
	return ui.MaxChars(scriptFooter().W-230, 1)
}

func liveEditSendHint() string {
	return ui.Bridge("Ctrl+Enter sends it. Esc closes.", "Ctrl+Enter sends it with server-side apply. Esc closes.")
}

func readOnlySend() string {
	return ui.Bridge("Read only: this visit sends nothing to the realm. Esc closes.", "READ ONLY: no writes are sent; reconnect at WRITES to send. Esc closes.")
}

func (h *HUD) drawLiveEditAsk(dst *ebiten.Image, editor ui.Rect) {
	st := h.liveEdit
	if st.forcing {
		h.drawEditBox(dst, editor, ui.Bridge("FORCE?", "FORCE: --force-conflicts"), []string{ForceWarning(st.conflicts), ConflictLine(st.conflicts)})
		return
	}
	if st.run == nil || st.run.ask == nil {
		return
	}
	h.drawEditBox(dst, editor, st.run.ask.Title, st.run.ask.Lines)
}

func (h *HUD) drawEditBox(dst *ebiten.Image, editor ui.Rect, title string, lines []string) {
	a := StoneAsk{Title: title, Lines: lines}
	var rows []string
	for _, l := range a.Lines {
		rows = append(rows, Literal.Wrap(l, editor.W-28, 1)...)
	}
	box := ui.Rect{X: editor.X + 6, Y: editor.Y + editor.H - 16 - len(rows)*ui.LineHeight - 18, W: editor.W - 12, H: 18 + len(rows)*ui.LineHeight + 10}
	ui.DarkPanel(dst, box)
	ui.Stroke(dst, box, ui.Palette.Gold)
	h.Font.Draw(dst, a.Title, box.X+8, box.Y+6, 1, ui.Palette.GoldBright)
	y := box.Y + 20
	for _, row := range rows {
		Literal.Draw(dst, row, box.X+8, y, 1, ui.Palette.Cream)
		y += ui.LineHeight
	}
}

func (h *HUD) sendButton(foot ui.Rect, label string, busy bool) Button {
	b := Button{ID: EditSendID, Rect: ui.Rect{X: foot.X + foot.W - 104, Y: foot.Y + 2, W: 100, H: 18}, Label: label, Tip: ui.Bridge("Send this decree to the realm, as Kubernites' own hand [Ctrl+Enter]", "Server-side apply as field manager kubernites [Ctrl+Enter]"), Disabled: busy}
	if h.liveReadOnly() {
		b.Disabled, b.Tip = true, readOnlySend()
	}
	return b
}
