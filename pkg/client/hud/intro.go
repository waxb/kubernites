package hud

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	IntroW        = 300
	introPad      = 8
	introLabelW   = 34
	introGap      = 26
	introDim      = 160
	HolePad       = 3
	introButtonW  = 64
	IntroEdge     = 4
	introAlmanacW = 7*ui.Advance + 12
	introWhyMax   = 4
	introTryMax   = 2
	IntroAck      = introPrefix + ":ack"
	IntroShow     = introPrefix + ":show"
	moreW         = 44
	introRowH     = CloseFace + 2*ClosePad
	introRowGap   = 3
	scanStep      = 12
)

var surfaceRects = map[string]func(*HUD, Frame) ui.Rect{
	SurfaceTidings: func(*HUD, Frame) ui.Rect { return InfoWell },
	SurfaceMinimap: func(*HUD, Frame) ui.Rect { return Minimap },
	SurfaceGate:    func(*HUD, Frame) ui.Rect { return CardBox },
	SurfaceRail: func(h *HUD, f Frame) ui.Rect {
		if r := h.RailRect(f); r.W > 0 {
			return r
		}
		return RailSlot(0)
	},
	SurfaceSpeed:    speedRect,
	SurfaceFreight:  func(_ *HUD, f Frame) ui.Rect { return FreightRect(f) },
	SurfaceLosses:   func(_ *HUD, f Frame) ui.Rect { return LossesRect(f) },
	SurfaceOverlays: func(*HUD, Frame) ui.Rect { return LensButtonRect() },
	SurfaceHeraldry: func(*HUD, Frame) ui.Rect { return LensButtonRect() },
	SurfaceQoS:      qosRect,
	SurfaceDecree:   decreeAckRect,
	MeterUpkeep:     goldStatRect,
}

func goldStatRect(h *HUD, f Frame) ui.Rect {
	for _, st := range h.TopStats(f) {
		if st.Meter == MeterGold {
			return ui.Rect{X: st.X - 2, Y: 1, W: h.StatWidth(st) + 4, H: TopH - 3}
		}
	}
	return ui.Rect{}
}

func decreeAckRect(h *HUD, f Frame) ui.Rect {
	if b, ok := HitID(h.questButtons(f), questAck); ok {
		return b.Rect
	}
	return ui.Rect{}
}

func qosRect(_ *HUD, f Frame) ui.Rect {
	if !hearthSelected(f) && len(f.Beacons) > 0 {
		return f.Beacons[0]
	}
	return CardBox
}

func speedRect(h *HUD, f Frame) ui.Rect {
	var out ui.Rect
	for _, b := range h.TopButtons(f) {
		if !strings.HasPrefix(b.ID, "speed:") {
			continue
		}
		if out.W == 0 {
			out = b.Rect
			continue
		}
		right := max(out.X+out.W, b.Rect.X+b.Rect.W)
		out.X = min(out.X, b.Rect.X)
		out.W = right - out.X
	}
	return out
}

var surfaceButtons = map[string]string{SurfaceStone: "tab:" + TabKube, SurfaceScriptorium: "tab:" + TabYAML, SurfaceLedger: LedgerButton}

func (h *HUD) TargetRect(target string, f Frame) (ui.Rect, bool) {
	if get, ok := surfaceRects[target]; ok {
		return get(h, f), true
	}
	id := target
	if b, ok := surfaceButtons[target]; ok {
		id = b
	}
	if strings.HasPrefix(target, "meter:") {
		for _, st := range h.TopStats(f) {
			if st.Meter == target {
				return ui.Rect{X: st.X - 2, Y: 1, W: h.StatWidth(st) + 4, H: TopH - 3}, true
			}
		}
	}
	if b, ok := HitID(h.fixedButtons(f), id); ok {
		return b.Rect, true
	}
	return ui.Rect{}, false
}

func HitID(buttons []Button, id string) (Button, bool) {
	for _, b := range buttons {
		if b.ID == id {
			return b, true
		}
	}
	return Button{}, false
}

func (h *HUD) IntroShown(f Frame) bool {
	return f.Introducing() && !f.Unguided && !placingGuideTarget(f) && h.OverlayID != OverlayWelcome && !h.QuestCardShown(f) && !h.troubleUp(f) && !h.leavesWindow(f, h.voiceGoal(f, VoiceIntro)) && !h.introOutsideWindow(f)
}

