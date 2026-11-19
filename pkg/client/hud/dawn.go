package hud

import (
	"image/color"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	dawnLife    = 120.0
	DawnFlash   = 2.5
	dawnListMax = 4
	dawnGap     = 8
	DawnMore    = "dawn:more"
	dawnMoreW   = 48
)

type ageDawn struct {
	watching string
	kinds    []string
	age      string
	at       float64
	Opened   []string `json:"-"`
	shown    bool
	More     bool `json:"-"`
}

func buildNames(f Frame) []string {
	out := make([]string, 0, len(f.State.Buildables))
	for _, b := range f.State.Buildables {
		out = append(out, b.Name)
	}
	return out
}

func newPages(f Frame) []string {
	out := []string{}
	for _, p := range f.State.Grimoire {
		if p.Era == f.State.Age && p.State != "hidden" {
			out = append(out, p.Name)
		}
	}
	return out
}

func newBuilds(before []string, f Frame) []string {
	out := []string{}
	for _, name := range buildNames(f) {
		if !slices.Contains(before, name) {
			out = append(out, name)
		}
	}
	return out
}

func (h *HUD) watchDawn(f Frame) {
	d := &h.Dawn
	age := f.State.Age
	switch {
	case age == "":
		return
	case d.watching == "":
		d.watching = age
	case d.watching != age:
		h.StartDawn(age, newBuilds(d.kinds, f), newPages(f))
	}
	d.kinds = buildNames(f)
	switch {
	case d.age == "":
	case h.OverlayID == OverlayAge:
		d.shown = true
	case d.shown || h.Now-d.at > dawnLife:
		d.age, d.Opened, d.shown = "", nil, false
	}
}

func (h *HUD) StartDawn(age string, builds, pages []string) {
	h.Dawn = ageDawn{watching: age, age: age, at: h.Now, Opened: dawnLines(builds, pages)}
}

func dawnLines(builds, pages []string) []string {
	out := []string{}
	if len(builds) > 0 {
		out = append(out, "You may now BUILD: "+joinMax(builds, dawnListMax)+".")
	}
	if len(pages) > 0 {
		out = append(out, "The Grimoire opens "+countWord(len(pages), "page")+": "+joinMax(pages, dawnListMax)+".")
	}
	return out
}

func joinMax(names []string, limit int) string {
	if len(names) <= limit {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:limit], ", ") + " and " + countWord(len(names)-limit, "more")
}

