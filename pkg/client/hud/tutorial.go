package hud

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayDebrief = "debrief"
	TeacherPrefix  = "teacher"
	DebriefPrefix  = "debrief"
	TeacherRegion  = "teacher"

	TeacherShow        = "show"
	TeacherChip        = "chip"
	TeacherAlmanac     = "almanac"
	TeacherNext        = "next"
	TeacherBegin       = "begin"
	TeacherLeave       = "leave"
	TeacherDebrief     = "debrief"
	TeacherNextChapter = "nextchapter"
	TeacherMap         = "map"
	TeacherWho         = "who"
	PlotSuggested      = "plot:suggested"
)

type TeacherView struct {
	Chapter     view.TutorialChapter
	Status      view.ChapterStatus
	Step        int
	Progress    view.StepProgress
	Soaking     bool
	Chip        string
	Lesson      view.StepLesson
	Lessons     int
	LessonStep  int
	LessonTitle string
	Next        view.TutorialChapter
	HasNext     bool
	Hop         view.Hop
	Armed       bool
	LastHop     bool
	Verdict     string
	Asking      bool
	Question    view.HopQuestion
	Met         bool
	Closing     bool
}

func (v TeacherView) walking() bool { return v.running() && v.Hop.Line != "" }

func (v TeacherView) Current() (view.TutorialStep, bool) {
	if v.Step < 0 || v.Step >= len(v.Chapter.Steps) {
		return view.TutorialStep{}, false
	}
	return v.Chapter.Steps[v.Step], true
}

type teachState struct {
	View      *TeacherView `json:"-"`
	lessons   int
	lesson    string
	tmap      TutorialMap
	PageID    int `json:"-"`
	answers   map[int]int
	face      string
	faceFrom  float64
	faceWait  bool
	Line      string `json:"-"`
	lineAt    float64
	pressAt   float64
	place     ui.Rect
	doneAt    float64
	faceDone  bool
	soaked    bool
	debriefed bool
	rows      int
}

func (h *HUD) DebriefShown() bool { return h.Yard.debriefed }

func (h *HUD) closingView(v TeacherView) TeacherView {
	if v.Status != view.ChapterComplete || h.Yard.debriefed || len(v.Chapter.Steps) == 0 {
		return v
	}
	last := len(v.Chapter.Steps) - 1
	v.Status, v.Soaking, v.Step, v.LessonStep, v.Closing = view.ChapterRunning, true, last, last, true
	return v
}

func (h *HUD) SetTeacher(v *TeacherView) {
	if v == nil {
		h.Yard.View, h.Yard.lessons = nil, 0
		return
	}
	if prev := h.Yard.View; prev == nil || prev.Chapter.ID != v.Chapter.ID {
		h.Yard.lessons = v.Lessons
		h.Yard.PageID, h.Yard.answers, h.Yard.debriefed = 0, nil, false
		h.Stone.Greet(ChapterGreetingK8s(v.Chapter.Number))
	}
	next := h.closingView(*v)
	prev := h.Yard.View
	if h.praiseWaitsForItsCount(prev, next) {
		return
	}
	h.Yard.View = &next
	h.noteTeacherMet(next)
	if v.Lessons > h.Yard.lessons {
		h.Yard.lessons, h.Yard.lesson = v.Lessons, v.LessonTitle
	}
	h.noteTeacherFace(next)
	if prev != nil && prev.Chapter.ID == v.Chapter.ID && prev.Step != v.Step && h.OverlayID == OverlayStone {
		h.keepStoneFor(next)
	}
}

func (h *HUD) praiseWaitsForItsCount(prev *TeacherView, v TeacherView) bool {
	if prev == nil || prev.Chapter.ID != v.Chapter.ID || prev.Step != v.Step || prev.Status != v.Status || prev.Soaking || !v.Soaking {
		return false
	}
	return v.Progress.Line != h.Yard.Line && h.Now-h.Yard.lineAt < CounterTick && !h.teacherPressed()
}

