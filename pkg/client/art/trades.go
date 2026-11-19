package art

import "image/color"

const HoldAction = "hold"

var TradeRoles = []string{"porter", "hauler", "clerk", "sealwright", "courier"}

var TradeKinds = map[string]string{"porter": "parcel", "hauler": "crate", "clerk": "ledger", "courier": "dispatch", "sealwright": ""}

var (
	porterR     = ramp{rgb(36, 56, 98), rgb(52, 84, 138), rgb(78, 118, 174), rgb(132, 166, 206)}
	haulerR     = ramp{rgb(120, 62, 24), rgb(168, 94, 36), rgb(206, 130, 52), rgb(236, 178, 100)}
	sealwrightR = ramp{rgb(70, 22, 30), rgb(108, 34, 42), rgb(146, 52, 56), rgb(190, 88, 84)}
	clerkR      = ramp{rgb(30, 64, 70), rgb(44, 94, 98), rgb(70, 130, 128), rgb(126, 180, 168)}
	apronCol    = rgb(54, 42, 38)
	glowCol     = color.NRGBA{255, 170, 60, 150}
)

type tradeFrame struct {
	c      *canvas
	spec   roleSpec
	p      pose
	top    int
	action string
	frame  int
}

func tradePose(action string, frame int) pose {
	if action == HoldAction {
		return poseFor("carry", frame)
	}
	return poseFor(action, frame)
}

func paintTrade(c *canvas, spec roleSpec, action string, frame int) {
	body := newWalkerBody()
	p := tradePose(action, frame)
	tf := tradeFrame{c: body, spec: spec, p: p, top: figureTop + p.bob, action: action, frame: frame}
	body.ellipse(8, float64(WalkerHeight-2), 3.5, 1.2, shadow)
	tradeBehind(tf)
	tf.legs()
	tf.torso(action == HoldAction)
	drawHead(body, spec, tf.headPose(), tf.top)
	tradeFront(tf)
	finishWalker(c, body)
}

func (tf tradeFrame) broad() bool { return tf.spec.trade == "hauler" }

func (tf tradeFrame) headPose() pose {
	p := tf.p
	if tf.spec.trade == "courier" && (tf.action == "walk" || tf.action == "carry") {
		p.bow = 1
	}
	return p
}

func (tf tradeFrame) legs() {
	if !tf.broad() {
		drawLegs(tf.c, tf.spec, tf.p, tf.top)
		return
	}
	lx, rx := 5+tf.p.legShift, 9-tf.p.legShift
	for _, x := range []int{lx, rx} {
		tf.c.rect(x, tf.top+11, 2, 21-(tf.top+11), tf.spec.legs)
		tf.c.rect(x, 21, 3, 1, ink)
	}
}

func (tf tradeFrame) span() (int, int) {
	if tf.broad() {
		return 5, 7
	}
	return 6, 5
}

func (tf tradeFrame) torso(armsUp bool) {
	c, spec, top := tf.c, tf.spec, tf.top
	x0, w := tf.span()
	for y := top + 5; y < top+11; y++ {
		c.rect(x0, y, w, 1, spec.tunic[1])
		c.dot(x0, y, spec.tunic[2])
		c.dot(x0+w-1, y, spec.tunic[0])
	}
	tradeApron(tf, x0, w)
	c.rect(x0, top+8, w, 1, woodR[0])
	c.dot(8, top+8, spec.trim)
	drawGarb(c, spec, top, top+11)
	c.dot(9, top+6, goldR[3])
	c.dot(9, top+7, goldR[1])
	tf.arms(x0, w, armsUp)
}

func (tf tradeFrame) arms(x0, w int, up bool) {
	c, spec, top, swing := tf.c, tf.spec, tf.top, tf.p.armSwing
	if tf.action == "work" || tf.action == "carry" {
		swing = 0
	}
	if up {
		c.rect(x0-1, top+2, 1, 5, spec.tunic[2])
		c.rect(x0+w, top+2, 1, 5, spec.tunic[0])
		c.dot(x0-1, top+1, spec.skin[2])
		c.dot(x0+w, top+1, spec.skin[1])
		return
	}
	c.rect(x0-1, top+6-swing, 1, 3, spec.tunic[2])
	c.dot(x0-1, top+9-swing, spec.skin[2])
	c.rect(x0+w, top+6+swing, 1, 3, spec.tunic[0])
	c.dot(x0+w, top+9+swing, spec.skin[1])
}

func tradeApron(tf tradeFrame, x0, w int) {
	switch tf.spec.trade {
	case "hauler":
		tf.c.rect(x0+1, tf.top+9, w-2, 3, woodR[1])
		tf.c.dot(x0+1, tf.top+9, woodR[2])
	case "sealwright":
		tf.c.rect(x0+1, tf.top+6, w-2, 7, apronCol)
		tf.c.dot(x0+1, tf.top+7, rgb(84, 68, 60))
		tf.c.rect(x0+1, tf.top+6, w-2, 1, rgb(84, 68, 60))
	}
}

func tradeBehind(tf tradeFrame) {
	switch tf.spec.trade {
	case "hauler":
		if tf.action == "carry" {
			backCrate(tf.c, 3, tf.top-4)
		}
	case "courier":
		if tf.action == "walk" || tf.action == "carry" {
			flutter := []int{0, 1, 0, 1}[tf.frame]
			tf.c.rect(3, tf.top+5+flutter, 3, 1, goldR[2])
			tf.c.dot(2, tf.top+6-flutter, goldR[1])
		}
	}
}

func tradeFront(tf tradeFrame) {
	if tf.action == HoldAction {
		if tf.spec.trade == "courier" {
			satchel(tf.c, tf.top)
		}
		return
	}
	tradeItems[tf.spec.trade](tf)
}
