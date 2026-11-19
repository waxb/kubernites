package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	WoePrefix     = "woe"
	WoeMore       = WoePrefix + ":more"
	chronVigW     = 340
	chronVigS     = 2
	chronTop      = 50
	answerStep    = 24
	answerH       = 18
	answerMaxRows = 2
	shelfLabelW   = 64
	shelfGap      = 6
	ShelfPer      = 6
)

type WoeState struct {
	pick   string
	page   int
	last   string
	lastAt float64
}

const woeStale = 2.0

type ChronLayout struct {
	Question ui.Rect
	Picture  ui.Rect
	Timeline ui.Rect
	Scale    int
	Answers  int
	Shelf    ui.Rect
}

func ChronicleVig(c view.ChronicleView, f Frame) Vig {
	return Vig{Scene: art.AgedScene(gfx.ChronicleScene(c.Kind), gfx.AgeTier(f.State.Age)), Seed: gfx.VignetteSeed(c.ID)}
}

func (h *HUD) ChronicleLayout(c view.ChronicleView) ChronLayout {
	w := Overlay.W - 40
	q := ui.Rect{X: Overlay.X + 20, Y: Overlay.Y + chronTop, W: w, H: max(len(h.Font.Wrap(c.Question, w, 1)), 1) * ui.LineHeight}
	scale := chronVigS
	if c.Timeline != nil {
		scale = 1
	}
	pic := ui.Rect{X: Overlay.X + 20, Y: q.Y + q.H + vigGap, W: chronVigW * scale, H: art.VignetteH * scale}
	shelf := ui.Rect{X: Overlay.X + 20, Y: Overlay.Y + Overlay.H - 6 - art.ThumbH, W: w, H: art.ThumbH}
	l := ChronLayout{Question: q, Picture: pic, Scale: scale, Answers: pic.Y + pic.H + 6, Shelf: shelf}
	if rows := len(timelineRows(c)); rows > 0 {
		l.Timeline = timelineBeside(pic, rows)
		l.Answers = max(l.Answers, l.Timeline.Y+l.Timeline.H+6)
	}
	return l
}

func (h *HUD) answerRows(text string) []string {
	return h.Font.WrapLines(text, Overlay.W-50, 1, answerMaxRows)
}

func (h *HUD) AnswerRects(l ChronLayout, c view.ChronicleView) []ui.Rect {
	out := make([]ui.Rect, 0, len(c.Options))
	y := l.Answers
	for _, o := range c.Options {
		height := answerH + (max(len(h.answerRows(o.Text)), 1)-1)*ui.LineHeight
		out = append(out, ui.Rect{X: Overlay.X + 20, Y: y, W: Overlay.W - 40, H: height})
		y += height + answerStep - answerH
	}
	return out
}

func (h *HUD) pickedChronicle(f Frame) (view.ChronicleView, bool) {
	for _, c := range f.State.Chronicles {
		if c.ID == h.Woe.pick {
			return c, true
		}
	}
	return view.ChronicleView{}, false
}

func WoeShelf(list []view.ChronicleView) []view.ChronicleView {
	out := []view.ChronicleView{}
	for _, c := range list {
		if !c.Resolved {
			out = append(out, c)
		}
	}
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Resolved {
			out = append(out, list[i])
		}
	}
	return out
}

func shelfPages(n int) int {
	return max((n+ShelfPer-1)/ShelfPer, 1)
}

func ShelfThumbRect(shelf ui.Rect, i int) ui.Rect {
	return ui.Rect{X: shelf.X + shelfLabelW + i*(art.ThumbW+shelfGap), Y: shelf.Y, W: art.ThumbW, H: art.ThumbH}
}

