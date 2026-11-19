package hud

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestCoinArcsFromThePilgrimToTheCounter(t *testing.T) {
	from := [2]float64{400, 300}
	ax, ay := GoldAnchor()
	to := [2]float64{float64(ax), float64(ay)}
	if got := CoinPath(from, to, 0); got != from {
		t.Errorf("a dot starts at the request: %v", got)
	}
	if got := CoinPath(from, to, 1); math.Abs(got[0]-to[0]) > 1e-9 || math.Abs(got[1]-to[1]) > 1e-9 {
		t.Errorf("a dot lands on the counter: %v", got)
	}
	if got := CoinPath(from, to, 2); math.Abs(got[0]-to[0]) > 1e-9 {
		t.Errorf("the path clamps: %v", got)
	}
	mid := CoinPath(from, to, .5)
	if mid[1] >= (from[1]+to[1])/2 {
		t.Errorf("the dot should arc upward, mid %v", mid)
	}
	prev := from
	for i := 1; i <= 20; i++ {
		p := CoinPath(from, to, float64(i)/20)
		if math.Hypot(p[0]-prev[0], p[1]-prev[1]) > 60 {
			t.Fatalf("the dot jumps at step %d: %v -> %v", i, prev, p)
		}
		prev = p
	}
	if ax < 80 || ax > 110 || ay < 1 || ay > TopH {
		t.Errorf("counter anchor %d,%d is not on the counter icon", ax, ay)
	}
}

func TestCounterPulseFades(t *testing.T) {
	cases := []struct{ age, want float64 }{{-1, 0}, {0, 1}, {coinPulse / 2, .5}, {coinPulse, 0}, {9, 0}}
	for _, c := range cases {
		if got := CounterPulse(c.age); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("pulse(%v) = %v, want %v", c.age, got, c.want)
		}
	}
}

func TestCoinTapAggregatesTreasuryIncome(t *testing.T) {
	cases := []struct {
		name      string
		rate      float64
		seconds   float64
		minFlight int
		maxFlight int
		maxPerSec float64
	}{
		{"trickle", .3, 20, 5, 7, 4},
		{"steady", 2, 10, 18, 22, 4},
		{"flood", 400, 10, 36, 41, 4},
		{"nothing", 0, 10, 0, 0, 0},
		{"upkeep", -1, 10, 0, 0, 0},
	}
	for _, c := range cases {
		tap, flights, total, last := CoinTap{}, 0, 0, -1.0
		for now := 0.0; now < c.seconds; now += .25 {
			var v int
			tap, v = tap.Step(now, c.rate*.25)
			if v > 0 {
				if last >= 0 && now-last < coinGap-1e-9 {
					t.Fatalf("%s: flights %.2fs apart", c.name, now-last)
				}
				flights, total, last = flights+1, total+v, now
			}
		}
		if flights < c.minFlight || flights > c.maxFlight {
			t.Errorf("%s: %d flights, want %d..%d", c.name, flights, c.minFlight, c.maxFlight)
		}
		if c.rate > 0 && math.Abs(float64(total)+tap.Carry-c.rate*c.seconds) > 1e-6 {
			t.Errorf("%s: flights carry %d + %.2f, served %.2f", c.name, total, tap.Carry, c.rate*c.seconds)
		}
	}
}

func TestCoinsFlyLandAndPulse(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	h.Update(Input{}, f)
	h.LaunchCoin(400, 300, 1)
	h.LaunchCoin(410, 300, 3)
	if n := len(h.Coins.Flights); n != 2 {
		t.Fatalf("flights = %d", n)
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	f.Clock += coinFlight + .01
	h.Update(Input{}, f)
	h.Draw(dst, f)
	if len(h.Coins.Flights) != 0 || h.Coins.Landed != 4 {
		t.Fatalf("dots did not land: %+v", h.Coins)
	}
	if h.GoldPulse() <= 0 {
		t.Error("the counter should pulse when dots land")
	}
	f.Clock += coinPlus + 1
	h.Update(Input{}, f)
	if h.GoldPulse() != 0 || h.Coins.Landed != 0 {
		t.Error("the pulse fades")
	}
	for i := 0; i < maxFlights+5; i++ {
		h.LaunchCoin(300, 200, 1)
	}
	if len(h.Coins.Flights) > maxFlights {
		t.Errorf("flights pile up: %d", len(h.Coins.Flights))
	}
	f.Reduce = true
	h.Update(Input{}, f)
	h.Coins = purse{}
	h.LaunchCoin(300, 200, 2)
	if len(h.Coins.Flights) != 0 || h.Coins.Landed != 2 {
		t.Errorf("reduced motion lands dots at once: %+v", h.Coins)
	}
	h.Draw(dst, f)
}
