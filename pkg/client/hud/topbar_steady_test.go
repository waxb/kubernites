package hud

import (
	"math"
	"testing"
)

func TestTheTopBarsCountersAndClockTickAtMostOnceASecond(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	shown, at, changes := map[string]string{}, map[string]float64{}, map[string]int{}
	note := func(key, value string) {
		prev, seen := shown[key]
		switch {
		case !seen:
		case prev == value:
			return
		case h.Now-at[key] < CounterTick-1e-6:
			t.Fatalf("the top bar's %s changed %.2f s after its last change (%q to %q), under %.0f s", key, h.Now-at[key], prev, value, CounterTick)
		default:
			changes[key]++
		}
		shown[key], at[key] = value, h.Now
	}
	for range 80 {
		f.State.Gold += .37
		f.State.Pilgrims.DayPhase = math.Mod(f.State.Pilgrims.DayPhase+.01, 1)
		tick(h, &f, .1)
		s := h.Stage(f)
		for _, st := range h.TopStats(s) {
			note("stat "+st.Meter+st.icon, st.Text+" "+st.Rate+" "+st.Upkeep+" "+st.Net+" "+st.Floor)
		}
		clock, _ := h.ClockSlot(s)
		note("clock", clock)
	}
	if changes["stat "+MeterGold+"gold"] == 0 || changes["clock"] == 0 {
		t.Fatalf("the top bar never followed the realm: %v", changes)
	}
}

func TestThePlayersOwnSpendShowsOnThePurseAtOnce(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f.State.Gold = 40
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	f.State.Gold += .5
	tick(h, &f, .1)
	f.State.Gold -= 30
	f.Clock += .1
	h.Update(Input{MX: -10, MY: -10, Click: true}, f)
	for _, st := range h.TopStats(h.Stage(f)) {
		if st.Meter == MeterGold && st.Text != FitAmount(f.State.Gold, GoldChars) {
			t.Fatalf("the counter still says %q a frame after the player spent down to %s", st.Text, FitAmount(f.State.Gold, GoldChars))
		}
	}
}
