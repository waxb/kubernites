package app

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	FirstBatch = int(view.FirstShipBoxes)
	coinMargin = 12
)

type purse struct {
	Tap       hud.CoinTap `json:"-"`
	realmTime float64
	seen      bool
	pickX     float64
	pickY     float64
	picked    bool
}

func (s *Session) Beckon() bool {
	f := hud.Frame{State: s.State, Guide: s.shownGuide()}
	if s.GuideGoal() != view.ParcelPile {
		f.GoalAt = s.goalAt()
	}
	return s.Hud.Beckons(f)
}

func ParcelsWaiting(fv view.FreightView) float64 {
	for _, k := range fv.Kinds {
		if k.Kind == world.KindName(0) {
			return k.Waiting
		}
	}
	return 0
}

func ClickHint(st view.GameState) bool {
	return !st.Tutorial && hud.HandsNeeded(st.Freight) && st.Pilgrims.HandServed < float64(FirstBatch) && ParcelsWaiting(st.Freight) >= 1
}

func Servable(hit world.Hit, _ view.GameState) bool {
	_, ok := PileKind(hit)
	return ok
}

func (s *Session) ScreenOf(v world.Vec) (float64, float64) {
	x, y := world.ScreenOf(s.Cam, s.Site, v)
	return clampCoin(x, y)
}

func clampCoin(x, y float64) (float64, float64) {
	return min(max(x, coinMargin), hud.Width-coinMargin), min(max(y, hud.TopH+coinMargin), hud.ConsoleY-coinMargin)
}

func (s *Session) clickPile(hit world.Hit) []sound.SFX {
	kind, ok := PileKind(hit)
	if !ok {
		return nil
	}
	top, found := s.Renderer.PileFront(kind, hit.Serial, s.Cam, s.Site)
	if !found {
		return nil
	}
	return s.servePile(kind, top)
}

func (s *Session) HoverPile(in hud.Input, blocked bool) {
	s.Hover, s.Cursor, s.TipAt = "", false, [2]int{in.MX, in.MY}
	if blocked || s.Placing != "" {
		return
	}
	hit, ok := s.Renderer.Pick(in.MX, in.MY)
	switch {
	case ok && hit.Kind == world.HitPile:
		s.Hover, s.Cursor = hit.ID, Servable(hit, s.State)
	case ok && hit.Kind == world.HitGolden:
		s.Hover, s.Cursor = hit.ID, true
	case ok && hit.Kind == world.HitStone:
		s.Hover, s.Cursor = world.HitStone, true
	case ok && hit.Kind == world.HitHearth && hit.Pod.Name != "":
		s.Hover = hutHoverHead + hit.Pod.Namespace + "/" + hit.Pod.Name
	case ok && hit.Kind == world.HitStructure && s.liveTip(hit.ID) != "":
		s.Hover = liveHoverHead + hit.ID
	case !ok:
		s.Hover = s.forumUnder(in.MX, in.MY)
	}
}

func (s *Session) handCoin(value float64) {
	x, y := s.Coins.pickX, s.Coins.pickY
	if !s.Coins.picked {
		x, y = s.ScreenOf(world.HandSpot(s.Site))
		y -= coinMargin
	}
	s.Coins.picked = false
	s.Hud.LaunchHandCoin(x, y, value)
}

func (s *Session) TreasuryCoins() {
	now := s.State.RealmTime
	if !s.Coins.seen {
		s.Coins.realmTime, s.Coins.seen = now, true
		return
	}
	earned := max(s.State.GoldPerSec, 0) * max(now-s.Coins.realmTime, 0)
	s.Coins.realmTime = now
	var value int
	s.Coins.Tap, value = s.Coins.Tap.StepEvery(s.Clocks.Event, earned, s.Hud.TreasuryGap())
	if value <= 0 {
		return
	}
	treasury, ok := world.FirstOfKind(s.Site, view.StructTreasury, false)
	if !ok {
		return
	}
	x, y := s.ScreenOf(world.Door(treasury))
	s.Hud.LaunchCoin(x, y, float64(value))
}

