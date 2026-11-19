package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	invitePrefix = "invite"
	lockedMark   = "Locked: "
)

func InviteButton(page string) string { return invitePrefix + ":" + page }

func InviteLabel(inv view.Invitation) string {
	return "LEARN " + strings.ToUpper(inv.Rank) + "  " + costText(inv.Gold, inv.Wisdom)
}

func InviteTip(inv view.Invitation) string {
	return fmt.Sprintf("%s%s.\nLearn %s in the Grimoire and this answer is yours: the button opens its page.", lockedMark, inv.Label, inv.Rank)
}

func InviteHeight(inv *view.Invitation) int {
	if inv == nil {
		return 0
	}
	return remedyRowH
}

func InviteButtons(inv *view.Invitation, r ui.Rect, y int) []Button {
	if inv == nil {
		return nil
	}
	row := ui.Rect{X: r.X + CalamityPad, Y: y + remedyRowH - ChipH - lineGap, W: r.W - 2*CalamityPad, H: ChipH}
	label := ui.Clip(InviteLabel(*inv), ui.MaxChars(row.W-4, 1))
	return []Button{{ID: InviteButton(inv.Page), Rect: row, Label: label, Tip: InviteTip(*inv)}}
}

func (h *HUD) DrawInvite(dst *ebiten.Image, inv *view.Invitation, r ui.Rect, y int) {
	if inv == nil {
		return
	}
	text := ui.Clip(lockedMark+inv.Label, ui.MaxChars(r.W-2*CalamityPad, 1))
	h.Font.Draw(dst, text, r.X+CalamityPad, y+1, 1, ui.Palette.TextDim)
}

func (h *HUD) pressInvite(page string, f Frame) []Intent {
	h.OpenGrimoireAt(f, page)
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) OpenGrimoireAt(f Frame, page string) {
	h.OverlayID, h.Grimoire = OverlayGrimoire, page
	h.Kmap.Focus, h.Kmap.Held, h.Kmap.heldSet = "", true, false
	l := h.Knowledge(f)
	if p, ok := l.Find(page); ok {
		r := p.Rect()
		h.Kmap.Scroll = KmapScroll{Pos: float64(r.X + r.W/2 - KnowledgeView.W/2)}.clamp(KmapLimit(l))
	}
}

func init() {
	prefixHandlers[invitePrefix] = (*HUD).pressInvite
}
