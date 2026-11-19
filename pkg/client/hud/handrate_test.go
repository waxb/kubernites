package hud

import (
	"strings"

	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheTopBarRateLeavesOutGoldTheHandAlreadyBanked(t *testing.T) {
	st := view.GameState{}
	st.Gold, st.GoldPerSec = 5.8, 0.96
	st.Economy.Hand = 0.96
	if got := NetGoldRate(st); got > 0.001 {
		t.Fatalf("the bar shows +%.2f a second while the balance stands still: the carried boxes are counted already", got)
	}
	if rate := GoldStat(Frame{State: st}).Rate; rate != "" {
		t.Fatalf("the balance stat prints %q beside a balance that does not move", rate)
	}
	st.Economy.Income, st.Economy.Hand = 1.5, 0.5
	if got := IncomeRate(st); got < 0.99 || got > 1.01 {
		t.Fatalf("rate %.2f, want 1.0 without the hand's 0.5", got)
	}
}

func TestEnterBeforeTheStoneExistsSaysTheStoneSleeps(t *testing.T) {
	h := New(nil, Backend{})
	f := Frame{}
	out := h.Update(Input{MX: -5, MY: -5, Keys: []ebiten.Key{ebiten.KeyEnter}}, f)
	_ = out
	got := ""
	for _, ts := range h.Toasts() {
		got += ts.Text
	}
	if !strings.Contains(got, "Stone sleeps") {
		t.Fatalf("Enter before the Stone exists gave %q", got)
	}
}
