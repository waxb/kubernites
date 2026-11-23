package hud

import (
	"fmt"
	"image"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	SlotFeedback = "feedback"
	shadowW      = 2
	ShadowH      = 3
	pointerReach = 8
)

var (
	fullScreen   = ui.Rect{W: Width, H: Height}
	autoOverlays = map[string]bool{OverlayWelcome: true, OverlayDebrief: true, OverlayEnding: true}
	dimOverlays  = map[string]bool{OverlayWelcome: true, OverlayDebrief: true, OverlayEnding: true, OverlayChapters: true, OverlayTrials: true}
	veiled       = map[string]bool{OverlayBook: true}
	veil         = ui.Rect{Y: TopH, W: Width, H: Height - TopH}
)

type paintedSurface struct {
	Surface
	paint func(*ebiten.Image)
	base  string
}

func shadowedButtons(list []Button, dx, dy int) []Button {
	out := make([]Button, len(list))
	for i, b := range list {
		b.Rect.W, b.Rect.H = b.Rect.W+dx, b.Rect.H+dy
		out[i] = b
	}
	return out
}

func Shadowed(r ui.Rect) ui.Rect { return ui.Rect{X: r.X, Y: r.Y, W: r.W + shadowW, H: r.H + ShadowH} }

func spanOf(rects ...ui.Rect) ui.Rect {
	out, have := ui.Rect{}, false
	for _, r := range rects {
		if r.W <= 0 || r.H <= 0 {
			continue
		}
		out, have = unionRect(out, r, have), true
	}
	return out
}

func buttonSpan(r ui.Rect, buttons []Button) ui.Rect {
	rects := []ui.Rect{r}
	for _, b := range buttons {
		rects = append(rects, b.Rect, b.Face)
	}
	return spanOf(rects...)
}

func lineSurface(owner string, line [2][2]float32) Surface {
	x0, y0, x1, y1 := int(math.Round(float64(line[0][0]))), int(math.Round(float64(line[0][1]))), int(math.Round(float64(line[1][0]))), int(math.Round(float64(line[1][1])))
	r := ui.Rect{X: min(x0, x1), Y: min(y0, y1), W: absInt(x1-x0) + 1, H: absInt(y1-y0) + 1}.Inset(-pointerReach)
	return Surface{Slot: SlotPointer, Kind: "pointer", Key: "pointer:" + owner, Owner: owner, Line: [4]int{x0, y0, x1, y1}, Rect: r}
}

