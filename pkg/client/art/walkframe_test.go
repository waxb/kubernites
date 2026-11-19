package art

import (
	"image"
	"image/color"
	"strings"
	"sync"
	"testing"
)

var (
	framedOnce sync.Once
	framedSet  Atlas
)

func framedWalkers() Atlas {
	framedOnce.Do(func() { framedSet = reframeWalkers(built().Walkers) })
	return framedSet
}

func frameAnchorX(name string) int {
	if strings.Contains(name, "/sw/") {
		return walkerFrameW - walkerFrameAX
	}
	return walkerFrameAX
}

func reframeWalkers(a Atlas) Atlas {
	names := []string{}
	for _, n := range Names(a) {
		if strings.HasPrefix(n, "walker/") {
			names = append(names, n)
		}
	}
	cols := atlasWidth / walkerFrameW
	rows := (len(names) + cols - 1) / cols
	out := Atlas{Image: image.NewNRGBA(image.Rect(0, 0, cols*walkerFrameW, rows*walkerFrameH)), Regions: map[string]Region{}}
	for i, n := range names {
		r := a.Regions[n]
		f := Region{X: i % cols * walkerFrameW, Y: i / cols * walkerFrameH, W: walkerFrameW, H: walkerFrameH, AnchorX: frameAnchorX(n), AnchorY: walkerFrameAY}
		dx, dy := f.X+f.AnchorX-r.AnchorX, f.Y+f.AnchorY-r.AnchorY
		for y := 0; y < r.H; y++ {
			for x := 0; x < r.W; x++ {
				out.Image.SetNRGBA(dx+x, dy+y, a.Image.NRGBAAt(r.X+x, r.Y+y))
			}
		}
		out.Regions[n] = f
	}
	return out
}

func TestFramedCanvasDrawsAtItsOrigin(t *testing.T) {
	c := newFramedCanvas(6, 5, 2, 1)
	red := color.NRGBA{255, 0, 0, 255}
	c.dot(-2, -1, red)
	c.dot(0, 0, red)
	c.dot(4, 4, red)
	if c.img.NRGBAAt(0, 0) != red || c.img.NRGBAAt(2, 1) != red {
		t.Fatal("logical points land at their origin offset")
	}
	if c.at(0, 0) != red || c.at(-2, -1) != red || c.at(9, 9) != (color.NRGBA{}) {
		t.Fatal("at reads through the origin")
	}
	if c.img.NRGBAAt(5, 4) == red {
		t.Fatal("a point past the canvas is dropped")
	}
}

func TestTrimmedEntryFitsItsArtAndAnchor(t *testing.T) {
	red := color.NRGBA{255, 0, 0, 255}
	e := entry{name: "walker/x", w: 20, h: 20, ax: 10, ay: 18, paint: func(c *canvas) { c.rect(6, 4, 3, 2, red) }}
	got := trimmedEntry(e)
	if got.w != 6 || got.h != 16 || got.ax != 5 || got.ay != 15 {
		t.Fatalf("trimmed to %dx%d anchored %d,%d", got.w, got.h, got.ax, got.ay)
	}
	c := newCanvas(got.w, got.h)
	got.paint(c)
	if c.img.NRGBAAt(1, 1) != red || c.img.NRGBAAt(0, 0).A != 0 || c.img.NRGBAAt(3, 2) != red {
		t.Fatal("the trimmed paint keeps the art one pixel in from the edge")
	}
	blank := entry{name: "walker/y", w: 4, h: 4, ax: 2, ay: 3, paint: func(*canvas) {}}
	if b := trimmedEntry(blank); b.w != 4 || b.h != 4 || b.ax != 2 {
		t.Fatalf("a blank sprite keeps its canvas: %+v", b)
	}
}

func TestWalkersAreTrimmedAroundTheirArt(t *testing.T) {
	a := built().Walkers
	for n, r := range a.Regions {
		if !strings.HasPrefix(n, "walker/") {
			continue
		}
		if r.W > walkerFrameW || r.H > walkerFrameH {
			t.Fatalf("%s outgrew the walker frame: %+v", n, r)
		}
		b := contentBounds(a, r)
		loose := b.Min.X > walkerMargin && r.AnchorX != 0 || r.W-b.Max.X > walkerMargin && r.AnchorX != r.W-1 ||
			b.Min.Y > walkerMargin && r.AnchorY != 0 || r.H-b.Max.Y > walkerMargin && r.AnchorY != r.H-1
		if loose {
			t.Fatalf("%s keeps a loose border: region %+v, art %v", n, r, b)
		}
	}
}

func TestWalkerDirectionsShareOneFrame(t *testing.T) {
	a := built().Walkers
	for _, role := range append(append([]string{}, WalkerRoles...), CritterRoles...) {
		se, sw := a.Regions[walkerName(role, "walk", "se", 1)], a.Regions[walkerName(role, "walk", "sw", 1)]
		if se.W != sw.W || se.H != sw.H || sw.AnchorX != sw.W-se.AnchorX || sw.AnchorY != se.AnchorY {
			t.Errorf("%s: se %+v and sw %+v are not mirrors about the feet", role, se, sw)
		}
	}
}

func TestFramedWalkersKeepTheOldFeet(t *testing.T) {
	a := framedWalkers()
	for _, role := range WalkerRoles {
		r := a.Regions[walkerName(role, "idle", "se", 0)]
		if r.W != walkerFrameW || r.AnchorX != walkerFeetX+walkerPadX || r.AnchorY != walkerFeetY+walkerPadTop {
			t.Fatalf("%s framed region %+v", role, r)
		}
		if b := solidBounds(a, r); b.Max.Y-1 > r.AnchorY {
			t.Errorf("%s: feet at row %d below the anchor %d", role, b.Max.Y-1, r.AnchorY)
		}
	}
}
