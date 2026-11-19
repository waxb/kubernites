package hud

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	railSize     = 24
	railGap      = 4
	railRight    = 6
	railTop      = 8
	RailVisible  = 6
	railSlideFor = .35
	railSlideBy  = 30.0
	RailPulseFor = 4.0
	railPulseHz  = 1.2
	railBadgePad = 2
	railTipGap   = 6
	railGripH    = railTop - 1
	railGripDots = 3
)

var railFrames = map[string]color.RGBA{
	NoticeDanger: ui.Palette.RedText, NoticeDecision: ui.Palette.GoldBright, NoticeNews: ui.Palette.Parchment,
}

var railFills = map[string]color.RGBA{
	NoticeDanger: ui.Palette.RedDark, NoticeDecision: ui.Palette.Console, NoticeNews: ui.Palette.ConsoleEdge,
}

func Overlaps(a, b ui.Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func RailSlot(i int) ui.Rect {
	return railSlot(i)
}

func railSlot(i int) ui.Rect {
	return ui.Rect{X: Width - railRight - railSize, Y: TopH + railTop + i*(railSize+railGap), W: railSize, H: railSize}
}

func (h *HUD) railShown(f Frame) bool {
	return h.OverlayID == OverlayNone && (f.Shown(SurfaceRail) || slices.ContainsFunc(f.Notices, timelyNotice)) && !f.railQuiet()
}

func timelyNotice(n Notice) bool { return n.Timely }

func (h *HUD) noticeSeen(f Frame, id string) bool {
	n, ok := findNotice(f.Notices, id)
	if !ok || h.OverlayID != OverlayNone || !n.Timely && !f.Shown(SurfaceRail) {
		return false
	}
	in := f.Guide.Intro
	return in == nil || in.Target == SurfaceRail || !h.Director.Stage.Has(actIntro+in.ID)
}

func railCount(n int) (int, bool) {
	if n > RailVisible {
		return RailVisible, true
	}
	return n, false
}

func (h *HUD) RailRect(f Frame) ui.Rect {
	shown, more := railCount(len(f.Notices))
	if !h.railShown(f) || shown == 0 {
		return ui.Rect{}
	}
	if more {
		shown++
	}
	top, bottom := railSlot(0), railSlot(shown-1)
	return f.placed(PanelNotices, ui.Rect{X: top.X, Y: top.Y - railGripH, W: top.W, H: bottom.Y + bottom.H - top.Y + railGripH})
}

func railSlotIn(rail ui.Rect, i int) ui.Rect {
	return ui.Rect{X: rail.X, Y: rail.Y + railGripH + i*(railSize+railGap), W: railSize, H: railSize}
}

func noticeTip(n Notice) string {
	action := "Click: open."
	if n.At != nil {
		action = "Click: go there."
	}
	head := n.Title
	if n.Count > 1 {
		head += fmt.Sprintf(" (x%d notices like this)", n.Count)
	}
	return head + "\n" + n.Detail + "\n" + action + " Right click: dismiss."
}

func (h *HUD) RailButtons(f Frame) []Button {
	if !h.railShown(f) {
		return nil
	}
	shown, more := railCount(len(f.Notices))
	rail := h.RailRect(f)
	out := make([]Button, 0, shown+1)
	for i, n := range f.Notices[:shown] {
		out = append(out, Button{ID: noticePrefix + n.ID, Rect: railSlotIn(rail, i), Icon: n.Icon, Tip: noticeTip(n)})
	}
	if more {
		extra := len(f.Notices) - shown
		out = append(out, Button{ID: noticePrefix + noticeMore, Rect: railSlotIn(rail, shown), Label: fmt.Sprintf("+%d", extra),
			Tip: fmt.Sprintf("%d more notices.\nClick: open the tidings.", extra)})
	}
	return out
}

func (h *HUD) openTidings() []Intent {
	h.TabID, h.scroll = TabEvents, ui.Scroll{}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) PressNotice(id string, f Frame) []Intent {
	if id == noticeMore {
		return h.openTidings()
	}
	n, ok := findNotice(f.Notices, id)
	if !ok {
		return nil
	}
	out := []Intent{{Kind: IntentNotice, Target: id, Command: NoticeOpen}}
	if kind, ok := alarmCard[strings.TrimPrefix(id, dangerPrefix)]; ok && strings.HasPrefix(id, dangerPrefix) {
		h.FocusCalamity(kind, f)
	}
	switch {
	case n.At != nil:
		return append(out, h.focus(*n.At)...)
	case n.Intent != "":
		return append(out, h.Press(n.Intent, f)...)
	}
	return append(out, h.openTidings()...)
}

const dangerPrefix = "danger:"

var alarmCard = map[string]string{
	AlertPending: view.TagUnhoused, AlertFire: view.TagCrashLoop, AlertGhost: view.TagImagePull, AlertFallen: view.TagEvicted,
}

func (h *HUD) dismissNotice() ([]Intent, bool) {
	id, ok := strings.CutPrefix(h.Hover.ID, noticePrefix)
	if !h.Hovering || !ok || id == noticeMore {
		return nil, false
	}
	return []Intent{{Kind: IntentNotice, Target: id, Command: NoticeDismiss}, {Kind: IntentSound, Command: "click"}}, true
}

func RailSlide(age float64, reduce bool) float64 {
	if reduce || age < 0 || age >= railSlideFor {
		return 0
	}
	p := 1 - age/railSlideFor
	return railSlideBy * p * p
}

func RailPulse(age float64, reduce bool) float64 {
	if reduce || age < 0 || age >= RailPulseFor {
		return 0
	}
	return (.5 + .5*math.Sin(age*2*math.Pi*railPulseHz)) * (1 - age/RailPulseFor)
}

func (h *HUD) drawRail(dst *ebiten.Image, f Frame) {
	if rail := h.RailRect(f); rail.W > 0 {
		drawRailGrip(dst, rail)
	}
	for _, b := range h.RailButtons(f) {
		id := strings.TrimPrefix(b.ID, noticePrefix)
		n, ok := findNotice(f.Notices, id)
		if !ok {
			h.drawRailMore(dst, b)
			continue
		}
		age := f.Clock - n.Since
		h.drawNotice(dst, b.Rect.Offset(int(RailSlide(age, f.Reduce)), 0), n, RailPulse(age, f.Reduce), h.Hovering && h.Hover.ID == b.ID)
	}
}

func (h *HUD) drawNotice(dst *ebiten.Image, r ui.Rect, n Notice, pulse float64, hot bool) {
	frame := railFrames[n.Kind]
	ui.Fill(dst, r.Offset(1, 2), ui.Palette.Shadow)
	fill := railFills[n.Kind]
	if hot {
		fill = ui.Palette.BlueDark
	}
	ui.Fill(dst, r, fill)
	ui.Stroke(dst, r, ui.Palette.Ink)
	ui.Stroke(dst, r.Inset(1), frame)
	ui.Stroke(dst, r.Inset(2), ui.Tint(frame, uint8(60+160*pulse)))
	if pulse > 0 {
		ui.Stroke(dst, r.Inset(-1), ui.Tint(ui.Palette.GoldBright, uint8(200*pulse)))
	}
	h.icon(dst, n.Icon, r.X+r.W/2-8, r.Y+r.H/2-8)
	if n.Count > 1 {
		h.drawBadge(dst, r, n.Count, frame)
	}
}

func BadgeRect(tile ui.Rect, textW int) ui.Rect {
	w, h := textW+2*railBadgePad, ui.CapHeight+2*railBadgePad
	return ui.Rect{X: tile.X + tile.W - w - 1, Y: tile.Y + tile.H - h - 1, W: w, H: h}
}

func (h *HUD) drawBadge(dst *ebiten.Image, r ui.Rect, count int, frame color.RGBA) {
	text := countBadge(count)
	w, _ := h.Font.Measure(text, 1)
	badge := BadgeRect(r, w)
	ui.Fill(dst, badge, frame)
	ui.Stroke(dst, badge, ui.Palette.Ink)
	h.Font.Draw(dst, text, badge.X+railBadgePad, badge.Y+railBadgePad, 1, ui.Palette.Ink)
}

func (h *HUD) drawRailMore(dst *ebiten.Image, b Button) {
	ui.Fill(dst, b.Rect.Offset(1, 2), ui.Palette.Shadow)
	ui.Fill(dst, b.Rect, ui.Palette.Console)
	ui.Stroke(dst, b.Rect, ui.Palette.ParchmentDark)
	h.Font.DrawCentered(dst, b.Label, b.Rect, 1, ui.Palette.Cream)
}

func drawRailGrip(dst *ebiten.Image, rail ui.Rect) {
	grip := ui.Rect{X: rail.X, Y: rail.Y, W: rail.W, H: railGripH - 1}
	ui.Fill(dst, grip, ui.Tint(ui.Palette.Console, 220))
	ui.Stroke(dst, grip, ui.Palette.ParchmentDark)
	step := grip.W / (railGripDots + 1)
	for i := 1; i <= railGripDots; i++ {
		ui.Fill(dst, ui.Rect{X: grip.X + i*step - 1, Y: grip.Y + grip.H/2 - 1, W: 2, H: 2}, ui.Palette.Gold)
	}
}

func countBadge(n int) string {
	return "x" + strconv.Itoa(min(n, 99))
}

func (h *HUD) noticeOnTurn(f Frame) (Notice, bool) {
	key, ok := strings.CutPrefix(h.Director.Stage.Slot.Key, ActNotice)
	if !ok {
		return Notice{}, false
	}
	return findNotice(f.Notices, key)
}
