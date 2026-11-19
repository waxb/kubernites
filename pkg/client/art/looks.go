package art

import (
	"fmt"
	"hash/fnv"
	"image/color"
)

const WalkerLooks = 8

var VariedRoles = []string{
	"lamplighter", "baker", "scribe", "granary-keeper", "archivist", "courier", "tallyman", "oracle",
	"pilgrim", "wanderer", "builder", "steward", "quartermaster", "clerk",
	"porter", "hauler", "sealwright",
}

var tierTools = map[string][]string{"lantern-pole": {"lantern-pole", "lantern-pole-brass", "lantern-pole-twin"}, "parcel": {"parcel", "parcel", "parcel"}}

var tierTrims = []color.NRGBA{{}, greyR[3], goldR[3]}

var headgear = map[string][]string{
	"lamplighter": {"cap", "", "hood", "straw"}, "scribe": {"", "hood", "beret"}, "granary-keeper": {"straw", "", "cap"},
	"archivist": {"", "hood"}, "courier": {"feather", "", "cap"}, "tallyman": {"", "cap", "beret"},
	"pilgrim": {"hood", "", "straw"}, "wanderer": {"", "cap", "straw", "hood"}, "builder": {"leather", "", "cap"},
	"steward": {"", "beret"}, "clerk": {"beret", "", "cap"},
	"porter": {"cap", "", "straw", "hood"}, "hauler": {"", "leather", "cap"}, "sealwright": {"leather", "", "cap"},
}

var (
	paleSkin = ramp{rgb(150, 98, 76), rgb(210, 150, 116), rgb(240, 196, 160), rgb(252, 226, 196)}
	tanSkin  = ramp{rgb(110, 66, 44), rgb(166, 110, 72), rgb(204, 146, 96), rgb(230, 186, 134)}
	skins    = []ramp{skinR, darkSkin, paleSkin, tanSkin}
	hairs    = []color.NRGBA{brownHair, blondHair, darkHair, greyHair, rgb(170, 78, 40), rgb(30, 24, 26)}
	trims    = []color.NRGBA{cream, redR[2], blueR[2], greenR[3], goldR[2]}
	scarves  = []color.NRGBA{redR[2], blueR[2], goldR[2], greenR[2]}
)

func WalkerTiers(role string) int {
	if tiers, ok := tierTools[roleSpecs[role].tool]; ok {
		return len(tiers)
	}
	return 1
}

func roleSeed(role string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(role))
	return h.Sum32()
}

func lookSpec(role string, variant, tier int) roleSpec {
	if role == "pilgrim" && variant > 0 {
		return pilgrimLook(variant)
	}
	spec := roleSpecs[role]
	cue := 0
	if tools, ok := tierTools[spec.tool]; ok && tier > 1 {
		t := min(tier, len(tools))
		spec.tool, spec.trim, cue = tools[t-1], tierTrims[t-1], t+2
	}
	if variant <= 0 {
		spec.extra = cue
		return spec
	}
	seed := int(roleSeed(role) % 997)
	h := hash(seed, variant, 1201)
	spec.skin = skins[(variant+seed)%len(skins)]
	spec.hair = hairs[(variant*5+seed)%len(hairs)]
	spec.style = int(h % 4)
	if spec.style%2 == 0 && h>>4%3 != 0 {
		spec.beard = 1 + int(h>>6%2)
	}
	if hats, ok := headgear[role]; ok {
		spec.hat = hats[(variant+int(h>>8))%len(hats)]
	}
	if tier <= 1 {
		spec.trim = trims[(variant+int(h>>11))%len(trims)]
	}
	spec.patch = h>>14%3 == 0
	spec.extra = int(h>>16) % 4
	if cue > 0 {
		spec.extra = cue
	}
	spec.tunic = shiftRamp(spec.tunic, []float64{-0.1, 0.08, -0.04, 0.12, 0}[h>>19%5])
	return spec
}

func shiftRamp(r ramp, t float64) ramp {
	if t == 0 {
		return r
	}
	out := r
	for i, c := range r {
		if t < 0 {
			out[i] = darken(c, -t)
		} else {
			out[i] = mix(c, white, t*0.6)
		}
	}
	return out
}

func walkerLookName(role string, tier, variant int, action, dir string, frame int) string {
	return fmt.Sprintf("walker/%s/t%dv%d/%s/%s/%d", role, tier, variant, action, dir, frame)
}

func drawHairStyle(c *canvas, spec roleSpec, x, top int) {
	switch spec.style {
	case 1:
		c.rect(x-1, top, 1, 5, spec.hair)
		c.rect(x, top+1, 1, 4, spec.hair)
	case 2:
		c.rect(x, top+1, 1, 2, spec.skin[1])
		c.dot(x, top+1, spec.hair)
	case 3:
		c.rect(x-2, top+1, 2, 2, spec.hair)
	}
	switch spec.beard {
	case 1:
		c.rect(x+1, top+3, 4, 2, spec.hair)
		c.dot(x+3, top+3, spec.skin[1])
	case 2:
		c.rect(x+2, top+3, 3, 1, spec.hair)
	}
}

func drawGarb(c *canvas, spec roleSpec, top, bottom int) {
	if spec.patch {
		c.rect(7, min(top+9, bottom-2), 2, 2, mix(spec.tunic[1], woodR[2], 0.6))
	}
	switch spec.extra {
	case 1:
		c.rect(6, top+5, 5, 1, scarves[int(spec.hair.R)%len(scarves)])
	case 2:
		c.rect(9, top+9, 2, 2, woodR[1])
		c.dot(9, top+9, woodR[2])
	case 3:
		c.line(6, top+5, 10, top+9, woodR[0])
	case 4:
		c.rect(6, top+5, 5, 1, greyR[3])
		c.dot(8, top+6, white)
	case 5:
		c.line(6, top+5, 10, top+9, goldR[2])
		c.line(6, top+6, 9, top+9, goldR[3])
	}
}

func drawLanternPole(c *canvas, tool string, hx, hy, lift int) {
	c.line(hx, hy+2, hx+2, hy-9+lift, woodR[1])
	switch tool {
	case "lantern-pole-brass":
		c.rect(hx+2, hy-9+lift, 2, 3, white)
		c.rect(hx+2, hy-10+lift, 3, 1, goldR[2])
		c.dot(hx+2, hy-7+lift, goldR[1])
	case "lantern-pole-twin":
		c.rect(hx, hy-10+lift, 6, 1, woodR[2])
		c.rect(hx, hy-9+lift, 2, 3, lampLit)
		c.rect(hx+4, hy-9+lift, 2, 3, lampLit)
		c.dot(hx+2, hy-11+lift, goldR[3])
	default:
		c.rect(hx+2, hy-9+lift, 2, 3, lampLit)
		c.dot(hx+3, hy-10+lift, goldR[3])
	}
}
