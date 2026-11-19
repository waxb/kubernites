package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	IntentEcho     = "echo"
	ActionPrefix   = "act"
	actionH        = 15
	wardKey        = "ward"
	drainRune      = "drain"
	defaultNS      = "default"
	systemNS       = "kube-system"
	detailOutput   = "output"
	drainLockTip   = "Learn Masonry I in the Grimoire to drain a Keep (kubectl drain)."
	citadelLockTip = "The Citadel's own huts (kube-system) are not yours to delete."
)

type CardAction struct {
	Key      string
	Label    string
	Command  string
	Tip      string
	Disabled bool
	Details  string
	Choices  []CardAction
	Act      view.Action
}

func RuneKnown(f Frame, rune string) bool {
	for _, p := range f.State.Grimoire {
		for _, r := range p.Ranks {
			if r.Learned && slices.Contains(r.Runes, rune) {
				return true
			}
		}
	}
	return false
}

func (f Frame) actionsShown() bool {
	return f.StoneStands()
}

func SelectionActions(f Frame) []CardAction {
	if !f.actionsShown() {
		return nil
	}
	sel := f.Selection
	switch {
	case sel.Gone:
		return nil
	case sel.Kind == "hearth" || sel.Kind == "walker":
		return taughtActions(withRollActions(withDeedActions(withDoorReseat(HearthActions(sel.Pod), sel.Pod, f), sel.Pod, f), sel.Pod, f), f)
	case sel.Kind != "structure" || sel.Structure.State == view.StateConstruction:
		return nil
	}
	return taughtActions(StructureActions(sel.Structure, f), f)
}

func StructureActions(s view.Structure, f Frame) []CardAction {
	switch {
	case s.Kind == view.StructKeep && s.Object.Name != "":
		return KeepActions(s, f)
	case s.Kind == view.StructCodex:
		return CodexActions(f)
	case s.Kind == view.StructChancery:
		return ChanceryActions()
	case s.Kind == view.StructObservatory:
		return ObservatoryActions(f)
	case s.Kind == view.StructVault:
		return VaultActions(s, f)
	}
	if d, ok := depotOfStructure(s); ok {
		return withRollDepot(DepotActions(d.Service), d.Service, f)
	}
	return nil
}

func CardSource(f Frame) string {
	sel := f.Selection
	if sel.Kind != "structure" {
		return FromHearthCard
	}
	switch sel.Structure.Kind {
	case view.StructKeep:
		return FromKeepCard
	case view.StructCodex:
		return FromCodexCard
	case view.StructChancery:
		return FromChanceryCard
	case view.StructObservatory:
		return FromObservatoryCard
	}
	return FromDepotCard
}

func ActionTip(a CardAction) string {
	if len(a.Choices) > 0 {
		return a.Tip + "\nChoose what to " + strings.ToLower(a.Label) + "."
	}
	return a.Tip + SpeakTip(a.Command)
}

func ActionByKey(actions []CardAction, key string) (CardAction, bool) {
	for _, a := range actions {
		if a.Key == key {
			return a, true
		}
	}
	return CardAction{}, false
}

func HasDescribe(actions []CardAction) bool {
	_, ok := ActionByKey(actions, KeyDescribe)
	return ok
}

func FirstLine(output string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(output), "\n")
	return strings.TrimSpace(line)
}

func (h *HUD) PressAction(rest string, f Frame) []Intent {
	a, ok := ActionByKey(SelectionActions(f), rest)
	if !ok || a.Disabled {
		return nil
	}
	if len(a.Choices) > 0 {
		h.openChooser(a, f)
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	intents := h.RunAction(a, f.Selection, CardSource(f))
	if a.Key == keyEndpoints && !ui.Naming() && h.OverlayID == OverlayDetails {
		h.DetailsCard.Lines = append(h.DetailsCard.Lines, RollByHut(f, a.Command)...)
	}
	return intents
}

const rollHutWidth = 26

func RollByHut(f Frame, cmd string) []string {
	_, service, _ := strings.Cut(cmd, "kubernetes.io/service-name=")
	for _, r := range f.Rolls {
		if r.Service != service || len(r.Members) == 0 {
			continue
		}
		out := []string{"", "The roll by hut (each answers at its address):"}
		for _, m := range r.Members {
			name := m.Name
			switch {
			case isGone(f, m.Name):
				name += " (gone)"
			case !m.Ready:
				name += " (not ready)"
			}
			out = append(out, "  "+name+strings.Repeat(" ", max(rollHutWidth-len(name), 1))+RollAddressMark+m.Address)
		}
		return out
	}
	return nil
}

func (h *HUD) RunAction(a CardAction, sel Selection, from string) []Intent {
	if a.Act.Action != "" {
		return []Intent{ActFrom(a.Act, from), {Kind: IntentSound, Command: "click"}}
	}
	res, intents := h.RunFrom(a.Command, from)
	if a.Details != "" {
		h.showOutput(sel, a.Details, a.Command, res)
		return append(intents, Intent{Kind: IntentSound, Command: "click"})
	}
	h.Toast(answerLine(res.Output), !res.OK)
	sound := "click"
	if !res.OK {
		sound = "error"
	}
	return append(intents, Intent{Kind: IntentSound, Command: sound})
}

func answerLine(output string) string {
	if line := FirstLine(output); line != "" {
		return line
	}
	return "(no output)"
}

func echoState(ok bool) string {
	if ok {
		return "ok"
	}
	return "error"
}

func (h *HUD) showOutput(sel Selection, tab, cmd string, res view.CommandResult) {
	h.DetailsCard = Details{Title: SelectionTitle(sel), Sel: sel, Tab: tab, OK: res.OK}
	h.DetailsCard.Lines = h.AnswerLines(SpeakLabel(cmd), res)
	h.OverlayID = OverlayDetails
}

func GuildhallLines(s view.Structure, f Frame) []Line {
	for _, r := range f.Rolls {
		if r.Service != s.Object.Name || s.Object.Namespace != "" && r.Namespace != s.Object.Namespace {
			continue
		}
		gone := len(goneMembers(f))
		head := fmt.Sprintf("GUILD ROLL %d of %d ready", r.Ready-gone, r.Endpoints)
		if d, ok := depotFlowOf(f.State.Freight, r.Service); ok {
			if n := rollOutsiders(f, r.Service, d.crew); n > 0 {
				head += fmt.Sprintf(", %d outside the crew", n)
			}
		}
		out := []Line{{Text: head, Clr: toneColor(rollTone(r))}}
		for _, m := range r.Members {
			mark, clr := "ready", ui.Palette.Green
			switch {
			case isGone(f, m.Name):
				mark, clr = "gone: its boxes spoil", ui.Palette.RedText
			case !m.Ready:
				mark, clr = "not ready", ui.Palette.RedText
			}
			out = append(out, Line{Text: m.Name + "  " + mark, Clr: clr})
		}
		return out
	}
	return []Line{{Text: "GUILD ROLL empty: no ready hut answers", Clr: ui.Palette.RedText}}
}