func (h *HUD) introOutsideWindow(f Frame) bool {
	w, open := h.windowRect()
	if !open || h.ShortStone() {
		return false
	}
	target, ok := h.IntroHole(f)
	return !ok || !w.Contains(target.X+target.W/2, target.Y+target.H/2)
}

func (h *HUD) introCovers(f Frame) bool {
	return f.Introducing() && !h.troubleUp(f)
}

func (h *HUD) troubleUp(f Frame) bool {
	_, up := h.ShownCalamity(f)
	return up
}

func (h *HUD) IntroLines(in Intro) ([]string, []string) {
	w := IntroW - 2*introPad - introLabelW
	why := []string{}
	if in.Purpose != "" {
		why = h.Font.Wrap(in.Purpose, w, 1)
	}
	return why, h.Font.Wrap(QuoteButton(in.Try, in.Target), w, 1)
}

func (h *HUD) IntroTitle(in Intro) []string {
	return h.Font.Wrap(in.Title, IntroW-2*introPad-introLabelW, 1)
}

func (h *HUD) introLore(in Intro) []string {
	if in.Lore == "" {
		return nil
	}
	return h.Font.Wrap(in.Lore, IntroW-2*introPad-introLabelW, 1)
}

func (h *HUD) introBodyHeight(in Intro) int {
	why, try := h.IntroLines(in)
	rows := max(len(h.IntroTitle(in)), 1) + len(why) + len(try)
	gaps := 3
	if lore := h.introLore(in); len(lore) > 0 {
		rows += len(lore)
		gaps++
	}
	return introPad*2 + rows*ui.LineHeight + gaps*introRowGap
}

func (h *HUD) IntroCard(f Frame) ui.Rect {
	in := *f.Guide.Intro
	height := h.introBodyHeight(in) + 18
	target, ok := h.IntroHole(f)
	return f.placed(PanelIntro, h.placeGuideCard(f, target, ok, IntroW, height))
}

func (h *HUD) placeGuideCard(f Frame, target ui.Rect, ok bool, w, height int) ui.Rect {
	avoid := append(h.objectiveAvoid(f), inflated(FreightRects(f), 2)...)
	card := PlaceCard(target, ok, w, height, avoid...)
	if blockedArea(card, avoid) == 0 {
		return h.withClearLine(f, card, target, ok, avoid)
	}
	hard := append(append([]ui.Rect{}, avoid...), FreightRects(f)...)
	if ok {
		hard = append(hard, target.Inset(-HolePad-introGap/2))
	}
	if r := h.RailRect(f); r.W > 0 {
		hard = append(hard, r)
	}
	return scanClear(card, w, height, hard)
}

func (h *HUD) withClearLine(f Frame, card, target ui.Rect, ok bool, avoid []ui.Rect) ui.Rect {
	if !ok || h.lineClear(f, card, target) {
		return card
	}
	blocked := append([]ui.Rect{target.Inset(-HolePad - introGap/2)}, avoid...)
	for _, c := range clampAll(around(target, card.W, card.H, introGap)) {
		if blockedArea(c, blocked) == 0 && h.lineClear(f, c, target) {
			return c
		}
	}
	return card
}

func (h *HUD) lineBlockers(f Frame, target ui.Rect) []ui.Rect {
	all := h.crewControls(f)
	if h.ObjectiveShown(f) {
		all = append(all, h.ObjectiveRectOf(f))
	}
	if h.LessonShown(f) {
		all = append(all, h.LessonRect(f))
	}
	all = append(all, FreightRects(f)...)
	if r := h.RailRect(f); r.W > 0 {
		all = append(all, r)
	}
	hole := target.Inset(-HolePad - 2)
	out := make([]ui.Rect, 0, len(all))
	for _, r := range all {
		if !Overlaps(r, hole) {
			out = append(out, r)
		}
	}
	return out
}

func (h *HUD) lineClear(f Frame, card, target ui.Rect) bool {
	line, ok := pointerLine(card, target)
	if !ok {
		return true
	}
	for _, r := range h.lineBlockers(f, target) {
		if lineHits(line, r) {
			return false
		}
	}
	return true
}

func lineHits(line [2][2]float32, r ui.Rect) bool {
	_, ok := lineEntry(line, r)
	return ok
}

