package app

import (
	"maps"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	reweaveWindow = 8.0
	tapestryText  = "The Tapestry weaves itself"
	gatheringText = "The Rush Tide gathers"
	siegeWon      = "RUSH TIDE WEATHERED"
	gateFell      = "THE QUAY FLOODED"
	questKind     = "quest"
	GildedText    = "A gilded ship docks"
	GildedTitle   = "A GILDED SHIP DOCKED"
	impText       = "Tremor!"
	impTitle      = "THE SIMIAN IMP STRUCK"
	resultSpeed   = 5.0
)

func (s *Session) result(title, text string) {
	if s.State.Speed >= resultSpeed {
		s.Hud.Result(title, text)
	}
}

func (s *Session) resultWhile(title, text string, still func() bool) {
	if s.State.Speed >= resultSpeed {
		s.Hud.ResultWhile(title, text, still)
	}
}

var proclamationSounds = map[string]sound.SFX{
	"calamity": sound.SFXCalamity, "reveal": sound.SFXReveal, "unlock": sound.SFXUnlock, "age": sound.SFXVictory,
	"lesson": sound.SFXUnlock, view.ProclaimTrade: sound.SFXVictory, view.ProclaimTried: sound.SFXUnlock, questKind: sound.SFXVictory,
}

var juiceLife = map[string]float64{"sparkle": 0.5, "reweave": 0.9, "puff": 4.0 / 12}

func (s *Session) Juice(kind string, at world.Vec, pod string) {
	s.Transients = append(s.Transients, world.Transient{Kind: kind, At: at, Pod: pod, Start: s.Clocks.World, Life: juiceLife[kind]})
}

func (s *Session) Notices() []sound.SFX {
	out := s.StructureNotices()
	out = append(out, s.ProclamationNotices()...)
	out = append(out, s.treasureNotices()...)
	s.storageNotices()
	s.walkRounds()
	s.AgeNotice()
	s.TierNotice()
	out = append(out, s.HornNotices()...)
	out = append(out, s.HearthNotices()...)
	out = append(out, s.SeasonNotices()...)
	out = append(out, s.stepCues()...)
	for _, fx := range s.RailSounds() {
		if !slices.Contains(out, fx) {
			out = append(out, fx)
		}
	}
	return out
}

func (s *Session) Served(byHand bool, value float64) {
	if !byHand {
		return
	}
	s.ServedAt = s.Clocks.World
	s.handCoin(max(value, 0))
}

func (s *Session) AgeNotice() {
	if s.State.Age == s.Age || s.State.Age == "" {
		return
	}
	s.Age = s.State.Age
	s.Hud.FeatBanner("THE "+strings.ToUpper(s.Age)+" AGE", view.Content.StageTheme(s.Age))
}

func (s *Session) StructureNotices() []sound.SFX {
	var out []sound.SFX
	for _, st := range s.Site.Structures {
		prev, ok := s.Built[st.ID]
		s.Built[st.ID] = st.State
		if ok && prev == view.StateConstruction && completed(st.State) {
			s.Hud.Announce(hud.AnnounceBuilt, st.Label+" stands complete", false)
			s.Juice("puff", world.Anchor(st), "")
			out = append(out, sound.SFXBuilt)
		}
	}
	return out
}

func completed(state string) bool {
	return state == view.StateReady || state == view.StateJoining
}

func (s *Session) ProclamationNotices() []sound.SFX {
	var out []sound.SFX
	latest := s.ProcT
	for _, p := range s.State.Proclamations {
		if p.T <= s.ProcT {
			continue
		}
		latest = max(latest, p.T)
		if fx, ok := ProclamationSound(p); ok {
			out = append(out, fx)
		}
		s.Proclaimed(p)
	}
	s.ProcT = latest
	return out
}

