package gfx

import "fmt"

const (
	MiddenLevels     = 4
	MiddenVariants   = 2
	CodexBloatLevels = 4
)

var qosArt = map[string]string{
	"BestEffort": "vagrant", "Burstable": "freeman", "Guaranteed": "landed",
	"vagrant": "vagrant", "freeman": "freeman", "landed": "landed",
}

var CouncilSeats = []string{"chief", "quartermaster", "stewards", "wayfinder", "treasurer", "augur"}

var councilPortraits = map[string]string{
	"chief": "advisor-chief", "quartermaster": "advisor-quartermaster", "stewards": "steward",
	"wayfinder": "advisor-wayfinder", "treasurer": "advisor-treasurer", "augur": "augur",
}

var overlayIcons = map[string]string{
	"midden": "overlay-midden", "codex": "codex", "labor": "overlay-labor", "unhoused": "overlay-unhoused", "heraldry": "overlay-heraldry",
}

var ratingIcons = map[string]string{
	"reliability": "rating-reliability", "efficiency": "rating-efficiency", "resilience": "rating-resilience", "favor": "rating-favor",
}

func QoSArt(class string) string { return qosArt[class] }

func HearthQoSName(tier int, state, qos string) string {
	art := QoSArt(qos)
	if art == "" {
		return HearthName(tier, state)
	}
	return HearthName(tier, state) + "/" + art
}

func HearthQoSNames(tier int, state, qos string) []string {
	chain := []string{HearthQoSName(tier, state, qos), HearthName(tier, state), HearthQoSName(tier, "ready", qos), HearthName(tier, "ready")}
	out := make([]string, 0, len(chain))
	seen := map[string]bool{}
	for _, n := range chain {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

func MiddenName(level, variant int) string {
	return fmt.Sprintf("midden/%d/%d", min(max(level, 0), MiddenLevels-1), ((variant%MiddenVariants)+MiddenVariants)%MiddenVariants)
}

func CodexBloatName(level int) string {
	return fmt.Sprintf("codexbloat/%d", min(max(level, 0), CodexBloatLevels-1))
}

func AlarmName(frame int) string { return EffectName("alarm", frame) }

func MiddenLevel(used float64, pressure bool) int {
	switch {
	case pressure || used >= 0.9:
		return 3
	case used >= 0.75:
		return 2
	case used >= 0.4:
		return 1
	}
	return 0
}

func CodexBloatLevel(used float64, alarm bool) int {
	switch {
	case alarm || used >= 0.9:
		return 3
	case used >= 0.7:
		return 2
	case used >= 0.4:
		return 1
	}
	return 0
}

func CouncilPortrait(seat string) string {
	if p, ok := councilPortraits[seat]; ok {
		return p
	}
	return councilPortraits["chief"]
}

func OverlayIcon(mode string) string { return overlayIcons[mode] }

func RatingIcon(rating string) string { return ratingIcons[rating] }

func RankIcon(rank string) string {
	switch rank {
	case "high", "low":
		return "rank-" + rank
	}
	return "rank-normal"
}
