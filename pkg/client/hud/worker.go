package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	PanelWorker   = "worker"
	workerW       = 280
	workerPad     = 6
	workerTop     = 5
	workerLabelW  = 9 * ui.Advance
	workerRecent  = 5
	workerThought = 2
	workerClose   = PanelWorker + ":close"
)

type WorkerView struct {
	Title   string
	Line    string
	Status  string
	Last    string
	Needs   []WorkerNeed
	Mood    string
	Thought string
	Recent  []WorkerThought
	View    *ebiten.Image
	Gone    bool
	Names   bool
}

type WorkerNeed struct {
	Label string
	Value string
	Mark  string
	Alarm bool
}

type WorkerThought struct {
	Ago  string
	Text string
}

var workerFont = ui.Font{Literal: true}

func workerShown(f Frame) bool {
	return f.cardSelection() && f.Selection.Kind == "walker" && f.Worker != nil
}

func moodLabel(w *WorkerView) string {
	if w.Names {
		return "HEALTH"
	}
	return "MOOD"
}

func (w *WorkerView) thoughtRows() []string {
	if w.Thought == "" {
		return nil
	}
	return workerFont.WrapLines("\""+w.Thought+"\"", workerW-2*workerPad-workerLabelW, 1, workerThought)
}

func (w *WorkerView) recentRows() []WorkerThought {
	out := []WorkerThought{}
	for _, t := range w.Recent[:min(len(w.Recent), workerRecent)] {
		for i, line := range workerFont.WrapLines(t.Text, workerW-2*workerPad-workerLabelW, 1, workerThought) {
			ago := ""
			if i == 0 {
				ago = t.Ago
			}
			out = append(out, WorkerThought{Ago: ago, Text: line})
		}
	}
	return out
}

func (w *WorkerView) rows() int {
	n := 3 + max(len(w.thoughtRows()), 1)
	if w.Gone {
		n++
	}
	n += len(w.Needs)
	if len(w.Recent) > 0 {
		n += 1 + len(w.recentRows())
	}
	return n
}

func WorkerWindow(hud *HUD, f Frame) (ui.Rect, bool) {
	if !workerShown(f) {
		return ui.Rect{}, false
	}
	h := workerTop + f.Worker.rows()*ui.LineHeight + workerPad
	y := ConsoleY - crewFloat - h
	others := hud.OtherWindows(f)
	home := ui.Rect{X: CardBox.X, Y: y, W: workerW, H: h}
	for _, x := range []int{CardBox.X, InfoWell.X, Width - workerW - workerPad} {
		spot := ui.Rect{X: x, Y: y, W: workerW, H: h}
		if !OverlapsAny(spot, others) {
			home = spot
			break
		}
	}
	return f.placed(PanelWorker, home), true
}

func (h *HUD) OtherWindows(f Frame) []ui.Rect {
	out := []ui.Rect{}
	for _, id := range stackedPanels {
		if id == PanelWorker {
			continue
		}
		if r, ok := WindowSpecs[id].Rect(h, f); ok {
			out = append(out, r)
		}
	}
	if r, ok := h.introWindow(f); ok {
		out = append(out, r)
	}
	if r, ok := h.TrayRect(f); ok {
		out = append(out, r)
	}
	return out
}

func (h *HUD) workerButtons(f Frame) []Button {
	r, ok := WorkerWindow(h, f)
	if !ok {
		return nil
	}
	return []Button{closeButtonAs(r, workerClose, "Deselect [Esc or right click]")}
}

func (h *HUD) PressWorker(rest string, f Frame) []Intent {
	if rest != "close" {
		return nil
	}
	return append(h.leaveObject(f), Intent{Kind: IntentSound, Command: "click"})
}

