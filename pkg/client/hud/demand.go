package hud

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const DemandRowID = FreightTip + "demand"

func unserved(f Frame) []model.Unserved {
	if f.State.Tutorial {
		return nil
	}
	return model.UnservedKinds(f.State.Freight)
}

func StrandedCargo(f Frame) (model.Unserved, bool) {
	for _, u := range unserved(f) {
		if u.Stranded() {
			return u, true
		}
	}
	return model.Unserved{}, false
}

func DemandShown(f Frame) bool {
	_, ok := StrandedCargo(f)
	return ok
}

func DemandSegments(u model.Unserved, width int) []Segment {
	name, rate, gold := strings.ToUpper(u.Name), Rate(u.Rate)+"/S", goldAmount(u.Gold)
	named := ui.Name(name)
	seg := func(story, storyGold, k8s, k8sGold string) []Segment {
		return []Segment{{ui.Bridge(story, k8s), ui.Palette.RedText}, {ui.Bridge(storyGold, k8sGold), ui.Palette.RedText}}
	}
	if u.Queued {
		return FitSegments([][]Segment{
			seg(name+" "+rate+" BACKING UP", gold+" GOLD/S AT RISK", named+" "+rate+" QUEUEING", "$"+gold+"/S AT RISK"),
			seg(name+" "+rate+" BACKING UP", ui.IconGold+gold+"/S AT RISK", named+" "+rate+" QUEUEING", "$"+gold+"/S AT RISK"),
			seg(name+" "+rate, ui.IconGold+gold+"/S AT RISK", named+" "+rate, "$"+gold+"/S AT RISK"),
		}, width)
	}
	return FitSegments([][]Segment{
		seg(name+" "+rate+" SPOILING", gold+" GOLD/S ON THE FLOOR", named+" "+rate+" DROPPED", "$"+gold+"/S LOST"),
		seg(name+" "+rate+" SPOILING", "-"+ui.IconGold+gold+"/S", named+" "+rate+" DROPPED", "-$"+gold+"/S"),
		seg(name+" "+rate, "-"+ui.IconGold+gold+"/S", named+" "+rate, "-$"+gold+"/S"),
	}, width)
}

func DemandText(u model.Unserved) string {
	rate, gold := Rate(u.Rate), goldAmount(u.Gold)
	if u.Queued {
		return ui.Bridge(fmt.Sprintf("%s: %s/S BACKING UP, %s GOLD/S AT RISK", strings.ToUpper(u.Name), rate, gold),
			fmt.Sprintf("%s: %s/S QUEUEING, $%s/S AT RISK", ui.Name(strings.ToUpper(u.Name)), rate, gold))
	}
	return ui.Bridge(fmt.Sprintf("%s: %s/S SPOILING, %s GOLD/S ON THE FLOOR", strings.ToUpper(u.Name), rate, gold),
		fmt.Sprintf("%s: %s/S DROPPED, $%s/S LOST", ui.Name(strings.ToUpper(u.Name)), rate, gold))
}

func DemandTip(u model.Unserved) string {
	rate, gold := Rate(u.Rate), goldAmount(u.Gold)
	if u.Queued {
		return tipLines(ui.Bridge(
			fmt.Sprintf("%s arrive that no crew carries: they are backing up, %s a second more on the quay, %s gold a second at risk; boxes that wait too long spoil.", upperFirst(u.Name), rate, gold),
			fmt.Sprintf("%s arrive that no Deployment serves: they are queueing, %s a second more, $%s a second of revenue at risk.", upperFirst(ui.Name(u.Name)), rate, gold)),
			ui.Bridge(fmt.Sprintf("The %s carry %s: open the %s depot and raise the %s before they spoil.", u.Crew, u.Name, u.Depot, u.Crew),
				fmt.Sprintf("The %s serve them: create the %s Service and the %s Deployment before they are dropped.", u.Crew, u.Depot, u.Crew)),
			ui.Bridge("Click to see the pile on the quay.", "Click to see the queue."))
	}
	if u.Cause == model.UnservedNoCrew {
		return tipLines(ui.Bridge(
			fmt.Sprintf("The %s depot stands empty: %s a second of %s spoil at its door, %s gold a second on the floor.", u.Depot, rate, u.Name, gold),
			fmt.Sprintf("The %s Service has no ready Pods: %s %s a second are dropped, $%s a second of revenue lost.", u.Depot, rate, ui.Name(u.Name), gold)),
			ui.Bridge(fmt.Sprintf("Raise the %s, and this loss turns into income.", u.Crew),
				fmt.Sprintf("Create the %s Deployment, and this loss turns into revenue.", u.Crew)),
			ui.Bridge("Click to see the pile on the quay.", "Click to see the queue."))
	}
	return tipLines(ui.Bridge(
		fmt.Sprintf("%s arrive that no crew carries: %s a second spoil on the quay, %s gold a second on the floor.", upperFirst(u.Name), rate, gold),
		fmt.Sprintf("%s arrive that no Deployment serves: %s a second are dropped, $%s a second of revenue lost.", upperFirst(ui.Name(u.Name)), rate, gold)),
		ui.Bridge(fmt.Sprintf("The %s carry %s: open the %s depot and raise the %s, and this loss turns into income.", u.Crew, u.Name, u.Depot, u.Crew),
			fmt.Sprintf("The %s serve them: create the %s Service and the %s Deployment, and this loss turns into revenue.", u.Crew, u.Depot, u.Crew)),
		ui.Bridge("Click to see the pile on the quay.", "Click to see the queue."))
}

