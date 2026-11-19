package art

const (
	annexH       = 16.0
	stoneHallH   = 26.0
	greatHallH   = 30.0
	citadelHallH = 56.0
	chimneyRim   = 2.0
)

type yardRect struct{ x0, y0, x1, y1 float64 }

type keepYard struct {
	left, right yardRect
	front       [2]float64
}

type chimneySpot struct{ gx, gy, z0, z1 float64 }

func annexLeft(l look) bool { return l.pick(1, 2) == 0 }

func yardsSwapped(l look) bool { return l.pick(20, 2) == 1 }

func keepScale(l look) float64 { return iso{lk: l}.kit().scale }

func keepYards(k KeepLook, l look) keepYard {
	front := [2]float64{2.6, 2.6}
	switch k.Size {
	case "tower":
		if annexLeft(l) {
			return keepYard{left: yardRect{1.1, 2.3, 2.1, 2.95}, right: yardRect{2.3, 0.2, 2.95, 1.2}, front: front}
		}
		return keepYard{left: yardRect{0.2, 2.3, 1.45, 2.95}, right: yardRect{2.35, 1.2, 2.95, 2.1}, front: front}
	case "great":
		return keepYard{left: yardRect{0.75, 2.3, 1.95, 2.95}, right: yardRect{2.3, 0.75, 2.95, 1.95}, front: front}
	case "citadel":
		return keepYard{left: yardRect{0.62, 2.3, 1.95, 2.95}, right: yardRect{2.3, 0.62, 2.95, 1.95}, front: front}
	}
	return keepYard{left: yardRect{0.2, 2.3, 1.55, 2.95}, right: yardRect{2.3, 0.2, 2.95, 1.55}, front: front}
}

func keepChimney(k KeepLook, l look) chimneySpot {
	sc := keepScale(l)
	switch k.Size {
	case "tower":
		top := annexH * sc
		if annexLeft(l) {
			return chimneySpot{0.42, 1.42, top - 3, top + 9}
		}
		return chimneySpot{1.42, 0.42, top - 3, top + 9}
	case "great":
		return chimneySpot{1.3, 1.25, greatHallH*sc - 4, greatHallH*sc + 12}
	case "citadel":
		return chimneySpot{1.05, 1.2, citadelHallH*sc - 4, citadelHallH*sc + 10}
	}
	return chimneySpot{1.0, 1.0, stoneHallH*sc - 4, stoneHallH*sc + 12}
}

func forgeYard(k KeepLook, l look) yardRect {
	y := keepYards(k, l)
	if yardsSwapped(l) {
		return y.left.inset(0.04)
	}
	return y.right.inset(0.04)
}

func KeepVent(k KeepLook) (float64, float64) {
	l := keepLook(k)
	gx, gy, z := 0.0, 0.0, 0.0
	if k.Class == "compute" {
		cx, cy := forgeSpot(forgeYard(k, l))
		gx, gy, z = cx+chimneyW/2, cy+chimneyW/2, forgeTop*keepScale(l)+chimneyRim
	} else {
		c := keepChimney(k, l)
		gx, gy, z = c.gx+0.05, c.gy+0.05, c.z1
	}
	return (gx - gy) * TileWidth / 2, (gx+gy)*TileHeight/2 - keepFoot*TileHeight - z
}
