package art

import "math"

var AdvisorPortraits = []string{"advisor-chief", "advisor-quartermaster", "advisor-wayfinder", "advisor-treasurer", "advisor-borg"}

var advisorSpecs = map[string]roleSpec{
	"advisor-chief": {tunic: ramp{rgb(58, 22, 30), rgb(96, 34, 44), rgb(134, 52, 60), rgb(180, 90, 90)}, legs: ink, hair: greyHair,
		skin: skinR, robe: true, tool: "decree", trim: goldR[3]},
	"advisor-quartermaster": roleSpecs["quartermaster"],
	"advisor-wayfinder":     roleSpecs["wayfinder"],
	"advisor-treasurer": {tunic: ramp{rgb(22, 60, 48), rgb(34, 90, 70), rgb(52, 124, 94), rgb(96, 168, 130)}, legs: ink, hair: darkHair,
		skin: darkSkin, tool: "gold", trim: goldR[2]},
	"advisor-borg": {tunic: magisterRobe, legs: ink, hair: greyHair, skin: skinR, hat: "hood", robe: true, tool: "decree", trim: goldR[2]},
}

var advisorExtras = map[string]func(c *canvas){"advisor-chief": chainOfOffice, "advisor-treasurer": spectacles, "advisor-borg": magisterClasp}

var advisorTools = map[string]func(c *canvas){"decree": iconDecree, "gold": iconGold}

func paintAdvisorPortrait(c *canvas, name string) {
	spec := advisorSpecs[name]
	portraitBackdrop(c)
	subject := newCanvas(PortraitSize, PortraitSize)
	bust(subject, spec)
	if extra := advisorExtras[name]; extra != nil {
		extra(subject)
	}
	subject.outline(ink)
	c.paste(subject, 0, 0, false)
	paint := toolIcons[spec.tool]
	if paint == nil {
		paint = advisorTools[spec.tool]
	}
	if paint != nil {
		tool := newCanvas(IconSize, IconSize)
		paint(tool)
		tool.outline(ink)
		c.paste(tool, 30, 29, false)
	}
	portraitFrame(c)
}

func chainOfOffice(c *canvas) {
	for i := 0; i <= 14; i++ {
		t := float64(i) / 14
		x := 16 + t*16
		y := 37 + math.Sin(t*math.Pi)*6
		col := tkGold
		if i%2 == 0 {
			col = tkGoldBright
		}
		c.dot(int(x), int(y), col)
	}
	c.disc(24, 43.5, 2.4, tkGold)
	c.disc(24, 43.5, 1.2, redR[2])
	c.dot(23, 42, tkGoldBright)
}

func spectacles(c *canvas) {
	for _, cx := range []float64{20.5, 27.5} {
		for k := 0; k < 12; k++ {
			a := float64(k) * math.Pi / 6
			c.dot(int(math.Floor(cx+math.Cos(a)*2.4)), int(math.Floor(20.8+math.Sin(a)*2.2)), goldR[1])
		}
	}
	c.rect(23, 20, 2, 1, goldR[1])
}
