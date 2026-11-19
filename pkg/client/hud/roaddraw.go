package hud

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	roadSubtitle    = "Each Age teaches a Kubernetes skill; meet its goal to advance. Reforge to walk it again, stronger."
	roadSubtitleK8s = "Each Environment teaches a Kubernetes skill; meet its goal to promote. Rebuild to go again, stronger."
	sealsNote       = "Weather a Rush Tide, earn a Seal. Eight: Ascend."
	sealsNoteK8s    = "Pass a load test, earn a badge. Eight: Ascend."
	sealsLabel      = "CONCLAVE"
	sealsLabelW     = 60
	SealsTip        = "Weather a Rush Tide for a Conclave Seal. Eight Seals open the Conclave gate to Ascension."
	cobbleStep      = 7
	sealR           = 4
	sealGap         = 11
)

var romans = []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX"}

func (h *HUD) drawRoad(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, RoadBox)
	h.header(dst, RoadBox, "THE ROAD OF AGES")
	h.drawRoadTally(dst, f)
	h.Font.Draw(dst, ui.Clip(ui.Bridge(roadSubtitle, roadSubtitleK8s), ui.MaxChars(roadLeftW, 1)), RoadBox.X+roadInset, RoadBox.Y+26, 1, ui.Palette.TextDim)
	road := f.State.Road
	if len(road.Ages) == 0 {
		h.Font.Draw(dst, "The road ahead is not yet charted.", RoadBox.X+roadInset, RoadBox.Y+roadPathY, 1, ui.Palette.Text)
	}
	cols := RoadColumns(len(road.Ages))
	statuses := make([]string, len(road.Ages))
	for i, a := range road.Ages {
		statuses[i] = RoadStatus(a, road.Current)
	}
	h.drawCobbles(dst, cols, statuses)
	sel, _ := h.selectedAge(f)
	for i, a := range road.Ages {
		h.drawAgeColumn(dst, f, a, statuses[i], cols[i], i, a.ID == sel.ID)
	}
	h.drawUnlockPane(dst, sel, road.Current)
	h.drawLedgerPane(dst, f)
	h.DrawOverlayButtons(dst, []Button{roadLoomButton(), closeButton(RoadBox)})
}

func (h *HUD) drawRoadTally(dst *ebiten.Image, f Frame) {
	rf, sg := f.State.Reforge, f.State.Siege
	right := RoadBox.X + RoadBox.W - 30
	seals := SealsText(sg)
	sw, _ := h.Font.Measure(seals, 1)
	h.Font.Draw(dst, seals, right-sw, RoadBox.Y+9, 1, ui.Palette.Text)
	goal := sealsGoal(sg)
	x0 := right - sw - 10 - goal*sealGap
	for i := range goal {
		cx, cy := float32(x0+i*sealGap+sealR), float32(RoadBox.Y+13)
		if i < sg.Seals {
			vector.FillCircle(dst, cx, cy, sealR, ui.Palette.Red, false)
			vector.StrokeCircle(dst, cx, cy, sealR, 1, ui.Palette.Gold, false)
			continue
		}
		vector.StrokeCircle(dst, cx, cy, sealR, 1, ui.Palette.Muted, false)
	}
	label := sealsLabel
	lw, _ := h.Font.Measure(label, 1)
	h.Font.Draw(dst, label, x0-lw-8, RoadBox.Y+9, 1, ui.Palette.Gold)
	threads := Compact(rf.Threads) + " THREADS"
	if rf.RunThreads > 0 {
		threads += " +" + Compact(rf.RunThreads)
	}
	tw, _ := h.Font.Measure(threads, 1)
	tx := x0 - lw - 30 - tw
	h.icon(dst, "icon/thread", tx-20, RoadBox.Y+5)
	h.Font.Draw(dst, threads, tx, RoadBox.Y+9, 1, ui.Palette.GoldBright)
	h.Font.Draw(dst, ui.Clip(ui.Bridge(sealsNote, sealsNoteK8s), ui.MaxChars(roadRight().W, 1)), roadRight().X, RoadBox.Y+26, 1, ui.Palette.TextDim)
}

