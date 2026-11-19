package art

import "image/color"

const (
	FolkLooks = 4
	folkMute  = 0.24
)

var FolkRoles = []string{"farmer", "carrier", "merchant", "dockhand", "child"}

var folkSpecs = map[string]roleSpec{
	"farmer":   {tunic: ramp{rgb(70, 78, 44), rgb(100, 110, 60), rgb(132, 140, 80), rgb(172, 176, 110)}, legs: woodR[0], hair: brownHair, skin: tanSkin, hat: "straw", tool: "none", trim: strawR[1]},
	"carrier":  {tunic: ramp{rgb(54, 70, 88), rgb(78, 100, 120), rgb(108, 132, 150), rgb(150, 170, 182)}, legs: woodR[1], hair: darkHair, skin: skinR, tool: "none", trim: cream},
	"merchant": {tunic: ramp{rgb(92, 44, 50), rgb(132, 66, 68), rgb(166, 94, 88), rgb(204, 140, 124)}, legs: inkSft, hair: greyHair, skin: paleSkin, hat: "beret", tool: "none", trim: goldR[1]},
	"dockhand": {tunic: ramp{rgb(96, 76, 52), rgb(136, 110, 76), rgb(170, 142, 100), rgb(204, 178, 134)}, legs: blueR[0], hair: blondHair, skin: darkSkin, hat: "cap", tool: "none", trim: woodR[2]},
}

var childTunics = []ramp{
	{rgb(64, 84, 120), rgb(92, 118, 158), rgb(126, 152, 188), rgb(176, 196, 216)},
	{rgb(60, 96, 66), rgb(86, 132, 88), rgb(118, 166, 112), rgb(170, 204, 150)},
	{rgb(140, 70, 80), rgb(184, 100, 104), rgb(214, 138, 132), rgb(236, 184, 170)},
	{rgb(132, 110, 50), rgb(180, 150, 70), rgb(214, 184, 100), rgb(236, 216, 150)},
}

func FolkName(role string, look int, action, dir string, frame int) string {
	if look <= 0 {
		return walkerName(role, action, dir, frame)
	}
	return walkerLookName(role, 1, look, action, dir, frame)
}

func addFolk(b *atlasBuilder) {
	for _, r := range FolkRoles {
		for look := 0; look < FolkLooks; look++ {
			for _, a := range WalkerActions {
				for f := 0; f < WalkerFrames; f++ {
					role, lk, action, frame := r, look, a, f
					addWalker(b, func(dir string) string { return FolkName(role, lk, action, dir, frame) }, func(c *canvas) {
						paintFolk(c, role, lk, action, frame)
					})
				}
			}
		}
	}
}

func folkSpec(role string, look int) roleSpec {
	spec := folkSpecs[role]
	if look <= 0 {
		return spec
	}
	h := hash(int(roleSeed(role)%997), look, 733)
	spec.skin = skins[(look+int(h))%len(skins)]
	spec.hair = hairs[(look*3+int(h>>3))%len(hairs)]
	spec.style = int(h>>6) % 4
	spec.tunic = shiftRamp(spec.tunic, []float64{-0.12, 0.1, -0.05, 0.14}[look%4])
	if role == "carrier" || role == "dockhand" {
		spec.hat = []string{"", "cap", "hood", ""}[look%4]
	}
	return spec
}

func paintFolk(c *canvas, role string, look int, action string, frame int) {
	body := newWalkerBody()
	p := poseFor(action, frame)
	p.carry = false
	if role == "child" {
		paintChild(body, look, action, frame)
	} else {
		drawPerson(body, folkSpec(role, look), p)
		top := figureTop + p.bob
		folkItems[role](body, action, frame, top, p)
	}
	body.mapPixels(mute)
	finishWalker(c, body)
}

func mute(col color.NRGBA) color.NRGBA {
	if col.A == 0 {
		return col
	}
	return mix(col, grey(col), folkMute)
}

var folkItems = map[string]func(c *canvas, action string, frame, top int, p pose){
	"farmer": farmerItem, "carrier": carrierItem, "merchant": merchantItem, "dockhand": dockhandItem,
}

func farmerItem(c *canvas, action string, frame, top int, p pose) {
	if action == "carry" {
		basket(c, 8, top+8)
		return
	}
	lift := 0
	if action == "work" && frame < 2 {
		lift = -3
	}
	hx, hy := 11, top+8-p.armSwing
	c.line(hx, hy+3, hx+1, hy-7+lift, woodR[1])
	c.rect(hx-1, hy-8+lift, 4, 1, greyR[2])
	c.dot(hx+2, hy-7+lift, greyR[1])
}

