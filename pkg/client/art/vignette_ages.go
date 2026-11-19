package art

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	vgAgeTag  = "age"
	vgKeepArt = "keep-stone"
)

func AgedScene(scene string, age int) string {
	return fmt.Sprintf("%s:%s%d", scene, vgAgeTag, clampAge(age))
}

func (p *vgPaint) argAge() int {
	for _, part := range strings.Split(p.arg, ":") {
		digits, ok := strings.CutPrefix(part, vgAgeTag)
		if n, err := strconv.Atoi(digits); ok && err == nil {
			return clampAge(n)
		}
	}
	return ageDark
}

func vgKeep(age int) string { return BuildingLookName(vgKeepArt, "ready", age, 0) }

func vgAgedHut(tier, age, look int) string {
	return HearthLookName(min(max(tier, 1), 3), "ready", HearthQoS[look%len(HearthQoS)], age, 0)
}
