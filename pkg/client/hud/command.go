package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	CellW   = 72
	CellH   = 34
	cellGap = 3
)

type mainEntry struct {
	reveal string
	button Button
}

var mainEntries = []mainEntry{
	{"build", Button{ID: "cmd:build", Label: "BUILD", Menu: true, Icon: "icon/build", Tip: "Raise buildings: Keeps and the control plane [B]"}},
	{"summon", Button{ID: "cmd:summon", Label: "HIRE", Cost: "FREE", Menu: true, Icon: "icon/summon", Tip: "Hire a porter by hand (kubectl run): a worker trained at the Academy. " + summonCost}},
	{"grimoire", Button{ID: "cmd:grimoire", Label: "GRIMOIRE", Menu: true, Icon: "icon/tech", Tip: "Learn Kubernetes lore and its ranks"}},
	{"soulvault", Button{ID: "cmd:soulvault", Label: "ACADEMY", Menu: true, Icon: "icon/academy", Tip: "The Academy across the sea trains Trades: container images"}},
	{"chronicles", Button{ID: "cmd:chronicles", Label: "CHRONICLE", Menu: true, Icon: "icon/chronicle", Tip: "Chronicles of Woe: explain what went wrong"}},
	{"ageup", Button{ID: "cmd:age", Label: "AGE", Menu: true, Icon: "icon/age", Tip: "Advance to the next Age"}},
}

var DoorButton = Button{ID: "cmd:door", Label: "PORTER AT THE DOOR", Icon: "icon/pod",
	Tip: "Hut at the door: a free porter your Keep runs by itself (a static Pod on hostPort 80). It carries boxes beside you and needs no Chancery. It costs no gold, but it eats some of the Keep's food and housing."}

var voyageButton = Button{ID: "cmd:voyage", Label: "VOYAGE", Icon: "icon/voyage", Tip: "Charter the outer isles: new zones for your Keeps [V]"}

func DoorOffered(f Frame) bool {
	return f.Revealed("door") && !DoorBurning(f.Pods) && f.Shown(DoorButton.ID)
}

func DoorBurning(pods []model.PodView) bool {
	for _, p := range pods {
		if p.Static && p.Node == view.TowerName && !p.Terminating {
			return true
		}
	}
	return false
}

func CellRect(i int) ui.Rect {
	col, row := i%3, i/3
	return ui.Rect{X: Command.X + 4 + col*(CellW+cellGap), Y: Command.Y + 4 + row*(CellH+cellGap), W: CellW, H: CellH}
}

func place(buttons []Button) []Button {
	return placeWith(buttons, CellRect)
}

func MainCell(i int) ui.Rect { return CellRect(8 - i) }

func placeWith(buttons []Button, rect func(int) ui.Rect) []Button {
	out := make([]Button, 0, len(buttons))
	for i, b := range buttons {
		if i >= 9 || b.ID == "" {
			continue
		}
		b.Rect = rect(i)
		out = append(out, b)
	}
	return out
}

func backButton() Button {
	return Button{ID: "cmd:back", Label: "BACK", Icon: "icon/back", Tip: "Back [Esc or right click]"}
}

func (h *HUD) CommandButtons(f Frame) []Button {
	if h.objectShown(f) {
		return h.objectGrid(f)
	}
	switch h.PageID {
	case PageBuild:
		return place(withBack(h.buildPage(f)))
	case PageSummon:
		return place(withBack(SoulButtons(f, "summon:", "Hire")))
	case PageMore:
		_, more := h.MainCard(f)
		return place(withBack(more))
	}
	main, _ := h.MainCard(f)
	return main
}

func withBack(buttons []Button) []Button {
	cells := make([]Button, 9)
	copy(cells, buttons[:min(len(buttons), 8)])
	cells[8] = backButton()
	return cells
}

func (h *HUD) BuildButtons(f Frame) []Button {
	out := []Button{}
	for _, b := range f.State.Buildables {
		if !offered(b, f) {
			continue
		}
		if IsKeepKind(b.Kind) {
			out = append(out, KeepButton(b, f))
			continue
		}
		out = append(out, Button{ID: "build:" + b.Kind, Label: BuildLabel(b.Kind, b.Name), Icon: gfx.BuildIcon(b.Kind, gfx.AgeTier(f.State.Age)),
			Cost: chapterPrice(gatedPrice(b), f), Tip: buildTip(b), Disabled: !b.Available, Why: whyUnless(b.Available, b.Reason), Selected: f.Placing == b.Kind})
	}
	return out
}

