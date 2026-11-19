package hud

import (
	"fmt"
	"image/color"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	supplyWarn  = 0.8
	supplyCalm  = 0.5
	gib         = 1 << 30
	roomsPerGiB = 1024
)

const (
	supplyTip = "Pledged / available. The Quartermaster places a hut only where its Pledge fits."
)

type Supply struct {
	Used  float64
	Total float64
}

func (s Supply) Ratio() float64 {
	if s.Total <= 0 {
		return 0
	}
	return s.Used / s.Total
}

func SupplyOf(nodes map[string]view.NodeMetrics) (Supply, Supply) {
	var cpuUsed, cpuTotal, memUsed, memTotal int64
	for _, n := range nodes {
		cpuUsed += n.CPURequestedMilli
		cpuTotal += n.CPUAllocatableMilli
		memUsed += n.MemoryRequestedBytes
		memTotal += n.MemoryAllocatableBytes
	}
	hands := Supply{Used: float64(cpuUsed) / 1000, Total: float64(cpuTotal) / 1000}
	lore := Supply{Used: float64(memUsed) / gib, Total: float64(memTotal) / gib}
	return hands, lore
}

var supplyFull = color.RGBA{196, 44, 36, 255}

func SupplyColor(s Supply) color.RGBA {
	t := min(max((s.Ratio()-supplyCalm)/(1-supplyCalm), 0), 1)
	c := ui.Palette.Cream
	return color.RGBA{R: lerpByte(c.R, supplyFull.R, t), G: lerpByte(c.G, supplyFull.G, t), B: lerpByte(c.B, supplyFull.B, t), A: 255}
}

func lerpByte(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t + .5)
}

func SupplyText(label string, s Supply, unit string) string {
	text := fmt.Sprintf("%s %s/%s%s", label, SupplyAmount(s.Used), SupplyAmount(s.Total), unit)
	if s.Ratio() >= 1 {
		text += "!"
	}
	return text
}

func Rations(cores Supply) Supply {
	return Supply{Used: cores.Used * view.RationsPerCPU, Total: cores.Total * view.RationsPerCPU}
}

func Rooms(gibs Supply) Supply {
	return Supply{Used: gibs.Used * roomsPerGiB, Total: gibs.Total * roomsPerGiB}
}

func SupplyCount(s Supply) string {
	text := FitAmount(s.Used, supplyChars) + "/" + FitAmount(s.Total, supplyChars)
	if s.Ratio() >= 1 {
		text += "!"
	}
	return text
}

func supplyCountSlot() string { return digits(supplyChars) + "/" + digits(supplyChars) + "!" }

func SupplySlot(label, unit string) string {
	return label + " " + digits(supplyChars) + "/" + digits(supplyChars) + unit + "!"
}

const GateSettle = 5.0

type gateWatch struct {
	seen     bool
	open     bool
	reopened bool
	waiting  bool
	since    float64
}

func (h *HUD) watch(f Frame) {
	h.watchRanks(f)
	open, now := HandsNeeded(f.State.Freight), f.State.RealmTime
	g := h.gate
	switch {
	case !g.seen:
		g = gateWatch{seen: true, open: open}
	case g.open && !open:
		g = gateWatch{seen: true}
		h.Announce(AnnounceGate, CrewTakesToast, false)
	case !g.open && !open:
		g.waiting = false
	case !g.open && !g.waiting:
		g.waiting, g.since = true, now
	case !g.open && now-g.since >= GateSettle:
		g = gateWatch{seen: true, open: true, reopened: true}
		h.Announce(AnnounceGate, HandsNeededToast, true)
	}
	h.gate = g
}

func (h *HUD) GateReopened() bool { return h.gate.reopened }
