package hud

import (
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	crewW       = 404
	crewPad     = 6
	crewTop     = 28
	crewRowH    = 13
	crewRowStep = 14
	crewBtnH    = 15
	crewGap     = 3
	crewIndent  = 14
	crewPodW    = 8
	crewPodStep = 10
	crewFloat   = 4
	maxStanding = 3
	crewTextPad = 10
	stepperW    = 16
)

type crewRow struct {
	kind     string
	rect     ui.Rect
	foreman  model.Foreman
	selected bool
}

type crewView struct {
	chain  model.CrewChain
	mode   string
	box    ui.Rect
	rows   []crewRow
	pods   []Button
	body   ui.Rect
	fields []resField
	use    useFacts
	peak   view.UsePeakView
}

func (h *HUD) crewView(f Frame) (crewView, bool) {
	c, ok := ChainOf(f)
	if !ok || !h.ChainShown(f) {
		return crewView{}, false
	}
	v := crewView{chain: c, mode: h.CrewMode(c)}
	if v.mode == CrewModeRes {
		v.fields = knownFields(f)
		v.use = crewUse(c, f)
		v.peak = crewPeakOf(f, c)
	}
	width := crewW - 2*crewPad
	bodyH := v.bodyHeight(f)
	height := crewTop + bodyH + crewPad
	v.box = f.placed(PanelCrew, ui.Rect{X: CardBox.X, Y: ConsoleY - crewFloat - height, W: crewW, H: height})
	v.body = ui.Rect{X: v.box.X + crewPad, Y: v.box.Y + crewTop, W: width, H: bodyH}
	v.rows = chainRows(v, f)
	v.pods = h.podButtons(v, f)
	return v, true
}

func (v crewView) bodyHeight(f Frame) int {
	switch v.mode {
	case CrewModeImage:
		return 2*ui.LineHeight + v.rolloutRoomHeight(f)
	case CrewModeRes:
		return len(v.fields)*crewRowStep + (3+resEffectRows+resRoomRows)*ui.LineHeight + 6
	case CrewModeHorn:
		return 3*crewRowStep + ui.LineHeight
	case CrewModeRoll:
		return 2*ui.LineHeight + v.rolloutRoomHeight(f)
	case CrewModeProbes:
		return 3*ui.LineHeight + v.rolloutRoomHeight(f)
	}
	return len(chainLevels(v.chain)) * crewRowStep
}

func chainLevels(c model.CrewChain) []crewRow {
	out := []crewRow{}
	if !c.Bare {
		out = append(out, crewRow{kind: "deploy"})
	}
	standing := c.Standing()
	for _, fm := range standing[:min(len(standing), maxStanding)] {
		out = append(out, crewRow{kind: "rs", foreman: fm})
	}
	if len(c.History()) > 0 {
		out = append(out, crewRow{kind: "hist"})
	}
	return out
}

func chainRows(v crewView, f Frame) []crewRow {
	if v.mode != CrewModeChain {
		return nil
	}
	out := chainLevels(v.chain)
	for i := range out {
		out[i].rect = ui.Rect{X: v.body.X, Y: v.body.Y + i*crewRowStep, W: v.body.W, H: crewRowH}
		if out[i].kind == "rs" {
			out[i].rect.X, out[i].rect.W = v.body.X+crewIndent, v.body.W-crewIndent
		}
		out[i].selected = rowSelected(out[i], v.chain, f.Selection)
	}
	return out
}

func rowSelected(r crewRow, c model.CrewChain, sel Selection) bool {
	switch r.kind {
	case "deploy":
		return sel.Kind == SelectCrew
	case "rs":
		return sel.Kind == SelectForeman && sel.ID == r.foreman.ID()
	}
	return false
}

func rowButton(r crewRow) Button {
	id := CrewPrefix + ":pick:deploy"
	tip := "Select the crew: its Deployment. The commands below act on it."
	switch r.kind {
	case "rs":
		id, tip = CrewPrefix+":pick:rs:"+r.foreman.Name, "Select this shift: the ReplicaSet "+r.foreman.Name+" that keeps its huts."
	case "hist":
		id, tip = CrewPrefix+":pick:deploy", "Old revisions kept for rollback (revisionHistoryLimit). ROLLOUT UNDO returns to the last one."
	}
	return Button{ID: id, Rect: r.rect, Tip: tip, Selected: r.selected}
}

func (h *HUD) podButtons(v crewView, f Frame) []Button {
	out := []Button{}
	for _, r := range v.rows {
		if r.kind != "rs" {
			continue
		}
		x := r.rect.X + r.rect.W - crewPodStep*len(r.foreman.Pods)
		x = max(x, r.rect.X+h.Font.Width(foremanRowText(r.foreman), 1)+crewTextPad)
		for _, name := range r.foreman.Pods {
			if x+crewPodW > r.rect.X+r.rect.W {
				break
			}
			rect := ui.Rect{X: x, Y: r.rect.Y + 2, W: crewPodW, H: crewRowH - 4}
			out = append(out, Button{ID: CrewPrefix + ":pick:pod:" + name, Rect: rect, Tip: "Select the hut " + name + " (Pod)",
				Selected: f.Selection.Kind != "structure" && f.Selection.Pod.Name == name})
			x += crewPodStep
		}
	}
	return out
}

func (h *HUD) CrewButtons(f Frame) []Button {
	if b, ok := h.BoardView(f); ok {
		return h.BoardButtons(b)
	}
	if d, ok := h.DepotView(f); ok {
		return d.buttons()
	}
	v, ok := h.crewView(f)
	if !ok {
		return nil
	}
	out := []Button{closeButtonAs(v.box, CrewPrefix+":close", "Deselect [Esc or right click]")}
	for _, r := range v.rows {
		out = append(out, rowButton(r))
	}
	out = append(out, v.pods...)
	return append(out, h.stepperButtons(v)...)
}

func (h *HUD) CrewRects(f Frame) []ui.Rect {
	if b, ok := h.BoardView(f); ok {
		return []ui.Rect{b.box}
	}
	if d, ok := h.DepotView(f); ok {
		return []ui.Rect{d.box}
	}
	if v, ok := h.crewView(f); ok {
		return []ui.Rect{v.box}
	}
	return nil
}

func (v crewView) rolloutRoomHeight(f Frame) int {
	if rolloutRoomText(f, v.chain) == "" {
		return 0
	}
	return rolloutRoomRows*ui.LineHeight + 2
}
