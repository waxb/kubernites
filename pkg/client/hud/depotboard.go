package hud

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	dbInfoY    = 18
	dbFanY     = 32
	dbFanH     = 80
	dbButtonY  = 116
	dbQuayW    = 44
	dbDepotX   = 56
	dbDepotW   = 64
	dbHutsX    = 160
	dbFullMax  = 6
	dbFullStep = 44
	dbWideStep = 80
	dbStagger  = 26
	dbMiniStep = 18
	dbMiniRows = 2
	dbMiniMax  = 24
	dbCrateW   = 6
	dbCrates   = 12
	dbDotSpeed = 0.6
	dbDotGap   = 0.37
)

var (
	laneLit  = ui.Palette.GoldBright
	laneDim  = color.RGBA{90, 96, 92, 255}
	crateInk = color.RGBA{168, 124, 70, 255}
)

type depotView struct {
	chain model.CrewChain
	flow  depotFlow
	Known bool            `json:"-"`
	Pods  []model.PodView `json:"-"`
	box   ui.Rect
	fan   ui.Rect
	Huts  []Button `json:"-"`
}

func (h *HUD) DepotView(f Frame) (depotView, bool) {
	c, ok := ChainOf(f)
	if !ok || !h.ChainShown(f) || c.Bare || !h.depotBoardOn(f, c) || h.CrewMode(c) != CrewModeChain {
		return depotView{}, false
	}
	d := depotView{chain: c, Pods: crewPods(c, f)}
	d.flow, d.Known = depotFlowOf(f.State.Freight, c.Depot)
	height := dbButtonY + crewBtnH + cbPad
	d.box = f.placed(PanelCrew, ui.Rect{X: CardBox.X, Y: ConsoleY - crewFloat - height, W: cbW, H: height})
	d.fan = ui.Rect{X: d.box.X + cbPad, Y: d.box.Y + dbFanY, W: cbW - 2*cbPad, H: dbFanH}
	d.Huts = d.endpointButtons(f)
	return d, true
}

func (d depotView) HutRects() ([]ui.Rect, bool) {
	n := len(d.Pods)
	x0 := d.fan.X + dbHutsX
	if n <= dbFullMax {
		step := min(max((d.fan.X+d.fan.W-x0-dbFullStep)/max(n-1, 1), dbFullStep/2), dbWideStep)
		out := make([]ui.Rect, n)
		for i := range out {
			out[i] = ui.Rect{X: x0 + i*step, Y: d.fan.Y + (i%dbMiniRows)*dbStagger, W: dbFullStep - 2, H: d.fan.H - dbStagger}
		}
		return out, true
	}
	shown := min(n, dbMiniMax)
	per := (shown + dbMiniRows - 1) / dbMiniRows
	out := make([]ui.Rect, shown)
	for i := range out {
		row, col := i/per, i%per
		out[i] = ui.Rect{X: x0 + col*dbMiniStep, Y: d.fan.Y + 10 + row*(d.fan.H/dbMiniRows), W: dbMiniStep - 2, H: d.fan.H/dbMiniRows - 4}
	}
	return out, false
}

func (d depotView) endpointButtons(f Frame) []Button {
	rects, _ := d.HutRects()
	out := make([]Button, 0, len(rects))
	for i, r := range rects {
		p := d.Pods[i]
		tip := fmt.Sprintf(ui.Bridge("%s: a ready hut, the depot sends it boxes.", "%s: a ready endpoint of the Service."), p.Name)
		if !p.Ready {
			tip = fmt.Sprintf(ui.Bridge("%s: not ready, so the depot sends it nothing.", "%s: not Ready, so it is not an endpoint."), p.Name)
		}
		out = append(out, Button{ID: CrewPrefix + ":pick:pod:" + p.Name, Rect: r, Tip: tip,
			Selected: f.Selection.Kind != "structure" && f.Selection.Pod.Name == p.Name})
	}
	return out
}

func (d depotView) buttons() []Button {
	out := []Button{closeButtonAs(d.box, CrewPrefix+":close", "Deselect [Esc or right click]")}
	return append(out, d.Huts...)
}

func (d depotView) ReadyCount() int {
	n := 0
	for _, p := range d.Pods {
		if p.Ready {
			n++
		}
	}
	return n
}

func DepotInfo(d depotView) string {
	if !d.Known {
		return ui.Bridge("No boxes of its kind land yet.", "No requests of its kind arrive yet.")
	}
	return fmt.Sprintf(ui.Bridge("%s: %d WAIT ON THE QUAY, %s/S CARRIED OF %s", "%s: %d QUEUED, %s/S SERVED OF %s"),
		strings.ToUpper(d.flow.name), int(math.Round(d.flow.waiting)), Amount(d.flow.served), Amount(d.flow.arrival))
}

