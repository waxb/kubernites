package app

import "github.com/waxb/kubernites/pkg/client/hud"

const maxFollow = 8

type GuideFollow struct {
	Steps   []string
	Placing string
	Best    string
}

func (p *GuideProbe) Follow() GuideFollow {
	out := GuideFollow{}
	for range maxFollow {
		f := p.S.Frame(true)
		step, ok := p.S.Hud.Step(f)
		if !ok || step.Map || p.S.Placing != "" {
			break
		}
		out.Steps = append(out.Steps, step.ID)
		p.S.Update(hud.Input{MX: step.Rect.X + step.Rect.W/2, MY: step.Rect.Y + step.Rect.H/2, Click: true}, 0, &p.ptr, true)
	}
	out.Placing = p.S.Placing
	if m, ok := bestSpot(p.S.SpotMarks()); ok {
		out.Best = m.Note
	}
	return out
}
