package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestCoinPolicyFliesTheOpeningThenOnlyNotablePayments(t *testing.T) {
	cases := []struct {
		name   string
		policy CoinPolicy
		value  float64
		golden bool
		want   bool
	}{
		{"opening", CoinPolicy{Mode: CoinFlightsFew, Flown: 0, Income: 100}, 1, false, true},
		{"last of the opening", CoinPolicy{Mode: CoinFlightsFew, Flown: coinOpening - 1, Income: 100}, 1, false, true},
		{"small after the opening", CoinPolicy{Mode: CoinFlightsFew, Flown: coinOpening, Income: 100}, 999, false, false},
		{"ten seconds of throughput", CoinPolicy{Mode: CoinFlightsFew, Flown: coinOpening, Income: 100}, coinNotable * 100, false, true},
		{"flagged flight", CoinPolicy{Mode: CoinFlightsFew, Flown: coinOpening, Income: 100}, 1, true, true},
		{"no throughput yet", CoinPolicy{Mode: CoinFlightsFew, Flown: coinOpening, Income: 0}, 1, false, true},
		{"overhead exceeds throughput", CoinPolicy{Mode: CoinFlightsFew, Flown: coinOpening, Income: -4}, 1, false, true},
		{"missing mode is few", CoinPolicy{Flown: coinOpening, Income: 100}, 1, false, false},
		{"unknown mode is few", CoinPolicy{Mode: "loud", Flown: coinOpening, Income: 100}, 1, false, false},
		{"all flies everything", CoinPolicy{Mode: CoinFlightsAll, Flown: 5000, Income: 100}, 1, false, true},
		{"off flies nothing", CoinPolicy{Mode: CoinFlightsOff, Flown: 0, Income: 0}, 1e6, true, false},
	}
	for _, c := range cases {
		if got := c.policy.Flies(c.value, c.golden); got != c.want {
			t.Errorf("%s: flies = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCoinPolicyStreamsOnlyWhileTreasuryCoinsFly(t *testing.T) {
	cases := []struct {
		policy CoinPolicy
		want   float64
	}{
		{CoinPolicy{Mode: CoinFlightsFew}, coinGap},
		{CoinPolicy{Mode: CoinFlightsFew, Flown: coinOpening}, coinBeat},
		{CoinPolicy{Mode: CoinFlightsAll, Flown: 5000}, coinGap},
		{CoinPolicy{Mode: CoinFlightsOff}, coinBeat},
	}
	for _, c := range cases {
		if got := c.policy.TreasuryGap(); got != c.want {
			t.Errorf("%+v: gap = %v, want %v", c.policy, got, c.want)
		}
	}
	if coinBeat <= coinPlus {
		t.Errorf("a counter beat every %vs would overlap its %vs +N", coinBeat, coinPlus)
	}
}

func TestCoinFlightModesCycleAndName(t *testing.T) {
	cycle := map[string]string{CoinFlightsAll: CoinFlightsFew, CoinFlightsFew: CoinFlightsOff, CoinFlightsOff: CoinFlightsAll, "": CoinFlightsOff}
	for from, want := range cycle {
		if got := NextCoinFlights(from); got != want {
			t.Errorf("next(%q) = %q, want %q", from, got, want)
		}
	}
	labels := map[string]string{CoinFlightsAll: "COIN FLIGHTS: ALL", CoinFlightsFew: "COIN FLIGHTS: FEW", CoinFlightsOff: "COIN FLIGHTS: OFF", "": "COIN FLIGHTS: FEW"}
	for mode, want := range labels {
		if got := CoinFlightsLabel(mode); got != want {
			t.Errorf("label(%q) = %q, want %q", mode, got, want)
		}
	}
}

func TestPurseLabelSpeaksMoney(t *testing.T) {
	cases := []struct {
		v     float64
		names bool
		want  string
	}{{12, false, "+12"}, {12, true, "+$12"}, {1500, true, "+$1.5K"}, {2.5, false, "+2.5"}}
	for _, c := range cases {
		if got := PurseLabel(c.v, c.names); got != c.want {
			t.Errorf("label(%v, %v) = %q, want %q", c.v, c.names, got, c.want)
		}
	}
}

func pursedHUD(t *testing.T, mode string) (*HUD, Frame) {
	t.Helper()
	h := New(nil, Backend{})
	h.SetCoinFlights(mode)
	f := fullFrame(t)
	f.State.GoldPerSec = 10
	h.Update(Input{}, f)
	for i := 0; i < coinOpening; i++ {
		h.LaunchCoin(400, 300, 1)
	}
	return h, f
}

func TestSmallPaymentsPulseThePurseAfterTheOpening(t *testing.T) {
	h, f := pursedHUD(t, CoinFlightsFew)
	if got := h.CoinsInFlight(); got != min(coinOpening, maxFlights) {
		t.Fatalf("the opening flies %d dots", got)
	}
	f.Clock += coinFlight + .01
	h.Update(Input{}, f)
	f.Clock += coinPlus + .5
	h.Update(Input{}, f)
	h.LaunchHandCoin(400, 300, 3)
	if h.CoinsInFlight() != 0 || h.Coins.Landed != 3 || h.GoldPulse() <= 0 {
		t.Fatalf("a small payment should pulse the counter, not fly: %+v", h.Coins)
	}
	h.LaunchCoin(400, 300, coinNotable*10)
	h.LaunchGoldenCoin(400, 300, 1)
	if h.CoinsInFlight() != 2 {
		t.Fatalf("notable and flagged payments fly: %d", h.CoinsInFlight())
	}
	if h.TreasuryGap() != coinBeat {
		t.Error("after the opening the counter beats slowly")
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
}

func TestCoinFlightsOffAndAll(t *testing.T) {
	off, _ := pursedHUD(t, CoinFlightsOff)
	off.LaunchGoldenCoin(400, 300, 500)
	if off.CoinsInFlight() != 0 || off.Coins.Landed != coinOpening+500 {
		t.Fatalf("OFF never flies: %d in flight, landed %v", off.CoinsInFlight(), off.Coins.Landed)
	}
	all, f := pursedHUD(t, CoinFlightsAll)
	f.Clock += coinFlight + .01
	all.Update(Input{}, f)
	all.LaunchCoin(400, 300, 1)
	if all.CoinsInFlight() != 1 || all.TreasuryGap() != coinGap {
		t.Fatal("ALL flies every payment")
	}
	if all.CoinFlights() != CoinFlightsAll || New(nil, Backend{}).CoinFlights() != CoinFlightsFew {
		t.Error("a new HUD defaults to FEW")
	}
}

func TestPurseTallyStaysBoundedUnderAStream(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	most := 0.0
	for i := 0; i < 80; i++ {
		f.Clock = float64(i) * coinGap
		h.Update(Input{}, f)
		h.land(1)
		most = max(most, h.Coins.Landed)
	}
	if limit := coinPlus/coinGap + 2; most > limit {
		t.Fatalf("the +N tally grows without bound: %v > %v", most, limit)
	}
}

func TestMenuOffersCoinFlights(t *testing.T) {
	f := fullFrame(t)
	h := New(nil, Backend{})
	h.Open(OverlayMenu)
	for _, mode := range []string{CoinFlightsAll, CoinFlightsFew, CoinFlightsOff} {
		h.SetCoinFlights(mode)
		b := buttonByID(h.MenuButtons(f), "menu:coins")
		if b.Label != CoinFlightsLabel(mode) || !MenuBox.Contains(b.Rect.X+b.Rect.W-1, b.Rect.Y+b.Rect.H-1) {
			t.Fatalf("flights button = %+v", b)
		}
		if w, _ := h.Font.Measure(b.Label, 1); w > b.Rect.W-8 {
			t.Errorf("%q does not fit its button", b.Label)
		}
	}
	if _, ok := hasIntent(clickOn(t, h, f, "menu:coins"), IntentCoinFlights); !ok {
		t.Fatal("the flights button does not ask to change the setting")
	}
}

func TestPurseTabHangsUnderThePurseClearOfCardText(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.TrueNames = true
	h.Update(Input{}, f)
	if _, _, ok := h.PurseTab(); ok {
		t.Fatal("no payment, no tab")
	}
	h.land(1500)
	tab, text, ok := h.PurseTab()
	ax, _ := GoldAnchor()
	if !ok || text != "+$1.5K" || tab.Y != TopH || tab.X+tab.W/2 < ax-1 || tab.X+tab.W/2 > ax+1 {
		t.Fatalf("tab %+v %q", tab, text)
	}
	if bottom, title := tab.Y+tab.H, Objective.Y+6; bottom > title {
		t.Errorf("the tab reaches y %d, over the step title at %d", bottom, title)
	}
	if w, _ := h.Font.Measure(text, 1); w+6 > tab.W {
		t.Errorf("%q does not fit its tab", text)
	}
}
