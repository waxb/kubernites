package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	chiefPortrait = "advisor-chief"
	chiefFallback = "chancellor"
	counselLines  = 3
	faceSize      = 40
	QuietRealm    = "The realm is quiet. Your advisors await your questions."
)

func ChiefFace() ui.Rect { return ui.Rect{X: Overlay.X + 10, Y: Overlay.Y + 26, W: 50, H: 52} }

func CounselBox() ui.Rect {
	return ui.Rect{X: Overlay.X + 68, Y: Overlay.Y + 40, W: DialsBox().X - Overlay.X - 80, H: counselLines * ui.LineHeight}
}

func Counsel(f Frame) string {
	switch {
	case f.Caesar.Counsel != "":
		return f.Caesar.Counsel
	case f.State.Advice != "":
		return f.State.Advice
	}
	return QuietRealm
}

func (h *HUD) face(dst *ebiten.Image, name, fallback string, box ui.Rect) {
	if h.Lib.Portrait(dst, name, box) {
		return
	}
	h.portrait(dst, fallback, box)
}

func (h *HUD) drawCouncil(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, Overlay)
	h.header(dst, Overlay, "THE COUNCIL")
	ui.Well(dst, ChiefFace())
	h.face(dst, chiefPortrait, chiefFallback, ChiefFace().Inset(2))
	box := CounselBox()
	h.Font.Draw(dst, "CHIEF ADVISOR", box.X, box.Y-12, 1, ui.Palette.Gold)
	lines := counselLines
	if len(counselButtons(f)) > 0 {
		lines--
	}
	for i, row := range h.Font.WrapLines(Counsel(f), box.W, 1, lines) {
		h.Font.Draw(dst, row, box.X, box.Y+2+i*ui.LineHeight, 1, ui.Palette.Cream)
	}
	h.drawDials(dst, f)
	active := h.CouncilTab()
	for i, a := range Advisors {
		h.drawCouncilTab(dst, CouncilTabID(i), a, active == a.Id)
	}
	body := CouncilBody()
	ui.Fill(dst, body, consoleInk)
	ui.Stroke(dst, body, ui.Palette.BlueDark)
	h.drawAdvisor(dst, f, active)
	h.DrawOverlayButtons(dst, h.councilPlainButtons(f))
	h.drawCouncilChip(dst, f)
}

func (h *HUD) drawCouncilTab(dst *ebiten.Image, r ui.Rect, a advisor, active bool) {
	state := ui.ButtonState{Hovered: h.Hovering && h.Hover.ID == CouncilPrefix+":tab:"+a.Id, Selected: active}
	text := ui.ButtonFace(dst, r, state)
	h.face(dst, a.portrait, a.fallback, ui.Rect{X: r.X + 2, Y: r.Y + 2, W: 18, H: r.H - 4})
	h.Font.Draw(dst, ui.Clip(a.title(), ui.MaxChars(r.W-26, 1)), r.X+23, r.Y+r.H/2-4, 1, text)
}

func (h *HUD) drawAdvisor(dst *ebiten.Image, f Frame, tab string) {
	body := CouncilBody()
	for _, a := range Advisors {
		if a.Id == tab {
			h.face(dst, a.portrait, a.fallback, ui.Rect{X: body.X + 4, Y: body.Y + 4, W: faceSize, H: faceSize})
		}
	}
	page := CouncilPageFor(tab, f)
	for i, row := range h.Font.WrapLines(page.Line, body.W-faceSize-16, 1, 3) {
		h.Font.Draw(dst, row, body.X+faceSize+12, body.Y+8+i*ui.LineHeight, 1, ui.Palette.Text)
	}
	if tab == TabDecrees {
		h.drawDecrees(dst, f)
		return
	}
	h.drawSections(dst, page, body.X+4, body.Y+PageTop)
}

func (h *HUD) drawSections(dst *ebiten.Image, p CouncilPage, x, top int) {
	rows := SectionRows(p, top)
	for i, s := range p.Sections {
		y := rows[i][0] - ui.LineHeight
		for c, head := range s.Head {
			h.Font.Draw(dst, head, x+s.Cols[c], y, 1, ui.Palette.TextDim)
		}
		if len(s.Head) > 0 {
			y -= ui.LineHeight
		}
		h.Font.Draw(dst, s.Title, x, y, 1, ui.Palette.Gold)
		if len(s.Rows) == 0 {
			h.Font.Draw(dst, s.Empty, x, rows[i][0], 1, ui.Palette.TextDim)
		}
		for r, cells := range s.Rows {
			h.drawCells(dst, s.Cols, cells, x, rows[i][r])
		}
	}
}

func (h *HUD) drawCells(dst *ebiten.Image, cols []int, cells []Cell, x, y int) {
	for c, v := range cells {
		width := CouncilBody().W - 8 - cols[c]
		if c+1 < len(cols) {
			width = cols[c+1] - cols[c] - ui.Advance
		}
		h.Font.Draw(dst, ui.Clip(v.Text, ui.MaxChars(width, 1)), x+cols[c], y, 1, toneColor(v.Tone))
	}
}

func (h *HUD) councilPlainButtons(f Frame) []Button {
	out := []Button{}
	for _, b := range h.CouncilButtons(f) {
		if b.Label != "" && b.Rect != councilTabRect(b.ID) {
			out = append(out, b)
		}
	}
	return out
}

func councilTabRect(id string) ui.Rect {
	for i, a := range Advisors {
		if id == CouncilPrefix+":tab:"+a.Id {
			return CouncilTabID(i)
		}
	}
	return ui.Rect{W: -1}
}

func (h *HUD) drawCouncilChip(dst *ebiten.Image, f Frame) {
	for _, b := range h.CouncilButtons(f) {
		if b.ID == CouncilPrefix+":disarm" {
			h.drawChip(dst, b.Rect, SpeakLabel(DisarmCommand), h.Hovering && h.Hover.ID == b.ID)
		}
	}
}
