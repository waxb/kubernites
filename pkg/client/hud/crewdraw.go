package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	crewGlyphW = 7
	crewTextX  = 11
	hutSprite  = 1
	hutInset   = 60
	qosRows    = 2
)

var rowHover = color.RGBA{44, 62, 70, 255}

func DeployRowText(c model.CrewChain) string {
	text := fmt.Sprintf("%s  %s  ready %d/%d", c.Name, c.Image, c.Ready, c.Desired)
	if c.MaxSurge != "" {
		text += fmt.Sprintf("  %s %s/%s", c.Strategy, c.MaxSurge, c.MaxUnavailable)
	} else if c.Strategy != "" {
		text += "  " + c.Strategy
	}
	return text
}

func foremanRowText(fm model.Foreman) string {
	story := fmt.Sprintf("SHIFT %s  rev %d  %d/%d", fm.Name, fm.Revision, fm.Ready, fm.Desired)
	k8s := fmt.Sprintf("ReplicaSet %s  rev %d  %d/%d", fm.Name, fm.Revision, fm.Ready, fm.Desired)
	return ui.Bridge(story, k8s)
}

func HistoryRowText(c model.CrewChain) string {
	revs := []string{}
	for _, f := range c.History() {
		revs = append(revs, fmt.Sprintf("rev %d", f.Revision))
	}
	return ui.Bridge("HISTORY  ", "rollout history  ") + strings.Join(revs, "  ")
}

func CrewTitle(c model.CrewChain) (string, string) {
	if c.Bare && len(c.Foremen) > 0 {
		return ui.Bridge("SHIFT "+c.Foremen[0].Name, "ReplicaSet "+c.Foremen[0].Name), ui.Bridge("(shift, no crew)", "(no Deployment)")
	}
	return ui.Bridge("CREW "+c.Name, "Deployment "+c.Name), ui.Bridge("(Deployment)", "")
}

func chainTitle(c model.CrewChain, f Frame) (string, string) {
	title, kind := CrewTitle(c)
	if c.Bare && foremenOnly(f) {
		return title, ""
	}
	return title, kind
}

var foremanLegend = [2]string{"SHIFT (ReplicaSet) > HUTS (Pods)", "ReplicaSet > Pods"}

func crewLegend(c model.CrewChain, mode string, f Frame) string {
	legend := modeLegends[mode]
	if mode == CrewModeProbes {
		legend = watchLegend(f)
	}
	if mode == CrewModeChain && c.Bare && foremenOnly(f) {
		legend = foremanLegend
	}
	return ui.Bridge(legend[0], legend[1])
}

var modeLegends = map[string][2]string{
	CrewModeChain:  {"CREW (Deployment) > SHIFT (ReplicaSet) > HUTS (Pods)", "Deployment > ReplicaSet > Pods"},
	CrewModeImage:  {"UPGRADE: pick a newer class of the trade", "SET IMAGE: pick a newer tag"},
	CrewModeRes:    {"RESOURCES: what each worker pledges and may use", "RESOURCES: requests and limits per container"},
	CrewModeHorn:   {"WAR HORN: the crew's count follows its CPU use", "AUTOSCALE: a HorizontalPodAutoscaler"},
	CrewModeRoll:   {"ROLLOUT: raise the huts anew, undo, pause", "ROLLOUT: restart, undo, pause, resume"},
	CrewModeProbes: {"WATCH: the Shutters and the Bell on every hut of the crew", "PROBES: readiness and liveness on the pod template"},
}

func (h *HUD) DrawCrewCard(dst *ebiten.Image, f Frame) {
	if b, ok := h.BoardView(f); ok {
		h.drawCrewBoard(dst, b, f)
		return
	}
	if d, ok := h.DepotView(f); ok {
		h.drawDepotBoard(dst, d, f)
		return
	}
	v, ok := h.crewView(f)
	if !ok {
		return
	}
	ui.DarkPanel(dst, v.box)
	title, kind := chainTitle(v.chain, f)
	x, y := v.box.X+crewPad, v.box.Y+5
	h.Font.DrawShadow(dst, title, x, y, 1, ui.Palette.GoldBright)
	h.Font.Draw(dst, kind, x+h.Font.Width(title, 1)+ui.Advance, y, 1, ui.Palette.Blue)
	h.Font.Draw(dst, crewLegend(v.chain, v.mode, f), x, y+ui.LineHeight, 1, ui.Palette.TextDim)
	h.drawCrewRows(dst, v, f)
	h.drawCrewBody(dst, v, f)
	for _, b := range h.stepperButtons(v) {
		h.drawButton(dst, b)
	}
	h.drawButton(dst, closeButtonAs(v.box, CrewPrefix+":close", ""))
}

