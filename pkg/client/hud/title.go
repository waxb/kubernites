package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

type Title struct {
	LiveOnly bool
	HasSave  bool
	Note     string
	Version  string
	Settings TitleSettings
	font     ui.Font
	hover    Button
	hovered  bool
	held     int
	settings bool
}

const (
	TitleTipTicks = 120
	titleButtonW  = 176
	titleRowH     = 22
	titleRowGap   = 6
	titleTop      = 318
	titlePad      = 16
	titleTipW     = 230
	titleTipGap   = 24
	titleTipPad   = 6
	titleTipLines = 8
)

type titleChoice struct {
	id, label, caption, keys string
	key                      ebiten.Key
}

func (t *Title) choices() []titleChoice {
	if t.LiveOnly {
		return []titleChoice{
			{id: "title:live", label: liveTitle, caption: LiveCaption, keys: "3", key: ebiten.Key3},
			{id: TitleSettingsID, label: "SETTINGS", caption: titleSettingsCaption},
			{id: "title:quit", label: "QUIT"},
		}
	}
	out := []titleChoice{{id: "title:new", label: "1  NEW REALM", caption: "Build a realm run on magic. Every part is real Kubernetes.", keys: "1 or N", key: ebiten.Key1}}
	if t.HasSave {
		out = append(out, titleChoice{id: "title:load", label: "CONTINUE", caption: "Return to your saved realm.", keys: "Enter"})
	}
	return append(out,
		titleChoice{id: TitleLearn, label: "2  LEARN KUBERNETES", caption: "Chapters with a teacher, real terms first: Pod (hut).", keys: "2", key: ebiten.Key2},
		titleChoice{id: "title:live", label: "3  LIVE CLUSTER", caption: LiveCaption, keys: "3", key: ebiten.Key3},
		titleChoice{id: "title:quit", label: "QUIT"},
	)
}

func titleLeft() int { return Width/2 - titleButtonW/2 }

func (c titleChoice) tip() string {
	if c.caption == "" || c.keys == "" {
		return c.caption
	}
	return c.caption + " [" + c.keys + "]"
}

func (t *Title) Buttons() []Button {
	if t.settings {
		return t.settingsButtons()
	}
	out := []Button{}
	y := titleTop
	for _, c := range t.choices() {
		out = append(out, Button{ID: c.id, Rect: ui.Rect{X: titleLeft(), Y: y, W: titleButtonW, H: titleRowH}, Label: c.label, Tip: c.tip()})
		y += titleRowH + titleRowGap
	}
	return out
}

func (t *Title) Update(in Input) []Intent {
	buttons := t.Buttons()
	hover, hovered := HitButton(buttons, in.MX, in.MY)
	switch {
	case !hovered:
		t.held = 0
	case t.hovered && hover.ID == t.hover.ID:
		t.held++
	default:
		t.held = 1
	}
	t.hover, t.hovered = hover, hovered
	if t.settings {
		return t.updateSettings(in)
	}
	switch {
	case in.Pressed(ebiten.KeyEnter) && t.HasSave && !t.LiveOnly:
		return []Intent{{Kind: IntentLoad}}
	case in.Pressed(ebiten.KeyN) && !t.LiveOnly:
		return []Intent{{Kind: IntentNew}}
	}
	for _, c := range t.choices() {
		if c.key != 0 && in.Pressed(c.key) {
			return []Intent{simpleIntents[c.id]}
		}
	}
	if !in.Click || !t.hovered || t.hover.Disabled {
		return nil
	}
	if t.hover.ID == TitleSettingsID {
		t.openSettings(true)
		return []Intent{titleClick}
	}
	if in, ok := simpleIntents[t.hover.ID]; ok {
		return []Intent{in}
	}
	return nil
}

