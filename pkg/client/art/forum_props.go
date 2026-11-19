package art

import "fmt"

const (
	ForumKerbKinds = 2
	ForumPostKinds = 2
	ForumTermini   = 3
	ForumStatues   = 3
	forumKerbW     = 19
	forumKerbH     = 16
	forumKerbAX    = 9
	forumKerbAY    = 8
	forumKerbReach = 8
	forumKerbJoint = 6
	termW, termH   = 12, 14
	termAX, termAY = 6, 11
	postW2, postH2 = 11, 19
	postAX2        = 5
	postAY2        = 16
	statueW        = 16
	statueH        = 40
	statueAX       = 8
	statueAY       = 34
	fountainW      = 34
	fountainH      = 36
	fountainAX     = 17
	fountainAY     = 27
)

var (
	kerbR          = ramp{rgb(84, 76, 72), rgb(146, 134, 116), rgb(206, 196, 170), rgb(238, 230, 204)}
	ironDark       = rgb(52, 46, 50)
	forumKerbRamps = [AgeTiers]ramp{kerbR, kerbR, mixRamp(kerbR, slateR, 0.25), marbleR}
)

func ForumKerbName(axis string, age, v int) string {
	return fmt.Sprintf("forum/kerb/%s/%d/%d", axis, clampAge(age), v)
}
func ForumPostName(age, v int) string  { return fmt.Sprintf("forum/post/%d/%d", clampAge(age), v) }
func ForumTerminusName(v int) string   { return fmt.Sprintf("forum/terminus/%d", v) }
func ForumPlinthName(age int) string   { return fmt.Sprintf("forum/plinth/%d", clampAge(age)) }
func ForumStatueName(v int) string     { return fmt.Sprintf("forum/statue/%d", v) }
func ForumFountainName(age int) string { return fmt.Sprintf("forum/fountain/%d", clampAge(age)) }

func addForum(b *atlasBuilder) {
	addForumPaving(b)
	addForumEdges(b)
	addForumMonuments(b)
	addMarketGrowth(b)
}

func addForumEdges(b *atlasBuilder) {
	for age := ageFeudal; age < AgeTiers; age++ {
		for v := 0; v < ForumKerbKinds; v++ {
			aa, vv := age, v
			b.add(ForumKerbName("x", age, v), forumKerbW, forumKerbH, forumKerbAX, forumKerbAY, func(c *canvas) { paintForumKerb(c, -1, vv, forumKerbRamps[aa]) })
			b.add(ForumKerbName("y", age, v), forumKerbW, forumKerbH, forumKerbAX, forumKerbAY, func(c *canvas) { paintForumKerb(c, 1, vv, forumKerbRamps[aa]) })
		}
		for v := 0; v < ForumPostKinds; v++ {
			aa, vv := age, v
			b.add(ForumPostName(age, v), postW2, postH2, postAX2, postAY2, func(c *canvas) { paintForumPost(c, aa, vv) })
		}
	}
	for v := 0; v < ForumTermini; v++ {
		vv := v
		b.add(ForumTerminusName(v), termW, termH, termAX, termAY, func(c *canvas) { paintTerminus(c, vv) })
	}
}

func forumKerbGround(slope, x int) int {
	d := x - forumKerbAX
	if slope < 0 {
		return forumKerbAY - (d+1)/2
	}
	return forumKerbAY + d/2
}

func paintForumKerb(c *canvas, slope, v int, r ramp) {
	for x := forumKerbAX - forumKerbReach; x <= forumKerbAX+forumKerbReach; x++ {
		y := forumKerbGround(slope, x)
		joint := (x+v*3)%forumKerbJoint == 0
		top, face := r[2], r[1]
		if joint {
			top, face = r[1], r[0]
		} else if (x+v)%2 == 0 {
			top = r[3]
		}
		c.dot(x, y-3, top)
		c.dot(x, y-2, darken(top, 0.06))
		c.dot(x, y-1, face)
		c.dot(x, y, r[0])
		c.dot(x, y+1, shadow)
	}
}

func paintForumPost(c *canvas, age, v int) {
	body := forumKerbRamps[age]
	top, foot := 8-btoi(age >= ageCastle)*2, postAY2-2
	c.rect(4, top, 3, foot-top, body[2])
	c.rect(4, top, 1, foot-top, body[3])
	c.rect(6, top, 1, foot-top, body[1])
	c.rect(3, top-2, 5, 2, body[3])
	c.dot(5, top-3, body[3])
	c.rect(4, foot, 3, 1, body[0])
	switch {
	case age == ageCastle:
		c.rect(4, top-3, 3, 1, ironDark)
		c.dot(5, top+3, ironDark)
	case age == ageEmpire:
		c.rect(4, top-3, 3, 1, goldR[2])
		c.dot(5, top-4, goldR[3])
	case v == 1:
		c.dot(5, top+3, ironDark)
	}
	c.outline(ink)
	propShadow(c, 5.5, float64(foot)+1.5, 3)
}

