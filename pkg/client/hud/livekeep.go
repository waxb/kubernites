package hud

import (
	"fmt"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	labelInstanceType = "node.kubernetes.io/instance-type"
	labelZone         = "topology.kubernetes.io/zone"
)

func liveCores(milli float64) string { return LiveAmount(milli / 1000) }

func liveGiB(bytes float64) string { return LiveAmount(bytes / gib) }

func LiveKeepLines(s view.Structure, f Frame) []Line {
	m, ok := f.Metrics.Nodes[s.Object.Name]
	if !ok || m.CPUAllocatableMilli <= 0 {
		return liveKeepLabels(s, f)
	}
	cpu := supplyBar(ui.Bridge("FOOD", "CPU"), float64(m.CPURequestedMilli), float64(m.CPUAllocatableMilli), float64(m.ControlPlaneCPUMilli))
	cpu.Suffix = ui.Bridge(foodCount(float64(m.CPURequestedMilli))+"/"+foodCount(float64(m.CPUAllocatableMilli)),
		liveCores(float64(m.CPURequestedMilli))+"/"+liveCores(float64(m.CPUAllocatableMilli)))
	bars := []Line{cpu}
	if m.MemoryAllocatableBytes > 0 {
		mem := supplyBar(ui.Bridge("HOUSING", "MEMORY"), float64(m.MemoryRequestedBytes), float64(m.MemoryAllocatableBytes), float64(m.ControlPlaneMemoryBytes))
		mem.Suffix = ui.Bridge(housingCount(float64(m.MemoryRequestedBytes))+"/"+housingCount(float64(m.MemoryAllocatableBytes)),
			liveGiB(float64(m.MemoryRequestedBytes))+"/"+liveGiB(float64(m.MemoryAllocatableBytes))+"Gi")
		bars = append(bars, mem)
	}
	if m.PodsAllocatable > 0 {
		pods := supplyBar(ui.Bridge("HUTS", "PODS"), float64(m.Pods), float64(m.PodsAllocatable), 0)
		pods.Suffix = fmt.Sprintf("%d/%d", m.Pods, m.PodsAllocatable)
		bars = append(bars, pods)
	}
	out := append(alignLabels(bars), Line{Text: ui.Bridge("pledged of available", "requests of allocatable"), Clr: ui.Palette.TextDim})
	if m.Measured {
		out = append(out, Line{Text: ui.Bridge(fmt.Sprintf("eaten: %s, %s", foodTag(m.CPUMilli), housingTag(m.MemoryBytes)),
			fmt.Sprintf("in use: %.0fm CPU, %s", m.CPUMilli, MemQuantity(int64(m.MemoryBytes)))), Clr: ui.Palette.Text})
	}
	if m.DiskPressure {
		out = append(out, Line{Text: ui.Bridge("CELLAR WARD: ground runs short", "DiskPressure: disk runs short"), Clr: ui.Palette.RedText})
	}
	if m.ControlPlaneCPUMilli > 0 || m.ControlPlaneMemoryBytes > 0 {
		legend := fmt.Sprintf(reserveWord, ui.Bridge(foodTag(float64(m.ControlPlaneCPUMilli)), fmt.Sprintf("%dm", m.ControlPlaneCPUMilli)),
			ui.Bridge(housingTag(float64(m.ControlPlaneMemoryBytes)), MemQuantity(m.ControlPlaneMemoryBytes)))
		out = append(out, Line{Text: legend, Clr: reserveFill})
	}
	return append(out, liveKeepLabels(s, f)...)
}

func liveKeepLabels(s view.Structure, f Frame) []Line {
	labels := f.Nodes[s.Object.Name].Labels
	kind, zone := labels[labelInstanceType], labels[labelZone]
	var text string
	switch {
	case kind != "" && zone != "":
		text = ui.Bridge(kind+" on isle "+zone, kind+" in zone "+zone)
	case kind != "":
		text = ui.Bridge("Keep size "+kind, "instance type "+kind)
	case zone != "":
		text = ui.Bridge("on isle "+zone, "zone "+zone)
	default:
		return nil
	}
	return []Line{{Text: text, Clr: ui.Palette.TextDim, Optional: true}}
}
