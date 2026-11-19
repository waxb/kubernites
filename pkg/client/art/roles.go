package art

import "image/color"

type roleSpec struct {
	tunic  ramp
	legs   color.NRGBA
	hair   color.NRGBA
	skin   ramp
	hat    string
	robe   bool
	tool   string
	trim   color.NRGBA
	emblem bool
	style  int
	beard  int
	patch  bool
	extra  int
	short  int
	trade  string
}

var brownHair, blondHair, darkHair, greyHair = rgb(96, 58, 34), rgb(214, 170, 84), rgb(52, 38, 34), rgb(170, 166, 160)

var darkSkin = ramp{rgb(84, 50, 36), rgb(128, 82, 56), rgb(162, 110, 74), rgb(196, 146, 104)}

var roleSpecs = map[string]roleSpec{
	"lamplighter":    {tunic: orangeR, legs: woodR[0], hair: brownHair, skin: skinR, hat: "cap", tool: "lantern-pole", trim: goldR[2]},
	"baker":          {tunic: ramp{rgb(170, 160, 150), rgb(214, 206, 196), rgb(240, 236, 228), white}, legs: woodR[1], hair: blondHair, skin: skinR, hat: "chef", tool: "loaf", trim: sandR[2]},
	"scribe":         {tunic: blueR, legs: blueR[0], hair: darkHair, skin: skinR, robe: true, tool: "quill", trim: cream},
	"granary-keeper": {tunic: greenR, legs: woodR[0], hair: brownHair, skin: darkSkin, hat: "straw", tool: "sack", trim: strawR[2]},
	"archivist":      {tunic: greyR, legs: greyR[0], hair: greyHair, skin: skinR, robe: true, tool: "tome", trim: redR[2]},
	"courier":        {tunic: purpleR, legs: woodR[0], hair: blondHair, skin: skinR, hat: "feather", tool: "satchel", trim: goldR[2], trade: "courier"},
	"tallyman":       {tunic: woodR, legs: inkSft, hair: darkHair, skin: darkSkin, tool: "tally", trim: cream},
	"oracle":         {tunic: purpleR, legs: purpleR[0], hair: white, skin: skinR, robe: true, hat: "hood", tool: "orb", trim: voidR[3]},
	"chancellor":     {tunic: ramp{rgb(70, 20, 50), rgb(120, 34, 70), rgb(160, 54, 92), rgb(206, 96, 130)}, legs: ink, hair: greyHair, skin: skinR, robe: true, hat: "crown", tool: "seal", trim: goldR[3]},
	"codex-keeper":   {tunic: tealR, legs: tealR[0], hair: brownHair, skin: skinR, robe: true, hat: "hood", tool: "book", trim: goldR[2]},
	"quartermaster":  {tunic: ramp{rgb(70, 74, 36), rgb(106, 110, 52), rgb(140, 146, 70), rgb(186, 190, 110)}, legs: woodR[0], hair: darkHair, skin: darkSkin, hat: "tricorn", tool: "pennant", trim: goldR[2]},
	"steward":        {tunic: ramp{rgb(26, 34, 70), rgb(40, 54, 104), rgb(62, 80, 140), rgb(110, 130, 190)}, legs: ink, hair: brownHair, skin: skinR, tool: "keys", trim: goldR[3]},
	"herald":         {tunic: ramp{redR[0], redR[1], goldR[2], goldR[3]}, legs: redR[0], hair: blondHair, skin: skinR, hat: "feather", tool: "horn", trim: goldR[3]},
	"wayfinder":      {tunic: forestR, legs: woodR[0], hair: brownHair, skin: darkSkin, hat: "hood", tool: "staff", trim: sandR[2]},
	"augur":          {tunic: ramp{rgb(30, 26, 70), rgb(50, 44, 110), rgb(76, 70, 150), rgb(130, 124, 200)}, legs: ink, hair: greyHair, skin: skinR, robe: true, hat: "star", tool: "lens", trim: goldR[3]},
	"reeve":          {tunic: dirtR, legs: woodR[0], hair: greyHair, skin: skinR, hat: "cap", tool: "lantern", trim: goldR[1]},
	"pilgrim":        {tunic: ramp{rgb(90, 86, 80), rgb(128, 122, 112), rgb(160, 154, 142), rgb(196, 190, 176)}, legs: woodR[0], hair: brownHair, skin: skinR, robe: true, hat: "hood", tool: "bindle", trim: rgb(112, 86, 64)},
	"builder":        {tunic: orangeR, legs: blueR[0], hair: darkHair, skin: darkSkin, hat: "leather", tool: "hammer", trim: woodR[2]},
	"wanderer":       {tunic: sandR, legs: woodR[1], hair: brownHair, skin: skinR, tool: "none", trim: woodR[2]},
	"helmbearer":     {tunic: blueR, legs: woodR[0], hair: blondHair, skin: skinR, tool: "lantern", trim: goldR[3], emblem: true},
	"clerk":          {tunic: clerkR, legs: inkSft, hair: brownHair, skin: skinR, hat: "beret", tool: "quill-ledger", trim: cream, trade: "clerk"},
	"porter":         {tunic: porterR, legs: woodR[0], hair: brownHair, skin: skinR, hat: "cap", tool: "parcel", trim: cream, trade: "porter"},
	"hauler":         {tunic: haulerR, legs: blueR[0], hair: darkHair, skin: darkSkin, tool: "crate-back", trim: woodR[2], trade: "hauler"},
	"sealwright":     {tunic: sealwrightR, legs: ink, hair: greyHair, skin: paleSkin, hat: "leather", tool: "seal-press", trim: sealR[2], trade: "sealwright"},
	"woodcutter":     {tunic: redR, legs: woodR[0], hair: brownHair, skin: skinR, hat: "cap", tool: "axe", trim: woodR[3]},
	"mason":          {tunic: blueR, legs: woodR[1], hair: darkHair, skin: darkSkin, hat: "leather", tool: "pick", trim: stoneR[3]},
	"watchman":       {tunic: slateR, legs: woodR[0], hair: greyHair, skin: skinR, hat: "hood", tool: "lantern", trim: goldR[2], beard: 1},
}

