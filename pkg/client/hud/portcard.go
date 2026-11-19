package hud

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	portPrefix    = "harbor:"
	PortVerbOpen  = "open"
	PortVerbRaise = "upgrade"
	PortVerbBerth = "berth"
	PortVerbClose = "close"
)

var portActions = map[string]string{PortVerbOpen: view.ActOpenPort, PortVerbRaise: view.ActUpgradePort, PortVerbBerth: view.ActOpenBerth, PortVerbClose: view.ActCloseBerth}

func PortButtonID(verb, id string) string {
	return portPrefix + verb + ":" + id
}

func PortTarget(target string) (string, string, bool) {
	rest, ok := strings.CutPrefix(target, portPrefix)
	if !ok {
		return "", "", false
	}
	verb, id, ok := strings.Cut(rest, ":")
	_, known := portActions[verb]
	return verb, id, ok && known && id != ""
}

func PortAction(target string) (view.Action, bool) {
	verb, id, ok := PortTarget(target)
	if !ok {
		return view.Action{}, false
	}
	return view.Action{Action: portActions[verb], ID: id}, true
}

func PressPort(_ *HUD, rest string, _ Frame) []Intent {
	if a, ok := PortAction(portPrefix + rest); ok {
		return []Intent{act(a)}
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func SelectedPort(f Frame) (view.PortView, bool) {
	id := world.GateHavenID
	if HavenSelected(f) && f.Selection.ID != "" {
		id = f.Selection.ID
	}
	for _, p := range f.State.Ports {
		if p.ID == id {
			return p, true
		}
	}
	return view.PortView{}, false
}

func PortLine(p view.PortView) string {
	if !p.Open {
		return ui.Bridge(fmt.Sprintf("A bare shore: open it for %s gold", Compact(p.OpenCost)), fmt.Sprintf("No Gateway yet: create it for $%s", Compact(p.OpenCost)))
	}
	return ui.Bridge(fmt.Sprintf("Level %d: %s", p.Level, p.Ships),
		fmt.Sprintf("%s listeners", strings.Join(p.Listeners, ", ")))
}

func portBar(p view.PortView, gold float64) Line {
	switch {
	case !p.Open:
		return Line{Text: ui.Bridge("OPEN THE PORT", "CREATE GATEWAY"), Clr: ui.Palette.Text, Bar: true, Value: share01(gold, p.OpenCost), Fill: ui.Palette.Green, Suffix: Compact(p.OpenCost)}
	case p.Next == nil:
		return Line{Text: "The top of the harbor ladder", Clr: ui.Palette.Cream}
	}
	return Line{Text: "NEXT: " + strings.ToUpper(p.Next.Ships), Clr: ui.Palette.Text, Bar: true, Value: share01(gold, p.UpgradeCost), Fill: ui.Palette.Green,
		Suffix: Compact(p.UpgradeCost)}
}

func share01(have, need float64) float64 {
	if need <= 0 {
		return 1
	}
	return math.Min(1, math.Max(0, have/need))
}

func PortBerthsLine(p view.PortView) string {
	ships := strconv.FormatFloat(math.Round(math.Pow(1.25, float64(p.Berths))*100)/100, 'f', -1, 64)
	return ui.Bridge(fmt.Sprintf("%d/%d berths: ships here x%s", p.Berths, p.MaxBerths, ships), fmt.Sprintf("%d/%d ramps: traffic x%s", p.Berths, p.MaxBerths, ships))
}

func portNote(p view.PortView, b view.BerthView) Line {
	switch {
	case p.Berths >= p.MaxBerths:
		return Line{Text: ui.Bridge("Its quay is full: upgrade it for more berths.", "Its ramps are full: add a listener for more."), Clr: ui.Palette.TextDim}
	case b.Gate != "":
		return Line{Text: upperFirst(b.Gate) + ".", Clr: ui.Palette.RedText}
	}
	return Line{Text: "Next: " + p.Effects.Berth, Clr: ui.Palette.TextDim}
}

func PortCard(f Frame, p view.PortView) Card {
	c := Card{Portrait: SpritePortrait + gfx.ShipName(world.ShipClassOf(max(p.Level, 1)), 0, "se", 0), Title: HarborTitle(f), progress: -1}
	c.Lines = append(c.Lines, Line{Text: PortLine(p), Clr: ui.Palette.GoldBright}, portBar(p, f.State.Gold))
	c = withPortUnserved(c, f, p)
	if !BerthsOffered(f) || !p.Open {
		return c
	}
	c.Lines = append(c.Lines, Line{Text: PortBerthsLine(p), Clr: ui.Palette.Text})
	return withLine(c, portNote(p, f.State.Berths))
}

func disabledBy(b Button, gate string) Button {
	next := b
	if gate != "" {
		next.Disabled, next.Tip, next.Why = true, upperFirst(gate)+".", WhyOf(gate)
	}
	return next
}

func upgradeLabel(p view.PortView) string {
	listener := "ADMIT HUGE BURSTS"
	if n := len(p.Next.Listeners); n > len(p.Listeners) {
		listener = "ADD " + p.Next.Listeners[n-1] + " LISTENER"
	}
	return ui.Bridge("UPGRADE: "+strings.ToUpper(p.Next.Ships), listener)
}