func paintTerminus(c *canvas, v int) {
	s := isoFrame(c, 0, termAX, termAY)
	h := 5 + float64(v)
	lean := float64(v%2)*2 - 1
	s.box(-0.07, -0.05, 0.07, 0.05, 0, h, faces{top: flat(stoneR[3]), left: flat(stoneR[2]), right: flat(stoneR[1])})
	tip := s.p(0, 0, h)
	c.dot(int(tip.x+lean), int(tip.y)-1, stoneR[2])
	c.dot(int(tip.x)-1, int(tip.y)+3, mossR[2])
	c.dot(int(tip.x)+1, int(tip.y)+5, mossR[1])
	c.outline(ink)
	base := s.p(0, 0, 0)
	c.dot(int(base.x)-3, int(base.y), grassR[3])
	c.dot(int(base.x)+3, int(base.y)-1, grassR[2])
	c.dot(int(base.x)+2, int(base.y), wildflowers[v%len(wildflowers)])
	propShadow(c, base.x+0.5, base.y+0.5, 3)
}

func paintPlinth(s iso, r ramp, z float64) float64 {
	s.box(-0.16, -0.16, 0.16, 0.16, 0, 2, faces{top: flat(r[3]), left: flat(r[2]), right: flat(r[1])})
	s.box(-0.11, -0.11, 0.11, 0.11, 2, z, faces{top: flat(r[3]), left: flat(r[2]), right: flat(r[1])})
	s.box(-0.14, -0.14, 0.14, 0.14, z, z+1.5, faces{top: flat(r[3]), left: flat(r[3]), right: flat(r[2])})
	return z + 1.5
}

func statueRamp(v int) ramp {
	switch v {
	case 1:
		return copperR
	case 2:
		return goldR
	}
	return marbleR
}

func paintStatue(c *canvas, v int, figure bool) {
	s := isoFrame(c, 0, statueAX, statueAY)
	top := paintPlinth(s, marbleR, 8)
	if figure {
		r := statueRamp(v)
		s.turn(lathe{cx: 0, cy: 0, z: top, h: 11, r: 0.075, profile: bellProfile}, toneMat(r), matStone)
		head := s.p(0, 0, top+13)
		c.disc(head.x, head.y, 1.8, r[2])
		c.dot(int(head.x)-1, int(head.y)-1, r[3])
		arm := s.p(0.05, 0.02, top+9)
		c.line(int(arm.x), int(arm.y), int(arm.x)+3, int(arm.y)-5+v, r[1])
	} else {
		urn := s.p(0, 0, top)
		c.ellipse(urn.x, urn.y-3, 2.5, 3, copperR[2])
		c.dot(int(urn.x)-1, int(urn.y)-4, copperR[3])
	}
	c.outline(ink)
	base := s.p(0, 0, 0)
	propShadow(c, base.x+0.5, base.y+0.5, 5)
}

func paintFountain(c *canvas, age int) {
	s := isoFrame(c, 0, fountainAX, fountainAY)
	r := forumKerbRamps[age]
	s.turn(lathe{cx: 0, cy: 0, z: 0, h: 4, r: 0.36}, toneMat(r), matStone)
	s.turn(lathe{cx: 0, cy: 0, z: 3.5, h: 0.5, r: 0.3}, toneMat(waterR), matNone)
	s.turn(lathe{cx: 0, cy: 0, z: 4, h: 7, r: 0.05}, toneMat(r), matStone)
	top := 11.0
	if age == ageEmpire {
		s.turn(lathe{cx: 0, cy: 0, z: 10, h: 2, r: 0.14, profile: domeProfile}, toneMat(r), matStone)
		s.finial(0, 0, 12, 4)
		top = 16
	}
	jet := s.p(0, 0, top)
	for i, d := range [][2]int{{-3, 2}, {3, 2}, {-4, 5}, {4, 5}, {0, -1}} {
		c.dot(int(jet.x)+d[0], int(jet.y)+d[1], waterR[3-i%2])
	}
	c.outline(ink)
	base := s.p(0, 0, 0)
	propShadow(c, base.x+0.5, base.y+0.5, 9)
}
