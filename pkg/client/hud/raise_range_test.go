package hud

import (
	"math"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestANewCrewStartsWithAtLeastOneWorker(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := chooserFrame(t)
	openChooser(t, h, f, "clerks")
	clickOn(t, h, f, raiseKey+":workers:down")
	clickOn(t, h, f, raiseKey+":workers:down")
	r, _ := h.Ops.Raise.row(creatableRows(RaiseRows(f)))
	if got := h.Ops.Raise.workersOf(r); got != 1 {
		t.Errorf("a new crew starts with at least one worker, got %d", got)
	}
}

func TestSignedGoldNeverPrintsMinusZero(t *testing.T) {
	defer ui.SetNamer(nil)
	cases := []struct {
		v            float64
		story, names string
	}{
		{math.Copysign(0, -1), "+0 GOLD/S", "+$0/S"},
		{0, "+0 GOLD/S", "+$0/S"},
		{-0.001, "+0 GOLD/S", "+$0/S"},
		{0.004, "+0 GOLD/S", "+$0/S"},
		{-0.4, "-0.4 GOLD/S", "-$0.4/S"},
		{2.5, "+2.5 GOLD/S", "+$2.5/S"},
		{-1234, "-1.23K GOLD/S", "-$1.23K/S"},
	}
	for _, c := range cases {
		ui.SetNamer(nil)
		if got := GoldPerSecond(c.v); got != c.story {
			t.Errorf("story %v = %q, want %q", c.v, got, c.story)
		}
		ui.SetNamer(names.Kubernetes(64))
		if got := GoldPerSecond(c.v); got != c.names {
			t.Errorf("names %v = %q, want %q", c.v, got, c.names)
		}
	}
	ui.SetNamer(nil)
	for v, want := range map[float64]string{math.Copysign(0, -1): "+0", -0.001: "+0", -2: "-2", 3.25: "+3.3"} {
		if got := SignedMoney(v); got != want {
			t.Errorf("SignedMoney(%v) = %q, want %q", v, got, want)
		}
	}
}
