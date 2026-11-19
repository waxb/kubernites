package app

import "github.com/waxb/kubernites/pkg/client/hud"

func (p *GuideProbe) Numbers() []hud.CrewNumbers {
	return hud.CrewPanelNumbers(p.S.Frame(true))
}
