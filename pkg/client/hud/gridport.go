package hud

import (
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	BerthCell   = 3
	UnberthCell = 5
)

func portSlots(f Frame) []gridSlot {
	if !BerthsOffered(f) {
		return nil
	}
	if p, ok := SelectedPort(f); ok {
		return portCells(p)
	}
	return []gridSlot{{cell: BerthCell, button: berthCellOf(OpenBerth(f), f.State.Berths.NextCost)}, {cell: UnberthCell, button: unberthCellOf(CloseBerth(f))}}
}

func berthCellOf(b Button, cost float64) Button {
	next := b
	next.Label, next.Icon, next.Cost = ui.Bridge("OPEN BERTH", "RAMP UP"), "icon/scale-up", priceTag(cost)
	return next
}

func unberthCellOf(b Button) Button {
	next := b
	next.Label, next.Icon = ui.Bridge("CLOSE BERTH", "RAMP DOWN"), "icon/scale-down"
	return next
}

func portCells(p view.PortView) []gridSlot {
	if !p.Open {
		open := Button{ID: PortButtonID(PortVerbOpen, p.ID), Label: ui.Bridge("OPEN PORT", "CREATE GATEWAY"), Icon: "icon/key-gate",
			Cost: priceTag(p.OpenCost), Tip: PortTip(p.Effects.Open, portOpenK8s())}
		return []gridSlot{{cell: CentreCell, button: disabledBy(open, p.Gates.Open)}}
	}
	berth := Button{ID: PortButtonID(PortVerbBerth, p.ID), Tip: withEffect("Draws more ships to this port.", p.Effects.Berth)}
	shut := Button{ID: PortButtonID(PortVerbClose, p.ID), Tip: closeBerthTip}
	out := []gridSlot{
		{cell: BerthCell, button: disabledBy(berthCellOf(berth, p.BerthCost), p.Gates.Berth)},
		{cell: UnberthCell, button: disabledBy(unberthCellOf(shut), p.Gates.Close)},
	}
	if p.Next == nil {
		return out
	}
	up := Button{ID: PortButtonID(PortVerbRaise, p.ID), Label: upgradeLabel(p), Icon: "icon/rank-high", Cost: priceTag(p.UpgradeCost),
		Tip: PortTip(p.Effects.Upgrade, p.K8s)}
	return append(out, gridSlot{cell: CentreCell, button: disabledBy(up, p.Gates.Upgrade)})
}

func portOpenK8s() string {
	return ui.Bridge("A port is a Gateway in Kubernetes: it brings its own load balancer, so it has its own price.",
		"Creating a Gateway (gateway.networking.k8s.io/v1) provisions its own load balancer.")
}

func PortTip(effect, k8s string) string {
	if ui.StoryNaming() {
		k8s = ""
	}
	switch {
	case effect == "":
		return k8s
	case k8s == "":
		return effect
	}
	return effect + "\n" + k8s
}

func portGoal(goal string, f Frame) string {
	if goal != BerthButton && goal != SurfaceBerths || !HavenSelected(f) {
		return goal
	}
	if p, ok := SelectedPort(f); ok && p.Open {
		return PortButtonID(PortVerbBerth, p.ID)
	}
	if _, ok := SelectedPort(f); ok {
		return GateHaven
	}
	return goal
}

func priceTag(cost float64) string {
	if cost <= 0 {
		return ""
	}
	return Compact(cost)
}
