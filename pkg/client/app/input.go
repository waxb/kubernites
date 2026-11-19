package app

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/waxb/kubernites/pkg/client/hud"
)

const (
	doubleClickSeconds = 0.35
	repeatDelay        = 24
	repeatEvery        = 3
	panSpeed           = 420.0
	edgeMargin         = 6
)

var repeatable = []ebiten.Key{
	ebiten.KeyBackspace, ebiten.KeyDelete, ebiten.KeyArrowLeft, ebiten.KeyArrowRight,
	ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyPageUp, ebiten.KeyPageDown,
}

type Pointer struct {
	lastClick  float64
	lastX      int
	lastY      int
	Dragging   bool `json:"-"`
	Grabbed    bool `json:"-"`
	dragX      int
	dragY      int
	Edge       bool `json:"-"`
	toLogical  func(x, y float64) (int, int)
	keysBuffer []ebiten.Key
}

func (p *Pointer) Gather(now float64) hud.Input {
	cx, cy := ebiten.CursorPositionF()
	mx, my := int(math.Floor(cx)), int(math.Floor(cy))
	if p.toLogical != nil {
		mx, my = p.toLogical(cx, cy)
	}
	in := hud.Input{MX: mx, MY: my}
	in.Click = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	in.RightClick = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	in.Held = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	in.Shift = ebiten.IsKeyPressed(ebiten.KeyShift)
	in.Ctrl = ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	in.Control = ebiten.IsKeyPressed(ebiten.KeyControl)
	in.Alt = ebiten.IsKeyPressed(ebiten.KeyAlt)
	in.WheelX, in.Wheel = ebiten.Wheel()
	in.Chars = ebiten.AppendInputChars(nil)
	p.keysBuffer = inpututil.AppendJustPressedKeys(p.keysBuffer[:0])
	in.Keys = append([]ebiten.Key(nil), p.keysBuffer...)
	for _, k := range repeatable {
		if d := inpututil.KeyPressDuration(k); d > repeatDelay && d%repeatEvery == 0 {
			in.Keys = append(in.Keys, k)
		}
	}
	if in.Click {
		in.DoubleClick = now-p.lastClick < doubleClickSeconds && Abs(mx-p.lastX) < 5 && Abs(my-p.lastY) < 5
		p.lastClick, p.lastX, p.lastY = now, mx, my
	}
	return in
}

func Abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

var HeldKey = ebiten.IsKeyPressed

func panKey(arrow, letter ebiten.Key) bool {
	return HeldKey(arrow) || HeldKey(letter)
}

func PanVector(dt float64) (float64, float64) {
	dx, dy := 0.0, 0.0
	if HeldKey(ebiten.KeyControl) {
		return 0, 0
	}
	if panKey(ebiten.KeyArrowLeft, ebiten.KeyA) {
		dx--
	}
	if panKey(ebiten.KeyArrowRight, ebiten.KeyD) {
		dx++
	}
	if panKey(ebiten.KeyArrowUp, ebiten.KeyW) {
		dy--
	}
	if panKey(ebiten.KeyArrowDown, ebiten.KeyS) {
		dy++
	}
	if dx != 0 && dy != 0 {
		dx, dy = dx/math.Sqrt2, dy/math.Sqrt2
	}
	return dx * panSpeed * dt, dy * panSpeed * dt
}

func (p *Pointer) Grab(mx, my int) {
	p.Grabbed, p.Dragging, p.dragX, p.dragY = true, true, mx, my
}

func (p *Pointer) drag(mx, my int, held bool) (float64, float64) {
	p.Grabbed = p.Grabbed && held
	if !p.Grabbed && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
		p.Dragging = false
		return 0, 0
	}
	if !p.Dragging {
		p.Dragging, p.dragX, p.dragY = true, mx, my
		return 0, 0
	}
	dx, dy := float64(p.dragX-mx), float64(p.dragY-my)
	p.dragX, p.dragY = mx, my
	return dx, dy
}

func EdgePan(mx, my int, dt float64) (float64, float64) {
	if mx < 0 || my < 0 || mx >= hud.Width || my >= hud.Height {
		return 0, 0
	}
	dx, dy := 0.0, 0.0
	switch {
	case mx < edgeMargin:
		dx = -1
	case mx >= hud.Width-edgeMargin:
		dx = 1
	}
	switch {
	case my < edgeMargin:
		dy = -1
	case my >= hud.Height-edgeMargin:
		dy = 1
	}
	if dx != 0 && dy != 0 {
		dx, dy = dx/math.Sqrt2, dy/math.Sqrt2
	}
	return dx * panSpeed * dt, dy * panSpeed * dt
}

func (p *Pointer) EdgePan(mx, my int, dt float64) (float64, float64) {
	if !p.Edge || !ebiten.IsFocused() {
		return 0, 0
	}
	return EdgePan(mx, my, dt)
}

func ZoomKey(in hud.Input) int {
	switch {
	case in.Pressed(ebiten.KeyEqual) || in.Pressed(ebiten.KeyKPAdd) || in.Pressed(ebiten.KeyPageUp):
		return 1
	case in.Pressed(ebiten.KeyMinus) || in.Pressed(ebiten.KeyKPSubtract) || in.Pressed(ebiten.KeyPageDown):
		return -1
	}
	return 0
}
