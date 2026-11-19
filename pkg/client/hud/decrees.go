package hud

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	decreeRows    = 5
	decreeH       = 38
	decreeBarW    = 220
	payW          = 96
	urgentSeconds = 60
	DecreesLine   = "Hold at most two contracts and deliver before the deadline. Open TRADE to accept offers."
	NoDecrees     = "No merchant has offered a contract yet."
)

type DecreeCard struct {
	Status   string
	Asks     string
	ID       string
	Title    string
	Text     string
	State    string
	Left     string
	Reward   string
	Progress float64
	Tone     int
	Tribute  bool
}

func DecreeCards(f Frame) []DecreeCard {
	rows := ContractRows(f)
	out := []DecreeCard{}
	for _, r := range rows[:min(len(rows), decreeRows)] {
		out = append(out, contractCard(r))
	}
	return out
}

func contractCard(r ContractRow) DecreeCard {
	tone := ToneText
	if r.Left < urgentSeconds {
		tone = ToneWarn
	}
	state := "OFFERED"
	if r.State == StateActive {
		state = "HELD"
	}
	return DecreeCard{ID: r.ID, Title: r.Title, Text: r.Merchant + ": " + r.Goal, State: state, Left: r.Clock(),
		Reward: r.Reward, Progress: r.Share(), Status: r.Progress, Tone: tone}
}

func DecreesPage(f Frame) CouncilPage {
	if len(ContractRows(f)) == 0 {
		return CouncilPage{Line: NoDecrees}
	}
	return CouncilPage{Line: DecreesLine}
}

func decreeRect(i int) ui.Rect {
	b := CouncilBody()
	return ui.Rect{X: b.X + 4, Y: b.Y + PageTop + i*decreeH, W: b.W - 8, H: decreeH - 4}
}

func (h *HUD) drawDecrees(dst *ebiten.Image, f Frame) {
	for i, c := range DecreeCards(f) {
		r := decreeRect(i)
		ui.Fill(dst, r, consoleInk)
		ui.Stroke(dst, r, ui.Palette.BlueDark)
		text := r.W - payW - 20
		h.Font.Draw(dst, ui.Clip(c.Title, ui.MaxChars(text, 1)), r.X+6, r.Y+3, 1, ui.Palette.GoldBright)
		sw, _ := h.Font.Measure(c.State, 1)
		h.Font.Draw(dst, c.State, r.X+r.W-sw-6, r.Y+3, 1, toneColor(c.Tone))
		h.Font.Draw(dst, ui.Clip(c.Text, ui.MaxChars(text, 1)), r.X+6, r.Y+13, 1, ui.Palette.TextDim)
		bar := ui.Rect{X: r.X + 6, Y: r.Y + 25, W: decreeBarW, H: 5}
		h.drawMiniBar(dst, bar, c.Progress, toneColor(max(c.Tone, ToneGood)))
		tail := fmt.Sprintf("%s  %s  %s", c.ProgressText(), c.Left, c.Reward)
		h.Font.Draw(dst, ui.Clip(tail, ui.MaxChars(r.W-decreeBarW-payW-32, 1)), bar.X+bar.W+6, r.Y+23, 1, toneColor(c.Tone))
	}
}

func (c DecreeCard) ProgressText() string {
	if c.Status != "" {
		return c.Status
	}
	return fmt.Sprintf("%d%%", percent(c.Progress))
}
