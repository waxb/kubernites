package hud

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

type useFacts struct {
	CPU    float64
	Memory float64
	ReqCPU int64
	ReqMem int64
	Pods   int
}

func crewUse(c model.CrewChain, f Frame) useFacts {
	u := useFacts{}
	for _, p := range crewPods(c, f) {
		m, ok := f.Metrics.Pods[p.Namespace+"/"+p.Name]
		if p.State == model.HearthPending || !ok {
			continue
		}
		u.Pods++
		u.CPU += m.CPUMilli
		u.Memory += m.MemoryBytes
		for _, r := range p.Requests {
			u.ReqCPU += r.CPUMilli
			u.ReqMem += r.MemoryBytes
		}
	}
	return u
}

func cpuText(milli int64) string {
	return ui.Bridge(shout(foodOf(float64(milli))), fmt.Sprintf("%dm CPU", milli))
}

func memText(bytes int64) string {
	return ui.Bridge(shout(housingOf(float64(bytes))), MemQuantity(bytes)+" MEMORY")
}

func landFree(v BoardView) (int64, int64) {
	if v.Facts.Booked {
		return max(v.Facts.Free.CPU, 0), max(v.Facts.Free.Memory, 0)
	}
	cpu, mem := v.Facts.LandTotals()
	return max(cpu[1]-cpu[0], 0), max(mem[1]-mem[0], 0)
}

func UsePieces(v BoardView) []deltaPiece {
	if !v.Facts.Bounded() && v.Facts.Use.Pods == 0 {
		return nil
	}
	if d := BoardDeltaOf(v); d.From != d.To {
		return previewUsePieces(v, d)
	}
	free, freeMem := landFree(v)
	out := workerPieces(v.Facts)
	if v.Chain.Uncounted && noHutYet(v.Facts) {
		out = []deltaPiece{{uncountedText(), ui.Palette.TextDim}}
	}
	return append(out, deltaPiece{ui.Bridge("KEEPS FREE ", "NODES FREE ") + cpuText(free) + " " + memText(freeMem), ui.Palette.TextDim})
}

func noHutYet(k CrewFacts) bool {
	return k.Use.Pods == 0 && k.Per.CPU <= 0 && k.Per.Memory <= 0
}

func workerPieces(k CrewFacts) []deltaPiece {
	if noHutYet(k) {
		return []deltaPiece{{ui.Bridge("NO HUT RUNS YET", "NO REPLICA RUNS YET"), ui.Palette.TextDim}}
	}
	out := []deltaPiece{{ui.Bridge("EACH WORKER", "PER CONTAINER"), ui.Palette.Cream},
		{foodTrio(k), overTone(float64(k.Peak.CPU), k.Per.CPU, ui.Palette.Blue)},
		{roomTrio(k), overTone(float64(k.Peak.Memory), k.Per.Memory, ui.Palette.Purple)}}
	if k.Squire.CPU > 0 || k.Squire.Memory > 0 {
		out = append(out, deltaPiece{squireShare(k.Squire), ui.Palette.TextDim})
	}
	return out
}

func squireShare(p view.ResourcePair) string {
	return ui.Bridge("SQUIRES ADD "+ui.IconFood+foodCount(float64(p.CPU))+" "+ui.IconHousing+housingCount(float64(p.Memory)),
		fmt.Sprintf("SIDECARS ADD %dm CPU %s", p.CPU, MemQuantity(p.Memory)))
}

func foodTrio(k CrewFacts) string {
	uses := notReadYet()
	if k.Peak.Known() {
		uses = ui.Bridge("USES "+foodCount(float64(k.Peak.CPU)), fmt.Sprintf("USES %dm", k.Peak.CPU))
	}
	pledge := k.pledge().CPU
	out := ui.Bridge(ui.IconFood+"PLEDGE "+foodCount(float64(pledge)), fmt.Sprintf("CPU REQ %dm", pledge)) + ", " + uses
	if k.Binding.CPU > 0 {
		out += ", " + ui.Bridge("BINDING "+foodCount(float64(k.Binding.CPU)), fmt.Sprintf("LIM %dm", k.Binding.CPU))
	}
	return out
}

func roomTrio(k CrewFacts) string {
	uses := notReadYet()
	if k.Peak.Known() {
		uses = ui.Bridge("USES "+housingCount(float64(k.Peak.Memory)), "USES "+MemQuantity(k.Peak.Memory))
	}
	pledge := k.pledge().Memory
	out := ui.Bridge(ui.IconHousing+"PLEDGE "+housingCount(float64(pledge)), "MEM REQ "+MemQuantity(pledge)) + ", " + uses
	if k.Binding.Memory > 0 {
		out += ", " + ui.Bridge("BINDING "+housingCount(float64(k.Binding.Memory)), "LIM "+MemQuantity(k.Binding.Memory))
	}
	return out
}

func previewUsePieces(v BoardView, d BoardDelta) []deltaPiece {
	free, freeMem := landFree(v)
	afterCPU, afterMem := max(free-d.CPU, 0), max(freeMem-d.Memory, 0)
	out := []deltaPiece{{ui.Bridge("KEEPS FREE ", "NODES FREE ") + cpuText(free) + " " + memText(freeMem), ui.Palette.TextDim},
		{"->", ui.Palette.TextDim}, {cpuText(afterCPU) + " " + memText(afterMem), ui.Palette.Cream}}
	step := d.To - d.From
	switch {
	case step > 0 && d.Waits == 0 && d.Hungry == 0:
		out = append(out, deltaPiece{ui.Bridge("ALL FIT", "ALL SCHEDULE"), ui.Palette.Green})
	case step > 0 && d.Waits == 0:
		out = append(out, deltaPiece{shortCount(v.Facts.Limit, d.Hungry), ui.Palette.GoldBright})
	case step > 0:
		out = append(out, deltaPiece{fmt.Sprintf(ui.Bridge("%d OF %d FIT", "%d OF %d SCHEDULE"), d.Earning, step), ui.Palette.RedText})
	}
	if charterCrossed(v.Facts.CharterAt, d.From, d.To) {
		out = append(out, deltaPiece{"A CHARTER FOLLOWS", ui.Palette.RedText})
	}
	return out
}

func overTone(use float64, req int64, calm color.RGBA) color.RGBA {
	if req > 0 && use > float64(req) {
		return ui.Palette.RedText
	}
	return calm
}

func (h *HUD) drawBoardUse(dst *ebiten.Image, v BoardView) {
	h.drawPieces(dst, UsePieces(v), v.box.X+cbPad, v.box.Y+cbUseY, v.box.X+v.box.W-cbPad)
}

func shortCount(limit string, n int) string {
	if limit == view.FlowMemory {
		return fmt.Sprintf(ui.Bridge("ALL PLACED, %d CROWD A KEEP", "ALL SCHEDULE, %d OVERCOMMIT MEMORY"), n)
	}
	return fmt.Sprintf(ui.Bridge("ALL PLACED, %d GO HUNGRY", "ALL SCHEDULE, %d OVERCOMMIT CPU"), n)
}

func notReadYet() string {
	return ui.Bridge("USE NOT READ YET", "USE NOT MEASURED YET")
}