func lineEntry(line [2][2]float32, r ui.Rect) (float32, bool) {
	lo, hi := float32(0), float32(1)
	a, b := line[0], line[1]
	dx, dy := b[0]-a[0], b[1]-a[1]
	for _, e := range [][2]float32{{-dx, a[0] - float32(r.X)}, {dx, float32(r.X+r.W) - a[0]}, {-dy, a[1] - float32(r.Y)}, {dy, float32(r.Y+r.H) - a[1]}} {
		p, q := e[0], e[1]
		if p == 0 {
			if q < 0 {
				return 0, false
			}
			continue
		}
		if u := q / p; p < 0 {
			lo = max(lo, u)
		} else {
			hi = min(hi, u)
		}
		if lo > hi {
			return 0, false
		}
	}
	return lo, true
}

func scanClear(fallback ui.Rect, w, height int, blocked []ui.Rect) ui.Rect {
	cx, cy := fallback.X+w/2, fallback.Y+height/2
	best, bestDist := fallback, -1
	for y := TopH + cardMargin; y <= ConsoleY-height-cardMargin; y += scanStep {
		for x := cardMargin; x <= Width-w-cardMargin; x += scanStep {
			r := ui.Rect{X: x, Y: y, W: w, H: height}
			if blockedArea(r, blocked) > 0 {
				continue
			}
			dx, dy := x+w/2-cx, y+height/2-cy
			if d := dx*dx + dy*dy; bestDist < 0 || d < bestDist {
				best, bestDist = r, d
			}
		}
	}
	return best
}

func (h *HUD) objectiveAvoid(f Frame) []ui.Rect {
	out := inflated(h.crewControls(f), 2)
	if h.LessonShown(f) {
		out = append(out, h.LessonRect(f).Inset(-introGap/2))
	}
	if r, ok := h.CalamityRect(f); ok {
		out = append(out, r.Inset(-introGap/2))
	}
	if !h.ObjectiveShown(f) {
		return out
	}
	return append(out, h.ObjectiveRectOf(f).Inset(-introGap/2))
}

func PlaceCard(target ui.Rect, ok bool, w, height int, avoid ...ui.Rect) ui.Rect {
	x, y := Width/2-w/2, (TopH+ConsoleY)/2-height/2
	if !ok {
		return clearOfAll(ui.Rect{X: x, Y: y, W: w, H: height}, avoid)
	}
	switch {
	case target.Y >= ConsoleY-introGap:
		x, y = target.X+target.W/2-w/2, ConsoleY-height-introGap
	case target.Y < TopH+4:
		x, y = target.X+target.W/2-w/2, TopH+introGap
	default:
		x, y = target.X-w-introGap, target.Y
	}
	candidates := append([]ui.Rect{{X: x, Y: y, W: w, H: height}}, around(target, w, height, introGap)...)
	return FirstClear(clampAll(candidates), append([]ui.Rect{target.Inset(-HolePad - introGap/2)}, avoid...))
}

func clearOfAll(r ui.Rect, avoid []ui.Rect) ui.Rect {
	r = clampCard(r)
	if blockedArea(r, avoid) == 0 {
		return r
	}
	candidates := []ui.Rect{r}
	for _, a := range avoid {
		candidates = append(candidates, ui.Rect{X: a.X + a.W + introGap, Y: r.Y, W: r.W, H: r.H}, ui.Rect{X: r.X, Y: a.Y + a.H + introGap, W: r.W, H: r.H})
	}
	return FirstClear(clampAll(candidates), avoid)
}

func (h *HUD) IntroButtons(f Frame) []Button {
	if !h.IntroShown(f) {
		return nil
	}
	card := h.IntroCard(f)
	face := card.Y + card.H - 18 - 4
	row := introRow(card)
	lefts := h.introLeftButtons(f, card, face)
	if _, ok := introPage(f); !ok {
		return exactRow(facedRow(spanRow(lefts, row)))
	}
	almanac := Button{ID: IntroAlmanac, Face: ui.Rect{X: card.X + card.W - IntroEdge - introAlmanacW, Y: face, W: introAlmanacW, H: 16}, Label: "ALMANAC"}
	row.W /= 2
	return exactRow(append([]Button{almanac}, centeredRow(lefts, row)...))
}

func (h *HUD) introLeftButtons(f Frame, card ui.Rect, face int) []Button {
	out := []Button{}
	if h.IntroHidden(f) && f.GoalAt != nil {
		out = append(out, Button{ID: IntroShow, Face: ui.Rect{X: card.X + introPad, Y: face, W: moreW, H: 16}, Label: "SHOW"})
	}
	if tray, ok := h.trayButton(f, card, face); ok {
		tray.Tip = ""
		out = append(out, tray)
	}
	r := ui.Rect{X: card.X + card.W - introButtonW - introPad, Y: face, W: introButtonW, H: 16}
	return append(out, Button{ID: IntroAck, Face: r, Label: "GOT IT"})
}

