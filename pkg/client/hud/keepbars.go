package hud

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	mebi        = 1 << 20
	reserveWord = "Citadel holds %s, %s"
)

var reserveFill = ui.Palette.Purple

func KeepSupplyLines(s view.Structure, f Frame) []Line {
	m, ok := f.Metrics.Nodes[s.Object.Name]
	if !ok || s.State == view.StateConstruction || m.CPUAllocatableMilli <= 0 {
		return nil
	}
	sys := f.Nodes[s.Object.Name]
	cpu := supplyBar(ui.Bridge("FOOD", "CPU"), m.CPUMilli, float64(m.CPUAllocatableMilli), float64(sys.SystemCPUMilli))
	cpu.lesson = laborLesson
	cpu.Suffix = ui.Bridge(foodCount(m.CPUMilli)+"/"+foodCount(float64(m.CPUAllocatableMilli)), Amount(m.CPUMilli/1000)+"/"+Amount(float64(m.CPUAllocatableMilli)/1000))
	mem := supplyBar(ui.Bridge("HOUSING", "MEMORY"), m.MemoryBytes, float64(m.MemoryAllocatableBytes), float64(sys.SystemMemBytes))
	mem.lesson = laborLesson
	mem.Suffix = ui.Bridge(housingCount(m.MemoryBytes)+"/"+housingCount(float64(m.MemoryAllocatableBytes)), Amount(m.MemoryBytes/gib)+"/"+Amount(float64(m.MemoryAllocatableBytes)/gib)+" GiB")
	out := alignLabels([]Line{cpu, mem})
	if sys.SystemCPUMilli > 0 || sys.SystemMemBytes > 0 {
		legend := fmt.Sprintf(reserveWord, ui.Bridge(foodTag(float64(sys.SystemCPUMilli)), fmt.Sprintf("%dm", sys.SystemCPUMilli)), ui.Bridge(housingTag(float64(sys.SystemMemBytes)), MemQuantity(sys.SystemMemBytes)))
		out = append(out, Line{Text: legend, Clr: reserveFill, lesson: laborLesson})
	}
	return out
}

func alignLabels(lines []Line) []Line {
	var font ui.Font
	widest := 0
	for _, l := range lines {
		widest = max(widest, font.Width(l.Text, 1))
	}
	out := make([]Line, len(lines))
	for i, l := range lines {
		l.Text += strings.Repeat(" ", (widest-font.Width(l.Text, 1))/ui.Advance)
		out[i] = l
	}
	return out
}

func supplyBar(label string, used, total, reserved float64) Line {
	share := ratio(used, total)
	return Line{Text: label, Clr: ui.Palette.Text, Bar: true, Value: share, Fill: pledgeColor(share),
		reserved: ratio(reserved, total), reserveFill: reserveFill}
}

func ratio(v, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return v / total
}

func pledgeColor(share float64) color.RGBA {
	switch {
	case share >= 1:
		return ui.Palette.Red
	case share >= supplyWarn:
		return ui.Palette.GoldBright
	}
	return ui.Palette.Green
}

func (h *HUD) drawSplitBar(dst *ebiten.Image, r ui.Rect, l Line) {
	h.drawMiniBar(dst, r, l.Value, l.Fill)
	if seg := reservedRect(r, l.reserved); seg.W > 0 {
		ui.Fill(dst, seg, l.reserveFill)
	}
}

func reservedRect(r ui.Rect, share float64) ui.Rect {
	inner := r.Inset(1)
	inner.W = int(float64(inner.W) * min(max(share, 0), 1))
	return inner
}

func Amount(v float64) string {
	return strings.TrimSuffix(fmt.Sprintf("%.2f", v), "0")
}

func MemQuantity(bytes int64) string {
	if bytes < gib {
		return fmt.Sprintf("%dMi", bytes/mebi)
	}
	g := float64(bytes) / gib
	if g == math.Trunc(g) {
		return fmt.Sprintf("%.0fGi", g)
	}
	return fmt.Sprintf("%.1fGi", g)
}

func systemShare(f Frame) (int64, int64) {
	var cpu, mem int64
	for _, n := range f.Nodes {
		cpu, mem = cpu+n.SystemCPUMilli, mem+n.SystemMemBytes
	}
	return cpu, mem
}

func milli(cores float64) int64 {
	return int64(math.Round(cores * 1000))
}
