package art

import (
	"image"
	"sort"
)

const (
	TileWidth     = 32
	TileHeight    = 16
	ElevationStep = 5
	WalkerWidth   = 16
	WalkerHeight  = 24
	WalkerFrames  = 4
	EffectFrames  = 6
	BoatFrames    = 4
	BannerFrames  = 3
	HornFrames    = 4
	DecorVariants = 3
	IconSize      = 16
	PortraitSize  = 48
)

type Region struct {
	X, Y, W, H       int
	AnchorX, AnchorY int
	Slice            int
	Flags            []FlagMark
}

type Atlas struct {
	Image   *image.NRGBA
	Regions map[string]Region
}

type Set struct {
	Terrain, Buildings, Walkers, Effects, Icons, Portraits, UI Atlas
}

var DecorKinds = []string{
	"tree-oak", "tree-pine", "tree-birch", "bush", "rock", "reeds", "flowers", "stump", "log", "haystack", "fence",
	"tree-maple", "tree-spruce", "tree-old", "tree-willow", "fern", "boulder", "mushrooms", "tallgrass", "lily", "ley-stone",
	"outcrop", "quarry", "sapling",
}

var BuildingKinds = []string{"keep-tower", "keep-stone", "keep-great", "keep-citadel", "codex", "chancery", "quartermaster", "stewards-hall", "wayfinder", "observatory", "gate", "treasury", "warhorn",
	"seal-smithy", "key-gate", "bell-watch", "cache-store", "caravan-yard", "shrine", "signpost", "foreign", "harbor-gate", "sea-chancery"}

var BuildingStates = []string{"ready", "construction-0", "construction-1", "construction-2", "dark"}

var HearthStates = []string{"ready", "kindling", "guttering", "ghost", "banished", "pending"}

var WalkerRoles = []string{
	"lamplighter", "baker", "scribe", "granary-keeper", "archivist", "courier", "tallyman", "oracle",
	"chancellor", "codex-keeper", "quartermaster", "steward", "herald", "wayfinder", "augur",
	"reeve", "pilgrim", "builder", "imp", "wanderer", "sheep", "helmbearer", "clerk", "raven", "hound",
	"porter", "hauler", "sealwright", "woodcutter", "mason", "watchman",
}

var CritterRoles = []string{"deer", "stag", "rabbit", "fox", "boar", "finch", "gull"}

var WalkerActions = []string{"idle", "walk", "work", "carry"}

var WalkerDirs = []string{"se", "sw"}

var EffectKinds = []string{"fire", "smoke", "dust", "void", "sparkle", "coin", "raven", "explosion", "heal"}

var IconNames = []string{
	"gold", "wisdom", "renown", "pod", "node", "service", "build", "summon", "apply", "scale-up", "scale-down",
	"probe", "roll", "tech", "codex", "chronicle", "soulstone", "crier", "age", "speed", "pause", "alert",
	"describe", "logs", "yaml", "back", "close", "academy", "diploma",
	"thread", "loom-treasury", "loom-builders", "loom-fame", "siege", "archive", "warhorn", "food", "housing", "wheat",
	"decree", "rating-reliability", "rating-efficiency", "rating-resilience", "rating-favor",
	"overlay-midden", "overlay-labor", "overlay-unhoused", "overlay-heraldry", "rank-high", "rank-normal", "rank-low", "council",
}

var PortraitExtras = []string{"academy", "loom"}

var ExtraEffects = []string{"boat", "banner", "reweave", "horn", "fish", "ley", "charter", "wake", "request", "alarm", "spoil"}

func EffectFrameCount(kind string) int {
	switch kind {
	case "boat":
		return BoatFrames
	case "horn":
		return HornFrames
	case "banner":
		return BannerFrames
	case "fish":
		return FishFrames
	case "ley":
		return LeyFrames
	case "charter":
		return CharterFrames
	}
	return EffectFrames
}

var UINames = []string{
	"panel", "panel-dark", "button", "button-hover", "button-pressed", "button-disabled", "tab", "tab-active",
	"scroll", "tooltip", "bar-frame", "bar-fill-good", "bar-fill-bad", "minimap-frame", "woven-strip", "chip", "bar-mini",
	"kmap-frame", "kmap-ribbon", "plaque-locked", "plaque-available", "plaque-researching", "plaque-learned",
}

var Footprints = map[string]int{
	"keep-tower": 3, "keep-stone": 3, "keep-great": 3, "keep-citadel": 3, "codex": 2, "chancery": 3, "quartermaster": 2, "stewards-hall": 2,
	"wayfinder": 1, "observatory": 2, "gate": 2, "treasury": 1, "warhorn": 2,
	"seal-smithy": 2, "key-gate": 2, "bell-watch": 2, "cache-store": 2, "caravan-yard": 3,
	"shrine": 1, "signpost": 1, "foreign": 2, "harbor-gate": 2, "sea-chancery": 3,
}

func Names(a Atlas) []string {
	out := make([]string, 0, len(a.Regions))
	for name := range a.Regions {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func Build() Set {
	return Set{
		Terrain:   buildTerrain(),
		Buildings: buildBuildings(),
		Walkers:   buildWalkers(),
		Effects:   buildEffects(),
		Icons:     buildIcons(),
		Portraits: buildPortraits(),
		UI:        buildUI(),
	}
}
