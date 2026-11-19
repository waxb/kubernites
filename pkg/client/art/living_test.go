package art

import (
	"image"
	"testing"
)

func solidCount(img *image.NRGBA) int {
	n := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] > 0 {
			n++
		}
	}
	return n
}

func pixelsDiffer(a, b *image.NRGBA) bool {
	if !a.Rect.Eq(b.Rect) {
		return true
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return true
		}
	}
	return false
}

func TestLivingPiecesPaint(t *testing.T) {
	cases := map[string]Piece{"scaffold": PaintScaffold(), "chicken": PaintChicken(0), "goat": PaintGoat(0), "gull": PaintGull(0), "board": PaintBoard(3, 0)}
	for name, p := range cases {
		if solidCount(p.Img) < 20 {
			t.Errorf("%s paints almost nothing", name)
		}
	}
}

func TestLivingStagesDiffer(t *testing.T) {
	stages := map[string]struct {
		paint func(int) Piece
		n     int
	}{
		"wheat":  {func(l int) Piece { return PaintWheat(l, 0) }, WheatLevels},
		"bush":   {func(l int) Piece { return PaintBush(l, 0) }, BushFruits},
		"cairn":  {func(l int) Piece { return PaintCairn(l, 0) }, CairnStages},
		"rubble": {func(l int) Piece { return PaintRubble(l, 0) }, RubbleStages},
		"board":  {func(l int) Piece { return PaintBoard(l, 0) }, BoardSlots + 1},
	}
	for name, s := range stages {
		for l := 1; l < s.n; l++ {
			if !pixelsDiffer(s.paint(l-1).Img, s.paint(l).Img) {
				t.Errorf("%s stage %d looks like stage %d", name, l, l-1)
			}
		}
	}
}

func TestLivestockAnimates(t *testing.T) {
	if !pixelsDiffer(PaintChicken(0).Img, PaintChicken(1).Img) || !pixelsDiffer(PaintGoat(0).Img, PaintGoat(1).Img) {
		t.Error("livestock frames do not move")
	}
}
