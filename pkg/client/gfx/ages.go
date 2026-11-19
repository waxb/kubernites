package gfx

import (
	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

var ageTiers = map[string]int{"dark": 0, "feudal": 1, "castle": 2, "empire": 3}

func AgeTier(age string) int { return ageTiers[age] }

func LookVariant(id string) int { return art.LookVariant(id) }

func lookRange(age, variant int) (int, int) {
	return min(max(age, 0), art.AgeTiers-1), min(max(variant, 0), art.BuildingVariants-1)
}

func BuildingLookNames(kind, state string, age, variant int) []string {
	age, variant = lookRange(age, variant)
	states := []string{state}
	if state != "ready" {
		states = append(states, "ready")
	}
	out := make([]string, 0, len(states)*(age+1)*2)
	for _, st := range states {
		for a := age; a >= 0; a-- {
			out = append(out, art.BuildingLookName(kind, st, a, variant))
			if variant != 0 {
				out = append(out, art.BuildingLookName(kind, st, a, 0))
			}
		}
	}
	return out
}

func StructureLookNames(s view.Structure, age int) []string {
	return structureVariantNames(s, age, LookVariant(s.ID))
}

func PreviewNames(s view.Structure, age int) []string {
	if s.Kind == view.StructKeep {
		return KeepNames(s, age)
	}
	return structureVariantNames(s, age, 0)
}

func structureVariantNames(s view.Structure, age, v int) []string {
	return append(BuildingLookNames(ArtKind(s), BuildingState(s), age, v), BuildingNames(s)...)
}

func HearthLookNames(tier int, state, qos string, age, variant int) []string {
	look := QoSArt(qos)
	if look == "" {
		return HearthQoSNames(tier, state, qos)
	}
	age, variant = lookRange(age, variant)
	t := min(max(tier, 1), 3)
	out := make([]string, 0, (age+1)*2+4)
	for a := age; a >= 0; a-- {
		out = append(out, art.HearthLookName(t, state, look, a, variant))
		if variant != 0 {
			out = append(out, art.HearthLookName(t, state, look, a, 0))
		}
	}
	return append(out, HearthQoSNames(tier, state, qos)[1:]...)
}

var portraitKinds = func() map[string]bool {
	out := map[string]bool{}
	for _, k := range append(append([]string{}, art.BuildingKinds...), art.StallKinds...) {
		out[k] = true
	}
	return out
}()

func StructurePortrait(s view.Structure, age int) string {
	if s.Kind == view.StructKeep {
		return KeepLookOf(s, age).PortraitName()
	}
	return BuildPortrait(portraitKind(s), age)
}

const BuildIconPrefix = "buildicon/"

func BuildIcon(kind string, age int) string {
	kind = buildArt(kind)
	if !portraitKinds[kind] {
		return BuildPortrait(kind, age)
	}
	return art.BuildIconName(kind, age)
}

func BuildPortrait(kind string, age int) string {
	kind = buildArt(kind)
	if !portraitKinds[kind] {
		return "portrait/" + kind
	}
	return "portrait/" + art.AgedPortrait(kind, age)
}
