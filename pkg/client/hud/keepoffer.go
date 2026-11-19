package hud

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func KeepOfferFor(f Frame, kind string) (view.KeepOffer, bool) {
	k, ok := view.KeepTypeOf(kind)
	if !ok || !IsKeepKind(kind) || !hasKeep(f.Site) {
		return view.KeepOffer{}, false
	}
	for _, o := range f.State.KeepShop {
		if o.KeepType == k.KeepType {
			return o, true
		}
	}
	return view.KeepOffer{}, false
}

func KeepOfferLine(o view.KeepOffer) string {
	land := o.Class
	if land == "" {
		land = "open"
	}
	machine := o.Instance
	if machine == "" {
		machine = o.K8s
	}
	return fmt.Sprintf("Next plot: %s (%s land), %s of food and %s of housing; its borders reach %d tiles.", machine, land, view.RationsOf(o.CPU), view.RoomsOf(o.Memory), o.Territory)
}

func keepOfferLines(b view.Buildable, f Frame) []string {
	o, ok := KeepOfferFor(f, b.Kind)
	if !ok {
		return nil
	}
	out := []string{}
	if payoff := KeepPayoff(f, o); payoff != "" {
		out = append(out, payoff)
	}
	return append(out, KeepOfferLine(o))
}

func KeepLines(s view.Structure) []Line {
	if s.Instance == "" && s.CPU == 0 && s.MemoryGiB == 0 {
		return nil
	}
	size := ui.Bridge(foodTag(float64(s.CPU)*view.RationsPerCPU)+" "+housingTag(float64(s.MemoryGiB)*gib), fmt.Sprintf("%d CPU, %d GiB memory", s.CPU, s.MemoryGiB))
	if s.Instance != "" {
		size = s.Instance + ": " + size
	}
	out := []Line{{Text: size, Clr: model.ClassColor(s.Class), lesson: laborLesson}}
	if land := landLine(s); land != "" {
		out = append(out, Line{Text: land, Clr: ui.Palette.TextDim, lesson: laborLesson})
	}
	return out
}

var classK8s = map[string]string{"balanced": "general-purpose"}

func landLine(s view.Structure) string {
	story, k8s := []string{}, []string{}
	if s.Class != "" {
		story = append(story, s.Class+" land")
		k8s = append(k8s, cmp.Or(classK8s[s.Class], s.Class))
	}
	if s.Territory > 0 {
		reach := fmt.Sprintf("reach %d tiles", s.Territory)
		story, k8s = append(story, reach), append(k8s, reach)
	}
	return ui.Bridge(strings.Join(story, ", "), strings.Join(k8s, ", "))
}
