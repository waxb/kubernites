package app

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
)

const (
	LintEnv    = "KUBERNITES_LINT"
	LintSuffix = ".lint.json"
)

type LintReport struct {
	Scene    string        `json:"scene"`
	Strays   []hud.Stray   `json:"strays"`
	Overlaps []string      `json:"overlaps"`
	Surfaces []hud.Surface `json:"surfaces"`
}

func (c *Capture) lint(g *Game) error {
	s := g.Sess
	if g.Scene == SceneTitle || s == nil {
		return nil
	}
	f := s.Frame(true)
	scratch := ebiten.NewImage(hud.Width, hud.Height)
	defer scratch.Deallocate()
	report := LintReport{Scene: c.Scene, Strays: s.Hud.Strays(f, scratch, readScreen), Surfaces: s.SurfacesNow(), Overlaps: []string{}}
	frame := hud.AttnFrame{Surfaces: report.Surfaces}
	for _, fault := range hud.CheckAttention(hud.AttnLog{Frames: []hud.AttnFrame{frame}}, hud.AttnRules{OverlapOnly: true}) {
		report.Overlaps = append(report.Overlaps, fault.String())
	}
	raw, err := json.MarshalIndent(report, "", " ")
	if err != nil {
		return fmt.Errorf("lint report: %w", err)
	}
	return os.WriteFile(c.path+LintSuffix, raw, 0o644)
}

func (s *Session) SurfacesNow() []hud.Surface {
	return append(s.Hud.Drawn(), worldSurfaces(s.Renderer.Drawn())...)
}

func worldSurfaces(boxes []world.DrawnBox) []hud.Surface {
	out := make([]hud.Surface, 0, len(boxes))
	for _, b := range boxes {
		s := hud.Surface{Slot: hud.SlotWorld, Kind: b.Kind, Key: b.Key, Text: b.Text, Rect: b.Rect}
		switch b.Kind {
		case world.DrawnSpeech:
			s.Asked = true
		case world.DrawnCallout:
			s.Kind, s.Box = hud.BoxCallout, true
		}
		out = append(out, s)
	}
	return out
}

func speechRects(boxes []world.DrawnBox) []ui.Rect {
	out := []ui.Rect{}
	for _, b := range boxes {
		if b.Kind == world.DrawnSpeech {
			out = append(out, b.Rect)
		}
	}
	return out
}

func worldRects(boxes []world.DrawnBox) []ui.Rect {
	out := make([]ui.Rect, 0, len(boxes))
	for _, b := range boxes {
		out = append(out, b.Rect)
	}
	return out
}
