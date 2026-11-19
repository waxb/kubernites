package hud

import "testing"

func TestTheMenuSaysHowLongAgoTheRealmWasSavedAndWhenItWasNot(t *testing.T) {
	cases := []struct {
		f      Frame
		want   string
		failed bool
	}{
		{Frame{}, "", false},
		{Frame{Saved: true, SavedAgo: 12}, "Saved 12 s ago", false},
		{Frame{Saved: true, SavedAgo: 150}, "Saved 2 min ago", false},
		{Frame{SaveError: "disk full"}, "NOT SAVED", true},
		{Frame{Saved: true, SavedAgo: 61, SaveError: "disk full"}, "NOT SAVED (last 1 min ago)", true},
	}
	for _, c := range cases {
		if got, failed := SaveStatus(c.f); got != c.want || failed != c.failed {
			t.Errorf("%+v: %q failed %v, want %q failed %v", c.f, got, failed, c.want, c.failed)
		}
	}
}
