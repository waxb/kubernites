package hud

import "testing"

func TestABannerWhoseNewsIsNoLongerTrueIsDroppedBeforeItShows(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.Banner("THE CITADEL STANDS")
	waits := true
	h.BannerWhile("NEW CARGO: LEDGERS", "Ledgers arrive, and no crew carries them yet", func() bool { return waits })
	if len(h.Director.Banners) != 1 {
		t.Fatalf("the cargo banner should wait behind the first: %+v", h.Director.Banners)
	}
	waits = false
	h.pruneBanners()
	h.Juice.Banner = ""
	h.showBanner()
	if h.Juice.Banner == "NEW CARGO: LEDGERS" || len(h.Director.Banners) != 0 {
		t.Fatalf("a banner whose news has passed must not show: shown %q, queue %+v", h.Juice.Banner, h.Director.Banners)
	}
}

func TestAWhatHappenedCardWhoseNewsHasPassedNeverShows(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.LessonCard = LessonCard{Title: "A GILDED SHIP DOCKED", Shown: true}
	waits := true
	h.ResultWhile("NEW CARGO: LEDGERS", "Ledgers arrive, and no crew carries them yet", func() bool { return waits })
	waits = false
	h.LessonCard = LessonCard{}
	h.nextResult()
	if h.LessonCard.Title == "NEW CARGO: LEDGERS" {
		t.Fatal("a WHAT HAPPENED card whose news has passed must not show")
	}
}