func basket(c *canvas, x, y int) {
	c.rect(x-3, y, 7, 3, strawR[1])
	c.rect(x-3, y, 7, 1, strawR[2])
	c.dot(x-1, y-1, orangeR[2])
	c.dot(x+1, y-1, greenR[2])
	c.dot(x, y-1, redR[2])
	c.dot(x+2, y-1, orangeR[1])
}

func carrierItem(c *canvas, action string, frame, top int, p pose) {
	y := top + 5
	c.line(3, y+1, 13, y-1, woodR[2])
	sway := 0
	if action == "walk" || action == "carry" {
		sway = []int{0, 1, 0, -1}[frame]
	}
	bucket(c, 3+sway, y+2, action != "walk")
	bucket(c, 13+sway, y, action != "walk")
}

func bucket(c *canvas, x, y int, full bool) {
	c.line(x, y, x, y+2, inkSft)
	c.rect(x-1, y+3, 3, 3, woodR[1])
	c.dot(x-1, y+3, woodR[2])
	if full {
		c.rect(x-1, y+3, 3, 1, blueR[3])
	}
}

func merchantItem(c *canvas, action string, frame, top int, p pose) {
	c.rect(6, top+7, 5, 4, plasterR[3])
	c.rect(6, top+7, 5, 1, plasterR[2])
	if action != "work" {
		c.rect(10, top+8-p.armSwing, 3, 2, redR[2])
		return
	}
	up := []int{-4, -5, -4, -2}[frame]
	c.rect(11, top+6+up, 1, 4, woodR[0])
	c.disc(12, float64(top+5+up), 1.4, []color.NRGBA{redR[2], goldR[2], greenR[2], orangeR[2]}[frame])
}

func dockhandItem(c *canvas, action string, frame, top int, p pose) {
	switch action {
	case "carry":
		crate(c, 5, top+5)
	case "work":
		crate(c, 5, top+1+[]int{0, -2, -3, -1}[frame])
	default:
		c.rect(11, top+8-p.armSwing, 2, 2, woodR[1])
	}
}

func crate(c *canvas, x, y int) {
	c.rect(x, y, 7, 5, woodR[2])
	c.rect(x, y, 7, 1, woodR[3])
	c.line(x, y+4, x+6, y, woodR[1])
	c.rect(x, y+4, 7, 1, woodR[0])
}

func paintChild(c *canvas, look int, action string, frame int) {
	tunic := childTunics[look%len(childTunics)]
	skin := []ramp{skinR, paleSkin, tanSkin, darkSkin}[look%4]
	hair := hairs[(look*5+2)%len(hairs)]
	hop, legs, arms := childPose(action, frame)
	top := 10 + hop
	c.ellipse(8, float64(WalkerHeight-2), 2.4, 0.9, shadow)
	c.rect(7+legs, top+9, 1, 2, woodR[0])
	c.rect(9-legs, top+9, 1, 2, woodR[0])
	c.dot(7+legs, top+11, ink)
	c.dot(9-legs, top+11, ink)
	childBody(c, tunic, skin, top, arms)
	c.rect(6, top, 4, 4, skin[2])
	c.rect(6, top+3, 4, 1, skin[1])
	c.rect(6, top-1, 4, 1, hair)
	c.rect(6, top, 4, 1, hair)
	c.dot(6, top+1, hair)
	c.dot(7, top+2, ink)
	c.dot(9, top+2, ink)
	if action == "carry" {
		c.disc(12, float64(top+7), 1.5, redR[2])
		c.dot(11, top+6, white)
	}
}

func childBody(c *canvas, tunic, skin ramp, top, arms int) {
	for y := top + 4; y < top+9; y++ {
		c.rect(6, y, 5, 1, tunic[2])
		c.dot(6, y, tunic[3])
		c.dot(10, y, tunic[1])
	}
	c.rect(6, top+4, 5, 1, tunic[3])
	c.rect(6, top+8, 5, 1, tunic[0])
	c.rect(5, top+5-arms, 1, 3, tunic[2])
	c.rect(11, top+5-arms, 1, 3, tunic[1])
	c.dot(5, top+8-arms, skin[2])
	c.dot(11, top+8-arms, skin[1])
}

func childPose(action string, frame int) (int, int, int) {
	switch action {
	case "walk", "carry":
		return []int{0, -1, 0, -1}[frame], []int{0, 1, 0, -1}[frame], 0
	case "work":
		return []int{0, -2, -3, -1}[frame], 0, []int{0, 2, 3, 1}[frame]
	}
	return []int{0, 0, 0, -1}[frame], 0, 0
}
