package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/termview"
	"github.com/waxb/kubernites/pkg/client/ui"
)

type clusterScreen struct{ cell string }

func (clusterScreen) Live() bool               { return true }
func (clusterScreen) Resize(int, int)          {}
func (clusterScreen) Cursor() (int, int, bool) { return 0, 0, false }
func (clusterScreen) Key(termview.Key)         {}
func (clusterScreen) Type(string)              {}
func (s clusterScreen) Row(int) []termview.Cell {
	out := make([]termview.Cell, 400)
	for i := range out {
		out[i].Text = s.cell
	}
	return out
}

func TestTheStonesTerminalDrawsOneGlyphPerCell(t *testing.T) {
	h := New(nil, Backend{})
	cols, rows := h.TerminalCells()
	dst := ebiten.NewImage(Width, Height)
	for _, cell := range []string{"\u0436" + strings.Repeat("\u0301", 31), strings.Repeat("\U0001F468\u200d", 16), "\u0d4e\u0d4e\u0d4e"} {
		drawn := 0
		ui.WatchText(func(text string) {
			for _, r := range text {
				if strings.ContainsRune(cell, r) {
					drawn++
				}
			}
		})
		h.drawTerminal(dst, clusterScreen{cell: cell})
		ui.WatchText(nil)
		if drawn > cols*rows {
			t.Errorf("a screen of %dx%d cells, each holding %q, drew %d glyphs", cols, rows, cell, drawn)
		}
	}
}
