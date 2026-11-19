package hud

import (
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	resEffectRows = 3
	resRoomRows   = 2
)

func (h *HUD) focusField(v crewView) resField {
	if len(v.fields) == 0 {
		return resField{}
	}
	if h.Hovering {
		for _, rf := range v.fields {
			if strings.HasPrefix(h.Hover.ID, CrewPrefix+":res:"+rf.key+":") {
				return rf
			}
		}
	}
	for _, rf := range v.fields {
		if rf.get(h.Crew.Res) != rf.get(v.chain.Resources) {
			return rf
		}
	}
	return v.fields[len(v.fields)-1]
}

func pledgeRoom(keeps []view.KeepView, pick model.Resources) int {
	if pick.CPURequest <= 0 && pick.MemRequest <= 0 {
		return -1
	}
	room := 0
	for _, k := range keeps {
		if !k.Ready {
			continue
		}
		fits := -1
		if pick.CPURequest > 0 {
			fits = int(max(k.Allocatable.CPU-k.Requested.CPU, 0) / pick.CPURequest)
		}
		if pick.MemRequest > 0 {
			byMem := int(max(k.Allocatable.Memory-k.Requested.Memory, 0) / (pick.MemRequest * mebi))
			if fits < 0 || byMem < fits {
				fits = byMem
			}
		}
		room += fits
	}
	return room
}

func pledgeRoomLine(f Frame, c model.CrewChain, pick model.Resources) string {
	n := pledgeRoom(f.State.Keeps, EffectiveResources(pick, f))
	if n < 0 {
		return ui.Bridge("No pledge: Keeps reserve nothing, and any hut may be evicted first.", "No requests: the scheduler reserves nothing (BestEffort).")
	}
	if fit, count, ok := repledgeFit(f, c, pick); ok {
		return repledgeLine(fit, count)
	}
	return seatMore(n)
}

func pledgeShort(f Frame, c model.CrewChain, pick model.Resources) bool {
	fit, _, ok := repledgeFit(f, c, pick)
	return ok && (fit.Seated < 0 || fit.Fed < fit.Seated)
}

func inForce(f Frame, c model.CrewChain, pick model.Resources) bool {
	now, next := EffectiveResources(c.Resources, f), EffectiveResources(pick, f)
	return now.CPURequest == next.CPURequest && now.MemRequest == next.MemRequest
}

func repledgeFit(f Frame, c model.CrewChain, pick model.Resources) (view.RoomFit, int, bool) {
	if inForce(f, c, pick) {
		if _, ok := crewByName(f, c.Name); ok && len(f.State.Room.Keeps) > 0 {
			k := CrewFactsOf(f, c)
			return view.RoomFit{Seated: k.Seats, Fed: k.Fed, Limit: k.Limit}, 0, true
		}
	}
	eff := EffectiveResources(pick, f)
	cv, ok := crewByName(f, c.Name)
	if !ok || len(f.State.Room.Keeps) == 0 || eff.CPURequest <= 0 && eff.MemRequest <= 0 {
		return view.RoomFit{}, 0, false
	}
	asks := view.ResourcePair{CPU: eff.CPURequest, Memory: eff.MemRequest * mebi}
	old := view.ResourcePair{CPU: cv.PerReplica.CPU, Memory: cv.PerReplica.Memory}
	return f.State.Room.Repledged(c.Name, cv.Replicas, old, asks, cv.Use, cv.Work), cv.Replicas, true
}

