package render

import (
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestProjectionRoundTrip(t *testing.T) {
	c := NewCamera(480, 40)
	for _, zoom := range ZoomLevels {
		c.Zoom = zoom
		for _, p := range [][2]float64{{0, 0}, {3, 7}, {12.5, 4.25}} {
			sx, sy := c.GridToScreen(p[0], p[1])
			gx, gy := c.ScreenToGrid(sx, sy)
			if !near(gx, p[0]) || !near(gy, p[1]) {
				t.Fatalf("zoom %v round trip %v -> %v,%v", zoom, p, gx, gy)
			}
		}
	}
	sx, sy := c.GridToScreen(0, 0)
	ex, ey := c.GridToScreenElevated(0, 0, 2)
	if ex != sx || !near(sy-ey, 2*StepHeight*c.Scale()) {
		t.Fatal("elevation offset wrong")
	}
}

func TestTileAtAndPan(t *testing.T) {
	c := NewCamera(480, 40)
	sx, sy := c.GridToScreen(5, 9)
	if x, y := c.TileAt(int(sx), int(sy)); x != 5 || y != 9 {
		t.Fatalf("tile = %d,%d", x, y)
	}
	p := c.Pan(10, -4)
	if p.X != 10 || p.Y != -4 || c.X != 0 {
		t.Fatal("pan should return a new camera")
	}
}

func TestZoomKeepsCursorAnchored(t *testing.T) {
	c := NewCamera(480, 40)
	gx, gy := c.ScreenToGrid(300, 200)
	z := c.ZoomStep(1, 300, 200)
	if z.Scale() != 2 {
		t.Fatalf("zoom = %v", z.Scale())
	}
	ax, ay := z.ScreenToGrid(300, 200)
	if !near(ax, gx) || !near(ay, gy) {
		t.Fatal("zoom moved the point under the cursor")
	}
	if z.ZoomStep(5, 0, 0).Scale() != 2 || c.ZoomStep(-5, 0, 0).Scale() != .5 {
		t.Fatal("zoom should clamp")
	}
	c.Zoom = 99
	if c.Scale() != 2 {
		t.Fatal("scale clamp")
	}
}

func TestCenterAndClamp(t *testing.T) {
	c := NewCamera(0, 0).CenterOn(24, 24, 960, 540)
	sx, sy := c.GridToScreen(24, 24)
	if !near(sx, 480) || !near(sy, 270) {
		t.Fatalf("center = %v,%v", sx, sy)
	}
	far := c.Pan(1e6, 1e6).Clamp(48, 960, 540)
	fx, fy := far.ScreenToGrid(480, 270)
	if fx > 60 || fy > 60 {
		t.Fatalf("clamp failed: %v,%v", fx, fy)
	}
	if DepthKey(1, 2) <= DepthKey(2, 0) {
		t.Fatal("depth key should order by x+y then y")
	}
}

func TestDrawAtSmoke(t *testing.T) {
	dst := ebiten.NewImage(32, 32)
	sheet := ebiten.NewImage(16, 16)
	r := Region{0, 0, 8, 8, 4, 8}
	DrawAt(dst, sheet, r, 10, 10, 1, SpriteOptions{})
	DrawAt(dst, sheet, r, 10, 10, 2, SpriteOptions{Flip: true, Alpha: .5, Tint: [3]float32{1, .5, .5}, Scale: 1})
	DrawAt(dst, nil, r, 0, 0, 1, SpriteOptions{})
	if r.Rect().Dx() != 8 {
		t.Fatal("rect wrong")
	}
}

func TestSolidColorPaintsEveryOpaquePixel(t *testing.T) {
	cm := SolidColor(color.RGBA{255, 218, 105, 255})
	got := cm.Apply(color.RGBA{10, 20, 30, 255})
	r, g, b, a := got.RGBA()
	if r>>8 != 255 || g>>8 != 218 || b>>8 != 105 || a>>8 != 255 {
		t.Errorf("solid = %v", got)
	}
	sheet := ebiten.NewImage(8, 8)
	sheet.Fill(color.White)
	dst := ebiten.NewImage(32, 32)
	OutlineAt(dst, sheet, Region{W: 4, H: 4, AnchorX: 2, AnchorY: 4}, 16, 16, 1, color.RGBA{255, 0, 0, 255})
	OutlineAt(dst, nil, Region{}, 0, 0, 1, color.RGBA{})
	halfM := SolidColor(color.RGBA{0, 0, 0, 128})
	if _, _, _, a := halfM.Apply(color.RGBA{255, 255, 255, 255}).RGBA(); a>>8 < 126 || a>>8 > 130 {
		t.Errorf("alpha is kept: %d", a>>8)
	}
}

func TestFlippedOutlinesMirrorAboutTheAnchor(t *testing.T) {
	r := Region{W: 10, H: 4, AnchorX: 3, AnchorY: 4}
	if got := SpriteLeft(r, 20, 2, false); got != 14 {
		t.Errorf("left edge = %v, want 14", got)
	}
	if got := SpriteLeft(r, 20, 2, true); got != 6 {
		t.Errorf("flipped left edge = %v, want 6 (the anchor column keeps its place)", got)
	}
	sheet := ebiten.NewImage(10, 4)
	sheet.Fill(color.White)
	dst := ebiten.NewImage(32, 32)
	OutlineFlipAt(dst, sheet, r, 16, 16, 1, color.RGBA{255, 0, 0, 255}, true)
	OutlineFlipAt(dst, nil, r, 16, 16, 1, color.RGBA{}, true)
}
