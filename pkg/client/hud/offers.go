package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	CmdDepot    = "cmd:depot"
	CmdCrew     = "cmd:crew"
	raisePrefix = "raise-crew:"
	depotPrefix = "open-depot:"
	maxCardLine = 2
)

type CommandLine struct {
	Label     string
	Command   string
	Action    *view.Action
	Run       bool
	Payoff    string
	Show      bool
	Effect    string
	EffectBad bool
	Blocked   string
	Unwired   bool
	Order     string
	Shown     string
}

func (l CommandLine) Runnable() bool {
	return l.Action != nil || l.Run && l.Command != ""
}

var (
	crewButton = Button{ID: CmdCrew, Label: "CREW", Icon: "icon/roll", Menu: true,
		Tip: "Opens RAISE A CREW: pick a trade, its class and its count. Once a crew stands, it opens the list of your crews: scale one or open its board."}
)

func OfferLine(o view.CommandOffer) CommandLine {
	a := o.Action
	return CommandLine{Label: o.Label, Command: ShownCommand(o.Command), Action: &a}
}

func offersWith(offers []view.CommandOffer, prefix string) []view.CommandOffer {
	out := []view.CommandOffer{}
	for _, o := range offers {
		if strings.HasPrefix(o.ID, prefix) {
			out = append(out, o)
		}
	}
	return out
}

func offerByID(offers []view.CommandOffer, id string) (view.CommandOffer, bool) {
	for _, o := range offers {
		if o.ID == id {
			return o, true
		}
	}
	return view.CommandOffer{}, false
}

func commandWords(cmd string) []string {
	return strings.Fields(strings.TrimPrefix(strings.TrimSpace(cmd), "kubectl "))
}

func FlagValue(words []string, name string) string {
	for i, w := range words {
		if v, ok := strings.CutPrefix(w, name+"="); ok {
			return v
		}
		if w == name && i+1 < len(words) {
			return words[i+1]
		}
	}
	return ""
}

func isVerb(cmd string, verbs ...string) bool {
	w := commandWords(cmd)
	return len(w) >= len(verbs) && strings.Join(w[:len(verbs)], " ") == strings.Join(verbs, " ")
}

func LineFor(cmd string, offers []view.CommandOffer) CommandLine {
	w := commandWords(cmd)
	line := CommandLine{Command: cmd}
	switch {
	case len(w) >= 3 && w[0] == "create" && w[1] == "deployment":
		if o, ok := offerByID(offers, raisePrefix+w[2]); ok {
			line.Label, line.Action = o.Label, OfferLine(o).Action
		}
	case len(w) >= 3 && w[0] == "expose":
		d, ok := soulstone.DepotOfService(FlagValue(w, "--name"))
		if !ok {
			return line
		}
		line.Label, line.Action = "Open the "+d.Service+" depot", &view.Action{Action: "openDepot", Kind: d.Kind}
		if o, ok := offerByID(offers, depotPrefix+d.Kind); ok {
			line.Label, line.Action = o.Label, OfferLine(o).Action
		}
	}
	return line
}

func closedDepot(f Frame) (soulstone.Depot, bool) {
	for _, k := range f.State.Freight.Kinds {
		if k.Open && !k.DepotOpen {
			return soulstone.DepotFor(k.Kind)
		}
	}
	return soulstone.Depot{}, false
}

func DepotLine(f Frame) (CommandLine, bool) {
	o, current := f.current()
	if current && isVerb(o.Command, "expose") {
		return LineFor(o.Command, f.State.Commands), true
	}
	if list := offersWith(f.State.Commands, depotPrefix); len(list) > 0 {
		return OfferLine(list[0]), true
	}
	if !current || o.Target != CmdDepot {
		return CommandLine{}, false
	}
	d, ok := closedDepot(f)
	if !ok {
		return CommandLine{}, false
	}
	return CommandLine{Label: "Open the " + d.Service + " depot", Action: &view.Action{Action: "openDepot", Kind: d.Kind}}, true
}

func CrewLine(f Frame) (CommandLine, bool) {
	if o, ok := f.current(); ok && isVerb(o.Command, "create", "deployment") {
		if l := LineFor(o.Command, f.State.Commands); l.Action != nil {
			return l, true
		}
	}
	if list := offersWith(f.State.Commands, raisePrefix); len(list) > 0 {
		return OfferLine(list[0]), true
	}
	return CommandLine{}, false
}

const tryStep = "try:"

func (f Frame) ObjectiveLine() (CommandLine, bool) {
	o, ok := f.current()
	if !ok {
		return CommandLine{}, false
	}
	var l CommandLine
	switch {
	case o.Command != "" && ui.StoryNaming() && !lexicon.StoryCommandShown(o.Command):
		return CommandLine{}, false
	case o.Command != "":
		l = LineFor(o.Command, f.State.Commands)
	case strings.HasPrefix(o.ID, tryStep):
		return CommandLine{}, false
	case o.Target == CmdDepot:
		l, ok = DepotLine(f)
	case o.Target == CmdCrew:
		l, ok = CrewLine(f)
	default:
		return CommandLine{}, false
	}
	l = objectivePath(l, o)
	if o.ID == view.AskObjective {
		l.Action, l.Run = nil, false
	}
	return l, ok && l.Command != ""
}

func depotOfStructure(s view.Structure) (soulstone.Depot, bool) {
	if s.Object.Kind != "Service" {
		return soulstone.Depot{}, false
	}
	return soulstone.DepotOfService(s.Object.Name)
}

func CardLines(f Frame) []CommandLine {
	if !f.actionsShown() || f.Selection.Kind != "structure" {
		return nil
	}
	s := f.Selection.Structure
	d, isDepot := depotOfStructure(s)
	out := []CommandLine{}
	for _, o := range f.State.Commands {
		mine := isDepot && (o.ID == raisePrefix+d.Crew || o.ID == depotPrefix+d.Kind)
		if mine || s.Kind == view.StructChancery {
			out = append(out, OfferLine(o))
		}
	}
	limit := maxCardLine
	if s.Kind == view.StructChancery {
		limit = 1
	}
	return out[:min(len(out), limit)]
}

func offerButtons(f Frame) []Button {
	out := []Button{}
	for _, o := range []struct {
		reveal string
		base   Button
	}{{"crew", crewButtonFor(f)}} {
		if f.Revealed(o.reveal) && f.Shown(o.base.ID) {
			b := o.base
			b.Tip = withEffect(b.Tip, f.Effect(b.ID))
			out = append(out, WithClickNote(b))
		}
	}
	return out
}