func pointerShape(hand bool) {
	if hand {
		ebiten.SetCursorShape(ebiten.CursorShapePointer)
		return
	}
	ebiten.SetCursorShape(ebiten.CursorShapeDefault)
}

func PileNote(id string, st view.GameState) string {
	kind := strings.TrimPrefix(id, world.PileIDHead)
	for _, k := range st.Freight.Kinds {
		if k.Kind == kind {
			return KindNote(k, true)
		}
	}
	return ""
}

const (
	PileLead    = "Boxes waiting on the quay: each one is a request someone sent to your realm."
	pileLeadK8s = "Requests waiting in the queue: each one is a call someone made to your app."
)

func PileTip(id string, st view.GameState) string {
	note := PileNote(id, st)
	if note == "" {
		return ""
	}
	return ui.Bridge(PileLead, pileLeadK8s) + "\n" + hud.GoldLabel(hud.HandValueOf(st.Freight, strings.TrimPrefix(id, world.PileIDHead))) + "\n" + note
}

func KindNote(k view.KindView, gateOpen bool) string {
	switch {
	case k.Kind == world.KindName(0) && gateOpen:
		return "Click the parcels to carry one by hand: each click carries one box."
	case !k.DepotOpen && k.Door:
		return fmt.Sprintf("The door porter carries the %s until the %s depot opens.", k.Name, k.Depot)
	case !k.DepotOpen:
		return fmt.Sprintf("The %s wait for the %s depot: open it, then raise %s to carry them.", k.Name, k.Depot, k.CrewName)
	case k.Crew == 0:
		return fmt.Sprintf("The %s depot stands empty: raise %s to carry the %s.", k.Depot, k.CrewName, k.Name)
	}
	return fmt.Sprintf("%d of the %s carry the %s: %.1f a second.", k.Crew, k.CrewName, k.Name, k.ServedRate)
}

func (s *Session) TierNotice() {
	if len(s.State.Ports) > 0 {
		s.portNotice()
		return
	}
	tier := s.State.Freight.Tier
	switch {
	case tier <= 0:
		return
	case s.Tier > 0 && tier > s.Tier:
		title, cause := TierBanner(tier)
		s.Hud.BannerCause(title, cause)
	}
	s.Tier = tier
}

func (s *Session) portNotice() {
	next := make(map[string]int, len(s.State.Ports))
	for _, p := range s.State.Ports {
		next[p.ID] = p.Level
		if was := s.portLevels[p.ID]; was > 0 && p.Level > was {
			title, cause := PortBanner(p)
			s.Hud.BannerCause(title, cause)
		}
	}
	s.portLevels = next
}

func PortBanner(p view.PortView) (string, string) {
	def := view.HarborTierOf(p.Level)
	ships := strings.ToUpper(def.Ships[:1]) + def.Ships[1:]
	listener, adds := "HUGE BURSTS", "admits huge bursts"
	if def.Listener != "" {
		listener, adds = def.Listener+" LISTENER", fmt.Sprintf("adds %s %s listener on port %d", listenerArticle(def.Listener), def.Listener, def.ListenerPort)
	}
	title := ui.Bridge(strings.ToUpper(p.Name+": "+def.Ships), strings.ToUpper(p.Name)+": "+listener)
	cause := ui.Bridge(fmt.Sprintf("%s dock at %s now: they bring %s", ships, p.Name, def.Opens),
		fmt.Sprintf("%s's Gateway %s: it admits %s", p.Name, adds, ui.Name(def.Opens)))
	return title, cause
}

func TierBanner(tier int) (string, string) {
	def := view.HarborTierOf(tier)
	ships := strings.ToUpper(def.Ships[:1]) + def.Ships[1:]
	return fmt.Sprintf("HARBOR TIER %d", def.Tier), fmt.Sprintf("%s dock at your deepest port: they bring %s", ships, def.Opens)
}

func TideSpot(site view.Site) [2]float64 {
	if at, ok := world.HarborFocus(site); ok {
		return [2]float64{at.X, at.Y}
	}
	return [2]float64{float64(site.Gate.X), float64(site.Gate.Y)}
}

func listenerArticle(protocol string) string {
	if strings.HasPrefix(protocol, "H") {
		return "an"
	}
	return "a"
}
