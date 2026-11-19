package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	clueRowH      = art.IconSize + 1
	stepMax       = 2
	stepGap       = 3
	clueTextX     = art.IconSize + 4
	stepPrefix    = "NEXT: "
	fixLead       = "Fix it: "
	logsButton    = "sel:logs"
	detailsButton = "sel:details"
)

var clueArt = map[string]string{"keeps": "keep-room"}

type clueRow struct {
	icon string
	text string
	lit  bool
	step bool
}

func incidentOf(f Frame, id string) (view.IncidentReport, bool) {
	if id == "" {
		return view.IncidentReport{}, false
	}
	for _, inc := range f.State.Incidents {
		if inc.ID == id && inc.State == "open" {
			return inc, true
		}
	}
	return view.IncidentReport{}, false
}

func causeFound(inc view.IncidentReport) bool { return inc.Timeline.CauseAt > 0 }

func clueRows(inc view.IncidentReport) []clueRow {
	out := make([]clueRow, 0, len(inc.Clues))
	for _, c := range inc.Clues {
		kind := c.ID
		if alt, ok := clueArt[kind]; ok {
			kind = alt
		}
		out = append(out, clueRow{icon: art.ClueIconName(kind, c.Found), text: ui.Bridge(c.Text, c.K8s), lit: c.Found, step: !c.Found && inc.Guided && c.ID == inc.Step})
	}
	return out
}

func incidentTexts(f Frame, v view.CalamityView) (string, string) {
	inc, ok := incidentOf(f, v.Incident)
	if !ok || causeFound(inc) {
		return v.Cause, v.Why
	}
	return upperFirst(inc.Symptom) + ".", ""
}

func (h *HUD) stepLines(inc view.IncidentReport) []string {
	text := inc.StepText
	if stepIsClue(inc) {
		text = ""
		if inc.FixText != "" {
			text = fixLead + inc.FixText
		}
	}
	if !inc.Guided || text == "" {
		return nil
	}
	return h.Font.WrapLines(stepPrefix+text, calamityW-2*CalamityPad, 1, stepMax)
}

func stepIsClue(inc view.IncidentReport) bool {
	for _, c := range inc.Clues {
		if c.ID == inc.Step && !c.Found {
			return true
		}
	}
	return false
}

func (h *HUD) cardIncident(f Frame, v view.CalamityView) (view.IncidentReport, bool) {
	if c := h.steadyTrouble; c.key != "" && c.key == troubleKey(v) && c.bare {
		return view.IncidentReport{}, false
	}
	return incidentOf(f, v.Incident)
}

func (h *HUD) incidentHeight(f Frame, v view.CalamityView) int {
	inc, ok := h.cardIncident(f, v)
	if !ok {
		return 0
	}
	height := len(inc.Clues)*clueRowH + stepGap
	if steps := h.stepLines(inc); len(steps) > 0 {
		height += len(steps)*ui.LineHeight + stepGap
	}
	return height
}

func (h *HUD) drawIncident(dst *ebiten.Image, f Frame, v view.CalamityView, x, y int) {
	inc, ok := h.cardIncident(f, v)
	if !ok {
		return
	}
	for _, row := range h.stepLines(inc) {
		h.Font.Draw(dst, row, x, y, 1, ui.Palette.GoldBright)
		y += ui.LineHeight
	}
	if len(h.stepLines(inc)) > 0 {
		y += stepGap
	}
	width := calamityW - 2*CalamityPad - clueTextX
	for _, c := range clueRows(inc) {
		h.icon(dst, c.icon, x, y)
		clr := ui.Palette.TextDim
		switch {
		case c.lit:
			clr = ui.Palette.Green
		case c.step:
			clr = ui.Palette.GoldBright
		}
		h.Font.Draw(dst, ui.Clip(c.text, ui.MaxChars(width, 1)), x+clueTextX, y+(clueRowH-ui.GlyphHeight)/2, 1, clr)
		y += clueRowH
	}
}

func guidedIncident(f Frame) (view.IncidentReport, bool) {
	for _, inc := range f.State.Incidents {
		if inc.State == "open" && inc.Guided {
			return inc, true
		}
	}
	return view.IncidentReport{}, false
}

func selectsPod(f Frame, inc view.IncidentReport) bool {
	if f.Selection.Kind != SelectHearth && f.Selection.Kind != "walker" {
		return false
	}
	return f.Selection.ID == inc.Pod || inc.Crew != "" && f.Selection.Pod.Crew == inc.Crew
}

func selectsKeep(f Frame) bool {
	return f.Selection.Kind == "structure" && f.Selection.Structure.Kind == view.StructKeep
}

func IncidentWants(f Frame) string {
	inc, ok := guidedIncident(f)
	if !ok {
		return ""
	}
	switch {
	case inc.Step == "logs" && selectsPod(f, inc):
		return logsButton
	case inc.Step == "describe" && selectsPod(f, inc), inc.Step == "keeps" && selectsKeep(f):
		return ActionPrefix + ":" + KeyDescribe
	}
	return ""
}

func incidentLit(f Frame, id string) bool {
	want := IncidentWants(f)
	return want != "" && (id == want || want == ActionPrefix+":"+KeyDescribe && id == detailsButton)
}

func showLit(f Frame, v view.CalamityView) bool {
	inc, ok := incidentOf(f, v.Incident)
	return ok && inc.Guided && inc.Step == "hut"
}
