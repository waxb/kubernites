package hud

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	contractRowH   = 56
	contractRowGap = 6
	contractTop    = 46
	ContractLeft   = 28
	ContractRight  = 170
	contractBtnW   = 80
	contractBtnH   = 14
	ContractsShown = 5
	hurryBelow     = 30
	TradeTip       = "Contracts: merchants' offers with a deadline and a reward"
	contractsLine  = "Merchants offer work with a deadline. Hold at most 2 at once."
	noContracts    = "No merchant waits. Their ships bring offers every minute or two."
	tradeOpenHint  = "Click to open the contracts board"
)

func ContractRowRect(i int) ui.Rect {
	return ui.Rect{X: Overlay.X + 10, Y: Overlay.Y + contractTop + i*(contractRowH+contractRowGap), W: Overlay.W - 20, H: contractRowH}
}

func ContractButtonRect(row ui.Rect) ui.Rect {
	return ui.Rect{X: row.X + row.W - contractBtnW - 6, Y: row.Y + row.H - contractBtnH - 4, W: contractBtnW, H: contractBtnH}
}

func (h *HUD) ContractButton(r ContractRow, row ui.Rect) (Button, bool) {
	b := Button{Rect: ContractButtonRect(row)}
	switch r.State {
	case StateOffered:
		b.ID, b.Label, b.Tip = contractPrefix+":accept:"+r.ID, "ACCEPT", "Take the contract: its clock starts now"
		if !r.CanAccept {
			b.Disabled, b.Tip = true, FullHands
		}
	case StateActive:
		b.ID, b.Label, b.Tip = contractPrefix+":abandon:"+r.ID, "ABANDON", "Give it up: the merchants' Favor falls by 3"
		if h.abandonArmed(r.ID) {
			b.Label, b.Selected, b.Tip = "SURE?", true, "Click again to abandon it"
		}
	default:
		return Button{}, false
	}
	return b, true
}

func (h *HUD) contractsButtons(f Frame) []Button {
	out := []Button{closeButton(Overlay)}
	rows := ContractRows(f)
	for i, r := range rows[:min(len(rows), ContractsShown)] {
		if b, ok := h.ContractButton(r, ContractRowRect(i)); ok {
			out = append(out, b)
		}
	}
	return out
}

func (h *HUD) DrawContracts(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, Overlay)
	h.header(dst, Overlay, "CONTRACTS")
	line := fmt.Sprintf("%s  Held %d of %d.", contractsLine, Held(f), MaxHeld)
	h.Font.Draw(dst, line, Overlay.X+10, Overlay.Y+30, 1, ui.Palette.TextDim)
	rows := ContractRows(f)
	if len(rows) == 0 {
		h.Font.DrawCentered(dst, noContracts, ui.Rect{X: Overlay.X, Y: Overlay.Y + Overlay.H/2 - 6, W: Overlay.W, H: 12}, 1, ui.Palette.Text)
	}
	for i, r := range rows[:min(len(rows), ContractsShown)] {
		h.drawContractRow(dst, r, ContractRowRect(i))
	}
	h.DrawOverlayButtons(dst, h.contractsButtons(f))
}

func (h *HUD) drawContractMarks(dst *ebiten.Image, r ContractRow, row ui.Rect) {
	h.icon(dst, r.Icon, row.X+6, row.Y+4)
	y := row.Y + 21
	if r.Sealed {
		h.icon(dst, "icon/sealed", row.X+6, y)
		y += 17
	}
	if r.Crowned {
		h.icon(dst, "icon/priority", row.X+6, y)
	}
}

func (h *HUD) drawContractRow(dst *ebiten.Image, r ContractRow, row ui.Rect) {
	edge := ui.Palette.BlueDark
	if r.State == StateActive {
		edge = ui.Palette.Gold
	}
	ui.Fill(dst, row, consoleInk)
	ui.Stroke(dst, row, edge)
	h.drawContractMarks(dst, r, row)
	x, textW := row.X+ContractLeft, row.W-ContractLeft-ContractRight-contractPicW
	h.drawContractPic(dst, r, row)
	h.Font.Draw(dst, ui.Clip(r.Title, ui.MaxChars(textW, 1)), x, row.Y+5, 1, ui.Palette.Text)
	state, clr := "OFFER", ui.Palette.Blue
	if r.State == StateActive {
		state, clr = "HELD", ui.Palette.GoldBright
	}
	h.Font.Draw(dst, state, x, row.Y+17, 1, clr)
	at := h.Font.Span(state, 2)
	h.Font.Draw(dst, ui.Clip(r.Merchant, ui.MaxChars(textW-at, 1)), x+at, row.Y+17, 1, ui.Palette.TextDim)
	h.drawContractBody(dst, r, x, row.Y+29, textW)
	right := row.X + row.W - ContractRight + 6
	for i, l := range h.Font.WrapLines(r.Reward, ContractRight-12, 1, 2) {
		h.Font.Draw(dst, l, right, row.Y+5+i*ui.LineHeight, 1, ui.Palette.GoldBright)
	}
	tone := ui.Palette.Text
	if r.Left < hurryBelow {
		tone = ui.Palette.RedText
	}
	h.Font.Draw(dst, r.Clock(), right, row.Y+27, 1, tone)
}

func (h *HUD) drawContractBody(dst *ebiten.Image, r ContractRow, x, y, w int) {
	if r.State != StateActive {
		for i, l := range h.Font.WrapLines(r.Goal, w, 1, 2) {
			h.Font.Draw(dst, l, x, y+i*ui.LineHeight, 1, ui.Palette.TextDim)
		}
		return
	}
	ui.Bar(dst, ui.Rect{X: x, Y: y + 1, W: w, H: 7}, r.Share(), ui.Palette.Green)
	h.Font.Draw(dst, ui.Clip(r.Progress, ui.MaxChars(w, 1)), x, y+12, 1, ui.Palette.TextDim)
}

func (h *HUD) drawTradeSummary(dst *ebiten.Image, f Frame, body ui.Rect) {
	offers := len(Offered(f))
	h.Font.Draw(dst, fmt.Sprintf("HELD %d/%d   %s", Held(f), MaxHeld, plural(offers, "OFFER", "OFFERS")), body.X+4, body.Y+4, 1, ui.Palette.Gold)
	y := body.Y + 17
	for _, r := range ContractRows(f) {
		if r.State != StateActive {
			continue
		}
		h.icon(dst, r.Icon, body.X+4, y-3)
		h.Font.Draw(dst, ui.Clip(r.Title, ui.MaxChars(body.W-30, 1)), body.X+24, y, 1, ui.Palette.Text)
		ui.Bar(dst, ui.Rect{X: body.X + 24, Y: y + 11, W: body.W - 100, H: 5}, r.Share(), ui.Palette.Green)
		h.Font.Draw(dst, r.Countdown(), body.X+body.W-70, y+9, 1, ui.Palette.TextDim)
		y += 24
	}
	h.Font.Draw(dst, tradeOpenHint, body.X+4, body.Y+body.H-11, 1, ui.Palette.GoldBright)
}

func init() {
	Overlays[OverlayContracts] = OverlaySpec{(*HUD).contractsButtons, (*HUD).DrawContracts}
	prefixHandlers[contractPrefix] = (*HUD).PressContract
	surfaceButtons[SurfaceContracts] = "tab:" + TabTrade
	openers[SurfaceContracts] = []string{"tab:" + TabTrade, ContractsOpen}
	wellTabs = append(wellTabs, struct{ tab, reveal, surface, tip string }{TabTrade, ContractsReveal, SurfaceContracts, TradeTip})
}
