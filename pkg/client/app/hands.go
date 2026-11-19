package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	FromPile   = "pile"
	cycleReach = 4
)

type PickCycle struct {
	X, Y int    `json:"-"`
	Hit  string `json:"-"`
}

func pickKey(h world.Hit) string {
	return h.Kind + "/" + h.ID
}

func (c PickCycle) near(x, y int) bool {
	return c.Hit != "" && Abs(x-c.X) <= cycleReach && Abs(y-c.Y) <= cycleReach
}

func NextPick(all []world.Hit, last PickCycle, x, y int) world.Hit {
	if !last.near(x, y) {
		return all[0]
	}
	for i, h := range all {
		if pickKey(h) == last.Hit {
			return all[(i+1)%len(all)]
		}
	}
	return all[0]
}

func held(all []world.Hit, sel hud.Selection) (world.Hit, bool) {
	for _, h := range all {
		if h.ID == sel.ID && h.Kind == sel.Kind {
			return h, true
		}
	}
	return world.Hit{}, false
}

func (s *Session) choosePick(all []world.Hit, in hud.Input) world.Hit {
	if h, ok := held(all, s.Selection); ok && in.DoubleClick {
		return h
	}
	hit := NextPick(all, s.cycle, in.MX, in.MY)
	s.cycle = PickCycle{X: in.MX, Y: in.MY, Hit: pickKey(hit)}
	return hit
}

func PileKind(hit world.Hit) (int, bool) {
	if hit.Kind != world.HitPile {
		return 0, false
	}
	return world.PileKind(hit.ID)
}

func (s *Session) servePile(kind int, top world.PileTop) []sound.SFX {
	name := world.KindName(kind)
	s.Coins.pickX, s.Coins.pickY = clampCoin(top.X, top.Y)
	s.Coins.picked = true
	before := s.Realm.GameState().Pilgrims.HandServed
	fx := s.ActFrom(view.Action{Action: "serve", Kind: name, At: top.At}, FromPile)
	if s.Realm.GameState().Pilgrims.HandServed <= before {
		s.Coins.picked = false
		s.gleamHint()
		return []sound.SFX{sound.SFXError}
	}
	s.pileTip(name)
	return fx
}

func (s *Session) pileTip(kind string) {
	if kind == world.KindName(0) || s.tipped[kind] {
		return
	}
	for _, k := range s.State.Freight.Kinds {
		if k.Kind == kind && (!k.DepotOpen || k.Crew == 0) {
			s.tipped = withTip(s.tipped, kind)
			s.Hud.Toast(KindNote(k, false), false)
		}
	}
}

func withTip(tipped map[string]bool, kind string) map[string]bool {
	next := make(map[string]bool, len(tipped)+1)
	for k, v := range tipped {
		next[k] = v
	}
	next[kind] = true
	return next
}

func (s *Session) gleamHint() {
	ship := s.State.Golden.Ship
	if ship == nil {
		return
	}
	for _, b := range ship.Boxes {
		if b.Fades > s.State.RealmTime {
			s.Hud.Toast(gleamHintText, false)
			return
		}
	}
}

const gleamHintText = "The gleaming boxes wait on the gilded ship's own quay: open its notice on the rail to glide there."

const (
	quayKeyTip  = "quayKey"
	QuayKeyText = "The boxes wait at the quay, out of view. Press P to go back to it."
)

func (s *Session) QuayKeyHint() {
	if s.tipped[quayKeyTip] || !hud.HandsNeeded(s.State.Freight) {
		return
	}
	x, y, ok := s.Renderer.FrontBox(s.Cam, s.Site)
	if !ok || (x >= 0 && x < hud.Width && y >= hud.TopH && y < hud.ConsoleY) {
		return
	}
	s.tipped = withTip(s.tipped, quayKeyTip)
	s.Hud.Toast(QuayKeyText, false)
}
