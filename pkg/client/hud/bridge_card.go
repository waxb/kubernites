package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	BridgeRowH      = 16
	bridgeButtonH   = 13
	bridgeButtonPad = 10
	justDone        = "Just done: "
	speakItYourself = "Now speak it yourself:"
)

func (h *HUD) BridgeRowShown(f Frame) bool {
	if h.QuestCardShown(f) {
		return false
	}
	_, ok := h.ObjectiveOffer(f)
	return ok
}

func (h *HUD) bridgeRow(f Frame) (ui.Rect, bridgeOffer, bool) {
	o, ok := h.ObjectiveOffer(f)
	if !ok || h.QuestCardShown(f) {
		return ui.Rect{}, bridgeOffer{}, false
	}
	r := h.ObjectiveRectOf(f)
	y := r.Y + ObjectiveTop + len(h.ObjectiveLines(f))*ui.LineHeight + 1
	return ui.Rect{X: r.X + 6, Y: y, W: r.W - 12, H: BridgeRowH - 2}, o, true
}

func offerButton(row ui.Rect, o bridgeOffer, id, tip string) Button {
	label := map[string]string{OfferSee: bridgeLabel, OfferAlmanac: AlmanacLabel, OfferTry: tryLabel}[o.Kind]
	w := len(label)*ui.Advance + bridgeButtonPad
	return Button{ID: id, Rect: ui.Rect{X: row.X + row.W - w, Y: row.Y, W: w, H: bridgeButtonH}, Label: bridgeLabelText(label), Tip: tip}
}

func (h *HUD) bridgeRowButtons(f Frame) []Button {
	row, o, ok := h.bridgeRow(f)
	if !ok {
		return nil
	}
	switch o.Kind {
	case OfferSee:
		return []Button{offerButton(row, o, BridgeSee, bridgeSeeTip)}
	case OfferAlmanac:
		return []Button{offerButton(row, o, BridgeAlmanac, bridgeBookTip)}
	}
	return []Button{offerButton(row, o, BridgeTry, TryButtonTip(o.line.Command))}
}

func (h *HUD) drawBridgeRow(dst *ebiten.Image, f Frame, decree bool) {
	row, o, ok := h.bridgeRow(f)
	if !ok {
		return
	}
	buttons := h.bridgeRowButtons(f)
	text := justDone + o.Title
	if o.Kind == OfferTry {
		text = speakItYourself
	}
	clr := ui.Palette.TextDim
	if decree {
		clr = ui.Palette.Ink
	}
	room := buttons[0].Rect.X - row.X - 4
	h.Font.Draw(dst, ui.Clip(text, ui.MaxChars(room, 1)), row.X+2, row.Y+3, 1, clr)
	for _, b := range buttons {
		h.drawButton(dst, b)
	}
}

func (h *HUD) calamityOfferRow(f Frame) int {
	o, ok := h.CalamityOffer(f)
	if !ok {
		return 0
	}
	v, _ := h.ShownCalamity(f)
	if o.Kind == OfferSee && deadEnd(h.calamityLines(f), v) {
		return 0
	}
	return calamityOKRow
}

func (h *HUD) calamityOfferButtons(f Frame, r ui.Rect, y int) []Button {
	o, ok := h.CalamityOffer(f)
	if !ok {
		return nil
	}
	row := ui.Rect{X: r.X + CalamityPad, Y: y + 2, W: r.W - 2*CalamityPad - calamityOKW - 6, H: calamityOKH}
	if o.Kind == OfferSee {
		b := offerButton(row, o, BridgeMended, bridgeMendTip)
		b.Rect.X, b.Rect.H = row.X, calamityOKH
		return []Button{b}
	}
	b := offerButton(row, o, bridgePrefix+":calamity-try", TryButtonTip(o.line.Command))
	b.Rect.X, b.Rect.H = row.X, calamityOKH
	return []Button{b}
}

func TryButtonTip(cmd string) string {
	return tryTip + "\n" + EchoLabel(cmd)
}
