package hud

import "testing"

func tempoFrame(t *testing.T, real, events float64) Frame {
	f := fullFrame(t)
	f.Clock, f.EventClock = real, events
	return f
}

func TestCoinsRunOnTheRealmClockAndWordsStayReadableAtAnySpeed(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	h.Update(Input{}, tempoFrame(t, 10, 100))
	h.Toast("A ship docks", false)
	h.LaunchCoin(400, 300, 5)
	h.Banner("HARBOR TIER 2")
	h.Update(Input{}, tempoFrame(t, 10.05, 100+ToastLife))
	if h.CoinsInFlight() != 0 {
		t.Fatal("a token still flies after its realm-time flight")
	}
	if len(h.Toasts()) != 1 {
		t.Fatal("a toast vanished with realm time at x20 before anyone could read it")
	}
	if _, ok := BannerOffset(h.Now - h.Juice.BannerAt); !ok {
		t.Fatal("a banner vanished with realm time at x20 before anyone could read it")
	}
	h.Update(Input{}, tempoFrame(t, 10.1+ToastLife, 200))
	if len(h.Toasts()) != 0 {
		t.Fatal("a toast outlived its life on the wall clock")
	}
	if _, ok := BannerOffset(h.Now - h.Juice.BannerAt); ok {
		t.Fatal("a banner outlived its life on the wall clock")
	}
}

func TestInputFeedbackStaysOnRealTime(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.Update(Input{}, tempoFrame(t, 10, 100))
	h.Delta(MeterGold, 5)
	h.Update(Input{}, tempoFrame(t, 10.1, 200))
	if len(h.LiveDeltas()) == 0 {
		t.Fatal("a delta under the meter vanished with realm time")
	}
}

func TestFramesWithoutARealmClockFallBackToRealTime(t *testing.T) {
	if got := (Frame{Clock: 4}).GameClock(); got != 4 {
		t.Fatalf("GameClock = %v", got)
	}
	if got := (Frame{Clock: 4, EventClock: 9}).GameClock(); got != 9 {
		t.Fatalf("GameClock = %v", got)
	}
}