func (h *HUD) rowHot(id string) bool {
	return h.Hovering && h.Hover.ID == id
}

func (h *HUD) drawCrewRows(dst *ebiten.Image, v crewView, f Frame) {
	for i, r := range v.rows {
		b := rowButton(r)
		switch {
		case r.selected:
			ui.Fill(dst, r.rect, ui.Palette.GreenDark)
		case h.rowHot(b.ID):
			ui.Fill(dst, r.rect, rowHover)
		}
		if r.kind == "rs" && i > 0 {
			drawElbow(dst, v.body.X+3, v.rows[i-1].rect.Y+crewRowH, r.rect)
		}
		h.drawCrewRow(dst, r, v.chain)
	}
	pods := podStates(f.Pods)
	for _, b := range v.pods {
		drawHutIcon(dst, b.Rect, hearthColors[pods[strings.TrimPrefix(b.ID, CrewPrefix+":pick:pod:")]], b.Selected || h.rowHot(b.ID))
	}
}

func drawElbow(dst *ebiten.Image, x, top int, row ui.Rect) {
	mid := row.Y + row.H/2
	ui.Fill(dst, ui.Rect{X: x, Y: top, W: 1, H: mid - top}, ui.Palette.Muted)
	ui.Fill(dst, ui.Rect{X: x, Y: mid, W: row.X - x - 2, H: 1}, ui.Palette.Muted)
}

func (h *HUD) drawCrewRow(dst *ebiten.Image, r crewRow, c model.CrewChain) {
	x, y := r.rect.X+2, r.rect.Y+3
	switch r.kind {
	case "deploy":
		drawFlag(dst, x, y-1, clothColor, model.RevisionColor(newestOf(c)))
		h.Font.Draw(dst, ui.Clip(DeployRowText(c), ui.MaxChars(r.rect.W-crewTextX, 1)), x+crewTextX, y, 1, ui.Palette.Cream)
		if c.Paused {
			h.Font.Draw(dst, "PAUSED", r.rect.X+r.rect.W-h.Font.Width("PAUSED", 1)-2, y, 1, ui.Palette.RedText)
		}
	case "rs":
		tone := model.RevisionColor(r.foreman.Revision)
		drawFlag(dst, x, y-1, tone, tone)
		h.Font.Draw(dst, foremanRowText(r.foreman), x+crewTextX, y, 1, TallyInk(r.foreman))
	case "hist":
		for i, fm := range c.History()[:min(len(c.History()), 5)] {
			drawFlag(dst, x+i*3, y+1, ui.Tint(model.RevisionColor(fm.Revision), 140), ui.Palette.Muted)
		}
		h.Font.Draw(dst, ui.Clip(HistoryRowText(c), ui.MaxChars(r.rect.W-crewTextX-10, 1)), x+crewTextX+10, y, 1, ui.Palette.TextDim)
	}
}

var clothColor = color.RGBA{232, 216, 176, 255}

func newestOf(c model.CrewChain) int64 {
	for _, f := range c.Foremen {
		if f.Newest {
			return f.Revision
		}
	}
	return c.Revision
}

func TallyInk(fm model.Foreman) color.RGBA {
	switch {
	case fm.Desired == 0:
		return ui.Palette.TextDim
	case fm.Ready >= fm.Desired:
		return ui.Palette.Green
	}
	return ui.Palette.GoldBright
}

func drawFlag(dst *ebiten.Image, x, y int, cloth, stripe color.RGBA) {
	drawFlagAt(dst, x, y, cloth, stripe, 1)
}

func drawFlagAt(dst *ebiten.Image, x, y int, cloth, stripe color.RGBA, zoom int) {
	ui.Fill(dst, ui.Rect{X: x, Y: y, W: zoom, H: 10 * zoom}, ui.Palette.Muted)
	ui.Fill(dst, ui.Rect{X: x + zoom, Y: y, W: (crewGlyphW - 1) * zoom, H: 5 * zoom}, cloth)
	ui.Fill(dst, ui.Rect{X: x + zoom, Y: y + 5*zoom, W: (crewGlyphW - 1) * zoom, H: zoom}, stripe)
}

