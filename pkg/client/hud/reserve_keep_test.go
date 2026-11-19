package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheReservesKeepSaysWhoPaysForIt(t *testing.T) {
	f := fullFrame(t)
	b := view.Buildable{Kind: "keep", Name: "Stone Keep", Cost: 0, Reserve: true, Available: true}
	btn := KeepButton(b, f)
	if btn.Cost != reserveCost {
		t.Fatalf("the reserve's Keep is priced %q, want %q", btn.Cost, reserveCost)
	}
	if !strings.Contains(btn.Tip, "reserve") || strings.Contains(btn.Tip, buildPrice(b)) {
		t.Fatalf("the reserve's Keep tip reads %q", btn.Tip)
	}
	b.Reserve, b.Cost = false, 11000
	if btn := KeepButton(b, f); btn.Cost == reserveCost {
		t.Fatal("a bought Keep reads as the reserve's")
	}
}
