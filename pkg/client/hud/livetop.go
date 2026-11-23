package hud

import (
	"fmt"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	liveCPUID      = "info:live:cpu"
	liveMemoryID   = "info:live:memory"
	livePodsID     = "info:live:pods"
	liveRequestTip = "Requested of allocatable, summed over every node, as the scheduler counts requests: the Pods bound to a node that are neither Succeeded nor Failed."
)

func LiveTotals(nodes map[string]view.NodeMetrics) (Supply, Supply, Supply) {
	hands, lore := SupplyOf(nodes)
	var pods, places int
	for _, n := range nodes {
		pods += n.Pods
		places += n.PodsAllocatable
	}
	return hands, lore, Supply{Used: float64(pods), Total: float64(places)}
}

func liveUsage(nodes map[string]view.NodeMetrics) (float64, float64, bool) {
	var cpu, mem float64
	measured := false
	for _, n := range nodes {
		if n.Measured {
			cpu, mem, measured = cpu+n.CPUMilli, mem+n.MemoryBytes, true
		}
	}
	return cpu, mem, measured
}

func LiveAmount(v float64) string {
	for d := 2; d >= 0; d-- {
		if s := trimFloat(v, d); len(s) <= supplyChars {
			return s
		}
	}
	return FitAmount(v, supplyChars)
}

func liveCount(s Supply, unit string) string {
	text := LiveAmount(s.Used) + "/" + LiveAmount(s.Total) + unit
	if s.Ratio() >= 1 {
		text += "!"
	}
	return text
}

func liveSupplyStat(id, icon, word string, s Supply, story, unit, tip string) Stat {
	st := Stat{Id: id, clr: SupplyColor(s), Tip: tip}
	switch {
	case ui.Naming():
		st.Text, st.slot = word+" "+liveCount(s, unit), word+" "+supplyCountSlot()[:len(supplyCountSlot())-1]+unit+"!"
	case icon != "":
		st.icon, st.Text, st.slot = icon, story, supplyCountSlot()
	default:
		st.Text, st.slot = word+" "+story, word+" "+supplyCountSlot()
	}
	return st
}

func LiveNumbers(f Frame) []Stat {
	hands, lore, pods := LiveTotals(f.Metrics.Nodes)
	cpuUsed, memUsed, measured := liveUsage(f.Metrics.Nodes)
	out := []Stat{}
	if hands.Total > 0 {
		story := fmt.Sprintf("FOOD %s of %s pledged.", foodCount(hands.Used*view.RationsPerCPU), foodOfCores(hands.Total))
		kube := fmt.Sprintf("CPU requests %dm of %dm allocatable.", milli(hands.Used), milli(hands.Total))
		if measured {
			story += " The Augur sees " + foodOf(cpuUsed) + " eaten."
			kube += fmt.Sprintf(" metrics-server: %.0fm in use.", cpuUsed)
		}
		out = append(out, liveSupplyStat(liveCPUID, "food", "CPU", hands, SupplyCount(Rations(hands)), "", tipLines(ui.Bridge(story, kube), liveCountTip())))
	}
	if lore.Total > 0 {
		story := fmt.Sprintf("HOUSING %s of %s pledged.", housingCount(lore.Used*gib), housingOfGiB(lore.Total))
		kube := fmt.Sprintf("Memory requests %s of %s allocatable.", MemQuantity(int64(lore.Used*gib)), MemQuantity(int64(lore.Total*gib)))
		if measured {
			story += " The Augur sees " + housingOf(memUsed) + " in use."
			kube += " metrics-server: " + MemQuantity(int64(memUsed)) + " in use."
		}
		out = append(out, liveSupplyStat(liveMemoryID, "housing", "MEMORY", lore, SupplyCount(Rooms(lore)), "Gi", tipLines(ui.Bridge(story, kube), liveCountTip())))
	}
	if pods.Total > 0 {
		tip := ui.Bridge(fmt.Sprintf("%s on the Keeps, of %d places the Keeps offer.", plural(int(pods.Used), "hut", "huts"), int(pods.Total)),
			fmt.Sprintf("%s on the nodes, of %d the nodes allow (allocatable pods).", plural(int(pods.Used), "Pod", "Pods"), int(pods.Total)))
		out = append(out, liveSupplyStat(livePodsID, "", ui.Bridge("HUTS", "PODS"), pods, liveCount(pods, ""), "", tip))
	}
	return out
}

func liveCountTip() string {
	return ui.Bridge("Pledged of available, summed over every Keep, as the Quartermaster counts Pledges: every hut on a Keep but those whose work is done or failed.", liveRequestTip)
}

func (h *HUD) liveStats(f Frame) []Stat {
	name := liveFont.Clip(h.live.Context, 40)
	out := []Stat{
		{Text: h.live.badge(), slot: h.live.badge(), clr: h.live.badgeColour(), Tip: h.live.levelTip(), Id: liveBadgeID},
		{Text: name, slot: name, clr: ui.Palette.Text, Tip: h.live.Server, Id: liveBadgeID},
	}
	buttons := h.TopButtons(f)
	right := buttons[len(buttons)-1].Rect.X - clockGap - h.Font.Width(h.liveClock(), 1) - clockGap
	end := h.layoutStats(out) - statGap
	for _, st := range LiveNumbers(f) {
		w := statGap + h.StatWidth(st)
		if end+w > right {
			break
		}
		out, end = append(out, st), end+w
	}
	return out
}
