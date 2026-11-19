package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	ObjectiveAlmanac = "obj:almanac"
	CalamityAlmanac  = "calamity:almanac"
	IntroAlmanac     = "intro:almanac"
	HelpButton       = "top:help"
	cardAlmanacW     = 58
	noPageYet        = "No Almanac page for that yet"
	helpToast        = "Help: click anything to open its page in the Almanac. Esc leaves."
)

var (
	objectiveKinds = []string{view.SubjectObjective, view.SubjectStructure, view.SubjectResource, view.SubjectConcept}
	calamityKinds  = []string{view.SubjectCalamity, view.SubjectIncident, view.SubjectConcept, view.SubjectStructure}
	introKinds     = []string{view.SubjectIntro, view.SubjectReveal, view.SubjectConcept, view.SubjectStructure, view.SubjectResource}
)

func splitIDs(ids ...string) []string {
	out := []string{}
	for _, id := range ids {
		if id == "" {
			continue
		}
		out = append(out, id)
		out = append(out, pageCandidates(strings.ReplaceAll(id, "-", ":"))...)
	}
	return out
}

func objectivePage(f Frame) (view.Page, bool) {
	o, ok := f.ShownObjective()
	if !ok {
		return view.Page{}, false
	}
	return ownPage(f, objectiveKinds, splitIDs(o.ID, o.Build, o.Target))
}

func (h *HUD) objectiveAlmanacButton(f Frame) []Button {
	if _, ok := objectivePage(f); !ok {
		return nil
	}
	r, show := h.ObjectiveRectOf(f), showRect(h.ObjectiveRectOf(f))
	title, decree := h.CardTitle(f)
	w, _ := h.Font.Measure(title, 1)
	x := r.X + titleX(decree) + w + 8
	if decree {
		x += foldMarkW
	}
	return []Button{{ID: ObjectiveAlmanac, Rect: ui.Rect{X: x, Y: show.Y, W: cardAlmanacW, H: show.H}, Label: "ALMANAC"}}
}

func calamityPage(f Frame, kind, incident string) (view.Page, bool) {
	ids := splitIDs(kind)
	if inc, ok := incidentOf(f, incident); ok {
		ids = append(splitIDs(inc.Kind), ids...)
	}
	return ownPage(f, calamityKinds, ids)
}

func (h *HUD) calamityAlmanacButton(f Frame, r ui.Rect, x int, kind, incident string) []Button {
	p, ok := calamityPage(f, kind, incident)
	if !ok {
		return nil
	}
	return []Button{{ID: CalamityAlmanac, Rect: ui.Rect{X: x - cardAlmanacW - 4, Y: r.Y + 5, W: cardAlmanacW, H: calamityBtnH}, Label: "ALMANAC", Tip: "Open the Almanac: " + p.Title, Page: p.ID}}
}

func introPageID(f Frame) string {
	if f.Guide.Intro == nil {
		return ""
	}
	p, ok := view.Content.ForSubject(view.SubjectIntro, f.Guide.Intro.ID)
	if !ok {
		return ""
	}
	return p.ID
}

func introPage(f Frame) (view.Page, bool) {
	id := introPageID(f)
	if id == "" {
		return view.Page{}, false
	}
	return view.FindPage(id)
}

func ownPage(f Frame, kinds, ids []string) (view.Page, bool) {
	for _, id := range ids {
		for _, kind := range kinds {
			if p, ok := view.Content.ForSubject(kind, id); ok {
				return p, bookMet(f, p.ID)
			}
		}
	}
	return view.Page{}, false
}

func (h *HUD) HelpMode() bool { return h.help }

func (h *HUD) ExitHelp() { h.help = false }

func (h *HUD) helpTopButton(f Frame, x int) (Button, bool) {
	if len(f.State.Almanac) == 0 {
		return Button{}, false
	}
	return Button{ID: HelpButton, Rect: ui.Rect{X: x, Y: 2, W: 16, H: 14}, Label: "?", Tip: "Help: click anything to open its page in the Almanac [F2 opens the book]", Selected: h.help}, true
}

func (h *HUD) pressHelp() []Intent {
	h.help = !h.help
	if h.help {
		h.Toast(helpToast, false)
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) helpClick(in Input, f Frame) ([]Intent, bool) {
	if !h.help || !in.Click || !h.Hovering || h.Hover.ID == HelpButton {
		return nil, false
	}
	h.help = false
	if p, ok := ButtonPage(h.Hover, f); ok {
		return h.OpenBook(p.ID, f), true
	}
	h.Toast(noPageYet, false)
	return []Intent{{Kind: IntentSound, Command: "error"}}, true
}

func (h *HUD) drawHelpCursor(dst *ebiten.Image) {
	if !h.help {
		return
	}
	r := ui.Rect{X: h.MouseX + 12, Y: h.MouseY + 10, W: 11, H: 13}
	ui.Fill(dst, r, ui.Palette.Console)
	ui.Stroke(dst, r, ui.Palette.GoldBright)
	h.Font.Draw(dst, "?", r.X+3, r.Y+3, 1, ui.Palette.GoldBright)
}