func sealsRect(f Frame) ui.Rect {
	var font ui.Font
	sw, _ := font.Measure(SealsText(f.State.Siege), 1)
	right := RoadBox.X + RoadBox.W - 30
	x := right - sw - 10 - sealsGoal(f.State.Siege)*sealGap - max(sealsLabelW, font.Width(sealsLabel, 1)+10)
	return ui.Rect{X: x, Y: RoadBox.Y + 5, W: right - x, H: 14}
}

func medalCentre(c ui.Rect) (float32, float32) {
	return float32(c.X + c.W/2), float32(RoadBox.Y + roadPathY)
}

func (h *HUD) drawCobbles(dst *ebiten.Image, cols []ui.Rect, statuses []string) {
	for i := 0; i+1 < len(cols); i++ {
		x0, y := medalCentre(cols[i])
		x1, _ := medalCentre(cols[i+1])
		walked := statuses[i+1] != RoadAhead
		n := 0
		for x := x0 + medalR + 5; x < x1-medalR-4; x += cobbleStep {
			n++
			if !walked && n%2 == 0 {
				continue
			}
			wave := float32(math.Round(2 * math.Sin(float64(x-x0)/9)))
			clr := ui.Palette.Muted
			if walked {
				clr = ui.Palette.ParchmentDark
				if n%3 == 0 {
					clr = ui.Palette.Parchment
				}
			}
			vector.FillRect(dst, x, y+wave-1, 4, 2, clr, false)
		}
	}
}

func (h *HUD) drawAgeColumn(dst *ebiten.Image, f Frame, a view.AgeStep, status string, c ui.Rect, i int, selected bool) {
	switch {
	case status == RoadHere:
		ui.Fill(dst, c, ui.Tint(ui.Palette.BlueDark, 110))
		ui.Stroke(dst, c, ui.Palette.Gold)
	case selected:
		ui.Stroke(dst, c, ui.Palette.ParchmentDark)
	}
	hot := h.Hovering && h.Hover.ID == "road:age:"+a.ID
	h.drawMedal(dst, c, status, i, hot || selected)
	txt := roadText(a, status, c.W-8)
	nameClr := ui.Palette.Gold
	if status == RoadAhead {
		nameClr = ui.Palette.TextDim
	}
	h.Font.DrawCentered(dst, txt.name, ui.Rect{X: c.X, Y: RoadBox.Y + roadNameY, W: c.W, H: 9}, 1, nameClr)
	_, tagClr := roadTag(status)
	h.Font.DrawCentered(dst, txt.tag, ui.Rect{X: c.X, Y: RoadBox.Y + roadTagY, W: c.W, H: 9}, 1, tagClr)
	x := c.X + 4
	goalLabel := "GOAL"
	if progress, all := roadProgress(f); status == RoadHere && progress != "" {
		clr := ui.Palette.GoldBright
		if all {
			clr = ui.Palette.Green
		}
		pw, _ := h.Font.Measure(progress, 1)
		h.Font.Draw(dst, progress, c.X+c.W-4-pw, RoadBox.Y+roadGoalLabelY, 1, clr)
	}
	h.Font.Draw(dst, goalLabel, x, RoadBox.Y+roadGoalLabelY, 1, ui.Palette.TextDim)
	h.drawRows(dst, txt.goal, x, RoadBox.Y+roadGoalY, ui.Palette.Text)
}

func (h *HUD) drawRows(dst *ebiten.Image, rows []string, x, y int, clr color.RGBA) {
	for i, r := range rows {
		h.Font.Draw(dst, r, x, y+i*ui.LineHeight, 1, clr)
	}
}

