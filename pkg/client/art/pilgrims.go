package art

import "image/color"

var pilgrimCloaks = []ramp{
	{rgb(96, 82, 66), rgb(134, 118, 96), rgb(166, 150, 124), rgb(200, 186, 160)},
	{rgb(70, 78, 62), rgb(102, 112, 88), rgb(134, 144, 114), rgb(172, 180, 148)},
	{rgb(70, 76, 88), rgb(102, 110, 124), rgb(136, 144, 156), rgb(176, 182, 190)},
	{rgb(98, 74, 64), rgb(136, 108, 94), rgb(168, 138, 120), rgb(200, 174, 158)},
	{rgb(66, 62, 64), rgb(98, 94, 96), rgb(130, 124, 124), rgb(168, 162, 158)},
	{rgb(112, 98, 78), rgb(156, 140, 110), rgb(190, 174, 140), rgb(222, 208, 178)},
	{rgb(90, 86, 80), rgb(128, 122, 112), rgb(160, 154, 142), rgb(196, 190, 176)},
}

var pilgrimBindles = map[string]color.NRGBA{
	"bindle-oat": rgb(196, 170, 120), "bindle-slate": rgb(110, 124, 146), "bindle-moss": rgb(122, 138, 96),
}

func pilgrimLook(variant int) roleSpec {
	v := ((variant % WalkerLooks) + WalkerLooks) % WalkerLooks
	h := hash(v, 17, 2903)
	spec := roleSpecs["pilgrim"]
	spec.tunic = pilgrimCloaks[(v*3)%len(pilgrimCloaks)]
	spec.trim = spec.tunic[0]
	spec.skin = skins[(v+1)%len(skins)]
	spec.hair = hairs[(v*5+2)%len(hairs)]
	spec.hat = "hood"
	if v%3 == 2 {
		spec.hat = "straw"
	}
	spec.tool = []string{"bindle", "pilgrim-staff", "bindle-oat", "pilgrim-staff", "bindle-slate", "pilgrim-staff", "bindle-moss", "bindle"}[v]
	spec.short = int(h % 2)
	spec.beard = int(h>>3) % 3
	spec.style = int(h>>5) % 4
	spec.patch = v == 5
	if v%4 == 3 {
		spec.extra = 3
	}
	return spec
}

func drawPilgrimTool(c *canvas, tool string, hx, hy, lift int) {
	if tool == "pilgrim-staff" {
		c.line(hx, hy+5, hx+1, hy-7+lift, woodR[1])
		c.dot(hx+1, hy-8+lift, woodR[2])
		c.dot(hx+2, hy-4+lift, strawR[2])
		c.dot(hx+2, hy-3+lift, strawR[1])
		return
	}
	c.line(hx-5, hy-4, hx+1, hy, woodR[2])
	c.disc(float64(hx)-5, float64(hy)-5, 1.7, pilgrimBindles[tool])
}
