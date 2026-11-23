package hud

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const KMapCardID = "info:kmap-card"

const (
	kmapFrame     = 11
	kmapMargin    = 6
	kmapBannerW   = 80
	kmapRibbonH   = 18
	kmapStripH    = 22
	KmapCardID    = KMapCardID
	KmapEraPrefix = "era:"
	KmapPrefix    = "kmap:"
	pressedEmpty  = "empty"
	pressedPlaque = "plaque"
)

var (
	KnowledgeBox   = ui.Rect{X: kmapMargin, Y: TopH + 1, W: Width - 2*kmapMargin, H: Height - TopH - 1}
	KmapInner      = KnowledgeBox.Inset(kmapFrame)
	KnowledgeView  = ui.Rect{X: KmapInner.X + kmapBannerW, Y: KmapInner.Y + kmapRibbonH + 2, W: KmapInner.W - kmapBannerW, H: KmapRowsTiles * KMapTile}
	kmapRibbonRow  = ui.Rect{X: KnowledgeView.X, Y: KmapInner.Y, W: KnowledgeView.W, H: kmapRibbonH}
	kmapBannerCol  = ui.Rect{X: KmapInner.X, Y: KnowledgeView.Y, W: kmapBannerW, H: KnowledgeView.H}
	KnowledgeStrip = ui.Rect{X: KnowledgeView.X, Y: KmapInner.Y + KmapInner.H - kmapStripH - 2, W: KnowledgeView.W, H: kmapStripH}
	KmapHeld       = ebiten.IsKeyPressed
)

type kmapState struct {
	Scroll  KmapScroll `json:"-"`
	opened  bool
	clock   float64
	Focus   string `json:"-"`
	pressed string
	canvas  cachedImage
	card    cachedImage
	Held    bool `json:"-"`
	heldAt  [2]int
	heldSet bool
}

func (h *HUD) KeyFocus() bool {
	return h.TextFocus() || h.OverlayID == OverlayGrimoire || h.OverlayID == OverlayBook
}

func (h *HUD) Knowledge(f Frame) KMapLayout {
	l := LayoutKnowledge(f.State.Grimoire, KmapRowsTiles)
	h.openKnowledge(f, l)
	return l
}

func KmapLimit(l KMapLayout) float64 {
	return math.Max(float64(l.Width*KMapTile-KnowledgeView.W), 0)
}

func eraIndex(age string) int {
	return max(slices.Index(view.KnowledgeEras, age), 0)
}

func EraCentre(l KMapLayout, era int) float64 {
	e := l.Eras[era]
	return float64((2*e.Start+e.Size)*KMapTile/2 - KnowledgeView.W/2)
}

func (h *HUD) openKnowledge(f Frame, l KMapLayout) {
	if h.Kmap.opened || h.OverlayID != OverlayGrimoire {
		return
	}
	h.Kmap.Scroll = KmapScroll{Pos: EraCentre(l, eraIndex(f.State.Age))}.clamp(KmapLimit(l))
	h.Kmap.opened, h.Kmap.clock, h.Kmap.Focus, h.Kmap.pressed = true, f.Clock, "", ""
	h.focusCalling(f, l)
}

func (h *HUD) JumpKnowledge(f Frame, era string) {
	l := h.Knowledge(f)
	h.Kmap.Scroll = h.Kmap.Scroll.GlideTo(EraCentre(l, eraIndex(era)), KmapLimit(l), f.Reduce)
}

func (h *HUD) KnowledgeScroll() float64 { return h.Kmap.Scroll.Pos }

func (h *HUD) ScreenOf(r ui.Rect) ui.Rect {
	return h.viewPen().at(r)
}

func clipRect(a, b ui.Rect) (ui.Rect, bool) {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return ui.Rect{}, false
	}
	return ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, true
}

func (h *HUD) GrimoireButtons(f Frame) []Button {
	l := h.Knowledge(f)
	out := []Button{}
	for _, p := range l.Plaques {
		if r, ok := clipRect(h.ScreenOf(p.Rect()), KnowledgeView); ok {
			out = append(out, Button{ID: "grim:" + p.Page.ID, Rect: r})
		}
	}
	for i, r := range StripSegments(l) {
		out = append(out, Button{ID: KmapPrefix + KmapEraPrefix + view.KnowledgeEras[i], Rect: r, Tip: eraTip(f, i)})
	}
	out = append(out, h.CardButtons(f, l)...)
	return append(out, closeButton(KnowledgeBox))
}

func (h *HUD) focusPlaque(l KMapLayout) (KMapPlaque, bool) {
	id := h.Kmap.Focus
	if id == "" {
		id = h.Grimoire
	}
	p, ok := l.Find(id)
	if !ok {
		return p, false
	}
	_, seen := clipRect(h.ScreenOf(p.Rect()), KnowledgeView)
	return p, seen
}

