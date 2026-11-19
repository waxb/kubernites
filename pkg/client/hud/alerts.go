package hud

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	AlertFire    = "fire"
	AlertGhost   = "ghost"
	AlertFallen  = "fallen"
	AlertPending = "pending"
	AlertKeep    = "keep"
	AlertDark    = "dark"
	AlertImp     = "imp"
	AlertSiege   = "siege"
	AlertStorm   = "storm"

	LevelNotice = 0
	LevelWarn   = 1
	LevelDanger = 2

	arrowMargin  = arrowPoint + arrowWing + 2 + TipClear
	arrowHalf    = 10
	clusterDist  = 30
	maxArrows    = 10
	onScreenPad  = 12
	pingLife     = 1.2
	blipHalf     = 3
	stepAroundBy = 4
)

var alertLevels = map[string]int{
	AlertFire: LevelDanger, AlertDark: LevelDanger, AlertStorm: LevelDanger,
	AlertKeep: LevelWarn, AlertGhost: LevelWarn, AlertFallen: LevelWarn, AlertImp: LevelWarn, AlertSiege: LevelWarn,
	AlertPending: LevelNotice,
}

var alertIcons = map[string]string{
	AlertFire: "icon/alert", AlertDark: "icon/codex", AlertStorm: "icon/node", AlertKeep: "icon/node", AlertGhost: "icon/soulstone",
	AlertFallen: "icon/pod", AlertImp: "icon/probe", AlertSiege: "icon/siege", AlertPending: "icon/pod",
}

type AlertArrow struct {
	X, Y   int
	DX, DY float64
	Level  int
	Kind   string
	Count  int
	Target Alert
	Texts  []string
}

type ping struct {
	on    bool
	at    [2]float64
	start float64
}

func AlertLevel(kind string) int {
	return alertLevels[kind]
}

func SeverityOf(a Alert) int {
	if a.Quiet {
		return LevelNotice
	}
	return AlertLevel(a.Kind)
}

func alertIcon(kind string) string {
	if icon, ok := alertIcons[kind]; ok {
		return icon
	}
	return "icon/alert"
}

func ViewPoint(quad [4][2]float64, gx, gy float64) (float64, float64, bool) {
	o := quad[0]
	ax, ay := quad[1][0]-o[0], quad[1][1]-o[1]
	bx, by := quad[3][0]-o[0], quad[3][1]-o[1]
	det := ax*by - ay*bx
	if math.Abs(det) < 1e-9 {
		return 0, 0, false
	}
	px, py := gx-o[0], gy-o[1]
	u := (px*by - py*bx) / det
	v := (ax*py - ay*px) / det
	return u * Width, TopH + v*(ConsoleY-TopH), true
}

func worldView() ui.Rect {
	return ui.Rect{X: 0, Y: TopH, W: Width, H: ConsoleY - TopH}
}

func arrowTrack() ui.Rect {
	return worldView().Inset(arrowMargin)
}

func bySeverity(alerts []Alert) []Alert {
	out := append([]Alert(nil), alerts...)
	sort.SliceStable(out, func(i, j int) bool { return SeverityOf(out[i]) > SeverityOf(out[j]) })
	return out
}

func edgeAnchor(sx, sy float64) (float64, float64, float64, float64) {
	track := arrowTrack()
	cx, cy := float64(track.X)+float64(track.W)/2, float64(track.Y)+float64(track.H)/2
	dx, dy := sx-cx, sy-cy
	n := math.Hypot(dx, dy)
	if n == 0 {
		return cx, cy, 0, -1
	}
	t := math.Inf(1)
	if dx != 0 {
		t = math.Min(t, float64(track.W)/2/math.Abs(dx))
	}
	if dy != 0 {
		t = math.Min(t, float64(track.H)/2/math.Abs(dy))
	}
	return cx + dx*t, cy + dy*t, dx / n, dy / n
}

func stepAround(x, y int, blocked []ui.Rect) (int, int) {
	track := arrowTrack()
	for range stepAroundBy {
		hit := false
		for _, b := range blocked {
			zone := b.Inset(-arrowHalf)
			if !zone.Contains(x, y) {
				continue
			}
			hit = true
			x, y = nearestOutside(x, y, zone, track)
		}
		if !hit {
			break
		}
	}
	return x, y
}