func (h *HUD) teacherPressed() bool {
	return h.Yard.pressAt > 0 && h.Now-h.Yard.pressAt <= attnAnswerWindow
}

func (h *HUD) noteTeacherFace(v TeacherView) {
	face := fmt.Sprintf("%s:%d:%d", v.Chapter.ID, v.Step, v.Status)
	if face == h.Yard.face {
		if v.Soaking && !h.Yard.soaked {
			h.teacherAnswered()
			h.Yard.Line, h.Yard.lineAt = v.Progress.Line, h.Now
		}
		h.Yard.soaked = v.Soaking
		if v.Progress.Line != h.Yard.Line && (h.Now-h.Yard.lineAt >= CounterTick || h.teacherPressed()) {
			h.Yard.Line, h.Yard.lineAt = v.Progress.Line, h.Now
		}
		h.Yard.rows = max(h.Yard.rows, h.ProgressRows())
		return
	}
	h.teacherAnswered()
	h.Yard.Line, h.Yard.lineAt, h.Yard.soaked = v.Progress.Line, h.Now, v.Soaking
	h.Yard.rows = h.faceRows(v)
	h.Yard.face, h.Yard.faceFrom, h.Yard.faceWait, h.Yard.faceDone = face, h.Now, false, false
	h.takeTeacherSlot()
}

const teacherBuildRows = 2

func (h *HUD) faceRows(v TeacherView) int {
	if v.running() && len(v.step().Builds) > 0 {
		return max(teacherBuildRows, h.ProgressRows())
	}
	return h.ProgressRows()
}

func (h *HUD) ProgressRows() int {
	return len(Literal.WrapLines(h.Yard.Line, TeacherW-2*TeacherPad, 1, teacherProgressMax))
}

func (h *HUD) takeTeacherSlot() {
	s := &h.Director.Stage
	if !s.Primed || !h.slotFree() {
		return
	}
	key := actTeacher + h.Yard.face
	s.Slot = SlotBoxOf{Key: key, text: h.Yard.face, From: h.Now, Read: ReadTime(h.TeacherText())}
	s.pulledKeys = withKey(s.pulledKeys, key)
}

func (h *HUD) teacherAnswered() {
	if h.Yard.face == "" || h.Yard.faceWait || h.Yard.faceDone {
		return
	}
	h.Yard.doneAt, h.Yard.faceDone = h.Now, true
	if s := &h.Director.Stage; s.Primed && h.Now >= h.Yard.faceFrom {
		s.lastEnd = max(s.lastEnd, h.Now)
	}
	h.NoteAttention(EventStepDone, actTeacher+h.Yard.face, h.attn.realm)
}

func (h *HUD) teacherNextAnswers(v TeacherView) string {
	if s, ok := v.Current(); v.Soaking || ok && s.Talk {
		return actTeacher + h.Yard.face
	}
	return ""
}

func (h *HUD) TeacherText() string {
	l := h.TeacherLayout()
	happened := []string{}
	if len(l.Happened) > 0 {
		happened = append([]string{TeacherHappened}, l.Happened...)
	}
	return strings.Join(slices.Concat([]string{l.Header}, l.Body, happened, l.Course, l.Why, l.Try, []string{l.Ask}), " ")
}

func (h *HUD) keepStoneFor(v TeacherView) {
	s, ok := v.Current()
	if !ok || v.Status != view.ChapterRunning || !s.Speaks() {
		h.OverlayID = OverlayNone
		return
	}
	h.FocusStone()
	if v.Chip != "" {
		h.OpenStone(v.Chip)
	}
}

func (h *HUD) closeTeacherLesson() {
	if h.LessonCard.Shown && h.LessonCard.Title == h.Yard.lesson && h.onStage(actLesson+h.LessonCard.Title) {
		h.LessonCard.Shown = false
	}
}

