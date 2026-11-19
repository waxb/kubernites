package hud

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	placeNote = "Click: choose where it stands, then click there to build."
	showLabel = "SHOW"
	hintLife  = 8.0
	crewNS    = "default/"
)

type teachHint struct {
	crew    string
	button  string
	action  string
	desired int
	at      float64
}

func FixSource(from string) bool {
	return from == FromHarborBoard || from == FromCalamityCard
}

func fades(a view.Action) bool {
	return a.Action == view.ActScaleCrew || a.Action == "buyKeep"
}

func fixFade(e view.EconomyView, o view.CommandOffer) string {
	if !fades(o.Action) {
		return view.FixActs
	}
	return cmp.Or(e.FixMode, view.FixActs)
}

func FixFade(f Frame, o view.CommandOffer) string {
	return fixFade(EconomyOf(f.State), o)
}

func fixCrew(a view.Action) string {
	return cmp.Or(a.For, a.Name)
}

func ShowLabel(o view.CommandOffer, room int) string {
	full := showLabel + " " + strings.ToUpper(fixCrew(o.Action))
	if NamedFont.Width(full, 1)+FlowbarFixPad > room {
		return showLabel
	}
	return full
}

func ShowTip(a view.Action) string {
	if a.Action == "buyKeep" {
		return fmt.Sprintf(ui.Bridge("SHOW selects the %s and lights BUY KEEP on their board: pick the plot and place the Keep yourself.",
			"SHOW selects the Deployment %s and lights ADD NODE: pick the plot and place the Node yourself."), fixCrew(a))
	}
	return fmt.Sprintf(ui.Bridge("SHOW selects the %s and lights their scale control: raise them there, by hand.",
		"SHOW selects the Deployment %s and lights its scale control: scale it there, by hand."), fixCrew(a))
}

func whereToGo(a view.Action) string {
	if a.Action == "buyKeep" {
		return fmt.Sprintf(ui.Bridge("Go to the %[1]s: open their board and BUY KEEP, or BUILD a Keep and place it where the %[1]s fit best.",
			"Go to the Deployment %[1]s: select it and ADD NODE, or BUILD a Node and place it."), fixCrew(a))
	}
	return fmt.Sprintf(ui.Bridge("Go to the %[1]s: click one of their huts, open the crew board and raise the crew there, while the Keeps have room.",
		"Go to the Deployment %[1]s: select the Deployment and scale it, or kubectl scale deployment %[1]s, while the Nodes have room."), fixCrew(a))
}

func fixTipLine(e view.EconomyView) string {
	switch fixFade(e, *e.Fix) {
	case view.FixShows:
		return ShowTip(e.Fix.Action)
	case view.FixNames:
		return whereToGo(e.Fix.Action)
	}
	return "The button fixes it: " + e.Fix.Label + "."
}

func NeckText(f Frame) string {
	e := EconomyOf(f.State)
	name := NeckName(e.Bottleneck)
	if name == "" && e.Fix == nil {
		return lossNeck(f.State)
	}
	if e.Fix == nil || name == "" || fixFade(e, *e.Fix) != view.FixNames {
		return name
	}
	return name + " (" + strings.ToUpper(fixCrew(e.Fix.Action)) + ")"
}

func withFade(l CommandLine, f Frame) CommandLine {
	next := l
	next.Show = l.Action != nil && l.Action.Action == view.ActScaleCrew && EconomyOf(f.State).FixMode != "" && EconomyOf(f.State).FixMode != view.FixActs
	return next
}

func hintButton(a view.Action) string {
	if a.Action == "buyKeep" {
		return BuyKeepButton
	}
	return StepUpButton
}

func (h *HUD) showFix(a view.Action, f Frame) []Intent {
	id := crewNS + fixCrew(a)
	desired := 0
	if c, ok := chainByID(f, id); ok {
		desired = c.Desired
	}
	h.View.Chain = id
	h.teach = teachHint{crew: id, button: hintButton(a), action: a.Action, desired: desired, at: h.Now}
	if a.Action == view.ActScaleCrew {
		h.Spotlight(GridScale)
	}
	return []Intent{{Kind: IntentChoose, Target: SelectCrew, Command: id}, {Kind: IntentSound, Command: "click"}}
}

func chainByID(f Frame, id string) (model.CrewChain, bool) {
	for _, c := range f.Crews {
		if c.ID() == id {
			return c, true
		}
	}
	return model.CrewChain{}, false
}

func (h *HUD) hintLive(f Frame) bool {
	t := h.teach
	if t.button == "" || f.Clock-t.at > hintLife || f.Placing != "" {
		return false
	}
	c, ok := chainByID(f, t.crew)
	return !ok || c.Desired == t.desired
}

func (h *HUD) HintedButton(f Frame) string {
	if !h.hintLive(f) {
		return ""
	}
	return h.teach.button
}

func (h *HUD) HintedAction(f Frame) string {
	if !h.hintLive(f) {
		return ""
	}
	return h.teach.action
}

func (h *HUD) drawHint(dst *ebiten.Image, f Frame) {
	id := h.HintedButton(f)
	if id == "" {
		return
	}
	for _, b := range h.Buttons(f) {
		if b.ID == id {
			h.drawBeacon(dst, b.Rect)
			return
		}
	}
}

func fadedShortFix(st view.GameState, rem view.RemedyView) string {
	if a := rem.Action; a != nil && a.Action == view.ActScaleCrew && st.Economy.FixMode == view.FixNames {
		return "too few " + a.Name
	}
	return ShortFix(rem)
}

func OpensPlacement(a *view.Action) bool {
	return a != nil && (a.Action == "buyKeep" || a.Action == "openDepot")
}

func lossNeck(st view.GameState) string {
	lines := StateLossLines(st)
	if len(lines) == 0 {
		return ""
	}
	return strings.ToUpper(lossLabel(lines[0].Cause))
}