func exactRow(buttons []Button) []Button {
	out := make([]Button, 0, len(buttons))
	for _, b := range buttons {
		next := b
		next.Rect = b.Face
		out = append(out, next)
	}
	return out
}

func centeredRow(buttons []Button, row ui.Rect) []Button {
	out := make([]Button, 0, len(buttons))
	for i, b := range buttons {
		share := row.W / len(buttons)
		next := b
		next.Face.X = row.X + i*share + (share-b.Face.W)/2
		out = append(out, next)
	}
	return out
}

func facedRow(buttons []Button) []Button {
	out := make([]Button, 0, len(buttons))
	for _, b := range buttons {
		next := b
		if !b.Rect.Contains(b.Face.X, b.Face.Y) || !b.Rect.Contains(b.Face.X+b.Face.W-1, b.Face.Y) {
			w := min(b.Face.W, b.Rect.W-4)
			next.Face = ui.Rect{X: b.Rect.X + (b.Rect.W-w)/2, Y: b.Face.Y, W: w, H: b.Face.H}
		}
		out = append(out, next)
	}
	return out
}

func introRow(card ui.Rect) ui.Rect {
	return ui.Rect{X: card.X + 2, Y: card.Y + card.H - introRowH - 2, W: card.W - 4, H: introRowH}
}

func spanRow(buttons []Button, row ui.Rect) []Button {
	out := make([]Button, 0, len(buttons))
	x := row.X
	for i, b := range buttons {
		w := (row.X + row.W - x) / (len(buttons) - i)
		next := b
		next.Rect = ui.Rect{X: x, Y: row.Y, W: w, H: row.H}
		out = append(out, next)
		x += w
	}
	return out
}

func (h *HUD) PressIntro(rest string, f Frame) []Intent {
	if !f.Introducing() {
		return nil
	}
	switch rest {
	case "almanac":
		p, _ := introPage(f)
		return h.OpenBook(p.ID, f)
	case "show":
		if f.GoalAt != nil {
			return h.focus(*f.GoalAt)
		}
	case "tray":
		return h.toggleTray()
	case "ack":
		h.noteAsk(actIntro+f.Guide.Intro.ID, f.State.RealmTime)
		if introAtHome(f) && f.cardSelection() && !f.goalOnSelection() {
			h.grid.sel, h.grid.home = selKey(f.Selection), true
		}
		return []Intent{act(Ack(f.Guide.Intro.ID)), {Kind: IntentSound, Command: "click"}}
	}
	return nil
}

func dimAround(dst *ebiten.Image, hole ui.Rect, ok bool) {
	dimArea(dst, ui.Rect{W: Width, H: Height}, hole.Inset(-HolePad), ok, introDim)
}

func dimArea(dst *ebiten.Image, area, hole ui.Rect, cut bool, alpha uint8) {
	shade := ui.Tint(ui.Palette.Shadow, alpha)
	for _, r := range DimRects(area, hole, cut) {
		ui.Fill(dst, r, shade)
	}
}

func (h *HUD) drawIntro(dst *ebiten.Image, f Frame) {
	h.paintSurfaces(dst, introSurfaces(h, f))
}

func (h *HUD) strokeHole(dst *ebiten.Image, target ui.Rect) {
	a := uint8(120 + 135*pulse(h.Now, h.Reduce))
	ui.Stroke(dst, target.Inset(-HolePad), ui.Tint(ui.Palette.GoldBright, a))
	ui.Stroke(dst, target.Inset(-HolePad-1), ui.Tint(ui.Palette.Gold, a/2))
}

func (h *HUD) DrawIntroCard(dst *ebiten.Image, card ui.Rect, in Intro) {
	ui.Fill(dst, card.Offset(2, 3), ui.Palette.Shadow)
	ui.Fill(dst, card, ui.Palette.Console)
	ui.Stroke(dst, card, ui.Palette.GoldBright)
	ui.Stroke(dst, card.Inset(2), ui.Palette.Gold)
	x, y := card.X+introPad, card.Y+introPad
	why, try := h.IntroLines(in)
	y = h.drawLabelled(dst, "WHAT", h.IntroTitle(in), x, y, ui.Palette.GoldBright) + introRowGap
	if lore := h.introLore(in); len(lore) > 0 {
		y = h.drawLabelled(dst, "", lore, x, y, ui.Palette.TextDim) + introRowGap
	}
	y = h.drawLabelled(dst, "FOR", why, x, y, ui.Palette.Cream) + introRowGap
	h.drawLabelled(dst, "TRY", try, x, y, ui.Palette.Green)
}

