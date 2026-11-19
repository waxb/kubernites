package art

import "fmt"

const (
	PileWood         = "wood"
	PileStone        = "stone"
	RequestPileUnits = 4
	pileW            = 19
	pileH            = 13
	pileAX           = 8
	pileAY           = 10
	pileShadowW      = 13
	logEnd           = 4
	logRise          = 3
	blockW           = 4
	blockH           = 3
)

var RequestPileKinds = []string{PileWood, PileStone}

type pileCell struct{ x, y int }

var pileLayouts = [RequestPileUnits + 1][]pileCell{
	nil,
	{{4, 7}},
	{{3, 7}, {7, 7}},
	{{3, 7}, {7, 7}, {5, 4}},
	{{3, 7}, {7, 7}, {11, 7}, {5, 4}},
}

var woodLayouts = [RequestPileUnits + 1][]pileCell{
	nil,
	{{5, 6}},
	{{3, 6}, {7, 6}},
	{{3, 6}, {7, 6}, {5, 3}},
	{{2, 6}, {6, 6}, {10, 6}, {6, 3}},
}

func RequestPileName(kind string, units int) string {
	return fmt.Sprintf("pile/%s/%d", kind, units)
}

func addRequestPiles(b *atlasBuilder) {
	for n := 1; n <= RequestPileUnits; n++ {
		units := n
		b.add(RequestPileName(PileWood, n), pileW, pileH, pileAX, pileAY, func(c *canvas) { paintWoodPile(c, units) })
		b.add(RequestPileName(PileStone, n), pileW, pileH, pileAX, pileAY, func(c *canvas) { paintStoneStack(c, units) })
	}
}

func paintWoodPile(c *canvas, units int) {
	logs := woodLayouts[min(max(units, 1), RequestPileUnits)]
	for i := len(logs) - 1; i >= 0; i-- {
		logBody(c, logs[i])
	}
	for _, l := range logs {
		logFace(c, l)
	}
	finishPile(c)
}

func logBody(c *canvas, at pileCell) {
	for i := logRise; i >= 1; i-- {
		x, y := at.x+i, at.y-i/2
		c.rect(x, y, logEnd, logEnd, barkR[1])
		c.rect(x+1, y, logEnd-2, 1, barkR[2])
		c.dot(x+logEnd-1, y+logEnd-1, barkR[0])
	}
}

func logFace(c *canvas, at pileCell) {
	c.rect(at.x+1, at.y, logEnd-2, logEnd, woodR[3])
	c.rect(at.x, at.y+1, logEnd, logEnd-2, woodR[3])
	c.rect(at.x+1, at.y+1, logEnd-2, logEnd-2, woodR[2])
	c.dot(at.x+1, at.y+1, woodR[1])
}

func paintStoneStack(c *canvas, units int) {
	for _, b := range pileLayouts[min(max(units, 1), RequestPileUnits)] {
		stoneBlock(c, b)
	}
	finishPile(c)
}

func stoneBlock(c *canvas, at pileCell) {
	c.rect(at.x+1, at.y-1, blockW, 1, cutR[3])
	c.rect(at.x+blockW, at.y, 1, blockH, cutR[1])
	c.rect(at.x, at.y, blockW, blockH, cutR[2])
	c.rect(at.x, at.y, blockW, 1, cutR[3])
	c.rect(at.x, at.y, 1, blockH, cutR[0])
	c.rect(at.x, at.y+blockH-1, blockW, 1, cutR[1])
}

func finishPile(c *canvas) {
	c.outline(ink)
	groundShadow(c, size{w: pileShadowW, h: pileH, ax: pileAX, ay: pileAY})
}
