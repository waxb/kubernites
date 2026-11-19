package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const bookTrialGap = 6

func (h *HUD) BookTrialButtons() []Button {
	t, ok := view.TrialForPage(h.Book.PageID)
	if !ok || h.Book.PageID == "" {
		return nil
	}
	label := "PRACTISE IT: " + strings.ToUpper(trialText(t.Title))
	page := BookPageRect(0)
	w := h.Font.Width(label, 1) + 2*TrialBtnPad
	r := ui.Rect{X: page.X, Y: page.Y + page.H + bookTrialGap - TrialBtnH, W: w, H: TrialBtnH}
	return []Button{{ID: trialPrefix + ":" + TrialBoard + ":" + t.ID, Rect: r, Label: label,
		Tip: "A short scored drill outside your realm on this page's idea, with a gold, silver and bronze time"}}
}
