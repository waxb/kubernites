package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestAnOrderScrollFitsItsContentAndWrapsWithoutALoneWord(t *testing.T) {
	defer ui.SetNamer(nil)
	h, _ := liveHUD(t, true)
	line := "kubectl rollout restart: stamp restartedAt on the pod template so every Pod is replaced."
	h.OpenOrder(OrderForm{ID: 1, Key: "restart", Title: "RESTART", Object: "Deployment market/storefront", Level: LevelSeamless, Lines: []string{line}, Go: "RESTART [Y]"})
	box := h.orderBox()
	if box.H >= OrderBox.H {
		t.Errorf("a one-line ask fills the whole order scroll: %+v", box)
	}
	body := h.orderBody(box)
	if len(body) > 1 {
		last := strings.Fields(body[len(body)-1])
		if len(last) < 2 {
			t.Errorf("the ask's line wraps a lone word onto its last line: %q", body)
		}
	}
	for _, b := range h.orderButtons() {
		if b.Rect.Y+b.Rect.H > box.Y+box.H || b.Rect.Y < box.Y {
			t.Errorf("button %s at %+v lies outside the scroll %+v", b.ID, b.Rect, box)
		}
	}
	h.OpenOrder(OrderForm{ID: 2, Key: "scale", Title: "SCALE", Object: "Deployment market/storefront", Level: LevelWrites, Inputs: []OrderInput{{Label: "REPLICAS", Value: "3"}}, Changes: []OrderChange{{Label: "replicas", Before: "1", After: "3"}}, Go: "CONFIRM [Y]"})
	if full := h.orderBox(); full.H > OrderBox.H || full.Y != OrderBox.Y {
		t.Errorf("an order with inputs and changes leaves its scroll: %+v", full)
	}
}

func TestEveryParagraphOfAnOrderScrollWrapsWithoutALoneWord(t *testing.T) {
	defer ui.SetNamer(nil)
	h, _ := liveHUD(t, true)
	n := ui.MaxChars(OrderBox.W-32, 1)
	paragraph := func(end string) string {
		return strings.Repeat("x", n-len("tail")-1) + " tail " + end
	}
	h.OpenOrder(OrderForm{ID: 1, Key: "drain", Title: "DRAIN", Object: "Node worker", Level: LevelSeamless,
		Verdict:  paragraph("verdictend"),
		Lines:    []string{paragraph("bodyend")},
		Progress: []string{paragraph("progressend")},
		Note:     paragraph("noteend"),
		Stop:     true})
	dst := ebiten.NewImage(Width, Height)
	runs := recordRuns(func() { h.drawOrderScroll(dst, Frame{}) })
	for _, end := range []string{"verdictend", "bodyend", "progressend", "noteend"} {
		found := false
		for _, r := range runs {
			if strings.Contains(r.Text, end) {
				found = true
				if len(strings.Fields(r.Text)) < 2 {
					t.Errorf("the paragraph ending %s wraps its last word alone: %q", end, r.Text)
				}
			}
		}
		if !found {
			t.Errorf("the paragraph ending %s is not drawn", end)
		}
	}
}
