package art

import (
	"fmt"
	"image/color"
	"math"
)

type portraitJob struct {
	name  string
	paint func(c *canvas)
}

func portraitJobs(m *shotMemo) []portraitJob {
	out := []portraitJob{}
	building := map[string]bool{}
	for _, k := range BuildingKinds {
		building[k] = true
	}
	for _, r := range WalkerRoles {
		if role := r; !building[role] {
			out = append(out, portraitJob{role, func(c *canvas) { paintRolePortrait(c, role) }})
		}
	}
	for _, k := range append(append([]string{}, BuildingKinds...), StallKinds...) {
		kind := k
		for age := 0; age < AgeTiers; age++ {
			a := age
			out = append(out, portraitJob{AgedPortrait(kind, a), func(c *canvas) { paintSpritePortrait(c, m.get(kind, a)) }})
		}
	}
	out = append(out, portraitJob{"academy", paintAcademyPortrait}, portraitJob{"loom", paintLoomPortrait})
	for _, a := range AdvisorPortraits {
		name := a
		out = append(out, portraitJob{name, func(c *canvas) { paintAdvisorPortrait(c, name) }})
	}
	return out
}

func portraitEntries() []string {
	jobs := portraitJobs(newShotMemo())
	out := make([]string, len(jobs))
	for i, j := range jobs {
		out[i] = j.name
	}
	return out
}

func buildPortraits() Atlas {
	b, m := &atlasBuilder{}, newShotMemo()
	for _, j := range portraitJobs(m) {
		b.add(portraitName(j.name), PortraitSize, PortraitSize, PortraitSize/2, PortraitSize/2, j.paint)
	}
	addBuildIcons(b, m)
	return b.build()
}

func shadedEllipse(c *canvas, cx, cy, rx, ry float64, r ramp) {
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if dx*dx+dy*dy > 1 {
				continue
			}
			col := r[1]
			switch l := -dx*0.8 - dy*0.6; {
			case l > 0.8:
				col = r[3]
			case l > 0.05:
				col = r[2]
			case l < -0.6:
				col = r[0]
			}
			c.dot(x, y, col)
		}
	}
}

func portraitBackdrop(c *canvas) {
	top, low := rgb(128, 178, 214), rgb(226, 234, 214)
	for y := 0; y < PortraitSize; y++ {
		c.rect(0, y, PortraitSize, 1, mix(top, low, math.Min(float64(y)/34, 1)))
	}
	for x := 0; x < PortraitSize; x++ {
		far := 30 + int(math.Sin(float64(x)*0.17+1)*2.5)
		near := 36 + int(math.Sin(float64(x)*0.11+4)*2)
		c.rect(x, far, 1, PortraitSize-far, mix(greenR[1], top, 0.45))
		c.rect(x, near, 1, PortraitSize-near, grassR[1])
	}
	for _, cl := range [][3]float64{{12, 10, 4}, {18, 9, 3}, {37, 14, 3.5}} {
		c.ellipse(cl[0], cl[1], cl[2]*1.6, cl[2]*0.7, withAlpha(white, 210))
	}
}

func portraitFrame(c *canvas) {
	n := PortraitSize
	for i, col := range []color.NRGBA{ink, woodR[3], woodR[2], ink} {
		dark := []color.NRGBA{ink, woodR[1], woodR[0], ink}[i]
		c.rect(i, i, n-2*i, 1, col)
		c.rect(i, i, 1, n-2*i, col)
		c.rect(i, n-1-i, n-2*i, 1, dark)
		c.rect(n-1-i, i, 1, n-2*i, dark)
	}
	for _, p := range [][2]int{{1, 1}, {n - 3, 1}, {1, n - 3}, {n - 3, n - 3}} {
		c.rect(p[0], p[1], 2, 2, goldR[2])
		c.dot(p[0], p[1], goldR[3])
	}
}

func paintRolePortrait(c *canvas, role string) {
	portraitBackdrop(c)
	subject := newCanvas(PortraitSize, PortraitSize)
	switch role {
	case "imp":
		impFace(subject)
	case "sheep":
		sheepFace(subject)
	case "raven":
		ravenFace(subject)
	case "hound":
		houndFace(subject)
	default:
		bust(subject, roleSpecs[role])
		if roleSpecs[role].emblem {
			helmEmblem(subject, 24.5, 40.5)
		}
	}
	subject.outline(ink)
	c.paste(subject, 0, 0, false)
	if paint := toolIcons[roleSpecs[role].tool]; paint != nil {
		tool := newCanvas(IconSize, IconSize)
		paint(tool)
		tool.outline(ink)
		c.paste(tool, 30, 29, false)
	}
	portraitFrame(c)
}

func bust(c *canvas, s roleSpec) {
	if s.hat == "hood" {
		shadedEllipse(c, 24, 21, 12.5, 14, s.tunic)
	}
	shadedEllipse(c, 24, 50, 18, 15, s.tunic)
	if s.robe {
		c.rect(22, 36, 4, 12, s.trim)
	} else {
		if !s.emblem {
			c.shadePoly([]pt{{18, 36}, {24, 43}, {30, 36}}, flat(s.trim))
		}
		c.shadePoly([]pt{{20, 36}, {24, 40}, {28, 36}}, flat(s.skin[1]))
	}
	c.rect(20, 28, 8, 8, s.skin[1])
	if s.hat != "hood" {
		c.ellipse(24, 15, 9, 8, s.hair)
	}
	shadedEllipse(c, 24, 21, 7.5, 9, s.skin)
	c.rect(15, 19, 2, 4, s.skin[1])
	c.rect(32, 19, 2, 4, s.skin[1])
	face(c, s)
	portraitHat(c, s)
}

func helmEmblem(c *canvas, x, y float64) {
	c.disc(x, y, 3.4, tkGold)
	c.disc(x, y, 2.2, blueR[1])
	c.rect(int(x), int(y)-3, 1, 7, tkGold)
	c.rect(int(x)-3, int(y), 7, 1, tkGold)
	c.dot(int(x)-1, int(y)-3, tkGoldBright)
	c.dot(int(x), int(y), tkRed)
}

func face(c *canvas, s roleSpec) {
	brow := darken(s.hair, 0.25)
	for _, x := range []int{19, 26} {
		c.rect(x, 18, 3, 1, brow)
		c.rect(x+1, 20, 2, 2, ink)
		c.dot(x+1, 20, white)
	}
	mouth := darken(s.skin[0], 0.2)
	if s.hair == greyHair {
		c.shadePoly([]pt{{17, 24}, {31, 24}, {28, 31}, {24, 33}, {20, 31}}, flat(s.hair))
		mouth = darken(s.hair, 0.45)
	} else {
		c.dot(19, 25, mix(s.skin[2], redR[3], 0.4))
		c.dot(29, 25, mix(s.skin[1], redR[3], 0.4))
	}
	c.rect(23, 22, 2, 3, s.skin[1])
	c.dot(24, 24, s.skin[0])
	c.rect(22, 27, 5, 1, mouth)
}

func AgedPortrait(kind string, age int) string {
	if age <= 0 {
		return kind
	}
	return fmt.Sprintf("%s/%d", kind, min(age, AgeTiers-1))
}

func agedBuilding(kind string, age int) *canvas {
	if k, ok := CanonicalKeep(kind, age); ok {
		return readyKeep(k)
	}
	if isStallKind(kind) {
		return stallReady(kind, "sw", age, 0)
	}
	return agedReady(kind, buildingSize(kind), newLook(kind, age, 0))
}
