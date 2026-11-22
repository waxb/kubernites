package hud

import (
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var MenuBox = ui.Rect{X: Width/2 - 100, Y: 54, W: 200, H: 432}

const (
	menuTop      = 38
	menuVersionY = 22
	menuRowH     = 22
	menuFoot     = 18
)

func (h *HUD) Chronicle(f Frame) (view.ChronicleView, bool) {
	if c, ok := h.pickedChronicle(f); ok {
		return c, true
	}
	if h.Now-h.Woe.lastAt > woeStale {
		h.Woe.last = ""
	}
	h.Woe.lastAt = h.Now
	if c, ok := findChronicle(f.State.Chronicles, h.Woe.last); ok && c.Resolved {
		return c, true
	}
	for _, c := range f.State.Chronicles {
		if !c.Resolved {
			h.Woe.last = c.ID
			return c, true
		}
	}
	if n := len(f.State.Chronicles); n > 0 {
		return f.State.Chronicles[n-1], true
	}
	return view.ChronicleView{}, false
}

func (h *HUD) ChronicleButtons(f Frame) []Button {
	out := []Button{}
	c, ok := h.Chronicle(f)
	if ok && !c.Resolved {
		rects := h.AnswerRects(h.ChronicleLayout(c), c)
		for i, o := range c.Options {
			b := Button{ID: "chron:" + c.ID + ":" + o.ID, Rect: rects[i], Tip: o.Text}
			if rows := h.answerRows(o.Text); len(rows) == 1 {
				b.Label = rows[0]
			}
			out = append(out, b)
		}
	}
	if ok && c.Resolved {
		if next, more := firstUnresolved(f.State.Chronicles); more {
			l := h.ChronicleLayout(c)
			out = append(out, Button{ID: WoePrefix + ":" + next.ID, Rect: ui.Rect{X: l.Question.X + l.Question.W - 70, Y: l.Answers - 3, W: 70, H: 15}, Label: "NEXT", Tip: "Read the next calamity to write"})
		}
	}
	if ok {
		out = append(out, h.ShelfButtons(f, h.ChronicleLayout(c))...)
		out = append(out, h.ChronicleAlmanac(f, c)...)
	}
	return append(out, closeButton(Overlay))
}

func (h *HUD) DrawChronicles(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, Overlay)
	h.header(dst, Overlay, "CHRONICLES OF WOE")
	c, ok := h.Chronicle(f)
	if !ok {
		empty := "No calamity has struck the realm yet. When one does, write down what happened here."
		if h.live != nil {
			empty = ui.Bridge("No trouble that struck while the realm was watched has mended yet. Each one that does is written here: when it struck, where, the cluster's words, and when it mended.",
				"No incident seen while watching has resolved yet. Each one that does is listed here: when it started, the object, the cluster's reason, and when it resolved.")
		}
		h.drawTextLines(dst, []string{empty}, Overlay.X+20, Overlay.Y+36, Overlay.W-40, ui.Palette.Text)
		h.DrawOverlayButtons(dst, []Button{closeButton(Overlay)})
		return
	}
	title, gloss := Glossed(strings.ToUpper(c.Calamity), c.K8s)
	if ui.StoryNaming() {
		gloss = ""
	}
	h.Font.Draw(dst, title, Overlay.X+20, Overlay.Y+36, 1, ui.Palette.Gold)
	h.Font.Draw(dst, gloss, Overlay.X+20+h.Font.Span(title, 2), Overlay.Y+36, 1, ui.Palette.Blue)
	h.drawChronicleBody(dst, f, c)
	buttons := PlainButtons(h.ChronicleButtons(f))
	h.DrawOverlayButtons(dst, buttons)
	h.drawLongAnswers(dst, c, buttons)
}

const chronAlmanacH = 15

func (h *HUD) ChronicleAlmanac(f Frame, c view.ChronicleView) []Button {
	p, ok := firstMetPage(f, []string{view.SubjectCalamity, view.SubjectIncident}, splitIDs(c.Kind))
	if !ok || !c.Resolved {
		return nil
	}
	l := h.ChronicleLayout(c)
	verdict := h.Font.Span("Wrongly written.", 2)
	return []Button{{ID: BookGo + p.ID, Rect: ui.Rect{X: l.Question.X + verdict, Y: l.Answers - 3, W: cardAlmanacW, H: chronAlmanacH - 1}, Label: "ALMANAC", Tip: "Why: open the Almanac at " + p.Title, Page: p.ID}}
}

func (h *HUD) drawLongAnswers(dst *ebiten.Image, c view.ChronicleView, buttons []Button) {
	for _, b := range buttons {
		if b.Label != "" || !strings.HasPrefix(b.ID, "chron:"+c.ID+":") {
			continue
		}
		rows := h.answerRows(b.Tip)
		top := b.Rect.Y + (b.Rect.H-len(rows)*ui.LineHeight)/2 + 1
		for i, row := range rows {
			h.Font.Draw(dst, row, b.Rect.X+5, top+i*ui.LineHeight, 1, ui.Palette.Text)
		}
	}
}