func repledgeLine(fit view.RoomFit, count int) string {
	switch {
	case fit.Seated < 0:
		wait := min(-fit.Seated, count)
		return ui.Bridge(fmt.Sprintf("At this pledge %d of your %d huts wait Unhoused once the crew rolls over, and their boxes spoil: the Nines fall.", wait, count),
			fmt.Sprintf("At these requests %d of %d Pods stay Pending after the rollout.", wait, count))
	case fit.Fed >= fit.Seated:
		return seatMore(fit.Seated)
	case fit.Fed < 0 && fit.Limit == view.FlowMemory:
		return ui.Bridge(fmt.Sprintf("At this pledge all are placed, but %d crowd a Keep at what each really takes.", min(-fit.Fed, count)),
			fmt.Sprintf("At these requests all schedule, but %d overcommit memory at real use.", min(-fit.Fed, count)))
	case fit.Fed < 0:
		return ui.Bridge(fmt.Sprintf("At this pledge all are placed, but %d go hungry at what each really eats.", min(-fit.Fed, count)),
			fmt.Sprintf("At these requests all schedule, but %d overcommit CPU at real use.", min(-fit.Fed, count)))
	case fit.Limit == view.FlowMemory:
		return ui.Bridge(fmt.Sprintf("At this pledge your Keeps have room for %d more, but house only %d at what each takes.", fit.Seated, fit.Fed),
			fmt.Sprintf("At these requests %d more Pods fit, but memory holds only %d at real use.", fit.Seated, fit.Fed))
	}
	return ui.Bridge(fmt.Sprintf("At this pledge your Keeps have room for %d more, but feed only %d at what each eats.", fit.Seated, fit.Fed),
		fmt.Sprintf("At these requests %d more Pods fit, but CPU feeds only %d at real use.", fit.Seated, fit.Fed))
}

func (h *HUD) drawResourceFocus(dst *ebiten.Image, v crewView, f Frame, y int) {
	rf := h.focusField(v)
	lead := rf.label + ": "
	if _, inherited := resValue(rf, h.Crew.Res, f); inherited {
		lead = rf.label + ui.Bridge(" (none of its own: the realm's default applies): ", " (unset: the LimitRange default applies): ")
	}
	for i, row := range h.Font.WrapLines(lead+ui.Bridge(rf.story, rf.k8s), v.body.W, 1, resEffectRows) {
		h.Font.Draw(dst, row, v.body.X, y+i*ui.LineHeight, 1, ui.Palette.Text)
	}
	line, ink := pledgeRoomLine(f, v.chain, h.Crew.Res), ui.Palette.Blue
	if pledgeShort(f, v.chain, h.Crew.Res) {
		ink = ui.Palette.RedText
	}
	if warn := bindingWarning(h.Crew.Res, v.use); warn != "" {
		line, ink = warn, ui.Palette.RedText
	}
	for i, row := range h.Font.WrapLines(line, v.body.W, 1, resRoomRows) {
		h.Font.Draw(dst, row, v.body.X, y+(resEffectRows+i)*ui.LineHeight, 1, ink)
	}
}

func bindingWarning(pick model.Resources, u useFacts) string {
	if u.Pods == 0 {
		return ""
	}
	if room := int64(math.Round(u.Memory / float64(u.Pods) / mebi)); pick.MemLimit > 0 && pick.MemLimit < room {
		return ui.Bridge(fmt.Sprintf("A Housing Binding of %d is below the %d rooms a worker takes: it would be Voidtaken.", pick.MemLimit, room),
			fmt.Sprintf("limits.memory %dMi is below the %dMi a Pod uses: it would be OOMKilled.", pick.MemLimit, room))
	}
	if food := int64(math.Round(u.CPU / float64(u.Pods))); pick.CPULimit > 0 && pick.CPULimit < food {
		return ui.Bridge(fmt.Sprintf("A Food Binding of %d is below the %d rations a worker eats: it would slow down.", pick.CPULimit, food),
			fmt.Sprintf("limits.cpu %dm is below the %dm a Pod uses: it would be throttled.", pick.CPULimit, food))
	}
	return ""
}

func seatMore(n int) string {
	return ui.Bridge("At this pledge your Keeps have room for "+plural(n, "more hut", "more huts")+".", "At these requests your Nodes fit "+plural(n, "more Pod", "more Pods")+".")
}
