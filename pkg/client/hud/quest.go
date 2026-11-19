package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	questPrefix   = "quest"
	questAck      = questPrefix + ":ack"
	questCardW    = 440
	questPad      = 16
	questTitleH   = 22
	questButtonW  = 150
	questBodyMax  = 5
	questPartMax  = 3
	questGainMax  = 6
	questLayer    = "quest"
	recapPrefix   = "recap:"
	decreeSealR   = 7
	NextStepTitle = "NEXT STEP"
	lastStepTitle = "STEP DONE"
	markDone      = "[x] "
	MarkNow       = "[>] "
	MarkLater     = "[ ] "
	partIndent    = "    "
)

type questCard struct {
	key    string
	title  string
	giver  string
	Body   string               `json:"-"`
	Parts  []view.QuestPartView `json:"-"`
	button string
	Recap  bool `json:"-"`
}

func (f Frame) quest(id string) (view.QuestView, bool) {
	for _, q := range f.State.Quests {
		if q.ID == id {
			return q, true
		}
	}
	return view.QuestView{}, false
}

func (f Frame) questStep() (view.ObjectiveView, view.QuestView, bool) {
	o, ok := f.ShownObjective()
	if !ok || o.Quest == "" {
		return o, view.QuestView{}, false
	}
	q, ok := f.quest(o.Quest)
	return o, q, ok
}

func (f Frame) decreeIntro() bool {
	return f.Introducing() && f.Guide.Intro.ID == view.BorgIntro
}

func (h *HUD) QuestCard(f Frame) (questCard, bool) {
	if f.Unguided || !h.CardsDrawn(f) {
		return questCard{}, false
	}
	for _, q := range f.State.Quests {
		switch {
		case !q.Fulfilled && f.decreeIntro():
			return questCard{key: q.ID, title: q.Title, giver: q.Giver, Body: q.Intro, Parts: q.Parts, button: "I WILL RAISE THEM"}, true
		case q.Fulfilled && !q.Closed && !h.questSeen[recapPrefix+q.ID] && h.onStage(actQuest+recapPrefix+q.ID):
			return questCard{key: recapPrefix + q.ID, title: q.Title + ": fulfilled", giver: q.Giver, Body: q.Recap, Parts: q.Parts, button: "ONWARD", Recap: true}, true
		}
	}
	return questCard{}, false
}

func (h *HUD) recapWanted(f Frame) (string, bool) {
	if f.Unguided || h.OverlayID == OverlayWelcome {
		return "", false
	}
	for _, q := range f.State.Quests {
		if q.Fulfilled && !q.Closed && !h.questSeen[recapPrefix+q.ID] {
			return recapPrefix + q.ID, true
		}
	}
	return "", false
}

func (h *HUD) QuestCardShown(f Frame) bool {
	_, ok := h.QuestCard(f)
	return ok
}

func (h *HUD) questBody(c questCard) ([]string, [][]string) {
	w := questCardW - 2*questPad
	body := h.Font.WrapLines(c.Body, w, 1, questBodyMax)
	parts := make([][]string, 0, len(c.Parts))
	for _, p := range c.Parts {
		parts = append(parts, h.Font.WrapLines(p.Title+": "+p.Purpose, w-len(partIndent)*ui.Advance, 1, questPartMax))
	}
	return body, parts
}

func (h *HUD) questRect(c questCard) ui.Rect {
	body, parts := h.questBody(c)
	rows := 1 + len(body)
	for _, p := range parts {
		rows += len(p)
	}
	height := questPad + questTitleH + rows*ui.LineHeight + 2*questPad + 18 + 6
	return clampCard(ui.Rect{X: Width/2 - questCardW/2, Y: (TopH+ConsoleY)/2 - height/2, W: questCardW, H: height})
}

func (h *HUD) questButtons(f Frame) []Button {
	c, ok := h.QuestCard(f)
	if !ok {
		return nil
	}
	r := h.questRect(c)
	return []Button{{ID: questAck, Rect: ui.Rect{X: r.X + r.W/2 - questButtonW/2, Y: r.Y + r.H - questPad - 14, W: questButtonW, H: 18}, Label: c.button}}
}

func RecapKey(id string) string { return recapPrefix + id }

func (h *HUD) SeenQuest(key string) {
	if h.questSeen == nil {
		h.questSeen = map[string]bool{}
	}
	h.questSeen[key] = true
}

func (h *HUD) PressQuest(rest string, f Frame) []Intent {
	c, ok := h.QuestCard(f)
	if rest != "ack" || !ok {
		return nil
	}
	click := Intent{Kind: IntentSound, Command: "click"}
	if !c.Recap {
		return []Intent{act(Ack(view.BorgIntro)), click}
	}
	h.SeenQuest(c.key)
	return []Intent{act(view.Action{Action: view.ActCloseQuest, ID: strings.TrimPrefix(c.key, recapPrefix)}), click}
}

func (h *HUD) drawQuestCard(dst *ebiten.Image, f Frame) {
	c, ok := h.QuestCard(f)
	if !ok {
		return
	}
	r := h.questRect(c)
	dimAround(dst, r, true)
	drawScroll(dst, r)
	h.drawQuestBody(dst, f, c, r)
}

