package gfx

import (
	"fmt"

	"github.com/waxb/kubernites/pkg/view"
)

func ArtKind(s view.Structure) string {
	if kind, ok := stallArtKind(s); ok {
		return kind
	}
	if s.Kind == view.StructKeep {
		switch s.Variant {
		case "tower", "great", "citadel":
			return "keep-" + s.Variant
		}
		return "keep-stone"
	}
	return s.Kind
}

func BuildingState(s view.Structure) string {
	switch s.State {
	case view.StateConstruction:
		stage := int(s.Progress * 3)
		return fmt.Sprintf("construction-%d", min(max(stage, 0), 2))
	case view.StateDark, view.StateRuined:
		return "dark"
	}
	return "ready"
}

func BuildingName(s view.Structure) string {
	return fmt.Sprintf("building/%s/%s", ArtKind(s), BuildingState(s))
}

func BuildingNames(s view.Structure) []string {
	return []string{BuildingName(s), fmt.Sprintf("building/%s/ready", ArtKind(s))}
}

func HearthName(tier int, state string) string {
	return fmt.Sprintf("hearth/%d/%s", min(max(tier, 1), 3), state)
}

func WalkerName(role, action, dir string, frame int) string {
	return fmt.Sprintf("walker/%s/%s/%s/%d", role, action, dir, ((frame%4)+4)%4)
}

func WalkerNames(role, action, dir string, frame int) []string {
	return []string{
		WalkerName(role, action, dir, frame),
		WalkerName(role, "walk", dir, frame),
		WalkerName("wanderer", action, dir, frame),
		WalkerName("wanderer", "walk", dir, frame),
	}
}

func EffectName(kind string, frame int) string {
	return fmt.Sprintf("fx/%s/%d", kind, ((frame%6)+6)%6)
}

func DecorName(kind string, variant int) string {
	return fmt.Sprintf("decor/%s/%d", kind, ((variant%3)+3)%3)
}

var ShipClasses = []string{"cog", "caravel", "galleon", "carrack"}

const ShipVariants = 8

func ShipName(class string, variant int, dir string, frame int) string {
	return fmt.Sprintf("ship/%s/%d/%s/%d", class, ((variant%ShipVariants)+ShipVariants)%ShipVariants, dir, ((frame%4)+4)%4)
}

func HavenName(kind string, variant int) string {
	return fmt.Sprintf("haven/%s/%d", kind, max(variant, 0))
}
