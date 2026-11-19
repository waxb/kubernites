package app

import (
	"fmt"
	"math"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	BurstSparks = 6
	burstReach  = .55
	minSpan     = 1.0
)

func Gilded(g view.GoldenView) *world.Gilded {
	if g.Ship == nil || len(g.Ship.Boxes) == 0 {
		return nil
	}
	out := &world.Gilded{Harbor: g.Ship.Harbor}
	for _, b := range g.Ship.Boxes {
		out.Gleams = append(out.Gleams, world.Gleam{ID: b.ID, Fades: b.Fades, Span: max(b.Fades-g.Ship.Docked, minSpan)})
	}
	return out
}

func GleamTip(id string, st view.GameState) string {
	ship := st.Golden.Ship
	if ship == nil {
		return ""
	}
	for _, b := range ship.Boxes {
		if b.ID == id {
			return fmt.Sprintf("A gleaming box from the gilded ship.\nClick it before it fades: %s left.", hud.Duration(max(b.Fades-st.RealmTime, 0)))
		}
	}
	return ""
}

func (s *Session) HoverTip() string {
	switch {
	case s.Hover == "":
		return ""
	case strings.HasPrefix(s.Hover, world.PileIDHead):
		return PileTip(s.Hover, s.State)
	case strings.HasPrefix(s.Hover, hutHoverHead):
		return s.hutTip(strings.TrimPrefix(s.Hover, hutHoverHead))
	case strings.HasPrefix(s.Hover, liveHoverHead):
		return s.liveTip(strings.TrimPrefix(s.Hover, liveHoverHead))
	case s.Hover == world.HitForum:
		return ForumTip()
	case s.Hover == world.HitStone:
		return StoneTip()
	}
	return GleamTip(s.Hover, s.State)
}

func (s *Session) Burst(at world.Vec) {
	for i := range BurstSparks {
		a := 2 * math.Pi * float64(i) / BurstSparks
		s.Juice("sparkle", world.Vec{X: at.X + burstReach*math.Cos(a), Y: at.Y + burstReach*math.Sin(a)}, "")
	}
	s.Juice("sparkle", at, "")
}

func (s *Session) ClaimGleam(hit world.Hit) []sound.SFX {
	at, placed := s.Renderer.GleamSpot(hit.ID)
	gold := s.Realm.GameState().Gold
	fx := s.Act(view.Action{Action: "claimGolden", ID: hit.ID})
	if len(fx) > 0 && fx[0] == sound.SFXError {
		return fx
	}
	if placed {
		s.Burst(at)
		if earned := s.Realm.GameState().Gold - gold; earned > 0 {
			x, y := s.ScreenOf(at)
			s.Hud.LaunchGoldenCoin(x, y, earned)
		}
	}
	return []sound.SFX{sound.SFXGleamLast}
}

func (s *Session) crewPods(crew string) []string {
	out := []string{}
	for _, p := range s.Pods {
		if p.Crew == crew || p.Crew == "" && strings.HasPrefix(p.Name, crew+"-") {
			out = append(out, p.Name)
		}
	}
	return out
}

func (s *Session) SeasonNotices() []sound.SFX {
	seen := s.seasons
	s.seasons = make(map[string]int, len(s.State.Crews))
	var out []sound.SFX
	for _, c := range s.State.Crews {
		s.seasons[c.Name] = c.Milestone
		prev, known := seen[c.Name]
		if seen == nil || !known || c.Milestone <= prev {
			continue
		}
		if s.State.Economy.Bottleneck == view.FlowCrew {
			s.Echoes = Chronicled(s.Echoes, fmt.Sprintf("The %s are seasoned: x%s carry speed!", c.Name, hud.Compact(c.Multiplier)), s.echoClock())
		}
		for _, pod := range s.crewPods(c.Name) {
			s.Juice("sparkle", world.Vec{}, pod)
		}
		out = append(out, sound.SFXUnlock)
	}
	return out
}

func (s *Session) gildedClaim() hud.Intent {
	id := ""
	if ship := s.State.Golden.Ship; ship != nil {
		id = ship.ID
	}
	return hud.Intent{Kind: hud.IntentAction, Action: view.Action{Action: "claimGolden", ID: id}}
}

func (s *Session) gildedQuay() *[2]float64 {
	ship := s.State.Golden.Ship
	if ship == nil {
		return nil
	}
	h, ok := havenOf(s.Site, ship.Harbor)
	if !ok {
		return nil
	}
	return &[2]float64{float64(h.Quay.X), float64(h.Quay.Y)}
}