func (h *HUD) demandButton(f Frame, row ui.Rect) (Button, bool) {
	u, ok := StrandedCargo(f)
	if !ok {
		return Button{}, false
	}
	return Button{ID: DemandRowID, Rect: row, Tip: DemandTip(u)}, true
}

func (h *HUD) goalButton(f Frame, row ui.Rect) Button {
	if b, ok := h.demandButton(f, row); ok {
		return b
	}
	return Button{ID: FreightTip + "goal", Rect: row, Tip: TierTip(f.State.Freight, PortsRowShown(f))}
}

func (h *HUD) drawDemandRow(dst *ebiten.Image, f Frame, x, y, width int) bool {
	u, ok := StrandedCargo(f)
	if ok {
		h.drawSegments(dst, DemandSegments(u, width), x, y, width)
	}
	return ok
}

func (h *HUD) pressDemand(f Frame) []Intent {
	u, ok := StrandedCargo(f)
	if !ok {
		return nil
	}
	at, ok := world.YardFocus(f.Site, u.Kind)
	if !ok {
		return nil
	}
	return h.focus([2]float64{at.X, at.Y})
}

func unservedChip(c KindChip, u model.Unserved, ok bool) KindChip {
	if !ok {
		return c
	}
	next := c
	if u.Queued {
		next.Tip = c.Tip + "\n" + ui.Bridge(fmt.Sprintf("Backing up: %s a second more arrive than are carried; %s gold a second is at risk, and boxes that wait too long spoil.", Rate(u.Rate), goldAmount(u.Gold)),
			fmt.Sprintf("Queueing: %s a second more arrive than are served; $%s a second of revenue is at risk.", Rate(u.Rate), goldAmount(u.Gold)))
		return next
	}
	next.Lost = u.Gold
	next.Tip = c.Tip + "\n" + ui.Bridge(fmt.Sprintf("Unserved: %s a second spoil, %s gold a second on the floor.", Rate(u.Rate), goldAmount(u.Gold)),
		fmt.Sprintf("Unserved: %s a second dropped, $%s a second of revenue lost.", Rate(u.Rate), goldAmount(u.Gold)))
	return next
}

func KindChipLabel(c KindChip) string {
	if c.Lost > 0 {
		amount := goldAmount(c.Lost)
		return ui.Bridge("-"+amount, "-$"+amount)
	}
	return fmt.Sprint(c.Waiting)
}

func withUnserved(c Card, f Frame) Card {
	u, ok := model.WorstUnserved(f.State.Freight)
	if !ok || f.State.Tutorial {
		return c
	}
	return withLine(c, Line{Text: DemandText(u), Clr: ui.Palette.RedText})
}

func withPortUnserved(c Card, f Frame, p view.PortView) Card {
	for _, u := range unserved(f) {
		if slices.Contains(p.Kinds, u.Kind) {
			return withLine(c, Line{Text: DemandText(u), Clr: ui.Palette.RedText})
		}
	}
	return c
}

func nodeKeepLabel(f Frame, node string) string {
	for _, s := range f.Site.Structures {
		if s.Kind == view.StructKeep && s.Object.Name == node && s.Label != "" {
			return s.Label
		}
	}
	return node
}

func withStranded(c Card, f Frame) Card {
	worst, found := view.KeepView{}, false
	for _, k := range f.State.Keeps {
		if k.StrandedGold > 0 && (!found || k.StrandedGold > worst.StrandedGold) {
			worst, found = k, true
		}
	}
	if !found {
		return c
	}
	text := ui.Bridge(fmt.Sprintf("DISPATCHES WAITING AT %s: NO WARDEN THERE, %s GOLD/S", strings.ToUpper(nodeKeepLabel(f, worst.Name)), Compact(worst.StrandedGold)),
		fmt.Sprintf("courier Pods on %s have no local post endpoint: -$%s/s", worst.Name, Compact(worst.StrandedGold)))
	return withLine(c, Line{Text: text, Clr: ui.Palette.RedText})
}

func withTreasure(c Card, f Frame) Card {
	if len(f.State.Treasure) == 0 {
		return c
	}
	t := f.State.Treasure[0]
	text := ui.Bridge(upperFirst(t.Boxes)+" wait at sea until their depot stands: their treasure waits", upperFirst(ui.Name(t.Boxes))+" wait: no Service and no Deployment serve them yet")
	return withLine(c, Line{Text: text, Clr: ui.Palette.GoldBright})
}
