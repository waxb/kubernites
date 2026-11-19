package hud

import (
	"image/color"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func (h *HUD) OverlayRect() ui.Rect {
	switch h.OverlayID {
	case OverlayMenu:
		return h.menuBox()
	case OverlayLiveChoices:
		return ChoicesBox
	case OverlayStone:
		return h.StoneRect()
	case OverlaySiege:
		return SiegeBox
	case OverlayWelcome:
		return WelcomeBox
	case OverlayRoad, OverlayEnding:
		return RoadBox
	case OverlayGrimoire:
		return KnowledgeBox
	case OverlayChooser:
		return h.ChooserRect()
	case OverlayRaise:
		return RaiseRect()
	case OverlayBook:
		return BookBox
	case OverlayTrials:
		return h.TrialsRect()
	case OverlayChapters:
		return KnowledgeBox
	case OverlayDebrief:
		return h.DebriefRect()
	}
	return Overlay
}

type OverlaySpec struct {
	Buttons func(*HUD, Frame) []Button       `json:"-"`
	Draw    func(*HUD, *ebiten.Image, Frame) `json:"-"`
}

var Overlays = map[string]OverlaySpec{
	OverlayDetails:     {func(h *HUD, _ Frame) []Button { return h.detailButtons() }, func(h *HUD, d *ebiten.Image, _ Frame) { h.drawDetails(d) }},
	OverlayStone:       {func(h *HUD, _ Frame) []Button { return h.StoneButtons() }, func(h *HUD, d *ebiten.Image, _ Frame) { h.drawStone(d) }},
	OverlayScriptorium: {func(h *HUD, _ Frame) []Button { return h.scriptButtons() }, func(h *HUD, d *ebiten.Image, _ Frame) { h.drawScriptorium(d) }},
	OverlayGrimoire:    {(*HUD).GrimoireButtons, (*HUD).drawGrimoire},
	OverlaySoulvault:   {(*HUD).AcademyButtons, (*HUD).DrawAcademy},
	OverlayChronicles:  {(*HUD).ChronicleButtons, (*HUD).DrawChronicles},
	OverlayAge:         {(*HUD).ageButtons, (*HUD).DrawAge},
	OverlayReforge:     {func(h *HUD, _ Frame) []Button { return h.ReforgeButtons() }, (*HUD).DrawReforge},
	OverlayMenu:        {(*HUD).MenuButtons, (*HUD).drawMenu},
	OverlaySiege:       {(*HUD).SiegeButtons, (*HUD).DrawSiege},
	OverlayWelcome:     {(*HUD).welcomeButtons, (*HUD).drawWelcome},
	OverlayRoad:        {(*HUD).roadOverlayButtons, (*HUD).drawRoad},
	OverlayVoyage:      {(*HUD).voyageOverlayButtons, (*HUD).drawVoyage},
	OverlayCouncil:     {(*HUD).CouncilButtons, (*HUD).drawCouncil},
	OverlayLedger:      {(*HUD).ledgerButtons, (*HUD).drawLedger},
	OverlayChooser:     {(*HUD).chooserButtons, (*HUD).drawChooser},
	OverlayBook:        {(*HUD).BookButtons, (*HUD).drawBook},
	OverlayTrials:      {(*HUD).TrialsButtons, (*HUD).DrawTrials},
}

func (h *HUD) OverlayButtons(f Frame) []Button {
	if spec, ok := Overlays[h.OverlayID]; ok {
		return spec.Buttons(h, f)
	}
	return nil
}

func (h *HUD) Buttons(f Frame) []Button {
	f = h.Stage(f)
	return h.liveGated(ChapterTips(WithLeads(append(append(append(h.baseButtons(f), h.introWindowButtons(f)...), h.questButtons(f)...), h.tipButtons()...), f), f), f)
}

func (h *HUD) fixedButtons(f Frame) []Button {
	out := h.StatButtons(f)
	out = append(out, h.TopButtons(f)...)
	out = append(out, h.CommandButtons(f)...)
	out = append(out, h.wellButtons(f)...)
	out = append(out, h.SelectionButtons(f)...)
	out = append(out, h.realmCardButtons(f)...)
	out = append(out, h.RailButtons(f)...)
	return append(out, h.legendButtons(f)...)
}

func (h *HUD) baseButtons(f Frame) []Button {
	out := h.CauseButtons(f)
	out = append(out, h.StatButtons(f)...)
	out = append(out, h.TopButtons(f)...)
	out = append(out, h.CommandButtons(f)...)
	out = append(out, h.wellButtons(f)...)
	out = append(out, h.SelectionButtons(f)...)
	out = append(out, h.realmCardButtons(f)...)
	out = append(out, h.alertOverlayButtons(f)...)
	out = append(out, h.windowButtons(f)...)
	out = append(out, h.legendButtons(f)...)
	out = append(out, h.lensCycleButtons(f)...)
	if h.OverlayID != OverlayNone {
		out = append(out, Button{ID: "overlay:body", Rect: h.OverlayRect()})
		out = append(out, h.OverlayButtons(f)...)
	}
	return out
}

func (h *HUD) drawOverlay(dst *ebiten.Image, f Frame) {
	if spec, ok := Overlays[h.OverlayID]; ok {
		spec.Draw(h, dst, f)
	}
}

func (h *HUD) header(dst *ebiten.Image, r ui.Rect, title string) {
	h.Font.DrawShadow(dst, title, r.X+10, r.Y+6, 2, ui.Palette.Gold)
}

func closeButton(r ui.Rect) Button {
	return closeButtonAs(r, "overlay:close", "Close [Esc]")
}

func closeButtonAs(r ui.Rect, id, tip string) Button {
	hit := ui.Rect{X: r.X + r.W - CloseFace - 2*ClosePad - closeInset, Y: r.Y + closeInset, W: CloseFace + 2*ClosePad, H: CloseFace + 2*ClosePad}
	return Button{ID: id, Rect: hit, Face: hit.Inset(ClosePad), Label: "X", Tip: tip}
}

func IsClose(b Button) bool {
	return b.Label == "X" && strings.HasSuffix(b.ID, ":close")
}

func (h *HUD) DrawOverlayButtons(dst *ebiten.Image, buttons []Button) {
	for _, b := range buttons {
		h.drawButton(dst, b)
	}
}

func (h *HUD) drawDetails(dst *ebiten.Image) {
	d := h.DetailsCard
	ui.DarkPanel(dst, Overlay)
	h.Font.DrawShadow(dst, ui.Clip(strings.ToUpper(d.Title), detailTitleChars-editTitleRoom*len(h.editTabButton(d))), Overlay.X+10, Overlay.Y+6, 2, ui.Palette.Gold)
	body := detailBody()
	ui.Fill(dst, body, color.RGBA{10, 14, 16, 255})
	ui.Stroke(dst, body, ui.Palette.BlueDark)
	visible := detailVisible()
	width := ui.MaxChars(body.W-8, 1)
	y := body.Y + 3
	for _, l := range d.Scroll.Window(d.Lines, visible) {
		clr := ui.Palette.Text
		switch {
		case strings.HasPrefix(l, "$ ") || strings.HasPrefix(l, "--- "):
			clr = ui.Palette.GoldBright
		case !d.OK:
			clr = ui.Palette.RedText
		case isHeading(l):
			clr = ui.Palette.Cream
		}
		if head, addr, ok := strings.Cut(l, RollAddressMark); ok {
			h.detailFont().Draw(dst, head, body.X+4, y, 1, clr)
			h.detailFont().Draw(dst, h.detailFont().Clip(addr, max(width-len([]rune(head)), 4)), body.X+4+h.detailFont().Width(head, 1), y, 1, ui.Palette.TextDim)
			y += ui.LineHeight
			continue
		}
		h.detailFont().Draw(dst, h.detailFont().Clip(l, width), body.X+4, y, 1, clr)
		y += ui.LineHeight
	}
	h.drawScrollbar(dst, body, len(d.Lines), visible, d.Scroll.Offset)
	h.DrawOverlayButtons(dst, h.detailButtons())
}

const RollAddressMark = "\x00"

func (h *HUD) detailFont() ui.Font {
	if ui.Naming() {
		return Literal
	}
	return h.Font
}

func isHeading(l string) bool {
	return l != "" && l[0] != ' ' && strings.HasSuffix(strings.TrimSpace(l), ":")
}

func (h *HUD) drawScrollbar(dst *ebiten.Image, r ui.Rect, total, visible, offset int) {
	if total <= visible {
		return
	}
	track := ui.Rect{X: r.X + r.W - 5, Y: r.Y + 2, W: 3, H: r.H - 4}
	ui.Fill(dst, track, ui.Palette.BlueDark)
	h2 := max(track.H*visible/total, 6)
	y := track.Y + (track.H-h2)*offset/max(total-visible, 1)
	ui.Fill(dst, ui.Rect{X: track.X, Y: y, W: 3, H: h2}, ui.Palette.Gold)
}

func (h *HUD) toastTop(f Frame) int {
	y := TopH + 8
	if SiegeActive(f) {
		y = SiegeStripRect(f).Y + SiegeStripRect(f).H + 6
	}
	if r, ok := h.pinnedRect(f); ok {
		y = max(y, r.Y+r.H+6)
	}
	return y
}

func (h *HUD) toastClear(f Frame) []ui.Rect {
	keepClear := h.GuideTargets(f)
	if h.IntroShown(f) {
		keepClear = append(keepClear, h.IntroCard(f))
	}
	if h.CardsDrawn(f) && h.ObjectiveShown(f) {
		keepClear = append(keepClear, h.ObjectiveRectOf(f))
	}
	if h.LessonShown(f) {
		keepClear = append(keepClear, h.LessonRect(f))
	}
	if r, ok := h.CalamityRect(f); ok {
		keepClear = append(keepClear, r)
	}
	if r, ok := h.SurveyRect(f); ok {
		keepClear = append(keepClear, r)
	}
	for _, a := range h.Arrows(f) {
		keepClear = append(keepClear, h.ArrowSpan(a))
	}
	for _, id := range h.Win.Stacked() {
		if r, ok := WindowSpecs[id].Rect(h, f); ok {
			keepClear = append(keepClear, Shadowed(windowClaim(id, r)))
		}
	}
	return inflated(append(append(keepClear, FreightRects(f)...), f.WorldBoxes...), TipClear)
}

func (h *HUD) toastRoom(f Frame, t Toast) bool {
	rects := h.placeToasts(f, append(slices.Clone(h.ToastList), t))
	return !OverlapsAny(rects[len(rects)-1], h.toastClear(f))
}

func (h *HUD) toastBox(t Toast, x, y int) ui.Rect {
	w, th := h.Font.Measure(ToastLabel(t), 1)
	return ui.Rect{X: x - w/2 - 8, Y: y, W: w + 16, H: th + 4}
}

func (h *HUD) ToastRects(f Frame) []ui.Rect { return h.placeToasts(f, h.ToastList) }

func (h *HUD) placeToasts(f Frame, toasts []Toast) []ui.Rect {
	clear := h.toastClear(f)
	kept := make([]ui.Rect, len(toasts))
	y, blocked := h.toastTop(f), []ui.Rect(nil)
	for i, t := range toasts {
		if r := h.toastBox(t, t.Spot.X+t.Spot.W/2, t.Spot.Y); t.Spot.W > 0 && !OverlapsAny(r, clear) {
			kept[i], blocked, y = r, append(blocked, r), max(y, r.Y+r.H+3)
		}
	}
	keepClear := append(clear, blocked...)
	out := make([]ui.Rect, 0, len(toasts))
	for i, t := range toasts {
		if kept[i].W > 0 {
			out = append(out, kept[i])
			continue
		}
		r := below(h.toastBox(t, Width/2, y), keepClear)
		out, keepClear, y = append(out, r), append(keepClear, r), r.Y+r.H+3
	}
	return out
}

func (h *HUD) settleToasts(f Frame) {
	rects := h.ToastRects(f)
	next := make([]Toast, len(h.ToastList))
	for i, t := range h.ToastList {
		t.Spot = rects[i]
		next[i] = t
	}
	h.ToastList = next
}

func below(r ui.Rect, blocked []ui.Rect) ui.Rect {
	next := r
	for moved := true; moved; {
		moved = false
		for _, b := range blocked {
			if Overlaps(next, b) && b.Y+b.H+3 < ConsoleY-next.H {
				next.Y, moved = b.Y+b.H+3, true
			}
		}
	}
	return next
}

func (h *HUD) toastsHeld(f Frame) bool {
	if h.OverlayID == OverlayNone {
		return false
	}
	win := h.OverlayRect()
	for _, r := range h.ToastRects(f) {
		if r.X < win.X+win.W && win.X < r.X+r.W && r.Y < win.Y+win.H && win.Y < r.Y+r.H {
			return true
		}
	}
	return false
}

func (h *HUD) holdToasts(f Frame, elapsed float64) {
	if elapsed <= 0 || !h.toastsHeld(f) {
		return
	}
	held := make([]Toast, len(h.ToastList))
	for i, t := range h.ToastList {
		t.Start += elapsed
		held[i] = t
	}
	h.ToastList = held
}

func (h *HUD) DrawToasts(dst *ebiten.Image, f Frame) {
	if h.toastsHeld(f) {
		return
	}
	h.settleToasts(f)
	for i, r := range h.ToastRects(f) {
		h.drawToast(dst, h.ToastList[i], r)
	}
}

func (h *HUD) drawToast(dst *ebiten.Image, t Toast, r ui.Rect) {
	border, text := ui.Palette.Gold, ui.Palette.Text
	if t.Bad {
		border, text = ui.Palette.Red, ui.Palette.RedText
	}
	ui.Fill(dst, r.Offset(1, 2), ui.Palette.Shadow)
	ui.Fill(dst, r, ui.Palette.Console)
	ui.Stroke(dst, r, border)
	h.Font.Draw(dst, ToastLabel(t), r.X+8, r.Y+4, 1, text)
}

func (h *HUD) TooltipRect(lines []string, f Frame) ui.Rect {
	r, _, _ := h.hoverTip(lines, f)
	return r
}

func (h *HUD) hoverTip(lines []string, f Frame) (ui.Rect, string, bool) {
	blocked := append(h.otherTargets(f), h.TipBlocked(f)...)
	r := h.tipAvoiding(lines, f, blocked)
	if !OverlapsAny(r, blocked) {
		return r, "", true
	}
	key, holder, ok := h.tipHolder(f)
	if !ok {
		return r, "", false
	}
	inside := slices.DeleteFunc(slices.Clone(blocked), func(b ui.Rect) bool { return holderItself(holder, b) })
	if in := h.tipAvoiding(lines, f, inside); !OverlapsAny(in, inside) {
		return in, key, true
	}
	return r, "", false
}

func holderItself(holder, r ui.Rect) bool {
	shared := intersectArea(holder, r)
	return shared*10 >= holder.W*holder.H*9 && shared*10 >= r.W*r.H*7
}

func intersectArea(a, b ui.Rect) int {
	w := min(a.X+a.W, b.X+b.W) - max(a.X, b.X)
	h := min(a.Y+a.H, b.Y+b.H) - max(a.Y, b.Y)
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

func (h *HUD) tipHolder(f Frame) (string, ui.Rect, bool) {
	cx, cy := h.Hover.Rect.X+h.Hover.Rect.W/2, h.Hover.Rect.Y+h.Hover.Rect.H/2
	if w, ok := h.windowRect(); ok && w.Contains(cx, cy) {
		return "panel:" + h.OverlayID, w, true
	}
	for _, s := range WindowSurfaces(h, f) {
		if s.Rect.Contains(cx, cy) && s.Slot != SlotStep {
			return s.Key, s.Rect, true
		}
	}
	return "", ui.Rect{}, false
}

func (h *HUD) tipAvoiding(lines []string, f Frame, blocked []ui.Rect) ui.Rect {
	w := 0
	font := h.tipFont(h.Hover.Tip)
	for _, l := range lines {
		lw, _ := font.Measure(l, 1)
		w = max(w, lw)
	}
	r := ui.Rect{X: h.Hover.Rect.X, Y: h.Hover.Rect.Y - len(lines)*ui.LineHeight - 6 - tipGap, W: w + 10, H: len(lines)*ui.LineHeight + 6}
	if strings.HasPrefix(h.Hover.ID, noticePrefix) {
		r.X, r.Y = h.Hover.Rect.X-r.W-railTipGap, h.Hover.Rect.Y
		return FirstClear(slidPast(append(h.tipSpots(r, true), h.besideHolders(r, blocked)...), blocked), blocked)
	}
	if r.Y < TopH {
		r.Y = h.Hover.Rect.Y + h.Hover.Rect.H + tipGap
	}
	r.X = min(max(r.X, 2), Width-r.W-2)
	spots := append(h.tipSpots(r, false), h.besideHolders(r, blocked)...)
	return FirstClear(slidPast(append(spots, beyondTargets(r, h.otherTargets(f))...), blocked), blocked)
}

func slidPast(spots, blocked []ui.Rect) []ui.Rect {
	out := slices.Clone(spots)
	for _, c := range spots {
		for _, b := range blocked {
			if !Overlaps(c, b) {
				continue
			}
			for _, x := range []int{b.X - c.W - tipGap, b.X + b.W + tipGap} {
				s := c
				s.X = min(max(x, 2), Width-c.W-2)
				out = append(out, s)
			}
		}
	}
	return out
}

func (h *HUD) TipBlocked(f Frame) []ui.Rect {
	out := append([]ui.Rect{TopBar}, inflated(h.panelRects(f), TipClear)...)
	if r, ok := h.ConsoleSpan(f); ok {
		out = append(out, r)
	}
	if r, ok := h.SurveyRect(f); ok {
		out = append(out, r.Inset(-TipClear))
	}
	for i, a := range h.Arrows(f) {
		if h.Hover.ID != "alert:"+strconv.Itoa(i) {
			out = append(out, h.ArrowSpan(a).Inset(-TipClear))
		}
	}
	out = append(out, inflated(f.WorldBoxes, TipClear)...)
	for _, id := range h.Win.Stacked() {
		if r, ok := WindowSpecs[id].Rect(h, f); ok {
			out = append(out, Shadowed(r).Inset(-TipClear))
		}
	}
	return append(out, inflated(h.CrewRects(f), TipClear)...)
}

const TipClear = 6

func (h *HUD) besideHolders(r ui.Rect, blocked []ui.Rect) []ui.Rect {
	cx, cy := h.Hover.Rect.X+h.Hover.Rect.W/2, h.Hover.Rect.Y+h.Hover.Rect.H/2
	out := []ui.Rect{}
	for _, c := range blocked {
		if !c.Contains(cx, cy) {
			continue
		}
		for _, s := range []ui.Rect{{X: r.X, Y: c.Y - r.H - tipGap, W: r.W, H: r.H}, {X: r.X, Y: c.Y + c.H + tipGap, W: r.W, H: r.H}, {X: c.X - r.W - tipGap, Y: r.Y, W: r.W, H: r.H}, {X: c.X + c.W + tipGap, Y: r.Y, W: r.W, H: r.H}} {
			s.X, s.Y = min(max(s.X, 2), Width-s.W-2), min(max(s.Y, 2), Height-s.H-2)
			out = append(out, s)
		}
	}
	return out
}

func beyondTargets(r ui.Rect, targets []ui.Rect) []ui.Rect {
	out := []ui.Rect{}
	for _, t := range targets {
		for _, c := range []ui.Rect{{X: r.X, Y: t.Y + t.H + tipGap, W: r.W, H: r.H}, {X: t.X - r.W - tipGap, Y: r.Y, W: r.W, H: r.H}, {X: t.X + t.W + tipGap, Y: r.Y, W: r.W, H: r.H}} {
			c.X, c.Y = min(max(c.X, 2), Width-c.W-2), min(max(c.Y, 2), Height-c.H-2)
			out = append(out, c)
		}
	}
	return out
}

func (h *HUD) tipSpots(r ui.Rect, rail bool) []ui.Rect {
	b := h.Hover.Rect
	below := ui.Rect{X: r.X, Y: b.Y + b.H + tipGap, W: r.W, H: r.H}
	above := ui.Rect{X: r.X, Y: b.Y - r.H - tipGap, W: r.W, H: r.H}
	if rail {
		below.X, above.X = b.X+b.W-r.W, b.X+b.W-r.W
	}
	out := []ui.Rect{r, below, above, {X: b.X - r.W - railTipGap, Y: b.Y, W: r.W, H: r.H}, {X: b.X + b.W + railTipGap, Y: b.Y, W: r.W, H: r.H}}
	for i, c := range out {
		out[i].X = min(max(c.X, 2), Width-c.W-2)
		out[i].Y = min(max(c.Y, 2), Height-c.H-2)
	}
	return out
}

func (h *HUD) otherTargets(f Frame) []ui.Rect {
	out := []ui.Rect{}
	for _, t := range h.GuideTargets(f) {
		if t != h.Hover.Rect {
			out = append(out, t)
		}
	}
	if card, ok := h.GuideCardRect(f); ok && !Overlaps(card, h.Hover.Rect) {
		out = append(out, card)
	}
	return out
}

func (h *HUD) GuideCardRect(f Frame) (ui.Rect, bool) {
	if h.IntroShown(f) {
		return h.IntroCard(f), true
	}
	if hole, ok := h.objectiveSpot(f); ok {
		card, _ := h.objectiveGuideCard(f, hole)
		return card, true
	}
	return ui.Rect{}, false
}

func (h *HUD) TipRect(f Frame) (ui.Rect, bool) {
	if h.Pin.On() || !h.Hovering || h.Hover.Tip == "" {
		return ui.Rect{}, false
	}
	lines, _ := h.layoutTip(h.Hover.Tip)
	r, _, _ := h.hoverTip(lines, f)
	return r, true
}

func (h *HUD) drawTooltip(dst *ebiten.Image, f Frame) {
	h.drawHelpCursor(dst)
	if h.Pin.On() {
		h.drawPinnedTip(dst)
		return
	}
	if !h.Hovering || h.Hover.Tip == "" {
		h.DrawPlainTip(dst, f)
		return
	}
	h.drawHoverTip(dst, f)
}

func (h *HUD) drawHoverTip(dst *ebiten.Image, f Frame) {
	lines, styles := h.layoutTip(h.Hover.Tip)
	r, _, _ := h.hoverTip(lines, f)
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 235))
	ui.Stroke(dst, r, ui.Palette.Gold)
	h.drawTipLines(dst, r, lines, styles)
}

const (
	TipWidth = 220
	tipGap   = beaconPad + 2
)