func (s *Session) Proclaimed(p view.Proclamation) {
	text := strings.TrimSpace(p.Text)
	s.prideNotice(p)
	if p.Kind == "lesson" {
		return
	}
	if p.Kind == questKind {
		title, _ := CitadelBanner()
		s.Hud.FeatBanner(title, strings.TrimSpace(strings.TrimPrefix(text, view.CitadelText+":")))
		return
	}
	if title, cause, ok := SiegeVerdict(text); ok {
		s.Hud.FeatBanner(title, cause)
		s.result(title, cause)
		return
	}
	if strings.HasPrefix(text, view.CitadelText+":") {
		s.Hud.FeatBanner(CitadelBanner())
		return
	}
	if p.Kind == "calamity" || p.Kind == "reveal" && !s.Guide.Staged {
		s.Hud.Announce(p.Kind, text, p.Kind == "calamity")
	}
	if strings.HasPrefix(text, GildedText) && !slices.Contains(s.Hud.Taught(), hud.GildedTaught) {
		told := p.T
		s.Hud.ResultClaim(hud.GildedTaught, GildedTitle, text, s.gildedQuay(), func() bool { return gildedDocked(s.State, told) }, s.gildedClaim())
		if s.quietNews == nil {
			s.quietNews = map[string]bool{}
		}
		s.quietNews[newsID(p)] = true
	}
	if strings.HasPrefix(text, impText) && !s.impTold && s.State.Speed >= resultSpeed {
		s.impTold = true
		s.result(impTitle, strings.TrimSpace(strings.TrimPrefix(text, impText)))
	}
	if p.Kind == "calamity" && !strings.HasPrefix(text, gatheringText) && s.State.Calm <= 0 {
		s.Hud.Vignette()
	}
	if strings.Contains(text, tapestryText) {
		s.Reweave = s.Clock + reweaveWindow
	}
}

func CitadelBanner() (string, string) {
	return ui.Bridge("THE CITADEL STANDS", "THE CONTROL PLANE IS UP"),
		ui.Bridge("The Codex, the Chancery, the Quartermaster and the Stewards' Hall now rule the realm together.",
			"etcd, kube-apiserver, kube-scheduler and kube-controller-manager now run together.")
}

func SiegeVerdict(text string) (string, string, bool) {
	for _, title := range []string{siegeWon, gateFell} {
		if rest, ok := strings.CutPrefix(text, title+":"); ok {
			return title, strings.TrimSpace(rest), true
		}
	}
	return "", "", false
}

func ProclamationSound(p view.Proclamation) (sound.SFX, bool) {
	if strings.HasPrefix(strings.TrimSpace(p.Text), gatheringText) {
		return sound.SFXDrums, true
	}
	if p.Kind == "calamity" && strings.HasPrefix(strings.TrimSpace(p.Text), "Tremor!") {
		return sound.SFXTremor, true
	}
	if p.Kind == "lesson" && p.Major {
		return sound.SFXVictory, true
	}
	fx, ok := proclamationSounds[p.Kind]
	return fx, ok
}

func (s *Session) HearthNotices() []sound.SFX {
	var out []sound.SFX
	next := HearthStates(s.Pods)
	names := podNames(s.Pods)
	for _, uid := range slices.Sorted(maps.Keys(next)) {
		state := next[uid]
		prev, known := s.Hearths[uid]
		switch {
		case !known && s.Clock < s.Reweave:
			s.Juice("reweave", world.Vec{}, names[uid])
		case state == model.HearthKindling && prev != model.HearthKindling:
			out = append(out, sound.SFXBoat)
		case state == model.HearthReady && prev != model.HearthReady:
			out = append(out, sound.SFXPodStart)
			if known {
				s.Juice("sparkle", world.Vec{}, names[uid])
			}
		}
	}
	s.Hearths = next
	return out
}

func HearthStates(pods []model.PodView) map[string]string {
	out := make(map[string]string, len(pods))
	for _, p := range pods {
		if !p.System {
			out[p.UID] = p.State
		}
	}
	return out
}

func podNames(pods []model.PodView) map[string]string {
	out := make(map[string]string, len(pods))
	for _, p := range pods {
		out[p.UID] = p.Name
	}
	return out
}
