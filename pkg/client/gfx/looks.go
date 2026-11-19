package gfx

import (
	"fmt"

	"github.com/waxb/kubernites/pkg/client/art"
)

var varied = func() map[string]bool {
	out := map[string]bool{}
	for _, r := range art.VariedRoles {
		out[r] = true
	}
	return out
}()

func WalkerVariants(role string) int {
	if varied[role] {
		return art.WalkerLooks
	}
	return 1
}

func WalkerTiers(role string) int {
	if varied[role] {
		return art.WalkerTiers(role)
	}
	return 1
}

func WalkerLookName(role string, variant, tier int, action, dir string, frame int) string {
	n := WalkerVariants(role)
	v := ((variant % n) + n) % n
	t := min(max(tier, 1), WalkerTiers(role))
	if t == 1 && v == 0 {
		return WalkerName(role, action, dir, frame)
	}
	return fmt.Sprintf("walker/%s/t%dv%d/%s/%s/%d", role, t, v, action, dir, ((frame%4)+4)%4)
}

func WalkerVariantNames(role string, variant int, action, dir string, frame int) []string {
	return WalkerTierVariantNames(role, variant, 1, action, dir, frame)
}

func WalkerTierVariantNames(role string, variant, tier int, action, dir string, frame int) []string {
	base := WalkerNames(role, action, dir, frame)
	if !varied[role] {
		return base
	}
	out := []string{WalkerLookName(role, variant, tier, action, dir, frame)}
	if tier > 1 {
		out = append(out, WalkerLookName(role, 0, tier, action, dir, frame))
	}
	return append(out, base...)
}