func (h *HUD) drawMedal(dst *ebiten.Image, c ui.Rect, status string, i int, hot bool) {
	cx, cy := medalCentre(c)
	fill, rim, ink := ui.Palette.ConsoleEdge, ui.Palette.Muted, ui.Palette.TextDim
	switch status {
	case RoadDone:
		fill, rim, ink = ui.Palette.Gold, ui.Palette.Ink, ui.Palette.Ink
	case RoadHere:
		fill, rim, ink = ui.Palette.GoldBright, ui.Palette.Ink, ui.Palette.Ink
		a := uint8(90 + 165*pulse(h.Now, h.Reduce))
		vector.StrokeCircle(dst, cx, cy, medalR+3, 1, ui.Tint(ui.Palette.GoldBright, a), false)
		h.drawPennant(dst, cx+medalR-2, cy-medalR-6)
	}
	if hot {
		rim = ui.Palette.Cream
	}
	vector.FillCircle(dst, cx, cy+1, medalR, ui.Palette.Shadow, false)
	vector.FillCircle(dst, cx, cy, medalR, fill, false)
	vector.StrokeCircle(dst, cx, cy, medalR, 1, rim, false)
	numeral := romans[min(i, len(romans)-1)]
	h.Font.DrawCentered(dst, numeral, ui.Rect{X: int(cx) - medalR, Y: int(cy) - 4, W: 2 * medalR, H: 9}, 1, ink)
}

func (h *HUD) drawPennant(dst *ebiten.Image, x, y float32) {
	vector.FillRect(dst, x, y, 1, 11, ui.Palette.Parchment, false)
	var p vector.Path
	p.MoveTo(x+1, y)
	p.LineTo(x+9, y+3)
	p.LineTo(x+1, y+6)
	p.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(ui.Palette.Red)
	vector.FillPath(dst, &p, &vector.FillOptions{}, op)
}

func unlockTitle(a view.AgeStep, current string) string {
	name := strings.ToUpper(a.Name)
	if !strings.HasPrefix(name, "THE ") {
		name = "THE " + name
	}
	if a.ID == current {
		return "WHAT " + name + " OFFERS YOU NOW"
	}
	return "WHAT " + name + " UNLOCKS"
}

func (h *HUD) drawUnlockPane(dst *ebiten.Image, a view.AgeStep, current string) {
	pane := roadPane()
	ui.Fill(dst, ui.Rect{X: pane.X, Y: pane.Y - 6, W: pane.W, H: 1}, ui.Palette.Gold)
	if a.ID == "" {
		return
	}
	h.Font.Draw(dst, ui.Clip(unlockTitle(a, current), ui.MaxChars(pane.W, 1)), pane.X, pane.Y, 1, ui.Palette.Gold)
	colW := (pane.W - roadGap) / roadPaneCols
	items := make([]string, len(a.Unlocks))
	for i, item := range a.Unlocks {
		items[i] = Unechoed(item)
	}
	if len(items) == 0 {
		items = []string{"the end of the road"}
	}
	for i, col := range RoadPaneColumns(items, colW, roadPaneRows, roadPaneCols) {
		h.drawRows(dst, col, pane.X+i*(colW+roadGap), pane.Y+14, ui.Palette.Cream)
	}
}

func (h *HUD) drawLedgerPane(dst *ebiten.Image, f Frame) {
	r := roadRight()
	ui.Fill(dst, ui.Rect{X: r.X - roadRightGap/2, Y: r.Y, W: 1, H: RoadBox.Y + RoadBox.H - 34 - r.Y}, ui.Palette.Gold)
	keeps, loses := ReforgeLedger(f)
	k, l := LedgerLines(keeps, loses, r.W, roadLedgerRows-2)
	h.Font.Draw(dst, "A REFORGE KEEPS", r.X, r.Y, 1, ui.Palette.Green)
	h.drawRows(dst, k, r.X, r.Y+ui.LineHeight+2, ui.Palette.Text)
	y := r.Y + (len(k)+1)*ui.LineHeight + 8
	h.Font.Draw(dst, "A REFORGE LOSES", r.X, y, 1, ui.Palette.RedText)
	h.drawRows(dst, l, r.X, y+ui.LineHeight+2, ui.Palette.Text)
}