func ringSurface(owner string, target ui.Rect) Surface {
	return Surface{Slot: SlotPointer, Kind: "ring", Key: "ring:" + owner, Owner: owner, Rect: target.Inset(-HolePad - 1), Ring: true}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (h *HUD) Surfaces(f Frame) []Surface {
	f = h.Stage(f)
	out := []Surface{}
	for _, s := range h.paintedSurfaces(f) {
		out = append(out, s.Surface)
	}
	return out
}

func (h *HUD) paintedSurfaces(f Frame) []paintedSurface {
	out := []paintedSurface{}
	window, open := h.windowRect()
	under := open
	for _, l := range HudLayers {
		under = under && l.Name != overlayLayer
		for _, s := range l.Surfaces(h, f) {
			if s.Rect.W <= 0 || s.Rect.H <= 0 || under && Overlaps(s.Rect, window) {
				continue
			}
			if s.Layer == "" {
				s.Layer = l.Name
			}
			s.base = l.Name
			out = append(out, s)
		}
	}
	return aimPointers(out)
}

func aimPointers(list []paintedSurface) []paintedSurface {
	out := make([]paintedSurface, len(list))
	copy(out, list)
	for i, s := range out {
		if !s.pointer() || s.Target != "" {
			continue
		}
		x, y := s.Rect.X+s.Rect.W/2, s.Rect.Y+s.Rect.H/2
		if s.Line != [4]int{} {
			x, y = s.Line[2], s.Line[3]
		}
		for _, t := range list {
			if t.pointer() || t.Slot == SlotJuice || t.Key == s.Owner {
				continue
			}
			if t.Rect.Inset(-HolePad-pointerReach).Contains(x, y) {
				out[i].Target = t.Key
				break
			}
		}
	}
	return out
}

func (h *HUD) paintAll(dst *ebiten.Image, f Frame) {
	h.drawn = h.drawn[:0]
	layers := map[string]hudLayer{}
	for _, l := range HudLayers {
		layers[l.Name] = l
	}
	for _, s := range h.paintedSurfaces(f) {
		h.drawn = append(h.drawn, s.Surface)
		l := layers[s.base]
		name := l.Name
		if name == overlayLayer && h.OverlayID != OverlayNone {
			name = PanelRegion + h.OverlayID
		}
		clip := dst.SubImage(image.Rect(s.Rect.X, s.Rect.Y, s.Rect.X+s.Rect.W, s.Rect.Y+s.Rect.H)).(*ebiten.Image)
		paint := s.paint
		if paint == nil {
			paint = func(d *ebiten.Image) { l.Draw(h, d, f) }
		}
		ui.InRegion(name, func() { paint(clip) })
	}
}

func (h *HUD) Drawn() []Surface { return append([]Surface(nil), h.drawn...) }

func (h *HUD) paintSurfaces(dst *ebiten.Image, list []paintedSurface) {
	for _, s := range list {
		if s.Rect.W > 0 && s.Rect.H > 0 {
			s.paint(clipTo(dst, s.Rect))
		}
	}
}

func one(s Surface) []paintedSurface { return []paintedSurface{{Surface: s}} }

func vignetteSurfaces(h *HUD, _ Frame) []paintedSurface {
	if VignetteAlpha(h.events-h.Juice.vignetteAt) <= 0 || h.Reduce {
		return nil
	}
	return one(Surface{Slot: SlotJuice, Key: "vignette", Rect: fullScreen})
}

func dimSurfaces(h *HUD, f Frame) []paintedSurface {
	if !h.IntroShown(f) {
		if _, ok := h.objectiveSpot(f); !ok {
			return nil
		}
	}
	return one(Surface{Slot: SlotJuice, Key: "dim", Rect: fullScreen})
}

func topBarSurfaces(_ *HUD, _ Frame) []paintedSurface {
	return one(Surface{Slot: SlotTop, Key: "topbar", Rect: TopBar})
}

func coinSurfaces(h *HUD, _ Frame) []paintedSurface {
	if len(h.Coins.Flights) == 0 {
		return nil
	}
	return one(Surface{Slot: SlotJuice, Key: "coins", Rect: fullScreen})
}

func deltaSurfaces(h *HUD, f Frame) []paintedSurface {
	out := []paintedSurface{}
	for _, d := range h.deltaSpots(f) {
		out = append(out, paintedSurface{Surface: Surface{Slot: SlotFeedback, Key: "delta:" + d.meter, Text: d.Text, Rect: d.rect}, paint: func(dst *ebiten.Image) { h.drawDelta(dst, d) }})
	}
	return out
}

func siegeSurfaces(_ *HUD, f Frame) []paintedSurface {
	if !SiegeActive(f) {
		return nil
	}
	return one(Surface{Slot: SlotTop, Key: "siege", Rect: SiegeStripRect(f)})
}

func alertSurfaces(h *HUD, f Frame) []paintedSurface {
	out := []paintedSurface{}
	for i, a := range h.Arrows(f) {
		hot := h.Hovering && h.Hover.ID == "alert:"+strconv.Itoa(i)
		out = append(out, paintedSurface{Surface: Surface{Slot: SlotTop, Key: "alert:" + strconv.Itoa(i), Rect: h.ArrowSpan(a)}, paint: func(d *ebiten.Image) { h.drawArrow(d, a, hot) }})
	}
	if h.ping.on && h.Now-h.ping.start <= pingLife {
		out = append(out, paintedSurface{Surface: Surface{Slot: SlotJuice, Key: "ping", Rect: fullScreen}, paint: func(d *ebiten.Image) { h.drawPing(d, f) }})
	}
	return out
}

var windowSlots = map[string]string{
	PanelHarbor: SlotHarbor, PanelObjective: SlotStep, PanelLesson: SlotBox, PanelNotices: SlotRail, PanelCalamity: SlotBox,
	PanelCrews: SlotPanel, PanelCrew: SlotPanel, PanelKind: SlotPanel, PanelWorker: SlotWorker,
}

func WindowSurfaces(h *HUD, f Frame) []paintedSurface {
	out := []paintedSurface{}
	for _, id := range h.Win.Stacked() {
		spec := WindowSpecs[id]
		r, ok := spec.Rect(h, f)
		if !ok {
			continue
		}
		s := h.windowSurface(f, id, r)
		s.Layer = WindowRegion + id
		draw := spec.draw
		out = append(out, paintedSurface{Surface: s, paint: func(d *ebiten.Image) {
			ui.InRegion(WindowRegion+id, func() { draw(h, d, f) })
		}})
		if id == PanelNotices {
			if b, ok := h.noticeBox(f, r, s.Key); ok {
				out = append(out, b)
			}
		}
	}
	return out
}

func (h *HUD) windowSurface(f Frame, id string, r ui.Rect) Surface {
	s := Surface{Slot: windowSlots[id], Key: "window:" + id, Rect: Shadowed(windowClaim(id, r))}
	switch id {
	case PanelObjective:
		return h.stepSurface(f, s)
	case PanelLesson:
		s.Kind, s.Key, s.Box, s.Waits, s.Idea, s.Urgent = BoxLesson, "lesson:"+h.LessonCard.Title, true, true, !h.LessonCard.done && !h.LessonCard.timely, h.LessonCard.timely
		switch {
		case h.LessonCard.done:
			s.Kind = BoxDone
		case !h.LessonCard.Teach:
			s.Kind = BoxNews
		}
		s.Tray = h.Aside(s.Key)
		s.Text = SurfaceText(h.LessonLines())
		s.Rect = buttonSpan(s.Rect, h.LessonButtons(f))
	case PanelCalamity:
		return h.troubleSurface(f, s)
	case PanelHarbor:
		s.Kind, s.Text = BoxBoard, boardFace(f)
	case PanelCrews, PanelCrew, PanelKind, PanelWorker:
		s.Asked = true
	}
	return s
}

func (h *HUD) stepSurface(f Frame, s Surface) Surface {
	s.Rect = buttonSpan(s.Rect, h.ObjectiveButtons(f))
	before, status, after := splitStatus(h.ObjectiveLines(f))
	s.Text, s.Status = SurfaceText(append(slices.Clone(before), after...)), SurfaceText(status)
	if h.chipShown(f) {
		s.Status += " " + h.shownCommand(f)
	}
	if h.teacherShown() {
		v, l := h.Yard.View, h.TeacherLayout()
		s.Kind, s.Key, s.Box, s.Face = BoxStep, actTeacher+h.Yard.face, true, true
		if v.Status == view.ChapterComplete {
			s.Kind = BoxDone
		}
		s.Text = h.TeacherText()
		s.Status = strings.TrimSpace(strings.Join(append(slices.Clone(l.Progress), v.Chip), " "))
		return s
	}
	if h.trialShown() {
		s.Kind, s.Key = BoxStep, "teacher"
		return s
	}
	o, ok := f.ShownObjective()
	if !ok {
		return s
	}
	s.Kind, s.Key, s.Box, s.Face = BoxStep, "step:"+o.ID, true, true
	if o.Done {
		s.Kind, s.Key = BoxDone, "done:"+o.ID
	}
	return s
}

func (h *HUD) troubleSurface(f Frame, s Surface) Surface {
	v, _, _, ok := h.CurrentCalamity(f)
	if !ok {
		return s
	}
	s.Kind, s.Box, s.Waits = BoxTrouble, true, v.Mended == ""
	s.Key = "trouble:" + v.ID
	if v.Mended != "" {
		s.Key = "mended:" + v.ID
	}
	s.Urgent, s.Idea = urgentTrouble(v.Kind), v.Mended == "" && !urgentTrouble(v.Kind)
	s.Taught = !v.First
	cause, why := h.CalamityText(f, v)
	s.Text = strings.TrimSpace(strings.Join(append(append(append([]string{h.asideLine()}, cause...), why...), remedyLabels(h.calamityLines(f))), " "))
	s.Rect = buttonSpan(s.Rect, h.CalamityButtons(f))
	return s
}

func boardFace(f Frame) string {
	e := f.State.Economy
	face := e.Bottleneck
	if e.Fix != nil {
		face += " " + e.Fix.Label
	}
	return face
}

func purseSurfaces(h *HUD, f Frame) []paintedSurface {
	out := []paintedSurface{}
	if box, text, ok := h.PurseTab(); ok {
		out = append(out, paintedSurface{Surface: Surface{Slot: SlotFeedback, Key: "purse", Text: text, Rect: box}})
	}
	if r, ok := h.gainRect(f); ok {
		out = append(out, paintedSurface{Surface: Surface{Slot: SlotFeedback, Key: "gain", Text: h.Coins.Gain.Float.text, Rect: r}})
	}
	for i := range out {
		key := out[i].Key
		out[i].paint = func(d *ebiten.Image) {
			if key == "gain" {
				h.drawGain(d, f)
				return
			}
			h.drawPurseOnly(d)
		}
	}
	return out
}

func surveySurfaces(h *HUD, f Frame) []paintedSurface {
	r, ok := h.SurveyRect(f)
	if !ok {
		return nil
	}
	return one(Surface{Slot: SlotTip, Key: "survey", Rect: Shadowed(r), Asked: true})
}

func legendSurfaces(h *HUD, f Frame) []paintedSurface {
	if !h.LegendShown(f) {
		return nil
	}
	return one(Surface{Slot: SlotTip, Key: "legend", Rect: ui.Rect{X: LegendRect(h.lens).X, Y: LegendRect(h.lens).Y, W: LegendRect(h.lens).W + 1, H: LegendRect(h.lens).H + 2}, Asked: true})
}

func consoleSurfaces(h *HUD, f Frame) []paintedSurface {
	if h.StripShown(f) {
		r, ok := h.ConsoleSpan(f)
		if !ok {
			return nil
		}
		return one(Surface{Slot: SlotConsole, Key: "console", Rect: r})
	}
	out := []paintedSurface{}
	for i, r := range h.consolePieces(f) {
		key := "console"
		if i > 0 {
			key = fmt.Sprintf("console:%d", i)
		}
		out = append(out, paintedSurface{Surface: Surface{Slot: SlotConsole, Key: key, Rect: r}})
	}
	return out
}

func (h *HUD) consolePieces(f Frame) []ui.Rect {
	crew := h.CrewRects(f)
	rects := []ui.Rect{}
	for _, r := range h.consoleRects(f) {
		if !slices.Contains(crew, r) {
			rects = append(rects, Shadowed(r))
		}
	}
	return rects
}

func (h *HUD) ConsoleSpan(f Frame) (ui.Rect, bool) {
	rects := h.consolePieces(f)
	if len(rects) == 0 {
		return ui.Rect{}, false
	}
	return spanOf(rects...), true
}

func lensSurfaces(h *HUD, f Frame) []paintedSurface {
	buttons := h.lensCycleButtons(f)
	if len(buttons) == 0 {
		return nil
	}
	return one(Surface{Slot: SlotConsole, Key: "lens", Rect: Shadowed(buttonSpan(ui.Rect{}, buttons)), Owner: "console"})
}

func OverlaySurfaces(h *HUD, f Frame) []paintedSurface {
	if h.OverlayID == OverlayNone {
		return nil
	}
	s := Surface{Slot: SlotPanel, Key: "panel:" + h.OverlayID, Rect: Shadowed(h.OverlayRect()), Asked: !autoOverlays[h.OverlayID]}
	switch {
	case dimOverlays[h.OverlayID]:
		s.Rect, s.Modal = fullScreen, true
	case veiled[h.OverlayID]:
		s.Rect, s.Modal = veil, true
	}
	if autoOverlays[h.OverlayID] {
		s.Kind, s.Box, s.Waits = BoxModal, true, true
	}
	return one(s)
}

func introSurfaces(h *HUD, f Frame) []paintedSurface {
	if !h.IntroShown(f) {
		return nil
	}
	in := *f.Guide.Intro
	key := "intro:" + in.ID
	card := h.IntroCard(f)
	title, why, try := h.introText(in)
	out := []paintedSurface{{Surface: Surface{Slot: SlotBox, Kind: BoxCard, Key: key, Text: title + " " + why + " " + try, Rect: buttonSpan(Shadowed(card), h.IntroButtons(f)), Box: true, Waits: true, Idea: true, Tray: h.Aside(key)},
		paint: func(d *ebiten.Image) {
			h.DrawIntroCard(d, card, in)
			h.DrawOverlayButtons(d, h.IntroButtons(f))
		}}}
	if r, ok := h.TrayRect(f); ok {
		out = append(out, paintedSurface{Surface: Surface{Slot: SlotBox, Kind: "tray", Key: "tray:" + in.ID, Rect: Shadowed(r), Owner: key}, paint: func(d *ebiten.Image) { h.DrawTray(d, f) }})
	}
	target, ok := h.IntroHole(f)
	if !ok {
		return out
	}
	if h.IntroHidden(f) {
		out = append(out, paintedSurface{Surface: Surface{Slot: SlotPointer, Kind: "arrow", Key: "arrow:" + in.ID, Owner: key, Rect: h.edgeArrowRect(target, f), Ring: true}, paint: func(d *ebiten.Image) { h.drawEdgeArrow(d, target, f) }})
	} else {
		out = append(out, h.onMap(f, target, paintedSurface{Surface: ringSurface(key, target), paint: func(d *ebiten.Image) { h.strokeHole(d, target) }}))
	}
	if h.lineClear(f, card, target) {
		if line, lineOK := h.pointerPath(f, card, target); lineOK {
			out = append(out, h.onMap(f, target, paintedSurface{Surface: lineSurface(key, line), paint: func(d *ebiten.Image) { drawPointer(d, line) }}))
		}
	}
	return out
}

func (h *HUD) introText(in Intro) (string, string, string) {
	why, try := h.IntroLines(in)
	return strings.Join(h.IntroTitle(in), " "), strings.Join(why, " "), strings.Join(try, " ")
}

func questSurfaces(h *HUD, f Frame) []paintedSurface {
	c, ok := h.QuestCard(f)
	if !ok {
		return nil
	}
	r := h.questRect(c)
	text := c.title + " " + c.Body
	return []paintedSurface{
		{Surface: Surface{Slot: SlotJuice, Key: "dim:quest", Rect: fullScreen, Modal: true}, paint: func(d *ebiten.Image) { dimAround(d, r, true) }},
		{Surface: Surface{Slot: SlotBox, Kind: BoxQuest, Key: "quest:" + c.key, Text: text, Rect: buttonSpan(Shadowed(r), h.questButtons(f)), Box: true, Waits: true, Idea: !c.Recap, Modal: true}, paint: func(d *ebiten.Image) {
			drawScroll(d, r)
			h.drawQuestBody(d, f, c, r)
		}},
	}
}

func spotSurfaces(h *HUD, f Frame) []paintedSurface {
	hole, ok := h.objectiveSpot(f)
	if !ok {
		return nil
	}
	card, in := h.objectiveGuideCard(f, hole)
	key := "spot:" + strings.TrimPrefix(in.ID, "objective:")
	title, why, try := h.introText(in)
	out := []paintedSurface{
		{Surface: ringSurface(key, hole), paint: func(d *ebiten.Image) { h.strokeHole(d, hole) }},
		{Surface: Surface{Slot: SlotBox, Kind: BoxCard, Key: key, Text: title + " " + why + " " + try, Rect: Shadowed(card), Box: true, Face: true}, paint: func(d *ebiten.Image) { h.DrawIntroCard(d, card, in) }},
	}
	if h.lineClear(f, card, hole) {
		if line, lineOK := h.pointerPath(f, card, hole); lineOK {
			out = append(out, paintedSurface{Surface: lineSurface(key, line), paint: func(d *ebiten.Image) { drawPointer(d, line) }})
		}
	}
	return out
}

func bannerSurfaces(h *HUD, f Frame) []paintedSurface {
	r, ok := h.BannerRect(f)
	if !ok || !h.CardsDrawn(f) && !(h.Juice.feat && h.OverlayID == OverlayAge) {
		return nil
	}
	return one(Surface{Slot: SlotBox, Kind: BoxBanner, Key: "banner:" + h.Juice.Banner, Text: strings.TrimSpace(h.Juice.Banner + " " + h.Juice.Cause), Rect: r.Inset(-2), Box: true})
}

func toastSurfaces(h *HUD, f Frame) []paintedSurface {
	if h.toastsHeld(f) {
		return nil
	}
	out := []paintedSurface{}
	for i, r := range h.ToastRects(f) {
		t := h.ToastList[i]
		s := Surface{Slot: SlotBox, Kind: BoxToast, Key: "toast:" + t.Text, Text: ToastLabel(t), Rect: ui.Rect{X: r.X, Y: r.Y, W: r.W + 1, H: r.H + 2}, Box: t.News, Asked: !t.News}
		out = append(out, paintedSurface{Surface: s, paint: func(d *ebiten.Image) { h.drawToast(d, t, r) }})
	}
	return out
}

func pinnedSurfaces(h *HUD, f Frame) []paintedSurface {
	r, ok := h.pinnedRect(f)
	if !ok {
		return nil
	}
	return one(Surface{Slot: SlotBox, Kind: BoxBanner, Key: "pinned:" + h.Juice.pinned, Text: h.pinnedText(f), Rect: buttonSpan(ui.Rect{X: r.X, Y: r.Y, W: r.W + 1, H: r.H + 2}, shadowedButtons(h.pinnedButtons(f), 1, 2))})
}

func beaconSurface(key string, r ui.Rect) Surface {
	return Surface{Slot: SlotPointer, Kind: "beacon", Key: "beacon:" + key, Rect: r.Inset(-3), Ring: true}
}

func hintSurfaces(h *HUD, f Frame) []paintedSurface {
	id := h.HintedButton(f)
	if id == "" {
		return nil
	}
	for _, b := range h.Buttons(f) {
		if b.ID == id {
			rect := b.Rect
			return []paintedSurface{h.onMap(f, rect, paintedSurface{Surface: beaconSurface("hint:"+id, rect), paint: func(d *ebiten.Image) { h.drawBeacon(d, rect) }})}
		}
	}
	return nil
}

func teacherBeaconSurfaces(h *HUD, f Frame) []paintedSurface {
	b, ok := h.teacherLit(f)
	if !ok {
		return nil
	}
	return []paintedSurface{h.onMap(f, b.Rect, paintedSurface{Surface: beaconSurface("teacher:"+b.ID, b.Rect), paint: func(d *ebiten.Image) { h.drawBeacon(d, b.Rect) }})}
}

func guideSurfaces(h *HUD, f Frame) []paintedSurface {
	out := []paintedSurface{}
	if h.IntroShown(f) {
		target, ok := h.IntroHole(f)
		if !ok {
			return nil
		}
		key := "intro:" + f.Guide.Intro.ID
		card := h.IntroCard(f)
		if h.holeStroked(f) {
			s := ringSurface(key, target)
			s.Key = "guide-ring:" + key
			out = append(out, h.onMap(f, target, paintedSurface{Surface: s, paint: func(d *ebiten.Image) { h.strokeHole(d, target) }}))
		}
		if h.lineClear(f, card, target) {
			if line, lineOK := h.pointerPath(f, card, target); lineOK {
				s := lineSurface(key, line)
				s.Key = "guide-pointer:" + key
				out = append(out, h.onMap(f, target, paintedSurface{Surface: s, paint: func(d *ebiten.Image) { drawPointer(d, line) }}))
			}
		}
		return out
	}
	for _, r := range h.guideBeacons(f) {
		rect := r
		out = append(out, h.onMap(f, rect, paintedSurface{Surface: beaconSurface("guide", rect), paint: func(d *ebiten.Image) { h.drawBeacon(d, rect) }}))
	}
	return out
}

func tooltipSurfaces(h *HUD, f Frame) []paintedSurface {
	out := []paintedSurface{}
	if h.help {
		r := ui.Rect{X: h.MouseX + 12, Y: h.MouseY + 10, W: 11, H: 13}
		out = append(out, paintedSurface{Surface: Surface{Slot: SlotTip, Key: "help-cursor", Rect: r, Asked: true}, paint: func(d *ebiten.Image) { h.drawHelpCursor(d) }})
	}
	switch {
	case h.Pin.On() && !h.pinCovers(f, nil):
		saved := h.Hover
		h.Hover = h.Pin.button
		owner, _, _ := h.tipHolder(f)
		h.Hover = saved
		out = append(out, paintedSurface{Surface: Surface{Slot: SlotTip, Key: "pinned-tip", Rect: h.pinnedTipSpan(), Asked: true, Owner: owner}, paint: func(d *ebiten.Image) { h.drawPinnedTip(d) }})
	case !h.Hovering || h.Hover.Tip == "":
		if r, ok := h.PlainTipRect(f); ok {
			out = append(out, paintedSurface{Surface: Surface{Slot: SlotTip, Key: "plain-tip", Rect: r, Asked: true}, paint: func(d *ebiten.Image) { h.DrawPlainTip(d, f) }})
		}
	default:
		lines, _ := h.layoutTip(h.Hover.Tip)
		r, owner, clear := h.hoverTip(lines, f)
		tip := Surface{Slot: SlotTip, Key: "tooltip:" + h.Hover.ID, Text: h.Hover.Tip, Rect: r, Asked: true, Owner: owner, Crowded: !clear}
		out = append(out, paintedSurface{Surface: tip, paint: func(d *ebiten.Image) { h.drawHoverTip(d, f) }})
	}
	return out
}

func worldTipSurfaces(h *HUD, f Frame) []paintedSurface {
	r, ok, clear := h.worldTip(f)
	if !ok || h.Hovering {
		return nil
	}
	return one(Surface{Slot: SlotTip, Key: "worldtip", Text: f.Tip, Rect: r, Asked: true, Crowded: !clear})
}

func SurfaceText(lines []Line) string {
	parts := make([]string, 0, len(lines))
	for _, l := range lines {
		parts = append(parts, l.Text)
	}
	return strings.Join(parts, " ")
}

func (h *HUD) pinnedTipSpan() ui.Rect {
	rects := []ui.Rect{h.Pin.Rect}
	if h.Pin.Why {
		rects = append(rects, h.Pin.WhyRect)
	}
	return buttonSpan(spanOf(rects...), h.tipButtons())
}

func (h *HUD) windowRect() (ui.Rect, bool) {
	if h.OverlayID == OverlayNone || dimOverlays[h.OverlayID] {
		return ui.Rect{}, false
	}
	if _, ok := Overlays[h.OverlayID]; !ok {
		return ui.Rect{}, false
	}
	return Shadowed(h.OverlayRect()), true
}

func (h *HUD) noticeBox(f Frame, rail ui.Rect, owner string) (paintedSurface, bool) {
	n, ok := h.noticeOnTurn(f)
	if !ok {
		return paintedSurface{}, false
	}
	shown, _ := railCount(len(f.Notices))
	i := slices.IndexFunc(f.Notices[:shown], func(m Notice) bool { return m.ID == n.ID })
	if i < 0 {
		return paintedSurface{}, false
	}
	box := Surface{Slot: windowSlots[PanelNotices], Kind: BoxNews, Key: ActNotice + n.ID, Box: true, Owner: owner, Rect: Shadowed(railSlotIn(rail, i)),
		Text: SurfaceText([]Line{{Text: n.Title}, {Text: n.Detail}}), Idea: n.Kind == NoticeDecision, Urgent: n.Timely}
	return paintedSurface{Surface: box, paint: func(*ebiten.Image) {}}, true
}