func drawHutIcon(dst *ebiten.Image, r ui.Rect, c color.RGBA, hot bool) {
	if c == (color.RGBA{}) {
		c = ui.Palette.Muted
	}
	ui.Fill(dst, ui.Rect{X: r.X + 1, Y: r.Y + 3, W: r.W - 2, H: r.H - 3}, c)
	ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y + 2, W: r.W, H: 1}, ui.Palette.Ink)
	ui.Fill(dst, ui.Rect{X: r.X + 2, Y: r.Y + 1, W: r.W - 4, H: 1}, ui.Palette.Ink)
	ui.Fill(dst, ui.Rect{X: r.X + 3, Y: r.Y, W: r.W - 6, H: 1}, ui.Palette.Ink)
	if hot {
		ui.Stroke(dst, r.Inset(-2), ui.Palette.GoldBright)
	}
}

func podStates(pods []model.PodView) map[string]string {
	out := make(map[string]string, len(pods))
	for _, p := range pods {
		out[p.Name] = p.State
	}
	return out
}

func (h *HUD) drawCrewBody(dst *ebiten.Image, v crewView, f Frame) {
	switch v.mode {
	case CrewModeImage:
		h.drawBodyText(dst, bodyTop(v, 2), fmt.Sprintf(ui.Bridge("Now %s. A new shift (ReplicaSet) takes the huts over one at a time: %s surge, %s unavailable.",
			"Now %s. A new ReplicaSet scales up as the old one scales down: maxSurge %s, maxUnavailable %s."), v.chain.Image, v.chain.MaxSurge, v.chain.MaxUnavailable))
		h.drawRolloutRoom(dst, v, f, 2)
	case CrewModeRes:
		h.drawResources(dst, v, f)
	case CrewModeHorn:
		h.drawHorn(dst, v)
	case CrewModeRoll:
		h.drawBodyText(dst, bodyTop(v, 2), RolloutLine(v.chain))
		h.drawRolloutRoom(dst, v, f, 2)
	case CrewModeProbes:
		h.drawBodyText(dst, bodyTop(v, 3), watchLine(f))
		h.drawRolloutRoom(dst, v, f, 3)
	}
}

func watchLine(f Frame) string {
	parts := []string{}
	for _, p := range probeKinds {
		if RuneKnown(f, p.rune) {
			parts = append(parts, p.tip)
		}
	}
	return strings.Join(append(parts, ui.Bridge("Each switch rolls the crew over, a few huts at a time.", "Each change rolls out a new ReplicaSet.")), " ")
}

const rolloutRoomRows = 4

func bodyTop(v crewView, rows int) ui.Rect {
	return ui.Rect{X: v.body.X, Y: v.body.Y, W: v.body.W, H: rows * ui.LineHeight}
}

func rolloutRoomText(f Frame, c model.CrewChain) string {
	return roomCheck(crewRoom(f, c), c.Name)
}

func upToHutsWait(n int) string {
	if n == 1 {
		return hutsWait(n)
	}
	return "up to " + hutsWait(n)
}

func hutsWait(n int) string {
	if n == 1 {
		return ui.Bridge("1 hut waits", "1 Pod")
	}
	return fmt.Sprintf(ui.Bridge("%d huts wait", "%d Pods"), n)
}

func (h *HUD) drawRolloutRoom(dst *ebiten.Image, v crewView, f Frame, above int) {
	ro := h.modeRoom(v.chain, f)
	text := roomCheck(ro, v.chain.Name)
	if text == "" {
		return
	}
	ink := ui.Palette.Blue
	if ro != nil && ro.Dip {
		ink = ui.Palette.GoldBright
	}
	if ro != nil && ro.Lacks() {
		ink = ui.Palette.RedText
	}
	y := v.body.Y + above*ui.LineHeight + 2
	for i, row := range h.Font.WrapLines(text, v.body.W, 1, rolloutRoomRows) {
		h.Font.Draw(dst, row, v.body.X, y+i*ui.LineHeight, 1, ink)
	}
}

