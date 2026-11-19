package art

import "testing"

func TestTheStoneIsAMenhirUntilProductionAndAnObeliskThere(t *testing.T) {
	for tier, want := range []string{StoneMenhir, StoneMenhir, StoneMenhir, StoneObelisk} {
		if got := SpeakStoneLook(tier); got != want {
			t.Errorf("age tier %d stone = %q, want %q", tier, got, want)
		}
	}
	if SpeakStoneName(SpeakStoneLook(0), StoneLit) == SpeakStoneName(SpeakStoneLook(AgeTiers-1), StoneLit) {
		t.Fatal("the two looks share a sprite")
	}
	if SpeakStoneName(StoneMenhir, "") != SpeakStoneName(StoneMenhir, StoneDormant) {
		t.Fatal("an unknown state is not the dormant stone")
	}
	if SpeakStoneGlowLift(StoneMenhir) >= SpeakStoneGlowLift(StoneObelisk) {
		t.Fatal("the menhir's bloom sits higher than its low mark")
	}
}
