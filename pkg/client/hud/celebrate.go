package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func CelebrationTitle(title string) string {
	up := strings.ToUpper(title)
	return ui.Bridge(up, strings.Replace(ui.Name(up), "DELIVERED", "SERVED", 1))
}

func (h *HUD) Celebrate(f Frame, title string, gain float64) {
	h.Banner(CelebrationTitle(title))
	h.floatGain(f, gain)
}

func (h *HUD) CelebrateChest(f Frame, title string, gain, chest float64) {
	h.BannerCause(CelebrationTitle(title), view.TreasureLine(chest))
	h.floatGain(f, gain)
}

func RolloutBanner(d model.RolloutDone) (string, string) {
	_, tag, _ := strings.Cut(d.Image, ":")
	crew := strings.ToUpper(d.Crew)
	title := ui.Bridge(crew+": CLASS "+tag, crew+": "+strings.ToUpper(d.Image)+" ROLLED OUT")
	cause := ui.Bridge(plural(d.Huts, "hut", "huts")+" rebuilt one by one", plural(d.Huts, "Pod", "Pods")+" updated one by one")
	if d.Gain >= .01 {
		amount := goldAmount(d.Gain)
		cause += ui.Bridge(": +"+amount+" gold a second more than before", ": +$"+amount+"/s since the rollout began")
	}
	return title, cause
}

func (h *HUD) CelebrateRollout(f Frame, d model.RolloutDone) {
	if !classTrusted(f, d.Image) {
		h.Toast(untrustedRollout(d), true)
		return
	}
	h.FeatBanner(RolloutBannerIn(f, d))
	h.floatGain(f, d.Gain)
}

func RolloutBannerIn(f Frame, d model.RolloutDone) (string, string) {
	title, cause := RolloutBanner(d)
	words := classWords(f, d.Image)
	switch {
	case f.State.Age == view.AgeDark && view.RolloutCredited(d.Image):
		cause = ui.Bridge(strings.TrimSpace(plural(d.Huts, "hut", "huts")+" rebuilt a few at a time while the rest carried: the crew's roll-out. "+words),
			plural(d.Huts, "Pod", "Pods")+" replaced a few at a time: a rolling update.")
	case words == "":
		return title, cause
	default:
		cause = ui.Bridge(plural(d.Huts, "hut", "huts")+" rebuilt one by one. "+words, cause)
	}
	if crewKeepsUp(f, d.Crew) {
		cause += ui.Bridge(" They already carried every box, so gold grows when more ships come: open a berth or a port.", "")
	} else if d.Gain >= .01 {
		cause += ui.Bridge(" +"+goldAmount(d.Gain)+" gold a second so far.", "")
	}
	return title, cause
}

func classWords(f Frame, image string) string {
	for _, s := range f.State.Soulstones {
		if s.Image == image && strings.TrimSpace(s.Description) != "" {
			head, _, _ := strings.Cut(s.Description, ". ")
			return strings.TrimSuffix(head, ".") + "."
		}
	}
	return ""
}

func crewKeepsUp(f Frame, crew string) bool {
	for _, k := range f.State.Freight.Kinds {
		if k.CrewName == crew && k.Open {
			return k.SpoiledRate < .05 && k.Waiting < 2
		}
	}
	return false
}

func classTrusted(f Frame, image string) bool {
	for _, s := range f.State.Soulstones {
		if s.Image == image {
			return s.Owned && !s.Cursed
		}
	}
	return false
}

func untrustedRollout(d model.RolloutDone) string {
	_, tag, _ := strings.Cut(d.Image, ":")
	return ui.Bridge("The "+d.Crew+" now wear class "+tag+", a class no Academy has examined: watch them before you trust it.",
		"The "+d.Crew+" now run "+d.Image+", an image no one has tested: watch the Pods before you trust it.")
}

func (h *HUD) floatGain(f Frame, gain float64) {
	if gain <= 0 || h.CoinFlights() == CoinFlightsOff {
		return
	}
	w := &h.Coins.Gain
	w.Armed, w.shown = false, true
	w.Float = gainFloat{text: GainLabel(gain, f.TrueNames), Good: true, start: h.events, x: h.incomeX(f), life: bannerIn + bannerHold + bannerOut}
}