func drawHat(c *canvas, spec roleSpec, x, y int) {
	switch spec.hat {
	case "cap", "cap-red":
		col := woodR[1]
		if spec.hat == "cap-red" {
			col = redR[2]
		}
		c.rect(x, y-1, 5, 2, col)
		c.rect(x+4, y, 2, 1, col)
	case "chef":
		c.rect(x, y-2, 5, 3, white)
		c.rect(x+1, y-3, 3, 1, white)
	case "straw", "tricorn":
		col := strawR[2]
		if spec.hat == "tricorn" {
			col = inkSft
		}
		c.rect(x-1, y, 7, 1, col)
		c.rect(x+1, y-2, 3, 2, col)
	case "hood":
		c.rect(x-1, y-1, 7, 3, spec.tunic[1])
		c.rect(x-1, y+2, 2, 3, spec.tunic[1])
	case "crown":
		c.rect(x, y-1, 5, 2, goldR[2])
		c.dot(x, y-2, goldR[3])
		c.dot(x+2, y-2, goldR[3])
		c.dot(x+4, y-2, goldR[3])
		c.dot(x+2, y-1, redR[3])
	case "feather":
		c.rect(x, y-1, 5, 2, goldR[1])
		c.dot(x+4, y-2, white)
		c.dot(x+5, y-3, white)
	case "star":
		c.shadePoly([]pt{{float64(x) - 1, float64(y) + 1}, {float64(x) + 2.5, float64(y) - 4}, {float64(x) + 6, float64(y) + 1}}, flat(spec.tunic[1]))
		c.dot(x+2, y-2, goldR[3])
	case "beret":
		c.rect(x-1, y-1, 6, 2, redR[1])
		c.rect(x+3, y-2, 3, 1, redR[1])
		c.dot(x, y-1, redR[2])
	case "leather":
		c.rect(x, y-1, 5, 2, woodR[0])
		c.dot(x+1, y-1, woodR[2])
	}
}
