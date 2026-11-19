package hud

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

var portraitFallbacks = map[string]string{"keep-great": "keep-stone", "keep-citadel": "keep-stone"}

func IsKeepKind(kind string) bool {
	return kind == view.StructKeep || strings.HasSuffix(kind, "-keep")
}

func KeepSpec(kind string) string {
	k, ok := view.KeepTypeOf(kind)
	if !ok || !IsKeepKind(kind) {
		return ""
	}
	return fmt.Sprintf("%s, %s", view.RationsOf(k.CPU), view.RoomsOf(k.Memory))
}

func hasKeep(site view.Site) bool {
	for _, s := range site.Structures {
		if s.Kind == view.StructKeep {
			return true
		}
	}
	return false
}

func standing(site view.Site, kind string) bool {
	for _, s := range site.Structures {
		if s.Kind == kind {
			return true
		}
	}
	return false
}

func offered(b view.Buildable, f Frame) bool {
	if IsKeepKind(b.Kind) {
		return b.Kind == view.StructKeep || f.Revealed("keeps")
	}
	return b.Available || !standing(f.Site, b.Kind)
}

func keepVariant(kind string, site view.Site) string {
	if kind == view.StructKeep && !hasKeep(site) {
		return "tower"
	}
	if k, ok := view.KeepTypeOf(kind); ok {
		return k.Variant
	}
	return "stone"
}

func keepIcon(kind string, f Frame) string {
	offer, _ := KeepOfferFor(f, kind)
	return gfx.KeepPreview(keepVariant(kind, f.Site), offer.Class, gfx.AgeTier(f.State.Age)).BuildIconName()
}

func keepName(b view.Buildable, site view.Site) string {
	if b.Kind == view.StructKeep && !hasKeep(site) {
		return b.Name
	}
	if k, ok := view.KeepTypeOf(b.Kind); ok {
		return k.Name
	}
	return b.Name
}

func KeepLabel(b view.Buildable, f Frame) string {
	name := keepName(b, f.Site)
	label := strings.ToUpper(name)
	if len(label) > ui.MaxChars(CellW-4, 1) {
		label = strings.TrimSuffix(label, " KEEP")
	}
	_, rate, rent := keepRent(f, b.Kind)
	_, sized := strings.CutSuffix(name, " Keep")
	node := strings.ToUpper(lexicon.TrueNames(name))
	switch {
	case sized && rent:
		priced := node + " -$" + rate + "/S"
		if !fitsCell(priced) {
			priced = strings.TrimSuffix(node, " NODE") + " -$" + rate + "/S"
		}
		return ui.Bridge(label+" -"+rate+"/S", priced)
	case sized:
		return ui.Bridge(label, node)
	case rent:
		return label + " " + ui.Bridge("-"+rate+"/S", "-$"+rate+"/S")
	}
	return label
}

func fitsCell(label string) bool {
	for _, row := range CellLabel(ui.Settle(label), ui.MaxChars(CellW-4, 1)) {
		if strings.HasSuffix(row, "...") {
			return false
		}
	}
	return true
}

func keepSpecOf(kind string, f Frame) string {
	if o, ok := KeepOfferFor(f, kind); ok && o.CPU != "" && o.Memory != "" {
		return fmt.Sprintf("%s, %s", view.RationsOf(o.CPU), view.RoomsOf(o.Memory))
	}
	return KeepSpec(kind)
}

func keepRent(f Frame, kind string) (string, string, bool) {
	k, ok := view.KeepTypeOf(kind)
	if !ok {
		return "", "", false
	}
	for _, o := range f.State.KeepShop {
		if o.KeepType == k.KeepType && o.UpkeepPerSec > 0 {
			rate := Rate(o.UpkeepPerSec)
			return ui.Bridge(rentLine(rate, f), "Cost: $"+rate+"/s while the Node runs."), rate, true
		}
	}
	return "", "", false
}

