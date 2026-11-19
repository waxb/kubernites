package hud

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	lessonFade     = 1.5
	LessonMaxLines = 4
	lessonGap      = 6
	lessonClose    = "lesson:close"
	LessonAck      = "lesson:ack"
	LessonShow     = "lesson:show"
)

type LessonCard struct {
	Title  string `json:"-"`
	Text   string `json:"-"`
	start  float64
	Shown  bool `json:"-"`
	Teach  bool `json:"-"`
	done   bool
	hold   bool
	at     *[2]float64
	still  func() bool
	timely bool
	once   string
}

func (l LessonCard) stale() bool {
	return l.still != nil && !l.still()
}

func (h *HUD) Lesson(title, text string) { h.showLesson(title, text, false) }

func (h *HUD) showLesson(title, text string, done bool) {
	if h.LessonCard.Shown && h.LessonCard.Title != title && !h.Director.Stage.Has(actLesson+h.LessonCard.Title) {
		h.Results = append([]LessonCard{h.LessonCard}, h.Results...)
	}
	h.LessonCard = LessonCard{Title: title, Text: text, start: h.Now, Shown: true, done: done}
	if s := &h.Director.Stage; s.Primed && !done {
		s.On[actLesson+title], s.Waits[actLesson+title] = h.Now, true
		h.takeSlot(Frame{}, stageAct{key: actLesson + title, waits: true, idea: true})
	}
}

func (h *HUD) LessonTitle() string {
	if _, ok := h.lessonAge(); !ok {
		return ""
	}
	return h.LessonCard.Title
}

func (h *HUD) LessonText() string {
	if _, ok := h.lessonAge(); !ok {
		return ""
	}
	return h.LessonCard.Text
}

func (h *HUD) lessonSpan() float64 { return math.Inf(1) }

func (h *HUD) lessonAge() (float64, bool) {
	age := h.Now - h.LessonCard.start
	return age, h.LessonCard.Shown && age >= 0 && h.onStage(actLesson+h.LessonCard.Title)
}

func (h *HUD) LessonLines() []Line {
	if _, ok := h.lessonAge(); !ok {
		return nil
	}
	w := Objective.W - 16
	head := w - CloseFace - 2*ClosePad
	out := []Line{
		{Text: "WHAT JUST HAPPENED", Clr: ui.Palette.Gold},
		{Text: ui.Clip("DONE: "+h.LessonCard.Title, ui.MaxChars(head, 1)), Clr: ui.Palette.Green},
	}
	maxLines := LessonMaxLines
	if h.LessonCard.Teach || h.LessonCard.hold {
		kicker := "NEW ON YOUR SCREEN"
		if h.LessonCard.hold {
			kicker = "WHAT HAPPENED"
		}
		out, maxLines = []Line{
			{Text: kicker, Clr: ui.Palette.Gold},
			{Text: ui.Clip(h.LessonCard.Title, ui.MaxChars(head, 1)), Clr: ui.Palette.GoldBright},
		}, teachMaxLines
	}
	for _, row := range h.Font.WrapLines(h.LessonCard.Text, w, 1, maxLines) {
		out = append(out, Line{Text: row, Clr: ui.Palette.Cream})
	}
	return out
}

func (h *HUD) LessonRect(f Frame) ui.Rect {
	height := 8 + len(h.LessonLines())*ui.LineHeight + introRowH
	blocked := append(append(h.cardBlocked(f), inflated(h.crewControls(f), 2)...), inflated(FreightRects(f), 2)...)
	anchor, anchored := ui.Rect{}, h.ObjectiveShown(f)
	if anchored {
		anchor = h.ObjectiveRectOf(f)
		blocked = append(blocked, anchor.Inset(-2))
	}
	return f.placed(PanelLesson, FirstClear(clampAll(lessonSpots(Objective, height, anchor, anchored)), blocked))
}

func (h *HUD) LessonShown(f Frame) bool {
	return len(h.LessonLines()) > 0 && h.CardsDrawn(f)
}

func (h *HUD) CardsDrawn(f Frame) bool {
	return h.OverlayID == OverlayNone || h.OverlayID == OverlayMenu || (h.OverlayID == OverlayStone && !h.Stone.Tall)
}

func (h *HUD) LessonButtons(f Frame) []Button {
	if !h.LessonShown(f) {
		return nil
	}
	r := h.LessonRect(f)
	face := ui.Rect{X: r.X + r.W - introButtonW - introPad, Y: r.Y + r.H - 18 - 4, W: introButtonW, H: 16}
	out := []Button{closeButtonAs(r, lessonClose, "Close"), h.lessonAckButton(face)}
	if h.LessonCard.at != nil {
		show := ui.Rect{X: face.X - introButtonW - introPad, Y: face.Y, W: introButtonW, H: face.H}
		out = append(out, Button{ID: LessonShow, Rect: show, Label: "SHOW", Tip: "Glide to where it is"})
	}
	return out
}

func (h *HUD) pressLesson(rest string, f Frame) []Intent {
	if claim, ok := h.claimLesson(rest); ok {
		return claim
	}
	if rest == "show" && h.LessonCard.at != nil {
		return h.focus(*h.LessonCard.at)
	}
	folds := rest == "ack" && h.foldsNext()
	if rest == "ack" {
		h.noteAsk(actLesson+h.LessonCard.Title, f.State.RealmTime)
	}
	if rest == "close" || rest == "ack" {
		if h.lessonOfAPassedStep() {
			h.NoteAttention(EventNextStep, "", f.State.RealmTime)
		}
		h.LessonCard.Shown = false
	}
	if folds {
		h.NoteAttention(EventNextStep, "", f.State.RealmTime)
		return []Intent{{Kind: IntentSound, Command: "click"}, TutorialIntent(TeacherNext, h.Yard.View.Chapter.ID)}
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) lessonOfAPassedStep() bool {
	v := h.Yard.View
	return v != nil && v.Status == view.ChapterRunning && !v.Soaking && v.Step > v.LessonStep && h.LessonCard.Shown && h.LessonCard.Title == h.Yard.lesson
}

func (h *HUD) foldsNext() bool {
	v := h.Yard.View
	return v != nil && v.Soaking && h.LessonCard.Shown && h.LessonCard.Title == h.Yard.lesson
}

func (h *HUD) drawLesson(dst *ebiten.Image, f Frame) {
	lines := h.LessonLines()
	if len(lines) == 0 {
		return
	}
	age, _ := h.lessonAge()
	alpha := uint8(235 * max(0, min(1, (h.lessonSpan()-age)/lessonFade)))
	r := h.LessonRect(f)
	ui.Fill(dst, r.Offset(2, 3), color.RGBA{8, 11, 12, alpha * 3 / 4})
	ui.Fill(dst, r, ui.Tint(ui.Palette.Console, alpha))
	ui.Stroke(dst, r, ui.Tint(ui.Palette.Green, alpha))
	y := r.Y + 5
	for _, l := range lines {
		h.Font.Draw(dst, l.Text, r.X+8, y, 1, ui.Tint(l.Clr, alpha))
		y += ui.LineHeight
	}
	for _, b := range h.LessonButtons(f) {
		if b.ID == LessonAck || b.ID == LessonShow {
			h.drawButton(dst, b)
		}
	}
	h.DrawOverlayButtons(dst, h.LessonButtons(f)[:1])
}
