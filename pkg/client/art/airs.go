package art

import "image/color"

var AiredRoles = []string{"porter", "clerk", "hauler", "courier", "sealwright"}

func SkinRamps() [][4]color.NRGBA {
	out := make([][4]color.NRGBA, 0, len(skins))
	for _, s := range skins {
		out = append(out, [4]color.NRGBA(s))
	}
	return out
}

func HairColours() []color.NRGBA {
	return append([]color.NRGBA(nil), hairs...)
}

func AirColours() []color.NRGBA {
	out := HairColours()
	for _, s := range SkinRamps() {
		out = append(out, s[:]...)
	}
	return out
}