const (
	soundTip      = "Music and sounds"
	effectsTip    = "Reduce effects: no vignette, pulses or floating numbers"
	fullscreenTip = "Fill the whole screen, or return to a window [F11 or Alt+Enter]"
	edgeTip       = "Pan the map when the mouse touches the edge of the window"
	namesTip      = "Story names with Kubernetes in brackets, or the real Kubernetes names everywhere (Pod, Node, Service)"
)

func onOff(name string, on bool, yes, no string) string {
	if on {
		return name + ": " + yes
	}
	return name + ": " + no
}

func soundLabel(muted bool) string     { return onOff("SOUND", muted, "OFF", "ON") }
func effectsLabel(reduce bool) string  { return onOff("EFFECTS", reduce, "REDUCED", "FULL") }
func namesLabel(trueNames bool) string { return onOff("NAMES", trueNames, "KUBERNETES", "STORY") }
func edgeLabel(on bool) string         { return onOff("EDGE SCROLL", on, "ON", "OFF") }
func fullscreenLabel(on bool) string   { return onOff("FULLSCREEN", on, "ON", "OFF") }

func (h *HUD) menuLabels(f Frame) [][3]string {
	experience := "EXPERIENCE: " + experienceNames[f.Experience()]
	calm := "CALM REALM: OFF"
	if f.State.CalmRealm {
		calm = "CALM REALM: ON"
	}
	labels := [][3]string{
		{"menu:resume", "RESUME", "Back to the realm [Esc]"},
		{"menu:save", "SAVE REALM", "Write the realm to disk now"},
		{"menu:mute", soundLabel(f.Muted), soundTip},
		{"menu:reduce", effectsLabel(f.Reduce), effectsTip},
		{"menu:coins", CoinFlightsLabel(h.Coins.mode), "Coins that fly from the Treasury to your purse: ALL of them, a FEW (the first ones, then only big payments), or OFF (the purse just pulses)"},
		{"menu:pause", PauseLabel(h.pauseFor), pauseTip},
		{"menu:fullscreen", fullscreenLabel(f.Fullscreen), fullscreenTip},
		{"menu:edge", edgeLabel(h.edgeScroll), edgeTip},
		{"menu:names", namesLabel(f.TrueNames), namesTip},
		{"menu:experience", experience, "The choice from the start, kept in the save: NEW TO STRATEGY (the full guide), STRATEGY (no step holds you back; each lesson arrives as a one-line note and advice comes when a situation needs it) or KUBERNETES (a light guide, NAMES on, the Kubernetes side first). Press to change it."},
		{"menu:calm", calm, "A calm realm sounds no Rush Tide and sends its troubles half as often. Kept in the save; the realm skips the Rush Tide step."},
		{MenuPanels, "RESET PANELS", "Put every panel you moved back in its place. A double click on a panel's title bar puts just that one back."},
		{TrialsButton, "TRIALS", "Short scored drills outside your realm, one idea each: win gold, silver or bronze. Your realm waits as you left it."},
		{TutorialButton, ui.Settle(learnModeLabel), ui.Settle(learnModeTip)},
		{"menu:title", "TITLE SCREEN", "Save and return to the title"},
	}
	return h.liveMenu(labels)
}

func (h *HUD) MenuButtons(f Frame) []Button {
	box := h.menuBox()
	out := []Button{}
	row := 0
	for _, l := range h.menuLabels(f) {
		out = append(out, Button{ID: l[0], Rect: ui.Rect{X: box.X + 20, Y: box.Y + menuTop + row*menuRowH, W: box.W - 40, H: 18}, Label: l[1], Tip: l[2]})
		row++
		if l[0] == "menu:mute" {
			out = append(out, volumeButtons(box.Y+menuTop+row*menuRowH, f.Volumes)...)
			row += len(volumeRows)
		}
	}
	return append(out, closeButton(box))
}

func (h *HUD) menuBox() ui.Rect {
	if h.live == nil {
		return MenuBox
	}
	labels := h.menuLabels(Frame{})
	rows := len(labels)
	if slices.ContainsFunc(labels, func(l [3]string) bool { return l[0] == "menu:mute" }) {
		rows += len(volumeRows)
	}
	box := MenuBox
	box.H = min(MenuBox.H, menuTop+rows*menuRowH+menuFoot)
	return box
}

func (h *HUD) drawMenu(dst *ebiten.Image, f Frame) {
	box := h.menuBox()
	ui.DarkPanel(dst, box)
	h.Font.DrawCentered(dst, "MENU", ui.Rect{X: box.X, Y: box.Y + 6, W: box.W, H: 14}, 2, ui.Palette.Gold)
	if f.Version != "" {
		h.Font.Draw(dst, f.Version, box.X+8, box.Y+menuVersionY, 1, ui.Palette.TextDim)
	}
	h.drawSaveStatus(dst, f)
	h.DrawOverlayButtons(dst, h.MenuButtons(f))
}

func firstUnresolved(list []view.ChronicleView) (view.ChronicleView, bool) {
	for _, c := range list {
		if !c.Resolved {
			return c, true
		}
	}
	return view.ChronicleView{}, false
}
