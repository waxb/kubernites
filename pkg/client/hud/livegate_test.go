package hud

import "testing"

func TestEveryCloseButtonClosesInALiveRealm(t *testing.T) {
	for _, id := range []string{CrewPrefix + ":close", crewsCloseKey, CalamityClose, KindKey + ":close", trialPrefix + ":close", endingPrefix + ":close"} {
		if !LiveReadButton(id) {
			t.Errorf("%s is refused as a write in a live realm", id)
		}
	}
	for _, id := range []string{CrewPrefix + ":slider", CrewPrefix + ":pick:porter"} {
		if LiveReadButton(id) {
			t.Errorf("%s passes the live gate", id)
		}
	}
}
