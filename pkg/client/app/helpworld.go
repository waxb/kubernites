package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

var hitPages = map[string]string{
	world.HitHearth: "hut", world.HitWalker: "hut", world.HitFamiliar: "hut", world.HitForeman: "hut", world.HitCrew: "hut",
	world.HitPile: "harbor", world.HitHaven: "harbor", world.HitGolden: "harbor", world.HitForum: "quartermaster",
	world.HitStone:        "speaking-stone",
	world.HitDormantStone: "speaking-stone",
}

func HitPage(hit world.Hit) (string, bool) {
	if hit.Kind == world.HitStructure {
		if p, ok := view.Content.ForSubject(view.SubjectStructure, hit.Structure.Kind); ok {
			return p.ID, true
		}
		return "", false
	}
	page, ok := hitPages[hit.Kind]
	return page, ok
}

func (s *Session) HelpWorld(in hud.Input) []sound.SFX {
	s.Hud.ExitHelp()
	f := s.Frame(true)
	hit, ok := s.Renderer.Pick(in.MX, in.MY)
	page, found := HitPage(hit)
	if !ok || !found {
		s.Hud.Toast("No Almanac page for that yet", false)
		return []sound.SFX{sound.SFXError}
	}
	var sfx []sound.SFX
	for _, it := range s.Hud.OpenBook(page, f) {
		_, fx := s.Handle(it)
		sfx = append(sfx, fx...)
	}
	return sfx
}
