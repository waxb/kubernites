package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	crewsKindX    = 78
	crewsReadyX   = 146
	crewsBarX     = 178
	crewsBarW     = 40
	crewsReserveX = 226
	crewsNameW    = 74
)

func crewsTitle(f Frame) string {
	if multiStall(f) {
		return ui.Bridge(f.Selection.Structure.Object.Name+" depot: the crews behind it", f.Selection.Structure.Object.Name+" Service: the Deployments behind it")
	}
	return ui.Bridge("THE CREWS", "DEPLOYMENTS")
}

func crewKindWord(c model.CrewChain) string {
	if c.Bare {
		return ui.Bridge("SHIFT", "REPLICASET")
	}
	return ui.Bridge("CREW", "DEPLOYMENT")
}

func crewReserve(f Frame, c model.CrewChain) (string, string) {
	per := CrewFactsOf(f, c).Per
	n := int64(c.Desired)
	if per.CPU <= 0 && per.Memory <= 0 {
		none := ui.Bridge("pledges nothing", "no requests")
		return none, none
	}
	return reserveLine(per.CPU*n, per.Memory*n), reserveShort(per.CPU*n, per.Memory*n)
}

func reserveLine(cpu, mem int64) string { return cpuText(cpu) + " " + memText(mem) }

func reserveShort(cpu, mem int64) string {
	return ui.Bridge(foodTag(float64(cpu))+" "+housingTag(float64(mem)), fmt.Sprintf("%dm %s", cpu, MemQuantity(mem)))
}

func crewsReserveText(full, short string, w int) string {
	if strings.HasSuffix(ui.Clip(full, ui.MaxChars(w, 1)), "...") {
		return short
	}
	return full
}

func crewStateWord(f Frame, c model.CrewChain) string {
	name := strings.ToUpper(c.Name)
	for _, g := range Guilds(f) {
		if g.Name == name && g.State != "" {
			return g.State
		}
	}
	return ""
}

func (h *HUD) drawCrews(dst *ebiten.Image, f Frame) {
	if !h.crewsShown(f) {
		return
	}
	rows := CrewsRows(f)
	box := crewsRect(f, len(rows))
	drawPanel(dst, box)
	h.Font.DrawShadow(dst, crewsTitle(f), box.X+crewsPad, box.Y+6, 1, ui.Palette.GoldBright)
	for i, r := range rows {
		rr := crewsRowRect(box, i)
		if r.kind != "" {
			h.drawKindRow(dst, f, rr, r)
			continue
		}
		if r.Standing {
			h.drawCrewsRow(dst, f, rr, r)
			continue
		}
		h.drawRaiseRow(dst, rr, r)
	}
	for _, b := range h.crewsButtons(f) {
		if strings.HasPrefix(b.ID, CrewsKey+":open:") {
			continue
		}
		h.drawButton(dst, b)
	}
}

func crewsStateAt(rr ui.Rect, word string) int {
	return rr.X + rr.W - 2*crewsBtnW - 8 - (ui.Font{}).Width(word, 1)
}

func crewsReserveSpan(rr ui.Rect, word string) (int, int) {
	end := rr.X + rr.W - 2*crewsBtnW - 8
	if word != "" {
		end = crewsStateAt(rr, word) - 4
	}
	return rr.X + crewsReserveX, end - rr.X - crewsReserveX
}

func (h *HUD) drawRaiseRow(dst *ebiten.Image, rr ui.Rect, r CrewsRow) {
	ui.Well(dst, rr)
	y := rr.Y + (rr.H-ui.GlyphHeight)/2
	h.Font.Draw(dst, ui.Clip(strings.ToUpper(r.trade.Crew), ui.MaxChars(crewsNameW, 1)), rr.X+4, y, 1, ui.Palette.Text)
	status := ui.Bridge("no crew yet: "+r.trade.Name+" waits", "no Deployment yet: "+r.trade.Name)
	h.Font.Draw(dst, status, rr.X+crewsKindX, y, 1, ui.Palette.TextDim)
}