func (h *HUD) ShelfButtons(f Frame, l ChronLayout) []Button {
	shelf := WoeShelf(f.State.Chronicles)
	page := h.Woe.page % shelfPages(len(shelf))
	out := []Button{}
	for i, c := range shelf[min(page*ShelfPer, len(shelf)):min((page+1)*ShelfPer, len(shelf))] {
		out = append(out, Button{ID: WoePrefix + ":" + c.ID, Rect: ShelfThumbRect(l.Shelf, i), Tip: GlossLine(c.Calamity, c.K8s)})
	}
	if len(shelf) > ShelfPer {
		out = append(out, Button{ID: WoeMore, Rect: ui.Rect{X: l.Shelf.X, Y: l.Shelf.Y + 26, W: shelfLabelW - 10, H: 14}, Label: "MORE", Tip: "The next page of Chronicles"})
	}
	return out
}

func WoeButton(b Button) bool {
	return strings.HasPrefix(b.ID, WoePrefix+":")
}

func PlainButtons(list []Button) []Button {
	out := []Button{}
	for _, b := range list {
		if !WoeButton(b) || b.Label != "" {
			out = append(out, b)
		}
	}
	return out
}

func (h *HUD) drawChronicleBody(dst *ebiten.Image, f Frame, c view.ChronicleView) {
	l := h.ChronicleLayout(c)
	if c.Plain {
		h.drawHistoryEntry(dst, f, c, l)
		return
	}
	h.drawTextLines(dst, []string{c.Question}, l.Question.X, l.Question.Y, l.Question.W, ui.Palette.Text)
	h.DrawVig(dst, ChronicleVig(c, f), l.Picture, l.Scale)
	h.drawTimeline(dst, c, l.Timeline)
	if c.Resolved {
		verdict, clr := "Wrongly written.", ui.Palette.RedText
		if c.Correct {
			verdict, clr = "Truly written.", ui.Palette.Green
		}
		h.Font.Draw(dst, verdict, l.Question.X, l.Answers, 1, clr)
		switch {
		case !ui.StoryNaming():
			h.drawTextLines(dst, []string{c.Explanation}, l.Question.X, ChronicleTruthY(l), l.Question.W, ui.Palette.Text)
		case !c.Correct && c.Truth != "":
			h.drawTextLines(dst, []string{"The truth: " + c.Truth}, l.Question.X, ChronicleTruthY(l), l.Question.W, ui.Palette.Text)
		}
	}
	h.drawShelf(dst, f, c, l)
}

func ChronicleTruthY(l ChronLayout) int { return l.Answers + chronAlmanacH }

func (h *HUD) drawShelf(dst *ebiten.Image, f Frame, shown view.ChronicleView, l ChronLayout) {
	shelf := WoeShelf(f.State.Chronicles)
	written := len(shelf)
	if written > 0 && !shelf[0].Resolved {
		written--
	}
	h.Font.DrawShadow(dst, "SURVIVED", l.Shelf.X, l.Shelf.Y+2, 1, ui.Palette.Gold)
	h.Font.Draw(dst, fmt.Sprintf("%d", written), l.Shelf.X, l.Shelf.Y+14, 1, ui.Palette.Text)
	for _, b := range h.ShelfButtons(f, l) {
		id := strings.TrimPrefix(b.ID, WoePrefix+":")
		c, ok := findChronicle(shelf, id)
		if !ok {
			continue
		}
		h.DrawVigThumb(dst, ChronicleVig(c, f), b.Rect, c.ID == shown.ID || (h.Hovering && h.Hover.ID == b.ID))
		if !c.Resolved {
			h.Font.DrawShadow(dst, "?", b.Rect.X+b.Rect.W-8, b.Rect.Y+2, 1, ui.Palette.GoldBright)
		}
	}
}

func findChronicle(list []view.ChronicleView, id string) (view.ChronicleView, bool) {
	for _, c := range list {
		if c.ID == id {
			return c, true
		}
	}
	return view.ChronicleView{}, false
}

func (h *HUD) PressWoe(rest string, _ Frame) []Intent {
	if rest == "more" {
		h.Woe.page++
	} else {
		h.Woe.pick, h.Woe.last = rest, ""
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func init() {
	prefixHandlers[WoePrefix] = (*HUD).PressWoe
}