func countWord(n int, word string) string {
	if word == "more" {
		return strconv.Itoa(n) + " more"
	}
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

func (h *HUD) DawnLive(f Frame) bool {
	return h.Dawn.age != "" && h.Dawn.age == f.State.Age
}

func (h *HUD) AgeTitle(f Frame) string {
	title := "THE " + strings.ToUpper(f.State.Age) + " AGE"
	if h.DawnLive(f) {
		title += " DAWNS"
	}
	return title
}

func (h *HUD) DawnLinesShown(f Frame) bool {
	return h.OverlayID == OverlayAge && h.DawnLive(f)
}

func (h *HUD) DrawDawn(dst *ebiten.Image, f Frame, r ui.Rect, y int) int {
	if !h.DawnLive(f) {
		return y
	}
	ui.Fill(dst, ui.Rect{X: r.X, Y: y, W: r.W, H: 1}, ui.Palette.Gold)
	y += dawnGap / 2
	if !h.Dawn.More {
		y = h.drawDawnRows(dst, h.dawnThree(f), r, y)
	}
	if b, ok := h.dawnMoreButton(f); ok {
		h.drawButton(dst, b)
		y = b.Rect.Y + b.Rect.H + dawnGap/2
	}
	if !h.Dawn.More {
		return y + dawnGap
	}
	opened, changed := h.dawnRows(f)
	y = h.drawDawnList(dst, "WHAT OPENED", opened, r, y)
	if len(changed) > 0 {
		y = h.drawDawnList(dst, "WHAT CHANGED", changed, r, y+dawnGap/2)
	}
	return y + dawnGap
}

func (h *HUD) dawnThree(f Frame) []string {
	if m := f.State.AgeMoment; m != nil && m.Age == f.State.Age && len(m.Lines) > 0 {
		return m.Lines
	}
	return h.Dawn.Opened
}

func (h *HUD) DawnWrap(line string, w int) []string {
	return h.Font.Wrap(line, w-12, 1)
}

func (h *HUD) dawnRowsHeight(lines []string, w int) int {
	height := 0
	for _, line := range lines {
		height += len(h.DawnWrap(line, w))*ui.LineHeight + 2
	}
	return height
}

func (h *HUD) drawDawnRows(dst *ebiten.Image, lines []string, r ui.Rect, y int) int {
	for _, line := range lines {
		for i, row := range h.DawnWrap(line, r.W) {
			lead := "  "
			if i == 0 {
				lead = "+ "
			}
			h.Font.Draw(dst, lead+row, r.X, y, 1, ui.Palette.Cream)
			y += ui.LineHeight
		}
		y += 2
	}
	return y
}

func (h *HUD) dawnMoreButton(f Frame) (Button, bool) {
	if !h.DawnLinesShown(f) {
		return Button{}, false
	}
	opened, changed := h.dawnRows(f)
	if len(opened)+len(changed) == 0 {
		return Button{}, false
	}
	r := AgeLeft()
	y := h.stageThemeBottom(f, r) + dawnGap/2
	label, tip := "MORE", "What else the new Age opened and changed"
	if h.Dawn.More {
		label, tip = "LESS", "Back to the lines that matter now"
	} else {
		y += h.dawnRowsHeight(h.dawnThree(f), r.W)
	}
	return Button{ID: DawnMore, Rect: ui.Rect{X: r.X, Y: y, W: dawnMoreW, H: 14}, Label: label, Tip: tip}, true
}

func (h *HUD) stageThemeBottom(f Frame, r ui.Rect) int {
	theme := StageThemeLine(f)
	if theme == "" {
		return r.Y
	}
	return r.Y + len(h.Font.WrapLines(theme, r.W, 1, themeRows))*ui.LineHeight + themeGap
}

func (h *HUD) drawDawnList(dst *ebiten.Image, title string, lines []string, r ui.Rect, y int) int {
	if title != "" {
		h.Font.DrawShadow(dst, title, r.X, y, 1, ui.Palette.Gold)
		y += ui.LineHeight + 2
	}
	for _, line := range lines {
		for i, row := range h.DawnWrap(line, r.W) {
			lead := "  "
			if i == 0 {
				lead = "+ "
			}
			h.Font.Draw(dst, lead+row, r.X, y, 1, ui.Palette.Cream)
			y += ui.LineHeight
		}
		y += 2
	}
	return y
}

func (h *HUD) dawnRows(f Frame) ([]string, []string) {
	m := f.State.AgeMoment
	if m == nil || m.Age != f.State.Age || len(m.Opened)+len(m.Changed) == 0 {
		return h.Dawn.Opened, nil
	}
	return m.Opened, m.Changed
}

func (h *HUD) drawDawnPicture(dst *ebiten.Image, f Frame, textBottom int) {
	r := AgeVigRect()
	if !h.DawnLive(f) || f.Revealed("loom") || textBottom+vigGap > r.Y {
		return
	}
	h.DrawVig(dst, Vig{Scene: art.AgedScene(art.SceneAgeUp, gfx.AgeTier(f.State.Age)), Seed: gfx.VignetteSeed(f.State.Age)}, r, 1)
}

func (h *HUD) DawnRings() []ui.Rect {
	age := h.Now - h.Dawn.at
	if h.Dawn.age == "" || age < 0 || age > DawnFlash {
		return nil
	}
	return []ui.Rect{Overlay.Inset(1), Overlay.Inset(2), Overlay.Inset(3)}
}

func (h *HUD) drawDawnFlash(dst *ebiten.Image) {
	rings := h.DawnRings()
	if len(rings) == 0 {
		return
	}
	a := uint8(200 * (1 - (h.Now-h.Dawn.at)/DawnFlash))
	for i, tint := range []color.RGBA{ui.Tint(ui.Palette.GoldBright, a), ui.Tint(ui.Palette.Gold, a/2), ui.Tint(ui.Palette.Gold, a/4)} {
		ui.Stroke(dst, rings[i], tint)
	}
}