func (h *HUD) drawDepotBoard(dst *ebiten.Image, d depotView, f Frame) {
	ui.DarkPanel(dst, d.box)
	x, y := d.box.X+cbPad, d.box.Y+cbTitleY
	title := ui.Bridge("DEPOT ", "SERVICE ") + d.chain.Depot
	h.Font.DrawShadow(dst, title, x, y, 1, ui.Palette.GoldBright)
	h.Font.Draw(dst, ui.Bridge("(Service)", ""), x+h.Font.Width(title, 1)+ui.Advance, y, 1, ui.Palette.Blue)
	h.Font.Draw(dst, ui.Clip(DepotInfo(d), ui.MaxChars(d.box.W-2*cbPad, 1)), x, d.box.Y+dbInfoY, 1, ShareColor(CarriedShare(d.flow.arrival, d.flow.served, d.flow.waiting)))
	h.drawFan(dst, d, f)
	ready := fmt.Sprintf(ui.Bridge("%d OF %d HUTS READY: THE LIT ONES GET BOXES", "ENDPOINTS: %d OF %d PODS READY"), d.ReadyCount(), len(d.Pods))
	h.Font.Draw(dst, ready, x, d.box.Y+dbButtonY+4, 1, ui.Palette.TextDim)
	h.drawButton(dst, closeButtonAs(d.box, CrewPrefix+":close", ""))
}

type FanLane struct {
	From, To [2]float32
	Lit      bool
	Pod      string
}

func (d depotView) depotRect() ui.Rect {
	mid := d.fan.Y + d.fan.H/2
	return ui.Rect{X: d.fan.X + dbDepotX, Y: mid - dbDepotW/2, W: dbDepotW, H: dbDepotW}
}

func (d depotView) Lanes() []FanLane {
	depot := d.depotRect()
	mid := float32(depot.Y + depot.H/2)
	out := []FanLane{{From: [2]float32{float32(d.fan.X + dbQuayW), mid}, To: [2]float32{float32(depot.X), mid}, Lit: d.flow.served > 0}}
	rects, _ := d.HutRects()
	from := [2]float32{float32(depot.X + depot.W), mid}
	for i, r := range rects {
		p := d.Pods[i]
		out = append(out, FanLane{From: from, To: [2]float32{float32(r.X + r.W/2), float32(r.Y + r.H - 6)}, Lit: p.Ready, Pod: p.Name})
	}
	return out
}

func (h *HUD) drawFan(dst *ebiten.Image, d depotView, f Frame) {
	depot := d.depotRect()
	drawCrates(dst, d.fan.X+2, depot.Y+depot.H/2+8, int(math.Round(d.flow.waiting)))
	h.drawCardPortrait(dst, gfx.StructurePortrait(f.Selection.Structure, gfx.AgeTier(f.State.Age)), depot)
	for i, lane := range d.Lanes() {
		h.drawLane(dst, lane.From, lane.To, lane.Lit, float64(i)*dbDotGap)
	}
	rects, full := d.HutRects()
	for i, r := range rects {
		h.drawEndpoint(dst, d, d.Pods[i], r, full, f)
	}
}

func (h *HUD) drawLane(dst *ebiten.Image, a, b [2]float32, lit bool, phase float64) {
	if !lit {
		vector.StrokeLine(dst, a[0], a[1], b[0], b[1], 1, laneDim, false)
		return
	}
	vector.StrokeLine(dst, a[0], a[1], b[0], b[1], 2, laneLit, false)
	t := float32(0.5)
	if !h.Reduce {
		t = float32(math.Mod(h.Now*dbDotSpeed+phase, 1))
	}
	x, y := a[0]+(b[0]-a[0])*t, a[1]+(b[1]-a[1])*t
	ui.Fill(dst, ui.Rect{X: int(x) - 2, Y: int(y) - 2, W: 4, H: 4}, crateInk)
	ui.Stroke(dst, ui.Rect{X: int(x) - 2, Y: int(y) - 2, W: 4, H: 4}, ui.Palette.Ink)
}

func (h *HUD) drawEndpoint(dst *ebiten.Image, d depotView, p model.PodView, r ui.Rect, full bool, f Frame) {
	o := render.SpriteOptions{}
	if !p.Ready {
		o = render.SpriteOptions{Alpha: leaveAlpha}
	}
	hot := h.rowHot(CrewPrefix+":pick:pod:"+p.Name) || f.Selection.Kind != "structure" && f.Selection.Pod.Name == p.Name
	if !full {
		drawHutIcon(dst, ui.Rect{X: r.X + 2, Y: r.Y + r.H - miniHutH - 2, W: r.W - 4, H: miniHutH}, slotColor(Slot{Kind: SlotStanding, Pod: p, Real: true}), hot)
		return
	}
	names := gfx.HearthLookNames(p.Tier, p.State, p.QoS, gfx.AgeTier(f.State.Age), gfx.LookVariant(p.Name))
	if _, ok := h.Lib.DrawFirst(dst, names, float64(r.X+r.W/2), float64(r.Y+r.H-2), 1, o); !ok {
		drawHutIcon(dst, ui.Rect{X: r.X + 8, Y: r.Y + r.H - 18, W: r.W - 16, H: 16}, slotColor(Slot{Kind: SlotStanding, Pod: p, Real: true}), false)
	}
	if hot {
		ui.Stroke(dst, r, ui.Palette.GoldBright)
	}
}

func drawCrates(dst *ebiten.Image, x, y, n int) {
	for i := range min(n, dbCrates) {
		row, col := i/4, i%4
		c := ui.Rect{X: x + col*(dbCrateW+1) + row*3, Y: y - row*(dbCrateW-1), W: dbCrateW, H: dbCrateW - 1}
		ui.Fill(dst, c, crateInk)
		ui.Stroke(dst, c, ui.Palette.Ink)
	}
}