func (h *HUD) drawWorker(dst *ebiten.Image, f Frame) {
	r, ok := WorkerWindow(h, f)
	if !ok {
		return
	}
	w := f.Worker
	ui.DarkPanel(dst, r)
	h.drawButton(dst, closeButtonAs(r, workerClose, ""))
	x, y := r.X+workerPad, r.Y+workerTop
	workerFont.DrawShadow(dst, workerFont.Clip(w.Title, ui.MaxChars(workerW-2*workerPad-closeReach, 1)), x, y, 1, ui.Palette.GoldBright)
	y += ui.LineHeight
	if w.Line != "" {
		workerFont.Draw(dst, workerFont.Clip(w.Line, ui.MaxChars(workerW-2*workerPad, 1)), x, y, 1, ui.Palette.Blue)
	}
	y += ui.LineHeight
	if w.Gone {
		workerFont.Draw(dst, w.Status, x, y, 1, ui.Palette.GoldBright)
		y += ui.LineHeight
	}
	h.workerRow(dst, x, y, moodLabel(w), w.Mood, "", false)
	y += ui.LineHeight
	workerFont.Draw(dst, "THOUGHT", x, y, 1, ui.Palette.Gold)
	for _, row := range w.thoughtRows() {
		workerFont.Draw(dst, row, x+workerLabelW, y, 1, ui.Palette.Cream)
		y += ui.LineHeight
	}
	if len(w.thoughtRows()) == 0 {
		y += ui.LineHeight
	}
	for _, n := range w.Needs {
		h.workerRow(dst, x, y, n.Label, n.Value, n.Mark, n.Alarm)
		y += ui.LineHeight
	}
	if len(w.Recent) == 0 {
		return
	}
	workerFont.Draw(dst, "RECENT THOUGHTS", x, y, 1, ui.Palette.Gold)
	y += ui.LineHeight
	for _, t := range w.recentRows() {
		workerFont.Draw(dst, t.Ago, x, y, 1, ui.Palette.TextDim)
		workerFont.Draw(dst, t.Text, x+workerLabelW, y, 1, ui.Palette.Text)
		y += ui.LineHeight
	}
}

func (h *HUD) workerRow(dst *ebiten.Image, x, y int, label, value, mark string, alarm bool) {
	workerFont.Draw(dst, label, x, y, 1, ui.Palette.Gold)
	room := ui.MaxChars(workerW-2*workerPad-workerLabelW, 1)
	if mark != "" {
		room -= len(mark) + 1
	}
	shown := workerFont.Clip(value, room)
	workerFont.Draw(dst, shown, x+workerLabelW, y, 1, ui.Palette.Text)
	if mark == "" {
		return
	}
	clr := ui.Palette.GoldBright
	if alarm {
		clr = ui.Palette.RedText
	}
	workerFont.Draw(dst, mark, x+workerW-2*workerPad-workerFont.Width(mark, 1), y, 1, clr)
}

func init() {
	WindowSpecs[PanelWorker] = windowSpec{Rect: WorkerWindow, draw: (*HUD).drawWorker, buttons: (*HUD).workerButtons}
	prefixHandlers[PanelWorker] = (*HUD).PressWorker
}

func withWorker(c Card, f Frame) Card {
	w := f.Worker
	if f.Selection.Kind != "walker" || w == nil || w.Status == "" {
		return c
	}
	at := 0
	for i, l := range c.Lines {
		if l.Text == KuberniteLine(f.Selection.Pod) {
			at = i + 1
		}
	}
	status := Line{Text: w.Status, Clr: ui.Palette.Cream}
	return Card{Portrait: c.Portrait, Title: c.Title, K8s: c.K8s, progress: c.progress,
		Lines: append(append(append([]Line(nil), c.Lines[:at]...), status), c.Lines[at:]...)}
}

func goneCard(p model.PodView, f Frame) Card {
	c := Card{Portrait: "portrait/" + p.Sprite, Title: strings.ToUpper(p.Profession), K8s: "Pod/" + p.Name, progress: -1}
	if p.Focus != "" {
		c.Lines = append(c.Lines, Line{Text: KuberniteLine(p), Clr: ui.Palette.Cream})
	}
	if f.Worker != nil && f.Worker.Status != "" {
		c.Lines = append(c.Lines, Line{Text: f.Worker.Status, Clr: ui.Palette.GoldBright})
	}
	return c
}

func (h *HUD) DrawWorkerView(dst *ebiten.Image, box ui.Rect, f Frame) bool {
	if f.Selection.Kind != "walker" || f.Worker == nil || f.Worker.View == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(box.X), float64(box.Y))
	dst.DrawImage(f.Worker.View, op)
	return true
}
