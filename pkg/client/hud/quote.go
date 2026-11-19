package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	ButtonToken  = view.ButtonToken
	councilLabel = "COUNCIL"
	unknownQuote = "the highlighted button"
)

var namedLabels = map[string]string{"cmd:siege": siegeLabel, "cmd:council": councilLabel, SurfaceOverlays: lensLabel, SurfaceHeraldry: lensLabel, CmdDepot: "BUILD"}

func ButtonLabel(id string) (string, bool) {
	for _, e := range mainEntries {
		if e.button.ID == id {
			return e.button.Label, true
		}
	}
	for _, b := range []Button{DoorButton, crewButton, voyageButton, moreButton} {
		if b.ID == id {
			return b.Label, true
		}
	}
	if id == view.WatchButton {
		return ui.Bridge("WATCH", "PROBES"), true
	}
	label, ok := namedLabels[id]
	return label, ok
}

func QuoteButton(text, id string) string {
	if !strings.Contains(text, ButtonToken) {
		return text
	}
	label, ok := ButtonLabel(id)
	if !ok {
		label = unknownQuote
	}
	return strings.ReplaceAll(text, ButtonToken, label)
}

func (f Frame) QuoteObjective(text string) string {
	if f.Guide.Button != "" {
		if _, ok := ButtonLabel(f.Guide.Button); ok {
			return QuoteButton(text, f.Guide.Button)
		}
	}
	return QuoteButton(text, f.Guide.Target)
}