func (h *HUD) drawLabelled(dst *ebiten.Image, label string, rows []string, x, y int, clr color.Color) int {
	for i, row := range rows {
		if i == 0 {
			h.Font.Draw(dst, label, x, y, 1, ui.Palette.Gold)
		}
		h.Font.Draw(dst, row, x+introLabelW, y, 1, clr)
		y += ui.LineHeight
	}
	return y
}

const (
	pointerClear = 2
	minPointer   = 12
)

func pointerLine(card, target ui.Rect) ([2][2]float32, bool) {
	cx, cy := float32(card.X+card.W/2), float32(card.Y+card.H/2)
	tx, ty := float32(target.X+target.W/2), float32(target.Y+target.H/2)
	sx, sy := ClipToRect(cx, cy, tx, ty, card)
	ex, ey := ClipToRect(tx, ty, cx, cy, target.Inset(-HolePad-2))
	if math.Hypot(float64(ex-sx), float64(ey-sy)) < 4 {
		return [2][2]float32{}, false
	}
	return [2][2]float32{{sx, sy}, {ex, ey}}, true
}

func (h *HUD) IntroLine(f Frame) ([2][2]float32, bool) {
	f = h.Stage(f)
	if !h.IntroShown(f) {
		return [2][2]float32{}, false
	}
	target, ok := h.IntroHole(f)
	card := h.IntroCard(f)
	if !ok || !h.lineClear(f, card, target) {
		return [2][2]float32{}, false
	}
	return h.pointerPath(f, card, target)
}

func (h *HUD) pointerPath(f Frame, card, target ui.Rect) ([2][2]float32, bool) {
	line, ok := pointerLine(card, target)
	if !ok {
		return line, false
	}
	hole := target.Inset(-HolePad - 2)
	end := float32(1)
	for _, b := range h.CommandButtons(f) {
		if t, hit := lineEntry(line, b.Rect.Inset(-pointerClear)); hit && !Overlaps(b.Rect, hole) {
			end = min(end, t)
		}
	}
	a, b := line[0], line[1]
	b = [2]float32{a[0] + (b[0]-a[0])*end, a[1] + (b[1]-a[1])*end}
	if math.Hypot(float64(b[0]-a[0]), float64(b[1]-a[1])) < minPointer {
		return line, false
	}
	return [2][2]float32{a, b}, true
}

func (h *HUD) drawClearPointer(dst *ebiten.Image, f Frame, card, target ui.Rect) {
	if !h.lineClear(f, card, target) {
		return
	}
	if line, ok := h.pointerPath(f, card, target); ok {
		drawPointer(dst, line)
	}
}

func drawPointer(dst *ebiten.Image, line [2][2]float32) {
	sx, sy, ex, ey := line[0][0], line[0][1], line[1][0], line[1][1]
	vector.StrokeLine(dst, sx, sy, ex, ey, 2, ui.Palette.GoldBright, false)
	a := math.Atan2(float64(ey-sy), float64(ex-sx))
	var p vector.Path
	p.MoveTo(ex, ey)
	p.LineTo(ex-7*float32(math.Cos(a-.45)), ey-7*float32(math.Sin(a-.45)))
	p.LineTo(ex-7*float32(math.Cos(a+.45)), ey-7*float32(math.Sin(a+.45)))
	p.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(ui.Palette.GoldBright)
	vector.FillPath(dst, &p, &vector.FillOptions{}, op)
}

func ClipToRect(fromX, fromY, towardX, towardY float32, r ui.Rect) (float32, float32) {
	dx, dy := towardX-fromX, towardY-fromY
	t := float32(1)
	if dx != 0 {
		for _, edge := range []float32{float32(r.X), float32(r.X + r.W)} {
			if s := (edge - fromX) / dx; s > 0 && s < t {
				t = s
			}
		}
	}
	if dy != 0 {
		for _, edge := range []float32{float32(r.Y), float32(r.Y + r.H)} {
			if s := (edge - fromY) / dy; s > 0 && s < t {
				t = s
			}
		}
	}
	return fromX + dx*t, fromY + dy*t
}
