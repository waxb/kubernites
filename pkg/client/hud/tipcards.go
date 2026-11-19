package hud

import (
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const NoPage = "no page"

type TipCard struct {
	ID      string
	Label   string
	Tip     string
	Purpose string
	How     string
	Why     string
	Page    string
}

func (h *HUD) TipCards(f Frame) []TipCard {
	out := []TipCard{}
	for _, b := range h.Buttons(f) {
		if b.Tip == "" && b.Label == "" {
			continue
		}
		tc := TipCard{ID: b.ID, Label: ui.Plain(ui.Name(b.Label)), Tip: ui.Plain(ui.Name(b.Tip)), Page: NoPage}
		if p, ok := ButtonPage(b, f); ok {
			tc.Page, tc.Purpose, tc.How, tc.Why = p.ID, ui.Name(view.Plain(p.Purpose)), ui.Name(view.Plain(p.How)), ui.Name(view.Plain(p.Why))
		}
		out = append(out, tc)
	}
	return out
}
