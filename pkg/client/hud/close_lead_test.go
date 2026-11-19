package hud

import "testing"

func TestTheCloseLeadNeverDoublesTheCardsArticle(t *testing.T) {
	if got := closeLead("THE [parcel] PARCELS DEPOT"); got != "X on the [parcel] PARCELS DEPOT card" {
		t.Fatalf("the close lead reads %q", got)
	}
	if got := closeLead("PORTERS"); got != "X on the PORTERS card" {
		t.Fatalf("the close lead reads %q", got)
	}
}
