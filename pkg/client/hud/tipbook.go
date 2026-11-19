package hud

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	tipPrefix  = "tip"
	TipWhy     = "tip:why"
	TipAlmanac = "tip:almanac"
	tipActH    = 14
	tipActW    = 64
	WhyWidth   = 210
	WhyGap     = 4
)

type tipPin struct {
	button  Button
	page    view.Page
	lines   []string
	tones   []color.RGBA
	Rect    ui.Rect `json:"-"`
	Why     bool    `json:"-"`
	WhyRect ui.Rect `json:"-"`
	area    ui.Rect
	overlay string
}

func (p tipPin) On() bool { return p.button.ID != "" }

var subjectKinds = []string{view.SubjectStructure, view.SubjectResource, view.SubjectConcept}

func pageCandidates(id string) []string {
	fields := strings.FieldsFunc(id, func(r rune) bool { return r == ':' || r == '/' })
	out := make([]string, 0, len(fields))
	for i := len(fields) - 1; i >= 0; i-- {
		out = append(out, strings.ToLower(fields[i]))
	}
	return out
}

func ButtonPage(b Button, f Frame) (view.Page, bool) {
	ids := []string{b.Page}
	if b.Page == "" {
		ids = pageCandidates(b.ID)
	}
	if foremenOnly(f) {
		ids = shiftPages(ids)
	}
	return firstMetPage(f, subjectKinds, ids)
}

func shiftPages(ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id
		if id == "crew" {
			out[i] = "foreman"
		}
	}
	return out
}

func firstMetPage(f Frame, kinds, ids []string) (view.Page, bool) {
	for _, id := range ids {
		for _, kind := range kinds {
			if p, ok := view.Content.ForSubject(kind, id); ok && bookMet(f, p.ID) {
				return p, true
			}
		}
	}
	return view.Page{}, false
}

func TipButtonID(id string) bool { return id == TipWhy || id == TipAlmanac }

func (h *HUD) stepPin(in Input, f Frame, buttons []Button) {
	if h.Pin.On() && (!h.liveButton(h.Pin.button, buttons) || h.Pin.overlay != h.OverlayID || h.pinCovers(f, buttons)) {
		h.Pin = tipPin{}
	}
	if in.Click && h.Pin.On() && !(h.Hovering && TipButtonID(h.Hover.ID)) {
		h.Pin = tipPin{}
		return
	}
	switch {
	case h.Hovering && TipButtonID(h.Hover.ID) && h.Pin.On():
		return
	case h.Hovering && h.Hover.ID == h.Pin.button.ID && h.Pin.On():
		return
	case h.Hovering && h.Hover.Tip != "" && !(h.Pin.On() && h.Pin.area.Contains(in.MX, in.MY)):
		if p, ok := ButtonPage(h.Hover, f); ok {
			h.Pin = h.PinnedTip(h.Hover, p, f, buttons)
			return
		}
	}
	if !h.Pin.On() || !h.Pin.area.Contains(in.MX, in.MY) {
		h.Pin = tipPin{}
	}
}

func (h *HUD) PinText(b Button, p view.Page) ([]string, []color.RGBA) {
	if strings.HasPrefix(b.ID, "build:") {
		first, _, _ := strings.Cut(b.Tip, "\n")
		lore := len(h.Font.Wrap(first, TipWidth, 1))
		lines := h.Font.Wrap(b.Tip, TipWidth, 1)
		return lines, tones(len(lines), pinSpan{lore, ui.Palette.TextDim})
	}
	purpose := h.Font.Wrap(view.Plain(p.Purpose), TipWidth, 1)
	how := h.Font.Wrap(view.Plain(p.How), TipWidth, 1)
	lines := append(append([]string(nil), purpose...), how...)
	spans := []pinSpan{{len(purpose), ui.Palette.GoldBright}, {len(how), ui.Palette.Cream}}
	if b.Tip != "" {
		lines = append(append(lines, ""), h.Font.Wrap(b.Tip, TipWidth, 1)...)
		spans = append(spans, pinSpan{1 + len(h.Font.Wrap(b.Tip, TipWidth, 1)), ui.Palette.TextDim})
	}
	return lines, tones(len(lines), spans...)
}

type pinSpan struct {
	n   int
	clr color.RGBA
}

