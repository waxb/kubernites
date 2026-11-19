package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

var (
	turnEdge = [bookTurnFrames]float64{1, .97, .85, .62, .35, .12, -.15, -.45, -.75, -.95}
	turnLift = [bookTurnFrames]int{0, 2, 5, 8, 10, 11, 11, 9, 5, 2}
)

const (
	turnFold    = 4
	turnShadow  = 6
	turnCurlH   = 4
	turnBorder  = 8
	turnMinBody = 40
)

func (h *HUD) TurnFrame() (int, bool) {
	t := h.Book.Turn
	if t.Dir == 0 || h.Reduce || h.Now < t.at {
		return 0, false
	}
	frame := int((h.Now - t.at) / bookTurnStep)
	return frame, frame >= 1 && frame < bookTurnFrames
}

func snap(v int) int { return v / bookPx * bookPx }

func SpineX() int { return ArtX(artSpineL+2) - 1 }

func (h *HUD) drawTurn(dst *ebiten.Image) {
	frame, ok := h.TurnFrame()
	if !ok {
		return
	}
	half := float64(SpineX() - bookOuter(0))
	edge := SpineX() + snap(int(turnEdge[frame]*half*float64(h.Book.Turn.Dir)))
	lo, hi := min(edge, SpineX()), max(edge, SpineX())
	if hi-lo < bookPx {
		return
	}
	lift := turnLift[frame] * bookPx
	side := 1
	if edge < SpineX() {
		side = -1
	}
	height := pageBottomY() - PageTopY()
	shadow := ui.Rect{X: edge, Y: PageTopY(), W: turnShadow, H: height}
	if side < 0 {
		shadow.X = edge - turnShadow
	}
	ui.Fill(dst, shadow, ui.Tint(bookOutline, 50))
	for x := lo; x < hi; x += bookPx {
		h.drawLeafColumn(dst, x, float64(iabs(x-SpineX()))/float64(hi-lo), lift, hi-lo > turnMinBody && iabs(x-edge) > turnBorder && iabs(x-SpineX()) > turnBorder)
	}
	fold := ui.Rect{X: edge - turnFold, Y: PageTopY() - lift, W: turnFold, H: height + lift/2}
	if side < 0 {
		fold.X = edge
	}
	ui.Fill(dst, fold, ui.Tint(bookEdgeDark, 170))
}

func (h *HUD) drawLeafColumn(dst *ebiten.Image, x int, k float64, lift int, ruled bool) {
	top := PageTopY() - snap(int(float64(lift)*(.35+.65*k)))
	bottom := pageBottomY() - snap(int(float64(lift)*.5*k*k))
	ui.Fill(dst, ui.Rect{X: x, Y: top, W: bookPx, H: bottom - top}, bookPaper)
	ui.Fill(dst, ui.Rect{X: x, Y: top, W: bookPx, H: bookPx}, bookEdgeDark)
	ui.Fill(dst, ui.Rect{X: x, Y: bottom - bookPx, W: bookPx, H: bookPx}, bookEdgeLight)
	if k < .4 {
		ui.Fill(dst, ui.Rect{X: x, Y: bottom - turnCurlH*bookPx, W: bookPx, H: (turnCurlH - 1) * bookPx}, bookEdgeLight)
	}
	if ruled {
		ui.Fill(dst, ui.Rect{X: x, Y: top + turnBorder, W: bookPx, H: bookPx / 2}, bookRule)
		ui.Fill(dst, ui.Rect{X: x, Y: bottom - turnBorder - turnCurlH*bookPx, W: bookPx, H: bookPx / 2}, bookRule)
	}
}