func (h *HUD) stepKnowledge(in Input, f Frame) {
	if h.OverlayID != OverlayGrimoire {
		h.Kmap.opened, h.Kmap.clock = false, f.Clock
		return
	}
	l := h.Knowledge(f)
	dt := f.Clock - h.Kmap.clock
	h.Kmap.clock = f.Clock
	buttons := h.GrimoireButtons(f)
	hit, ok := HitButton(buttons, in.MX, in.MY)
	if !h.holdsOpenedPage(in) {
		h.trackFocus(h.cardReach(l, buttons), hit, ok, in)
	}
	if in.Click {
		h.grabMap(in, hit, ok)
	}
	was := h.Kmap.Scroll.Drag
	h.Kmap.Scroll = h.Kmap.Scroll.Step(h.scrollInput(in, dt), KmapLimit(l))
	if was && !h.Kmap.Scroll.Drag && !h.Kmap.Scroll.Moved && h.Kmap.pressed == pressedEmpty {
		h.Grimoire, h.Kmap.Focus = "", ""
	}
}

type cardReach struct {
	card, bridge ui.Rect
	open         bool
}

func (h *HUD) cardReach(l KMapLayout, buttons []Button) cardReach {
	card, ok := HitID(buttons, KmapCardID)
	if !ok {
		return cardReach{}
	}
	plate, _ := HitID(buttons, "grim:"+h.focusedPage(l))
	return cardReach{card: card.Rect, bridge: spanOf(plate.Rect, card.Rect), open: true}
}

func (h *HUD) holdsOpenedPage(in Input) bool {
	at := [2]int{in.MX, in.MY}
	switch {
	case !h.Kmap.Held:
		return false
	case !h.Kmap.heldSet:
		h.Kmap.heldAt, h.Kmap.heldSet = at, true
		return true
	case at == h.Kmap.heldAt:
		return true
	}
	h.Kmap.Held, h.Kmap.heldSet = false, false
	return false
}

func (h *HUD) closePageCard() bool {
	if h.Grimoire == "" && h.Kmap.Focus == "" {
		return false
	}
	h.Grimoire, h.Kmap.Focus, h.Kmap.Held, h.Kmap.heldSet = "", "", true, false
	return true
}

func (h *HUD) focusedPage(l KMapLayout) string {
	if p, ok := h.focusPlaque(l); ok {
		return p.Page.ID
	}
	return ""
}

func (h *HUD) trackFocus(reach cardReach, hit Button, ok bool, in Input) {
	switch {
	case reach.open && reach.card.Contains(in.MX, in.MY):
	case ok && strings.HasPrefix(hit.ID, "grim:"):
		h.Kmap.Focus = strings.TrimPrefix(hit.ID, "grim:")
	case !ok && reach.open && reach.bridge.Contains(in.MX, in.MY):
	default:
		h.Kmap.Focus = ""
	}
}

func (h *HUD) grabMap(in Input, hit Button, ok bool) {
	h.Kmap.pressed = ""
	if !KnowledgeView.Contains(in.MX, in.MY) {
		return
	}
	switch {
	case !ok:
		h.Kmap.pressed = pressedEmpty
	case strings.HasPrefix(hit.ID, "grim:"):
		h.Kmap.pressed = pressedPlaque
	default:
		return
	}
	h.Kmap.Scroll = h.Kmap.Scroll.Grab(in.MX)
}

func (h *HUD) scrollInput(in Input, dt float64) ScrollInput {
	wheel := 0.0
	if KnowledgeBox.Contains(in.MX, in.MY) {
		wheel = -(in.Wheel + in.WheelX)
	}
	return ScrollInput{Dt: dt, Push: kmapPush(), Wheel: wheel, Held: in.Held, X: in.MX, Reduce: h.Reduce}
}

func kmapPush() float64 {
	push := 0.0
	if KmapHeld(ebiten.KeyA) || KmapHeld(ebiten.KeyArrowLeft) {
		push--
	}
	if KmapHeld(ebiten.KeyD) || KmapHeld(ebiten.KeyArrowRight) {
		push++
	}
	return push
}

func (h *HUD) PressKnowledge(rest string, f Frame) []Intent {
	era, ok := strings.CutPrefix(rest, KmapEraPrefix)
	if !ok || !slices.Contains(view.KnowledgeEras, era) {
		return nil
	}
	h.JumpKnowledge(f, era)
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func learnLabel(p view.GrimoirePage) string {
	return learnWord(p) + "  " + costText(p.Cost.Gold, p.Cost.Wisdom)
}

func learnWord(p view.GrimoirePage) string {
	if p.MaxRank > 1 {
		return "LEARN " + romanOf(p.Rank)
	}
	return "LEARN"
}

func shortfall(p view.GrimoirePage, gold, wisdom float64) string {
	parts := []string{}
	if gold < p.Cost.Gold {
		more := Compact(math.Ceil(p.Cost.Gold - gold))
		parts = append(parts, ui.Bridge(more+" more gold", "$"+more+" more"))
	}
	if wisdom < p.Cost.Wisdom {
		parts = append(parts, fmt.Sprintf("%s more wisdom", Compact(math.Ceil(p.Cost.Wisdom-wisdom))))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Needs " + strings.Join(parts, " and ")
}

func eraTip(f Frame, era int) string {
	tip := "Scroll the map to " + eraNames[era].title
	if ages := f.State.Road.Ages; era < len(ages) && ages[era].Goal != "" {
		tip += "\n" + EraText(ages[era], era, f.TrueNames) + "."
	}
	return tip
}