func rentLine(rate string, f Frame) string {
	if view.RentDrains(f.State.Age) {
		return "Rent: " + rate + " gold a second, paid while it stands."
	}
	return "Rent: " + rate + " gold a second out of your income while it stands; it never takes your gold."
}

func keepTip(b view.Buildable, f Frame) string {
	lines := buildWords(b)
	if spec := keepSpecOf(b.Kind, f); spec != "" && hasKeep(f.Site) {
		lines = append(lines, spec+".")
	}
	lines = append(lines, keepOfferLines(b, f)...)
	if effect := f.Effect("build:" + b.Kind); effect != "" {
		lines = append(lines, effect)
	}
	lines = append(lines, buildReason(b)...)
	if b.Reserve {
		lines = append(lines, ui.Bridge(reserveTip, "Provisioned with the environment: Staging gets a second Node, paid from the upgrade. Place it; its cost per second is yours."))
	} else {
		lines = append(lines, buildPrice(b))
	}
	if rent, rate, ok := keepRent(f, b.Kind); ok && !strings.Contains(strings.Join(lines, "\n"), rate) {
		lines = append(lines, rent)
	}
	if idle := keepIdle(f, b.Kind); idle != "" {
		lines = append(lines, idle)
	}
	if net := keepNet(f, b.Kind); net != "" {
		lines = append(lines, net)
	}
	return strings.Join(lines, "\n")
}

func keepNet(f Frame, kind string) string {
	k, ok := view.KeepTypeOf(kind)
	if !ok {
		return ""
	}
	for _, o := range f.State.KeepShop {
		if o.KeepType == k.KeepType {
			return o.Net
		}
	}
	return ""
}

func keepIdle(f Frame, kind string) string {
	k, ok := view.KeepTypeOf(kind)
	if !ok {
		return ""
	}
	for _, o := range f.State.KeepShop {
		if o.KeepType == k.KeepType {
			return o.Idle
		}
	}
	return ""
}

const (
	reserveCost = "RESERVE"
	reserveTip  = "From the realm's reserve: the Feudal Age needs more room, and what you paid to advance raised this Keep. Place it; its upkeep is yours."
)

func KeepButton(b view.Buildable, f Frame) Button {
	cost := gatedPrice(b)
	if b.Reserve {
		cost = reserveCost
	}
	return Button{ID: "build:" + b.Kind, Label: KeepLabel(b, f), Icon: keepIcon(b.Kind, f), Cost: chapterPrice(cost, f),
		Tip: keepTip(b, f), Disabled: !b.Available, Why: whyUnless(b.Available, b.Reason), Selected: f.Placing == b.Kind}
}

const retireWindow = 3.0

type retireArm struct {
	id string
	at float64
}

func retirable(f Frame) (view.Structure, bool) {
	s := f.Selection.Structure
	ok := f.Selection.Kind == "structure" && s.Kind == view.StructKeep && s.Variant != "tower" &&
		s.State != view.StateConstruction && f.Revealed("keeps")
	return s, ok
}

func (h *HUD) armed(id string) bool {
	return h.retire.id == id && h.Now >= h.retire.at && h.Now-h.retire.at <= retireWindow
}

func (h *HUD) retireButton(f Frame, r ui.Rect) (Button, bool) {
	s, ok := retirable(f)
	if !ok {
		return Button{}, false
	}
	b := Button{ID: "keep:retire", Rect: r, Label: "RETIRE",
		Tip: fmt.Sprintf("Retire %s. Drain it first: kubectl drain %s", s.Label, s.Object.Name)}
	if h.armed(s.ID) {
		b.Label, b.Selected, b.Tip = "SURE?", true, "Click again to retire "+s.Label
	}
	return b, true
}

func (h *HUD) pressKeep(rest string, f Frame) []Intent {
	s, ok := retirable(f)
	if rest != "retire" || !ok {
		return nil
	}
	if !h.armed(s.ID) {
		h.retire = retireArm{id: s.ID, at: h.Now}
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	h.retire = retireArm{}
	return []Intent{act(view.Action{Action: "retireKeep", ID: s.ID})}
}