func (h *HUD) drawQuestBody(dst *ebiten.Image, f Frame, c questCard, r ui.Rect) {
	drawSeal(dst, r.X+questPad+decreeSealR, r.Y+questPad+decreeSealR)
	x, y := r.X+questPad, r.Y+questPad
	h.Font.Draw(dst, strings.ToUpper(c.title), x+2*decreeSealR+8, y, 2, ui.Palette.RedDark)
	y += questTitleH
	h.Font.Draw(dst, c.giver, x, y, 1, ui.Palette.Ink)
	y += ui.LineHeight + questPad/2
	body, parts := h.questBody(c)
	for _, row := range body {
		h.Font.Draw(dst, row, x, y, 1, ui.Palette.Ink)
		y += ui.LineHeight
	}
	y += questPad / 2
	for i, rows := range parts {
		mark, clr := MarkLater, ui.Palette.Ink
		if c.Recap || c.Parts[i].Done {
			mark, clr = markDone, ui.Palette.GreenDark
		}
		for j, row := range rows {
			lead := partIndent
			if j == 0 {
				lead = mark
			}
			h.Font.Draw(dst, lead+row, x, y, 1, clr)
			y += ui.LineHeight
		}
	}
	for _, b := range h.questButtons(f) {
		h.drawPrimary(dst, b)
	}
}

func drawScroll(dst *ebiten.Image, r ui.Rect) {
	ui.Fill(dst, r.Offset(2, 3), ui.Palette.Shadow)
	ui.Fill(dst, r, ui.Palette.Parchment)
	ui.Stroke(dst, r, ui.Palette.ParchmentDark)
	ui.Stroke(dst, r.Inset(2), ui.Palette.Gold)
	ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 3}, ui.Palette.ParchmentDark)
	ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y + r.H - 3, W: r.W, H: 3}, ui.Palette.ParchmentDark)
}

func drawSeal(dst *ebiten.Image, cx, cy int) {
	vector.DrawFilledCircle(dst, float32(cx), float32(cy), decreeSealR, ui.Palette.RedDark, false)
	vector.StrokeCircle(dst, float32(cx), float32(cy), decreeSealR-2, 1, ui.Palette.Red, false)
	vector.DrawFilledCircle(dst, float32(cx), float32(cy), 2, ui.Palette.Gold, false)
}

func (h *HUD) trackerLines(f Frame, o view.ObjectiveView, q view.QuestView) []Line {
	w := Objective.W - 16
	out := []Line{{Text: q.Giver, Clr: ui.Palette.Ink}}
	now := partNow(q, o)
	for i, p := range q.Parts {
		switch {
		case p.Done:
			out = append(out, Line{Text: markDone + p.Title, Clr: ui.Palette.GreenDark})
			if i == now-1 && p.Gained != "" {
				out = append(out, wrapped(h, partIndent, p.Gained, w, questGainMax, ui.Palette.GreenDark)...)
			}
		case i == now:
			out = append(out, Line{Text: MarkNow + p.Title + PartCount(p), Clr: ui.Palette.RedDark})
			out = append(out, wrapped(h, partIndent, p.Purpose, w, questPartMax, ui.Palette.Ink)...)
			out = append(out, h.trackerStep(f, o, w, partIndent, p.Purpose == "")...)
		default:
			out = append(out, Line{Text: MarkLater + p.Title, Clr: ui.Palette.Ink})
		}
	}
	return out
}

func PartCount(p view.QuestPartView) string {
	if p.Steps < 2 || p.StepsDone == 0 {
		return ""
	}
	return fmt.Sprintf("  %d/%d", p.StepsDone, p.Steps)
}

func partNow(q view.QuestView, o view.ObjectiveView) int {
	for i, p := range q.Parts {
		if p.Objective == o.ID && !p.Done {
			return i
		}
	}
	return q.Current
}

func (h *HUD) trackerStep(f Frame, o view.ObjectiveView, w int, indent string, withWhy bool) []Line {
	if o.Done {
		return nil
	}
	out := wrapped(h, indent, f.QuoteObjective(o.Text), w, objectiveTextLines, ui.Palette.Ink)
	if withWhy {
		out = append(out, wrapped(h, indent, f.Guide.Why, w, objectiveWhyLines, ui.Palette.GreenDark)...)
	}
	for _, l := range wrapped(h, indent, f.QuoteObjective(o.Progress), w, objectiveGoalLines, ui.Palette.RedDark) {
		l.status = true
		out = append(out, l)
	}
	return out
}

func wrapped(h *HUD, indent, text string, w, max int, clr color.RGBA) []Line {
	out := []Line{}
	if text == "" {
		return out
	}
	rest := strings.Repeat(" ", len(indent))
	for i, row := range h.Font.WrapLines(text, w-len(indent)*ui.Advance, 1, max) {
		lead := rest
		if i == 0 {
			lead = indent
		}
		out = append(out, Line{Text: lead + row, Clr: clr})
	}
	return out
}