func (h *HUD) InChapter() bool { return h.Yard.View != nil }

func (h *HUD) SetTutorialRecord(rec view.ChapterRecord) { h.Yard.tmap.Record = rec }

func (h *HUD) teacherShown() bool { return h.Yard.View != nil }

func (h *HUD) Teacher() *TeacherView { return h.Yard.View }

func (h *HUD) OpenChapters(rec view.ChapterRecord) {
	h.Yard.tmap.Open(rec)
	h.Yard.tmap.Almanac = true
	h.OverlayID = OverlayChapters
}

func (h *HUD) ChapterMap() *TutorialMap { return &h.Yard.tmap }

func (h *HUD) OpenDebrief() {
	h.OverlayID, h.Yard.PageID = OverlayDebrief, 0
	if v := h.Yard.View; v != nil && (v.Closing || v.Status == view.ChapterComplete) {
		h.Yard.debriefed = true
	}
}

var bracketedWord = regexp.MustCompile(`\s*\([^()]*\)`)

func KubernetesOnly(text string) string {
	return strings.TrimSpace(bracketedWord.ReplaceAllString(text, ""))
}

func init() {
	Overlays[OverlayChapters] = OverlaySpec{
		func(h *HUD, _ Frame) []Button { return h.Yard.tmap.Buttons() },
		func(h *HUD, d *ebiten.Image, _ Frame) {
			h.Yard.tmap.Reduce = h.Reduce
			h.Yard.tmap.now = h.Now
			h.Yard.tmap.Draw(d, h.Lib, h.hoverID())
		},
	}
	Overlays[OverlayDebrief] = OverlaySpec{(*HUD).DebriefButtons, (*HUD).drawDebrief}
	prefixHandlers[tutorialPrefix] = (*HUD).pressChapters
	prefixHandlers[TeacherPrefix] = (*HUD).PressTeacher
	prefixHandlers[DebriefPrefix] = (*HUD).pressDebrief
}

func (h *HUD) hoverID() string {
	if !h.Hovering {
		return ""
	}
	return h.Hover.ID
}

func (h *HUD) pressChapters(rest string, f Frame) []Intent {
	return h.ChapterIntents(h.Yard.tmap.Press(tutorialPrefix+":"+rest), f)
}

func (h *HUD) ChapterIntents(in []Intent, f Frame) []Intent {
	out := []Intent{}
	for _, it := range in {
		switch {
		case it.Kind != IntentTutorial:
			out = append(out, it)
		case it.Command == TutorialBack:
			h.OverlayID = OverlayNone
		case it.Command == TutorialAlmanac:
			out = append(out, h.openChapterPage(it.Target, f)...)
		default:
			if it.Command == TutorialStart {
				h.OverlayID = OverlayNone
			}
			out = append(out, it)
		}
	}
	return out
}

func (h *HUD) openChapterPage(page string, f Frame) []Intent {
	h.Book = h.Book.Searched("")
	return h.OpenBook(page, f)
}

func (h *HUD) chapterKeys(in Input, f Frame) ([]Intent, bool) {
	return h.ChapterIntents(h.Yard.tmap.Keys(in), f), len(in.Keys) > 0
}