func (h *HUD) drawBodyText(dst *ebiten.Image, r ui.Rect, text string) {
	for i, row := range h.Font.WrapLines(text, r.W, 1, max(r.H/ui.LineHeight, 1)) {
		h.Font.Draw(dst, row, r.X, r.Y+i*ui.LineHeight, 1, ui.Palette.Text)
	}
}

func (h *HUD) stepperRow(dst *ebiten.Image, v crewView, i int, label, value string, changed bool, flags ...bool) {
	y := v.body.Y + i*crewRowStep + 3
	labelInk := ui.Palette.Cream
	if len(flags) > 0 && flags[0] {
		labelInk = ui.Palette.GoldBright
	}
	h.Font.Draw(dst, label, v.body.X, y, 1, labelInk)
	ink := ui.Palette.Text
	switch {
	case changed:
		ink = ui.Palette.GoldBright
	case len(flags) > 1 && flags[1]:
		ink = ui.Palette.TextDim
	}
	box := ui.Rect{X: v.body.X + stepperLabelW + stepperW, Y: y - 3, W: stepperValueW, H: crewRowH}
	h.Font.DrawCentered(dst, value, box, 1, ink)
}

func (h *HUD) drawResources(dst *ebiten.Image, v crewView, f Frame) {
	pick, age, focus := h.Crew.Res, gfx.AgeTier(f.State.Age), h.focusField(v)
	for i, rf := range v.fields {
		value, inherited := resValue(rf, pick, f)
		h.stepperRow(dst, v, i, rf.label, value, rf.get(pick) != rf.get(v.chain.Resources), rf.key == focus.key, inherited)
	}
	qos := PickQoS(EffectiveResources(pick, f))
	y := v.body.Y + len(v.fields)*crewRowStep + 3
	hutX, hutY := v.body.X+v.body.W-hutInset, v.body.Y+len(v.fields)*crewRowStep
	h.Lib.DrawFirst(dst, gfx.HearthLookNames(hutSprite, model.HearthReady, qos, age, 0), float64(hutX), float64(hutY), 1, render.SpriteOptions{})
	text := ui.Bridge("Its standing: ", "QoS class: ") + model.QoSTitle(qos)
	h.Font.Draw(dst, text, v.body.X, y, 1, qosColors[qos])
	for i, row := range h.Font.WrapLines(model.QoSSentence(qos), v.body.W, 1, qosRows) {
		h.Font.Draw(dst, row, v.body.X, y+(i+1)*ui.LineHeight, 1, ui.Palette.TextDim)
	}
	h.drawResourceFocus(dst, v, f, y+(qosRows+1)*ui.LineHeight+2)
}

func (h *HUD) drawHorn(dst *ebiten.Image, v crewView) {
	p := h.Crew.Horn
	rows := [][2]string{{ui.Bridge("Floor", "minReplicas"), fmt.Sprint(p.Min)}, {ui.Bridge("Ceiling", "maxReplicas"), fmt.Sprint(p.Max)},
		{ui.Bridge("Pitch (CPU)", "target CPU"), fmt.Sprintf("%d%%", p.CPU)}}
	for i, r := range rows {
		h.stepperRow(dst, v, i, r[0], r[1], false)
	}
	note := ui.Bridge("It weighs the food eaten against each worker's Food Pledge: set the Food Pledge first.", "Utilization is measured against requests.cpu: set it first.")
	if hn := v.chain.Horn; hn != nil {
		note = ui.Bridge("Sounding now: "+HornLine(hn), fmt.Sprintf("HPA %s: min %d, max %d, target %d%% CPU; now %d at %d%%.", hn.Name, hn.Min, hn.Max, hn.Goal, hn.Current, hn.Utilization))
	}
	h.Font.Draw(dst, ui.Clip(note, ui.MaxChars(v.body.W, 1)), v.body.X, v.body.Y+3*crewRowStep+2, 1, ui.Palette.TextDim)
}

func RolloutLine(c model.CrewChain) string {
	return fmt.Sprintf(ui.Bridge("Revision %d, %d kept for UNDO. RESTART raises every hut anew with the same class, one shift at a time.",
		"Revision %d, %d kept in history. kubectl rollout restart replaces every Pod through a new ReplicaSet."), c.Revision, len(c.History()))
}
