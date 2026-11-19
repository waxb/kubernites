package gfx

import "github.com/waxb/kubernites/pkg/client/art"

const (
	KMapCell      = art.KMapCell
	PageIconSize  = art.PageIconSize
	MedallionSize = art.MedallionSize
	PlaqueHeight  = art.PlaqueH
)

var (
	pageSet      = setOf(art.PageIcons)
	kmapStateSet = setOf(art.KMapStates)
	roadStateSet = setOf(art.KMapRoadStates)
	branchSet    = setOf(art.KMapBranches)
	tintSet      = setOf(art.GroundTints)
)

func BorderStoneName(variant int) string { return art.BorderStoneName(wrap(variant, art.BorderStones)) }

func BorderFlagName(frame int) string { return art.BorderFlagName(wrap(frame, art.BorderFlags)) }

func GroundTintName(kind string) string {
	if tintSet[kind] {
		return art.GroundTintName(kind)
	}
	return art.GroundTintName("own")
}

var pageIconAlias = map[string]string{"foremen": "muster", "deployments": "crews"}

func PageIconName(id string) string {
	if alias, ok := pageIconAlias[id]; ok {
		id = alias
	}
	if pageSet[id] {
		return art.PageIconName(id)
	}
	return art.PageIconName("blank")
}

func KMapState(state string) string {
	if kmapStateSet[state] {
		return state
	}
	return "locked"
}

func PlaqueName(state string) string { return art.PlaqueName(KMapState(state)) }

func PlaqueTextColor(state string) [3]uint8 {
	c := art.PlaqueText[KMapState(state)]
	return [3]uint8{c.R, c.G, c.B}
}

func KMapMedallionName(state string) string { return art.KMapMedallionName(KMapState(state)) }

func KMapRoadState(state string) string {
	if roadStateSet[state] {
		return state
	}
	return "locked"
}

func KMapRoadName(state string, n, e, s, w bool) string {
	mask := art.KMapRoadMask(n, e, s, w)
	if mask == "" {
		return ""
	}
	return art.KMapRoadName(KMapRoadState(state), mask)
}

func KMapBannerName(branch string) string {
	if branchSet[branch] {
		return art.KMapBannerName(branch)
	}
	return ""
}

func KMapDecorName(kind string) string { return art.KMapDecorName(kind) }

func KMapParchmentName() string { return art.KMapParchmentName() }
