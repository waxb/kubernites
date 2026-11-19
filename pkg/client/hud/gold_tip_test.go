package hud

import (
	"strings"
	"testing"
)

func TestTheGoldTipSaysWhatItsRateIsAndItsBreakdownAddsUp(t *testing.T) {
	f := fullFrame(t)
	f.State.Gold, f.State.GoldPerSec = 1.9, .12
	f.State.Economy.Income, f.State.Economy.Profit, f.State.Economy.Upkeep, f.State.Economy.Hand = .12, .12, 0, 0
	f.State.Economy.Carried, f.State.Freight.FreshPerSec = .1, .01
	tip := GoldTip(f)
	lines := strings.Split(tip, "\n")
	if len(lines) < 2 || !strings.Contains(lines[1], "/S") || !strings.Contains(tip, "by hand") {
		t.Errorf("+/S is explained only minutes later; the rate tip reads:\n%s", tip)
	}
	if strings.Contains(tip, "0.01 each second") || !strings.Contains(tip, "0.10 boxes") || !strings.Contains(tip, "1.2 gold each") {
		t.Errorf("the breakdown must add up to the rate:\n%s", tip)
	}
}
