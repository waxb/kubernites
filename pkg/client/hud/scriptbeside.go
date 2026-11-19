package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

const (
	BesideButton = "scr:beside"
	besideTip    = "Show the Kubernetes YAML beside your script, line for line. It is what the Chancery stores when you apply."
	besideTitle  = "KUBERNETES YAML"
	besideGap    = 4
)

func (s *Script) BesideShown() bool {
	return s.Beside && ui.StoryNaming()
}

func (s *Script) EditorRect() (ui.Rect, ui.Rect) {
	r := ScriptEditor()
	if !s.BesideShown() {
		return r, ui.Rect{}
	}
	left := r
	left.W = r.W/2 - besideGap/2
	return left, ui.Rect{X: left.X + left.W + besideGap, Y: r.Y, W: r.W - left.W - besideGap, H: r.H}
}

func (s *Script) columns() int {
	r, _ := s.EditorRect()
	return ui.MaxChars(r.W-gutterW-10, 1)
}

func (h *HUD) BesideButton() []Button {
	if !ui.StoryNaming() {
		return nil
	}
	rail := scriptRail()
	return []Button{{ID: BesideButton, Rect: ui.Rect{X: rail.X + 4, Y: rail.Y + rail.H - 24, W: rail.W - 8, H: 18},
		Label: bridgeLabelText(bridgeLabel), Tip: besideTip, Selected: h.Script.Beside}}
}

func KubernetesBeside(story string) []string {
	if strings.TrimSpace(story) == "" {
		return nil
	}
	return strings.Split(lexicon.ManifestFromStory(story), "\n")
}

func (h *HUD) drawBeside(dst *ebiten.Image, r ui.Rect) {
	if r.W <= 0 {
		return
	}
	e := &h.Script.Editor
	ui.Fill(dst, r, consoleInk)
	ui.Stroke(dst, r, ui.Palette.BlueDark)
	title := besideTitle
	Literal.Draw(dst, title, r.X+r.W-len(title)*ui.Advance-4, r.Y+r.H-ui.LineHeight-1, 1, ui.Palette.TextDim)
	lines := h.Script.besideLines(e.Text())
	cols := ui.MaxChars(r.W-10, 1)
	for i := 0; i < scriptVisible() && e.Top+i < len(lines); i++ {
		text := []rune(lines[e.Top+i])
		visible := string(text[min(e.Left, len(text)):min(len(text), e.Left+cols)])
		Literal.Draw(dst, visible, r.X+5, r.Y+4+i*ui.LineHeight, 1, ui.Palette.Blue)
	}
}

type besideCache struct {
	text  string
	lines []string
}

func (s *Script) besideLines(text string) []string {
	if s.beside.text != text || s.beside.lines == nil {
		s.beside = besideCache{text: text, lines: KubernetesBeside(text)}
	}
	return s.beside.lines
}
