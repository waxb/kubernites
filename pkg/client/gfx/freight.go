package gfx

import (
	"math"
	"strings"

	"github.com/waxb/kubernites/pkg/client/art"
)

const (
	BoxSizes      = art.BoxSizes
	BoxVariants   = art.BoxVariants
	FreshLevels   = art.FreshLevels
	BoxPileLevels = art.BoxPileLevels
	HoldAction    = art.HoldAction
)

var (
	BoxKinds   = append([]string(nil), art.BoxKinds...)
	TradeRoles = append([]string(nil), art.TradeRoles...)
	boxKindSet = setOf(art.BoxKinds)
	tradeSet   = setOf(art.TradeRoles)
)

type BoxLook struct {
	Size     int
	Kind     string
	Variant  int
	Sealed   bool
	Priority bool
	Spoiled  bool
	Fresh    float64
}

func setOf(names []string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}

func clampSize(size int) int { return min(max(size, 0), BoxSizes-1) }

func BoxKind(kind string) string {
	if boxKindSet[kind] {
		return kind
	}
	return art.BoxKinds[0]
}

func BoxName(size int, kind string, variant int) string {
	return art.BoxName(clampSize(size), BoxKind(kind), wrap(variant, BoxVariants))
}

func BoxSpoiledName(size int, kind string, variant int) string {
	return art.BoxSpoiledName(clampSize(size), BoxKind(kind), wrap(variant, BoxVariants))
}

func BoxSealName(size int) string  { return art.BoxSealName(clampSize(size)) }
func BoxCrownName(size int) string { return art.BoxCrownName(clampSize(size)) }

func BoxFreshName(size, level int) string {
	return art.BoxFreshName(clampSize(size), min(max(level, 0), FreshLevels-1))
}

func FreshLevel(fresh float64) int {
	if math.IsNaN(fresh) {
		return FreshLevels - 1
	}
	f := min(max(fresh, 0), 1)
	return min(int((1-f)*FreshLevels), FreshLevels-1)
}

func BoxLayers(b BoxLook) []string {
	base := BoxName(b.Size, b.Kind, b.Variant)
	if b.Spoiled {
		base = BoxSpoiledName(b.Size, b.Kind, b.Variant)
	}
	out := []string{base}
	if b.Sealed {
		out = append(out, BoxSealName(b.Size))
	}
	if b.Priority {
		out = append(out, BoxCrownName(b.Size))
	}
	if !b.Spoiled {
		out = append(out, BoxFreshName(b.Size, FreshLevel(b.Fresh)))
	}
	return out
}

func BoxLift(size int) int { return art.BoxLift(clampSize(size)) }

func BoxPileName(kind string, level int) string {
	return art.BoxPileName(BoxKind(kind), min(max(level, 0), BoxPileLevels-1))
}

func BoxPileLevel(waiting int) int {
	switch {
	case waiting <= 0:
		return -1
	case waiting < 4:
		return 0
	case waiting < 10:
		return 1
	case waiting < 25:
		return 2
	}
	return 3
}

func SpoilName(frame int) string { return EffectName("spoil", frame) }

func CarryName(size int, kind, dir string, frame int) string {
	if dir != "sw" {
		dir = "se"
	}
	if kind != art.SealedCarry {
		kind = BoxKind(kind)
	}
	return art.CarryName(clampSize(size), kind, dir, wrap(frame, art.WalkerFrames))
}

func IsTrade(role string) bool { return tradeSet[role] }

func TradeKind(role string) string { return art.TradeKinds[role] }

func TradeHoldNames(role string, variant int, dir string, frame int) []string {
	if !IsTrade(role) {
		return WalkerVariantNames(role, variant, "carry", dir, frame)
	}
	return WalkerVariantNames(role, variant, HoldAction, dir, frame)
}

func HelperArtKind(project string) string { return art.HelperArt[strings.ToLower(project)] }

var helperIcons = map[string]string{"stall-herald": "herald"}

func HelperNames(project, state string) []string {
	kind := HelperArtKind(project)
	if kind == "" {
		return nil
	}
	if _, ok := portraitKinds[kind]; ok && !isBuildingKind(kind) {
		kind = art.StallArtKind(kind, "sw")
	}
	return []string{"building/" + kind + "/" + state, "building/" + kind + "/ready"}
}

func isBuildingKind(kind string) bool {
	for _, k := range art.BuildingKinds {
		if k == kind {
			return true
		}
	}
	return false
}

func HelperIcon(project string) string {
	kind := HelperArtKind(project)
	if icon, ok := helperIcons[kind]; ok {
		return icon
	}
	return kind
}

func KindIcon(kind string) string { return "kind-" + BoxKind(kind) }