func tones(total int, spans ...pinSpan) []color.RGBA {
	out := make([]color.RGBA, 0, total)
	for _, s := range spans {
		for range s.n {
			out = append(out, s.clr)
		}
	}
	for len(out) < total {
		out = append(out, ui.Palette.Cream)
	}
	return out[:total]
}

func (h *HUD) PinnedTip(b Button, p view.Page, f Frame, buttons []Button) tipPin {
	lines, tones := h.PinText(b, p)
	saved := h.Hover
	h.Hover = b
	r := h.TooltipRect(append(append([]string(nil), lines...), "", ""), f)
	r.W = max(r.W, 2*tipActW+16)
	r.X = min(max(r.X, 2), Width-r.W-2)
	blockers := h.pinBlockers(buttons, b, f)
	r = FirstClear(h.pinSpots(r), blockers)
	h.Hover = saved
	if OverlapsAny(r, blockers) {
		return tipPin{}
	}
	pin := tipPin{button: b, page: p, lines: lines, tones: tones, Rect: r, overlay: h.OverlayID}
	return pin.withArea()
}

func (h *HUD) pinCovers(f Frame, buttons []Button) bool {
	saved := h.Hover
	h.Hover = h.Pin.button
	defer func() { h.Hover = saved }()
	return OverlapsAny(h.Pin.Rect, h.pinBlockers(buttons, h.Pin.button, f))
}

func (p tipPin) withArea() tipPin {
	next := p
	area := union(p.Rect, p.button.Rect)
	if p.Why {
		area = union(area, p.WhyRect)
	}
	next.area = area.Inset(-2)
	return next
}

func union(a, b ui.Rect) ui.Rect {
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func (h *HUD) PinnedLines() []string { return h.Pin.lines }

func PinWhy(font ui.Font, p view.Page) []string {
	return font.Wrap(view.Plain(p.Why), WhyWidth-10, 1)
}

func (h *HUD) tipButtons() []Button {
	if !h.Pin.On() {
		return nil
	}
	r := h.Pin.Rect
	y := r.Y + r.H - tipActH - 3
	return []Button{
		{ID: TipWhy, Rect: ui.Rect{X: r.X + 4, Y: y, W: tipActW, H: tipActH}, Label: "WHY?", Tip: "Why it matters: more on the left"},
		{ID: TipAlmanac, Rect: ui.Rect{X: r.X + r.W - tipActW - 4, Y: y, W: tipActW, H: tipActH}, Label: "ALMANAC", Tip: "Open this page of the Almanac"},
	}
}

func (h *HUD) pressTip(rest string, f Frame) []Intent {
	switch rest {
	case "why":
		h.Pin = h.Pin.toggledWhy(h.Font)
		return []Intent{{Kind: IntentSound, Command: "click"}}
	case "almanac":
		page := h.Pin.page.ID
		h.Pin = tipPin{}
		return h.OpenBook(page, f)
	}
	return nil
}

func (p tipPin) toggledWhy(font ui.Font) tipPin {
	next := p
	next.Why = !p.Why
	lines := PinWhy(font, p.page)
	h := len(lines)*ui.LineHeight + 8
	x := p.Rect.X - WhyWidth - WhyGap
	if x < 2 {
		x = p.Rect.X + p.Rect.W + WhyGap
	}
	next.WhyRect = ui.Rect{X: x, Y: min(p.Rect.Y, Height-h-2), W: WhyWidth, H: h}
	return next.withArea()
}

func (h *HUD) drawPinnedTip(dst *ebiten.Image) {
	r := h.Pin.Rect
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 240))
	ui.Stroke(dst, r, ui.Palette.Gold)
	for i, l := range h.Pin.lines {
		h.Font.Draw(dst, l, r.X+5, r.Y+4+i*ui.LineHeight, 1, h.Pin.tones[i])
	}
	for _, b := range h.tipButtons() {
		h.drawButton(dst, b)
	}
	if h.Pin.Why {
		w := h.Pin.WhyRect
		ui.Fill(dst, w, ui.Tint(ui.Palette.Console, 245))
		ui.Stroke(dst, w, ui.Palette.Purple)
		for i, l := range PinWhy(h.Font, h.Pin.page) {
			h.Font.Draw(dst, l, w.X+5, w.Y+4+i*ui.LineHeight, 1, ui.Palette.Cream)
		}
	}
}
