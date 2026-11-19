package lexicon

import "testing"

func TestLessThanOneRoomIsNotZeroRooms(t *testing.T) {
	if got, want := StoryText("SIZE 251 KiB, IN USE 251 KiB"), "SIZE 0.25 rooms, IN USE 0.25 rooms"; got != want {
		t.Fatalf("StoryText = %q, want %q", got, want)
	}
	if got, want := StoryText("16 MiB"), "16 rooms"; got != want {
		t.Fatalf("StoryText = %q, want %q", got, want)
	}
}