func (t *Title) Draw(dst *ebiten.Image) {
	ui.Fill(dst, ui.Rect{W: Width, H: Height}, color.RGBA{10, 14, 26, 70})
	band := ui.Rect{X: 0, Y: 34, W: Width, H: 100}
	ui.Fill(dst, band, color.RGBA{12, 16, 24, 200})
	ui.Fill(dst, ui.Rect{X: 0, Y: band.Y, W: Width, H: 1}, ui.Palette.Gold)
	ui.Fill(dst, ui.Rect{X: 0, Y: band.Y + band.H - 1, W: Width, H: 1}, ui.Palette.Gold)
	t.font.DrawCentered(dst, "KUBERNITES", ui.Rect{X: 3, Y: 50 + 3, W: Width, H: 42}, 6, ui.Palette.Ink)
	t.font.DrawCentered(dst, "KUBERNITES", ui.Rect{X: 0, Y: 50, W: Width, H: 42}, 6, ui.Palette.GoldBright)
	if !t.LiveOnly {
		t.font.DrawCentered(dst, "RAISE A REALM. LEARN KUBERNETES BY RULING IT.", ui.Rect{X: 0, Y: 108, W: Width, H: 10}, 1, ui.Palette.Cream)
	}
	buttons := t.Buttons()
	panel := t.Panel()
	ui.Fill(dst, panel.Offset(2, 3), color.RGBA{8, 11, 12, 180})
	ui.Fill(dst, panel, ui.Tint(ui.Palette.Console, 235))
	ui.Stroke(dst, panel, ui.Palette.Gold)
	if t.settings {
		t.font.DrawCentered(dst, "SETTINGS", ui.Rect{X: panel.X, Y: titleSettingsHead, W: panel.W, H: 14}, 2, ui.Palette.Gold)
	}
	for _, b := range buttons {
		state := ui.ButtonState{Hovered: t.hovered && t.hover.ID == b.ID, Disabled: b.Disabled}
		ui.Button(dst, titleFont, b.Rect, b.Label, state)
	}
	if t.Note != "" {
		t.font.DrawCentered(dst, t.Note, ui.Rect{X: 0, Y: 484, W: Width, H: 10}, 1, ui.Palette.Red)
	}
	t.font.DrawCentered(dst, "WASD PAN   WHEEL ZOOM   CLICK INSPECT   ENTER SPEAKING STONE   SPACE PAUSE", ui.Rect{X: 0, Y: Height - 20, W: Width, H: 10}, 1, ui.Palette.TextDim)
	if t.Version != "" {
		t.font.Draw(dst, t.Version, titleVersionX, Height-20, 1, ui.Palette.TextDim)
	}
	ui.InRegion("tooltip", func() { t.drawTip(dst) })
}

func (t *Title) Panel() ui.Rect {
	buttons := t.Buttons()
	last := buttons[len(buttons)-1].Rect
	top := titleTop - 12
	if t.settings {
		top = titleSettingsHead - 10
	}
	return ui.Rect{X: titleLeft() - titlePad, Y: top, W: titleButtonW + 2*titlePad, H: last.Y + last.H + 12 - top}
}

func (t *Title) drawTip(dst *ebiten.Image) {
	if !t.hovered || t.held < TitleTipTicks || t.hover.Tip == "" {
		return
	}
	lines := titleFont.WrapLines(t.hover.Tip, titleTipW-2*titleTipPad, 1, titleTipLines)
	b := t.hover.Rect
	r := ui.Rect{X: b.X + b.W + titleTipGap, Y: b.Y, W: titleTipW, H: len(lines)*ui.LineHeight + 2*titleTipPad - (ui.LineHeight - ui.GlyphHeight)}
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 235))
	ui.Stroke(dst, r, ui.Palette.Gold)
	for i, line := range lines {
		titleFont.Draw(dst, line, r.X+titleTipPad, r.Y+titleTipPad+i*ui.LineHeight, 1, ui.Palette.Cream)
	}
}

const titleVersionX = 8

const TitleLearn = "title:learn"

const LiveCaption = "Connect a real cluster, read-only by default."

var titleFont = ui.Font{Literal: true}
