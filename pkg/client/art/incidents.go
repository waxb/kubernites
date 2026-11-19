package art

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
)

const (
	RestartBadgeMax = 99
	RestartBadgeW   = 19
	RestartBadgeH   = 13
	BackoffRingW    = 21
	BackoffRingH    = 21
	BackoffRingAX   = 10
	BackoffRingAY   = 10
	IncidentFrames  = 4
	IncidentMarkerW = 19
	IncidentMarkerH = 25
	backoffGap      = 3.0
)

var BackoffWaits = []int{10, 20, 40, 80, 160, 300}

var backoffLit = orangeR[2]

var digitGlyphs = [10][]string{
	{"###", "#.#", "#.#", "#.#", "###"},
	{".#.", "##.", ".#.", ".#.", "###"},
	{"###", "..#", "###", "#..", "###"},
	{"###", "..#", ".##", "..#", "###"},
	{"#.#", "#.#", "###", "..#", "..#"},
	{"###", "#..", "###", "..#", "###"},
	{"###", "#..", "###", "#.#", "###"},
	{"###", "..#", ".#.", ".#.", ".#."},
	{"###", "#.#", "###", "#.#", "###"},
	nineGlyph,
}

var restartGlyph = []string{".###", "#.##", "#...", "#..#", ".##."}

func RestartBadgeName(n int) string {
	return fmt.Sprintf("incident/restarts/%d", min(max(n, 1), RestartBadgeMax))
}

func BackoffRingName(step int) string {
	return fmt.Sprintf("incident/backoff/%d", min(max(step, 0), len(BackoffWaits)))
}

func IncidentMarkerName(frame int) string {
	return fmt.Sprintf("incident/marker/%d", ((frame%IncidentFrames)+IncidentFrames)%IncidentFrames)
}

func addIncidentArt(b *atlasBuilder) {
	for n := 1; n <= RestartBadgeMax; n++ {
		count := n
		b.add(RestartBadgeName(n), RestartBadgeW, RestartBadgeH, RestartBadgeW/2, RestartBadgeH-1, padded(func(c *canvas) { paintRestartBadge(c, count) }))
	}
	for s := 0; s <= len(BackoffWaits); s++ {
		step := s
		b.add(BackoffRingName(s), BackoffRingW, BackoffRingH, BackoffRingAX, BackoffRingAY, padded(func(c *canvas) { paintBackoffRing(c, step) }))
	}
	for f := 0; f < IncidentFrames; f++ {
		frame := f
		b.add(IncidentMarkerName(f), IncidentMarkerW, IncidentMarkerH, IncidentMarkerW/2, IncidentMarkerH-1, padded(func(c *canvas) { paintIncidentMarker(c, frame) }))
	}
	addTreasureArt(b)
}

func paintRestartBadge(c *canvas, n int) {
	bw, bh := c.w, c.h-3
	c.rect(0, 0, bw, bh, tagRedFill)
	c.rect(0, 0, bw, 1, tagRedEdge)
	c.rect(0, bh-1, bw, 1, tagRedEdge)
	c.rect(0, 0, 1, bh, tagRedEdge)
	c.rect(bw-1, 0, 1, bh, tagRedEdge)
	for i := 0; i < 3; i++ {
		c.rect(bw/2-(2-i), bh+i, 2*(2-i)+1, 1, tagRedEdge)
	}
	glyph(c, restartGlyph, 2, 2, tagGoldEdge)
	digits := strconv.Itoa(n)
	x := 7 + (2-len(digits))*2
	for i, d := range digits {
		glyph(c, digitGlyphs[d-'0'], x+i*4, 2, cream)
	}
	c.outline(ink)
}

type backoffArc struct {
	start, sweep float64
}

func backoffArcs() []backoffArc {
	n := float64(len(BackoffWaits))
	room := 360 - n*backoffGap
	base := room / (1.75 * n)
	grow := 1.5 * base / (n - 1)
	out := make([]backoffArc, 0, len(BackoffWaits))
	at := 0.0
	for i := range BackoffWaits {
		sw := base + grow*float64(i)
		out = append(out, backoffArc{start: at, sweep: sw})
		at += sw + backoffGap
	}
	return out
}

func paintBackoffRing(c *canvas, step int) {
	arcs := backoffArcs()
	cx, cy := float64(c.w)/2, float64(c.h)/2
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			r := math.Hypot(dx, dy)
			if r > cx-1 || r < cx-4.2 {
				continue
			}
			deg := math.Mod(math.Atan2(dx, -dy)*180/math.Pi+360, 360)
			for i, a := range arcs {
				if deg >= a.start && deg < a.start+a.sweep {
					c.dot(x, y, backoffTone(i, step, r < cx-3))
				}
			}
		}
	}
	c.outline(ink)
	c.disc(cx, cy, 1.6, cream)
	hand := arcs[max(min(step, len(arcs))-1, 0)]
	ang := (hand.start + hand.sweep) * math.Pi / 180
	c.line(int(cx), int(cy), int(cx+math.Sin(ang)*4.5), int(cy-math.Cos(ang)*4.5), inkSft)
}

func backoffTone(i, step int, inner bool) color.NRGBA {
	switch {
	case i < step && inner:
		return orangeR[1]
	case i < step:
		return backoffLit
	case inner:
		return slateR[0]
	}
	return slateR[1]
}

func paintIncidentMarker(c *canvas, frame int) {
	cx := float64(c.w) / 2
	radius := []float64{5.5, 6.8, 8.1, 6.8}[frame%IncidentFrames]
	alpha := []uint8{255, 200, 130, 200}[frame%IncidentFrames]
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			d := math.Hypot((float64(x)+0.5-cx)/1, (float64(y)+0.5-9.5)/1.1)
			if math.Abs(d-radius) < 0.7 {
				c.dot(x, y, withAlpha(tagRedEdge, alpha))
			}
		}
	}
	body := newCanvas(c.w, c.h)
	body.shadePoly([]pt{{cx, 1}, {cx + 5.5, 7}, {cx + 5.5, 12}, {cx, 18}, {cx - 5.5, 12}, {cx - 5.5, 7}}, litByX(redR, cx))
	body.rect(int(cx)-1, 4, 2, 7, cream)
	body.rect(int(cx)-1, 13, 2, 2, cream)
	body.rect(int(cx), 18, 1, c.h-19, redR[1])
	body.outline(ink)
	c.paste(body, 0, 0, false)
}
