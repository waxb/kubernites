package art

import "fmt"

var buildingHeights = map[string]int{
	"keep-tower": 78, "keep-stone": 58, "keep-great": 90, "keep-citadel": 116, "codex": 68, "chancery": 60, "quartermaster": 34, "stewards-hall": 46,
	"wayfinder": 38, "observatory": 54, "gate": 56, "treasury": 30, "warhorn": 80,
	"shrine": 34, "signpost": 38, "foreign": 56, "harbor-gate": 52, "sea-chancery": 88,
}

var freightHeights = map[string]int{
	"seal-smithy": 40, "key-gate": 50, "bell-watch": 58, "cache-store": 32, "caravan-yard": 24,
}

func buildingHeight(kind string) int {
	if h, ok := buildingHeights[kind]; ok {
		return h
	}
	return freightHeights[kind]
}

type size struct {
	w, h, ax, ay int
}

const buildingFoot = 4

func buildingSize(kind string) size {
	n := Footprints[kind]
	w := n*TileWidth + 4
	h := n*TileHeight + buildingHeight(kind) + 2 + buildingFoot
	return size{w: w, h: h, ax: w / 2, ay: h - buildingFoot}
}

func hearthSize() size {
	return size{w: 36, h: 58, ax: 18, ay: 56}
}

var decorSizes = map[string]size{
	"tree-oak": {28, 44, 14, 42}, "tree-pine": {23, 46, 11, 44}, "tree-birch": {20, 42, 9, 40},
	"tree-maple": {26, 40, 12, 38}, "tree-spruce": {17, 44, 8, 42}, "tree-old": {46, 62, 22, 60},
	"tree-willow": {32, 40, 16, 38}, "bush": {17, 14, 8, 12}, "fern": {15, 10, 7, 9}, "rock": {15, 11, 7, 9},
	"boulder": {23, 17, 11, 15}, "reeds": {15, 16, 7, 14}, "flowers": {13, 7, 6, 5}, "mushrooms": {14, 10, 7, 9},
	"tallgrass": {15, 9, 7, 8}, "lily": {13, 6, 6, 3}, "stump": {15, 12, 7, 10}, "log": {24, 12, 12, 9},
	"haystack": {18, 18, 9, 15}, "fence": {34, 20, 17, 16}, "ley-stone": {13, 19, 6, 17},
	"outcrop": {30, 22, 15, 19}, "quarry": {34, 24, 17, 20}, "sapling": {16, 24, 8, 22},
}

func effectSize(kind string) size {
	switch kind {
	case "explosion":
		return size{24, 24, 12, 20}
	case "raven":
		return size{16, 12, 8, 6}
	case "boat":
		return size{24, 22, 12, 18}
	case "banner":
		return size{240, 40, 120, 20}
	case "reweave":
		return size{24, 24, 12, 20}
	case "horn":
		return size{24, 24, 12, 12}
	case "charter":
		return size{24, 40, 12, 37}
	case "wake":
		return size{WakeWidth, WakeHeight, WakeWidth / 2, WakeHeight / 2}
	case "request":
		return size{RequestWidth, RequestHeight, RequestWidth / 2, RequestHeight - 1}
	case "alarm":
		return size{alarmW, alarmH, alarmW / 2, alarmH - 1}
	}
	return size{16, 16, 8, 15}
}

var uiSizes = map[string][3]int{
	"panel": {32, 32, 10}, "panel-dark": {32, 32, 10}, "button": {24, 16, 5}, "button-hover": {24, 16, 5},
	"button-pressed": {24, 16, 5}, "button-disabled": {24, 16, 5}, "tab": {24, 14, 5}, "tab-active": {24, 14, 5},
	"scroll": {40, 32, 12}, "tooltip": {16, 16, 5}, "bar-frame": {24, 8, 3}, "bar-fill-good": {8, 6, 2},
	"bar-fill-bad": {8, 6, 2}, "minimap-frame": {40, 40, 12},
	"woven-strip": {344, 40, 4}, "chip": {12, 12, 3}, "bar-mini": {6, 3, 1},
	"kmap-frame": {KMapFrameSize, KMapFrameSize, KMapFrameSlice}, "kmap-ribbon": {48, 18, 8},
	"plaque-locked": {PlaqueW, PlaqueH, PlaqueSlice}, "plaque-available": {PlaqueW, PlaqueH, PlaqueSlice},
	"plaque-researching": {PlaqueW, PlaqueH, PlaqueSlice}, "plaque-learned": {PlaqueW, PlaqueH, PlaqueSlice},
}

func decorName(kind string, v int) string    { return fmt.Sprintf("decor/%s/%d", kind, v) }
func buildingName(kind, state string) string { return "building/" + kind + "/" + state }
func hearthName(tier int, state string) string {
	return fmt.Sprintf("hearth/%d/%s", tier, state)
}
func walkerName(role, action, dir string, frame int) string {
	return fmt.Sprintf("walker/%s/%s/%s/%d", role, action, dir, frame)
}
func effectName(kind string, frame int) string { return fmt.Sprintf("fx/%s/%d", kind, frame) }
func iconName(name string) string              { return "icon/" + name }
func portraitName(name string) string          { return "portrait/" + name }
func uiName(name string) string                { return "ui/" + name }