func nearestOutside(x, y int, zone, track ui.Rect) (int, int) {
	cands := [][2]int{{zone.X - 1, y}, {zone.X + zone.W + 1, y}, {x, zone.Y - 1}, {x, zone.Y + zone.H + 1}}
	best, bx, by := math.MaxInt, x, y
	for _, c := range cands {
		if !track.Contains(c[0], c[1]) {
			continue
		}
		if d := iabs(c[0]-x) + iabs(c[1]-y); d < best {
			best, bx, by = d, c[0], c[1]
		}
	}
	return bx, by
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func AlertArrows(f Frame, blocked []ui.Rect) []AlertArrow {
	visible := worldView().Inset(onScreenPad)
	out := []AlertArrow{}
	for _, a := range bySeverity(f.Alerts) {
		sx, sy, ok := ViewPoint(f.View, a.At[0], a.At[1])
		if !ok {
			return nil
		}
		if visible.Contains(int(sx), int(sy)) {
			continue
		}
		ex, ey, dx, dy := edgeAnchor(sx, sy)
		x, y := stepAround(int(math.Round(ex)), int(math.Round(ey)), blocked)
		out = mergeArrow(out, AlertArrow{X: x, Y: y, DX: dx, DY: dy, Level: SeverityOf(a), Kind: a.Kind, Count: 1, Target: a, Texts: []string{a.Text}})
	}
	return out
}

var perCause = map[string]bool{AlertPending: true, AlertFire: true, AlertGhost: true, AlertFallen: true}

func mergeArrow(arrows []AlertArrow, a AlertArrow) []AlertArrow {
	for i, b := range arrows {
		if iabs(b.X-a.X)+iabs(b.Y-a.Y) < clusterDist || b.Kind == a.Kind && perCause[a.Kind] {
			next := b
			next.Count++
			next.Texts = append(append([]string(nil), b.Texts...), a.Texts...)
			out := append([]AlertArrow(nil), arrows...)
			out[i] = next
			return out
		}
	}
	if len(arrows) >= maxArrows {
		return arrows
	}
	return append(arrows, a)
}

func arrowTip(a AlertArrow) string {
	tip := a.Target.Text
	if a.Count > 1 {
		tip += fmt.Sprintf(" (x%d: +%d more)", a.Count, a.Count-1)
	}
	return tip + ". Click to look [Tab: next alert]"
}

func ArrowRect(a AlertArrow) ui.Rect {
	return ui.Rect{X: a.X - arrowHalf, Y: a.Y - arrowHalf, W: 2 * arrowHalf, H: 2 * arrowHalf}
}

func alertButtons(_ Frame, arrows []AlertArrow) []Button {
	out := make([]Button, 0, len(arrows))
	for i, a := range arrows {
		out = append(out, Button{ID: "alert:" + strconv.Itoa(i), Rect: ArrowRect(a), Tip: arrowTip(a)})
	}
	return out
}

func (h *HUD) arrowsShown() bool {
	return h.OverlayID == OverlayNone || (h.OverlayID == OverlayStone && !h.Stone.Tall)
}

func (h *HUD) BlockedRects(f Frame) []ui.Rect {
	out := []ui.Rect{}
	if h.ObjectiveShown(f) {
		out = append(out, h.ObjectiveRectOf(f))
	}
	if len(h.LessonLines()) > 0 {
		out = append(out, h.LessonRect(f))
	}
	if SiegeActive(f) {
		out = append(out, SiegeStripRect(f))
	}
	out = append(out, FreightRects(f)...)
	if h.IntroShown(f) {
		out = append(out, Shadowed(h.IntroCard(f)))
	}
	if h.IntroShown(f) && h.IntroHidden(f) {
		if r, ok := h.IntroHole(f); ok {
			out = append(out, r.Inset(-EdgeArrowTip))
		}
	}
	if r, ok := h.CalamityRect(f); ok {
		out = append(out, r)
	}
	if h.OverlayID == OverlayStone {
		out = append(out, h.StoneRect())
	}
	if rail := h.RailRect(f); rail.W > 0 {
		out = append(out, rail)
	}
	if h.LegendShown(f) {
		out = append(out, LegendRect(h.lens))
	}
	return append(out, h.CrewRects(f)...)
}

func (h *HUD) Arrows(f Frame) []AlertArrow {
	if !h.arrowsShown() {
		return nil
	}
	blocked := h.BlockedRects(f)
	clear := append(inflated(append(slices.Clone(blocked), f.WorldBoxes...), TipClear), TopBar)
	blocked = append(blocked, inflated(f.WorldBoxes, arrowMargin-arrowHalf)...)
	if r, ok := h.ConsoleSpan(f); ok {
		clear = append(clear, r)
	}
	if r, ok := h.SurveyRect(f); ok {
		clear = append(clear, r.Inset(-TipClear))
	}

	return slices.DeleteFunc(AlertArrows(f, blocked), func(a AlertArrow) bool { return OverlapsAny(h.ArrowSpan(a), clear) })
}

func (h *HUD) ArrowSpan(a AlertArrow) ui.Rect {
	r := ArrowRect(a)
	rects := []ui.Rect{Shadowed(r), ui.Rect{X: a.X + int(a.DX*arrowPoint), Y: a.Y + int(a.DY*arrowPoint), W: 1, H: 1}.Inset(-arrowWing - 2)}
	if a.Count > 1 {
		rects = append(rects, h.countBadgeRect(a.Count, r))
	}
	return spanOf(rects...)
}

func (h *HUD) alertOverlayButtons(f Frame) []Button {
	out := alertButtons(f, h.Arrows(f))
	return append(out, h.blipButtons(f)...)
}

func (h *HUD) blipButtons(f Frame) []Button {
	if f.Site.Width == 0 {
		return nil
	}
	v := h.minimapView(f.Site)
	inner := Minimap.Inset(4)
	out := make([]Button, 0, len(f.Alerts))
	for i, a := range f.Alerts {
		x, y := v.FromGrid(a.At[0], a.At[1])
		if !inner.Contains(int(x), int(y)) {
			continue
		}
		out = append(out, Button{ID: "blip:" + strconv.Itoa(i), Rect: ui.Rect{X: int(x) - blipHalf, Y: int(y) - blipHalf, W: 2*blipHalf + 1, H: 2*blipHalf + 1}, Tip: a.Text})
	}
	return out
}

func (h *HUD) focus(at [2]float64) []Intent {
	h.ping = ping{on: true, at: at, start: h.Now}
	return []Intent{{Kind: IntentFocus, X: at[0], Y: at[1]}, {Kind: IntentSound, Command: "click"}}
}

func (h *HUD) pressAlert(rest string, f Frame) []Intent {
	i, err := strconv.Atoi(rest)
	arrows := h.Arrows(f)
	if err != nil || i < 0 || i >= len(arrows) {
		return nil
	}
	if kind, ok := alarmCard[arrows[i].Target.Kind]; ok {
		h.FocusCalamity(kind, f)
	}
	return h.focus(arrows[i].Target.At)
}

func (h *HUD) pressBlip(rest string, f Frame) []Intent {
	i, err := strconv.Atoi(rest)
	if err != nil || i < 0 || i >= len(f.Alerts) {
		return nil
	}
	if kind, ok := alarmCard[f.Alerts[i].Kind]; ok {
		h.FocusCalamity(kind, f)
	}
	return h.focus(f.Alerts[i].At)
}

func (h *HUD) nextAlert(f Frame) []Intent {
	sorted := bySeverity(f.Alerts)
	if len(sorted) == 0 {
		return nil
	}
	a := sorted[h.alertCursor%len(sorted)]
	h.alertCursor = (h.alertCursor + 1) % len(sorted)
	return h.focus(a.At)
}

func HomeOf(site view.Site) [2]float64 {
	for _, s := range site.Structures {
		if s.Kind == view.StructKeep {
			return [2]float64{float64(s.X) + float64(s.W)/2, float64(s.Y) + float64(s.H)/2}
		}
	}
	return [2]float64{float64(site.Gate.X), float64(site.Gate.Y)}
}

func (h *HUD) AvoidRects(f Frame) []ui.Rect {
	f = h.Stage(f)
	out := h.BlockedRects(f)
	if card, ok := h.GuideCardRect(f); ok {
		out = append(out, card)
	}
	return append(out, h.GuideTargets(f)...)
}
