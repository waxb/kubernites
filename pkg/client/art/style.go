package art

import (
	"fmt"
	"hash/fnv"
)

const (
	AgeTiers         = 4
	BuildingVariants = 3
)

const (
	ageDark = iota
	ageFeudal
	ageCastle
	ageEmpire
)

const (
	matNone uint8 = iota
	matWall
	matStone
	matRoof
	matGlass
	matTrim
)

type look struct {
	age, variant int
	seed         int
	id           ramp
}

func NameSeed(name string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(name))
	return h.Sum32()
}

func LookVariant(id string) int {
	return int(NameSeed(id) % BuildingVariants)
}

func clampAge(age int) int { return min(max(age, ageDark), AgeTiers-1) }

func newLook(kind string, age, variant int) look {
	l := look{age: clampAge(age), variant: variant, seed: int(NameSeed(kind)%1000003)*7 + variant*7919}
	l.id = hueShift(kindRamp(kind), l)
	return l
}

func (l look) roll(k int) float64 { return float64(hash(l.seed, k, 977)%1000) / 1000 }

func (l look) pick(k, n int) int { return int(hash(l.seed, k, 983) % uint32(max(n, 1))) }

func (l look) jit(k int, span float64) float64 { return (l.roll(k)*2 - 1) * span }

func (l look) chance(k int, p float64) bool { return l.roll(k) < p }

func (l look) at(least int) bool { return l.age >= least }

func (s iso) with(l look) iso {
	s.lk = l
	return s
}

func BuildingLookName(kind, state string, age, variant int) string {
	if age == 0 && variant == 0 {
		return buildingName(kind, state)
	}
	return fmt.Sprintf("building/%s/%s/%d/%d", kind, state, age, variant)
}

func HearthLookName(tier int, state, qos string, age, variant int) string {
	if age == 0 && variant == 0 {
		return hearthQoSName(tier, state, qos)
	}
	return fmt.Sprintf("%s/%d/%d", hearthQoSName(tier, state, qos), age, variant)
}

var kindRamps = map[string]ramp{
	"codex": tealR, "chancery": redR, "quartermaster": redR, "stewards-hall": blueR,
	"wayfinder": woodR, "observatory": blueR, "gate": redR, "treasury": goldR,
	"warhorn": redR, "seal-smithy": slateR, "key-gate": tealR, "bell-watch": goldR,
	"cache-store": slateR, "caravan-yard": redR, "shrine": goldR, "signpost": blueR, "foreign": purpleR, "harbor-gate": blueR, "sea-chancery": redR,
	"stall-parcel": blueR, "stall-ledger": tealR, "stall-crate": orangeR, "stall-dispatch": purpleR, "stall-seal": redR,
	"stall-guildhall": greenR, "stall-archive": blueR, "stall-herald": redR, "stall-plaque": slateR,
}

func kindRamp(kind string) ramp {
	if r, ok := kindRamps[kind]; ok {
		return r
	}
	return redR
}

func hueKin(r ramp) [2]ramp {
	switch r {
	case redR:
		return [2]ramp{orangeR, purpleR}
	case blueR:
		return [2]ramp{slateR, purpleR}
	case tealR:
		return [2]ramp{greenR, blueR}
	case greenR:
		return [2]ramp{tealR, meadowR}
	case orangeR:
		return [2]ramp{redR, goldR}
	case purpleR:
		return [2]ramp{blueR, redR}
	case goldR:
		return [2]ramp{orangeR, strawR}
	case slateR:
		return [2]ramp{blueR, greyR}
	case woodR:
		return [2]ramp{orangeR, dirtR}
	}
	return [2]ramp{r, greyR}
}

func hueShift(r ramp, l look) ramp {
	if l.variant == 0 {
		return r
	}
	kin := hueKin(r)[(l.variant-1)%2]
	t := 0.18 + 0.12*l.roll(1)
	out := ramp{}
	for i := range r {
		out[i] = mix(r[i], kin[i], t)
	}
	return out
}