func (h *HUD) drawCrewsRow(dst *ebiten.Image, f Frame, rr ui.Rect, r CrewsRow) {
	id := CrewsKey + ":open:" + r.Id()
	ui.Well(dst, rr)
	if h.rowHot(id) {
		ui.Fill(dst, rr, rowHover)
	}
	c, y := r.Chain, rr.Y+(rr.H-ui.GlyphHeight)/2
	h.Font.Draw(dst, ui.Clip(strings.ToUpper(c.Name), ui.MaxChars(crewsNameW, 1)), rr.X+4, y, 1, ui.Palette.Text)
	h.Font.Draw(dst, crewKindWord(c), rr.X+crewsKindX, y, 1, ui.Palette.Blue)
	h.Font.Draw(dst, fmt.Sprintf("%d/%d", c.Ready, c.Desired), rr.X+crewsReadyX, y, 1, readyTone(c.Ready, c.Desired))
	fl := crewFlow(f.State.Freight, c.Name)
	if fl.Known {
		share := CarriedShare(fl.Arriving, fl.Carried, fl.Waiting)
		h.drawMiniBar(dst, ui.Rect{X: rr.X + crewsBarX, Y: rr.Y + 5, W: crewsBarW, H: 5}, share, ShareColor(share))
	}
	word := crewStateWord(f, c)
	x, w := crewsReserveSpan(rr, word)
	full, short := crewReserve(f, c)
	drawClipped(dst, crewsReserveText(full, short, w), x, y, w, ui.Palette.Blue)
	if word != "" {
		h.Font.Draw(dst, word, crewsStateAt(rr, word), y, 1, ui.Palette.RedText)
	}
}

func init() {
	prefixHandlers[CrewsKey] = (*HUD).PressCrews
	WindowSpecs[PanelCrews] = windowSpec{Rect: crewsWindow, draw: (*HUD).drawCrews, buttons: (*HUD).crewsButtons}
}

func (h *HUD) drawKindRow(dst *ebiten.Image, f Frame, rr ui.Rect, r CrewsRow) {
	ui.Well(dst, rr)
	if h.rowHot(CrewsKey + ":open:" + r.kind) {
		ui.Fill(dst, rr, rowHover)
	}
	y := rr.Y + (rr.H-ui.GlyphHeight)/2
	name, ready, desired := kindRowFacts(f, r.kind)
	h.Font.Draw(dst, ui.Settle(ui.Clip(strings.ToUpper(name), ui.MaxChars(crewsNameW, 1))), rr.X+4, y, 1, ui.Palette.Text)
	h.Font.Draw(dst, r.KindWord(), rr.X+crewsKindX, y, 1, ui.Palette.GoldBright)
	if r.kind == DsRaiseID {
		h.Font.Draw(dst, ui.Bridge("no wardens yet: couriers strand", "no DaemonSet yet"), rr.X+crewsReadyX, y, 1, ui.Palette.TextDim)
		return
	}
	if r.kind == archiveRaiseID {
		h.Font.Draw(dst, unregisteredNote(f), rr.X+crewsReadyX, y, 1, ui.Palette.RedText)
		return
	}
	if r.kind == archiveCardID {
		h.Font.Draw(dst, archiveRowNote(f), rr.X+crewsReadyX, y, 1, ui.Palette.GoldBright)
		return
	}
	h.Font.Draw(dst, fmt.Sprintf("%d/%d", ready, desired), rr.X+crewsReadyX, y, 1, readyTone(ready, desired))
}

func kindRowFacts(f Frame, id string) (string, int, int) {
	if d, ok := daemonSetNamed(f, strings.TrimPrefix(id, dsPrefix)); ok {
		return d.Name, int(d.Ready), int(d.Desired)
	}
	if s, ok := statefulSetNamed(f, strings.TrimPrefix(id, stsPrefix)); ok {
		return s.Name, s.Ready, s.Replicas
	}
	if id == archiveCardID || id == archiveRaiseID {
		return ui.Bridge("muniment room", view.ArchiveName), 0, 0
	}
	return "wardens", 0, 0
}