func buildTip(b view.Buildable) string {
	return strings.Join(append(buildWords(b), buildReason(b)...), "\n") + "\n" + buildPrice(b)
}

func buildWords(b view.Buildable) []string {
	out := []string{}
	for _, text := range []string{b.Lore, b.Description} {
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func buildReason(b view.Buildable) []string {
	if b.Available || b.Reason == "" {
		return nil
	}
	return []string{b.Reason + "."}
}

func buildPrice(b view.Buildable) string { return "Costs " + Gold(b.Cost) }

var shortLabels = map[string][2]string{"quartermaster": {"QMASTER", "SCHEDULER"}, "stewards-hall": {"STEWARDS", "CONTROLLERS"}, "chancery": {"CHANCERY", "APISERVER"}}

func BuildLabel(kind, name string) string {
	if label, ok := shortLabels[kind]; ok {
		return ui.Bridge(label[0], label[1])
	}
	return strings.TrimPrefix(strings.ToUpper(name), "THE ")
}

func SoulButtons(f Frame, prefix, verb string) []Button {
	out := []Button{}
	for _, s := range f.State.Soulstones {
		if !s.Owned {
			continue
		}
		profession, sprite := soulstone.ProfessionOf(s.Image)
		tip := verb + " " + s.Name + " (" + s.Image + ")"
		b := Button{ID: prefix + s.Image, Label: strings.ToUpper(profession), Icon: "portrait/" + sprite, Cost: ImageTag(s.Image)}
		if prefix == "summon:" {
			b.Tip = withEffect(tip+". "+summonCost, f.Effect(prefix+s.Image))
			out = append(out, summonCell(b, f))
			continue
		}
		b.Tip = withEffect(tip, f.Effect(prefix+s.Image))
		out = append(out, b)
	}
	return out
}

func (h *HUD) drawCommand(dst *ebiten.Image, f Frame) {
	full := h.fullGrid(f)
	if full {
		for i := range GridCells {
			ui.Well(dst, CellRect(i))
		}
	}
	beacon, spot, keyed := h.StepID(f), h.Spotlit(f), h.GridKeysTaken(f)
	if h.guideFaded(f) {
		beacon = ""
	}
	for _, b := range h.CommandButtons(f) {
		h.withAppear(dst, b.ID, f, func(d *ebiten.Image) {
			h.drawCommandCell(d, b, f, full, beacon, spot)
			h.drawCellKey(d, b, keyed)
		})
	}
}

func (h *HUD) fullGrid(f Frame) bool {
	return h.PageID != PageMain || h.objectShown(f)
}

func (h *HUD) drawCommandCell(dst *ebiten.Image, b Button, f Frame, full bool, beacon, spot string) {
	if !full {
		ui.Well(dst, b.Rect)
	}
	h.drawCell(dst, b)
	switch {
	case b.ID == beacon || b.ID == spot:
		h.drawBeacon(dst, b.Rect)
	case b.ID == "cmd:siege" && f.State.Siege.Available && !SiegeActive(f):
		a := uint8(80 + 175*pulse(h.Now, h.Reduce))
		ui.Stroke(dst, b.Rect.Inset(-1), ui.Tint(ui.Palette.GoldBright, a))
	}
}

func (h *HUD) drawCell(dst *ebiten.Image, b Button) {
	b = h.liveGate(b)
	state := ui.ButtonState{Hovered: h.Hovering && h.Hover.ID == b.ID, Disabled: b.Disabled, Selected: b.Selected}
	text := ui.ButtonFace(dst, b.Rect, state)
	r := b.Rect
	iconY := r.Y + 5
	if len(CellLabel(b.Label, ui.MaxChars(r.W-4, 1))) > 1 {
		iconY = r.Y + 2
	}
	badge := ui.Rect{X: r.X + r.W/2 - art.BuildIconW/2, Y: r.Y + 2, W: art.BuildIconW, H: art.BuildIconH}
	switch {
	case !cellShowsIcon(b):
	case strings.HasPrefix(b.Icon, gfx.BuildIconPrefix):
		h.Lib.Fitted(dst, b.Icon, badge)
	case strings.HasPrefix(b.Icon, "portrait/"):
		h.portrait(dst, strings.TrimPrefix(b.Icon, "portrait/"), badge)
	default:
		h.icon(dst, b.Icon, r.X+r.W/2-8, iconY)
	}
	rows := labelRows(b)
	for i, row := range CellLabel(b.Label, ui.MaxChars(r.W-4, 1)) {
		h.Font.DrawCentered(dst, row, rows[i], 1, text)
	}
	if b.Menu {
		drawMenuMark(dst, r)
	}
	h.drawWhy(dst, b)
	if b.Note != "" {
		nw, _ := h.Font.Measure(b.Note, 1)
		h.Font.Draw(dst, b.Note, r.X+r.W-nw-3, r.Y+3, 1, ui.Palette.TextDim)
	}
	h.drawLeyPips(dst, b.Pips, r.X+3, r.Y+3)
	if price, ok := h.priceRect(b); ok {
		ui.Fill(dst, price, ui.Tint(ui.Palette.ConsoleEdge, 200))
		h.Font.Draw(dst, b.Cost, price.X+2, r.Y+3, 1, ui.Palette.GoldBright)
	}
}

func (h *HUD) priceRect(b Button) (ui.Rect, bool) {
	if b.Cost == "" || cellWhy(b) != "" && whyOnTop(b) {
		return ui.Rect{}, false
	}
	cw, _ := h.Font.Measure(b.Cost, 1)
	return ui.Rect{X: b.Rect.X + b.Rect.W - cw - 4, Y: b.Rect.Y + 2, W: cw + 3, H: 9}, true
}

func labelRows(b Button) []ui.Rect {
	r := b.Rect
	out := []ui.Rect{}
	for i := range CellLabel(b.Label, ui.MaxChars(r.W-4, 1)) {
		out = append(out, ui.Rect{X: r.X, Y: r.Y + r.H - 11 - 9*i, W: r.W, H: 9})
	}
	return out
}

var iconFallbacks = map[string]string{"icon/academy": "icon/soulstone", "icon/voyage": "icon/node"}

func (h *HUD) buildPage(f Frame) []Button {
	out := []Button{}
	for _, b := range h.BuildButtons(f) {
		if !stalled(b, f) {
			out = append(out, b)
		}
	}
	if f.State.Tutorial {
		return out
	}
	out = append(out, stallButtons(f)...)
	if voyageOpen(f) {
		out = append(out, voyageButton)
	}
	return out
}

func (h *HUD) portrait(dst *ebiten.Image, name string, box ui.Rect) {
	if h.Lib.Portrait(dst, name, box) {
		return
	}
	if alt, ok := portraitFallbacks[name]; ok {
		h.Lib.Portrait(dst, alt, box)
	}
}

func (h *HUD) drawLeyPips(dst *ebiten.Image, n, x, y int) {
	for i := range n {
		px := x + i*6
		ui.Fill(dst, ui.Rect{X: px + 1, Y: y, W: 2, H: 1}, ui.Palette.Blue)
		ui.Fill(dst, ui.Rect{X: px, Y: y + 1, W: 4, H: 2}, ui.Palette.Blue)
		ui.Fill(dst, ui.Rect{X: px + 1, Y: y + 3, W: 2, H: 1}, ui.Palette.Blue)
	}
}

func (h *HUD) icon(dst *ebiten.Image, name string, x, y int) {
	if !h.Lib.Has(name) {
		if alt, ok := iconFallbacks[name]; ok {
			name = alt
		}
	}
	h.Lib.Icon(dst, strings.TrimPrefix(name, "icon/"), x, y)
}

func CellLabel(label string, width int) []string {
	named := ui.Name(label)
	runes := []rune(named)
	if len(runes) <= width {
		return []string{named}
	}
	cut := strings.LastIndex(string(runes[:width+1]), " ")
	if cut <= 0 {
		return []string{ui.Clip(named, width)}
	}
	return []string{ui.Clip(ui.Settle(strings.TrimSpace(named[cut:])), width), ui.Settle(named[:cut])}
}

func ImageTag(image string) string {
	i := strings.LastIndex(image, ":")
	if i < 0 || i < strings.LastIndex(image, "/") {
		return "latest"
	}
	return image[i+1:]
}