func (h *HUD) PressTeacher(rest string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	v := h.Yard.View
	if v == nil {
		return []Intent{click}
	}
	switch rest {
	case TeacherChip:
		if v.Chip != "" {
			h.OpenStone(v.Chip)
		}
		return []Intent{click}
	case TeacherAlmanac:
		if s, ok := v.Current(); ok && s.Page != "" && v.Status == view.ChapterRunning {
			return h.openChapterPage(s.Page, f)
		}
		if page := v.Chapter.OwnPage(); page != "" {
			return h.openChapterPage(page, f)
		}
	case TeacherDebrief:
		if v.Closing {
			h.Yard.pressAt = h.Now
			h.noteAsk(actTeacher+h.Yard.face, f.State.RealmTime)
		}
		h.OpenDebrief()
		return []Intent{click}
	case TeacherNextChapter:
		h.OverlayID = OverlayNone
		return []Intent{click, TutorialIntent(TutorialStart, v.Next.ID)}
	case TeacherNext:
		h.Yard.pressAt = h.Now
		h.NoteAttention(EventNextStep, "", f.State.RealmTime)
		h.noteAsk(h.teacherNextAnswers(*v), f.State.RealmTime)
		if h.OverlayID != OverlayStone {
			h.OverlayID = OverlayNone
		}
		h.closeTeacherLesson()
		return []Intent{click, TutorialIntent(rest, v.Chapter.ID)}
	case TeacherShow, TeacherBegin, TeacherLeave, TeacherMap:
		if rest == TeacherBegin {
			h.noteAsk(actTeacher+h.Yard.face, f.State.RealmTime)
		}
		if rest != TeacherShow {
			h.OverlayID = OverlayNone
		}
		return []Intent{click, TutorialIntent(rest, v.Chapter.ID)}
	}
	if n, ok := strings.CutPrefix(rest, TeacherWho+":"); ok && v.Asking && v.Question.Picked < 0 {
		h.Yard.pressAt = h.Now
		h.NoteAttention(EventAction, actTeacher+TeacherWho, f.State.RealmTime)
		return []Intent{click, TutorialIntent(TeacherWho, n)}
	}
	return []Intent{click}
}

func (h *HUD) pressDebrief(rest string, f Frame) []Intent {
	switch rest {
	case "close":
		h.OverlayID = OverlayNone
		return []Intent{{Kind: IntentSound, Command: "click"}}
	case TeacherNextChapter, TeacherMap:
		return h.PressTeacher(rest, f)
	case DebriefSee:
		h.Yard.PageID = 1
	case DebriefBack:
		h.Yard.PageID = 0
	default:
		check, ok := strings.CutPrefix(rest, DebriefCheck+":")
		if !ok {
			return nil
		}
		h.pressCheck(check)
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

const (
	chapterTopSpeed = 5
	learnModeLabel  = "LEARN KUBERNETES"
	learnModeTip    = "Chapters with a teacher, real terms first: Pod (hut). Your realm is saved first and waits as you left it."
)

var RealmWords = map[string]string{
	"cmd:door":            "the hut at the door",
	"build:keep":          "a Keep; the first is the Lone Tower",
	"build:codex":         "the Codex",
	"build:chancery":      "the Chancery",
	"build:stewards-hall": "the Stewards' Hall",
	"build:quartermaster": "the Quartermaster",
}

func chapterPrice(cost string, f Frame) string {
	if f.State.Tutorial {
		return ""
	}
	return cost
}

func ChapterTips(buttons []Button, f Frame) []Button {
	if !f.State.Tutorial {
		return buttons
	}
	out := make([]Button, len(buttons))
	for i, b := range buttons {
		out[i] = b
		word, ok := RealmWords[b.ID]
		if !ok || strings.HasPrefix(b.Tip, tipMark) {
			continue
		}
		tip := b.Tip
		if own, ok := chapterTipK8s(b.ID); ok {
			tip = own
		}
		out[i].Tip = ui.Settle(ui.Name(chapterTipText(tip)) + staticPodNoteK8s(b.ID) + "\nIn the realm: " + word + ".")
	}
	return out
}

func chapterTipK8s(id string) (string, bool) {
	if id != DoorButton.ID {
		return "", false
	}
	return "A static Pod is one the kubelet runs from a file on its own Node: here /etc/kubernetes/manifests/porter.yaml, on the host network (hostNetwork: true, port 80). It needs no kube-apiserver, and it uses some of the Node's CPU and memory.\n" + ActsNote, true
}

func chapterTipText(tip string) string {
	lines := []string{}
	for _, line := range strings.Split(tip, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Costs ") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
