package app

import (
	"testing"
	"time"
)

func TestStoodSecondsCountFromTheFoundingEpoch(t *testing.T) {
	founded := time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)
	got := stoodSeconds(map[string]time.Time{"tower": founded.Add(90 * time.Second), "keep": founded.Add(-time.Minute)}, founded)
	if got["tower"] != 90 || got["keep"] != -60 || len(got) != 2 {
		t.Errorf("stood = %v", got)
	}
}
