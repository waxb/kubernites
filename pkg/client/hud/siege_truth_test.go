package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheCallWhileTheKeepsLackTheSpareFoodForecastsTheFlood(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Siege.Available, f.State.Siege.Hands, f.State.Siege.Spare = true, 7.245, 3
	f.State.Siege.Forecast = view.TideFloodForecast
	if b := h.SiegeButtons(f)[1]; b.Disabled || b.Tip != view.TideFloodForecast {
		t.Fatalf("ANSWER THE CALL while the Keeps are short must stay a player's choice, with the flood forecast: %+v", b)
	}
	f.State.Siege.Spare = 8
	if b := h.SiegeButtons(f)[1]; b.Disabled {
		t.Fatalf("a realm with the spare food cannot answer: %+v", b)
	}
}

func TestTheTidePanelShowsOneSpareFoodFigure(t *testing.T) {
	f := fullFrame(t)
	lines := SiegeReadiness(f)
	if strings.Contains(lines[1].Text, "/") {
		t.Fatalf("the Keeps line shows a second food figure beside the spare one: %q", lines[1].Text)
	}
}

func TestTheTidePanelNamesACrewThatCannotCarryItsShareAndGivesNoGoAhead(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Siege.Available, f.State.Siege.Hands, f.State.Siege.Spare = true, 4.267, 5
	f.State.Siege.Crowded = "the haulers need 16 huts for the tide and have 2; your Keeps have room for 0 more (each asks 1,000 rations and 1,024 rooms)"
	f.State.Siege.Forecast = view.TideFloodForecast
	found := false
	for _, l := range SiegeReadiness(f) {
		found = found || strings.Contains(l.Text, "haulers need 16 huts")
	}
	if !found || !siegeShort(f) {
		t.Fatalf("the panel's go-ahead ignores haulers with no room: short %v, lines %+v", siegeShort(f), SiegeReadiness(f))
	}
	if b := h.SiegeButtons(f)[1]; b.Disabled || b.Tip != view.TideFloodForecast {
		t.Fatalf("the call stays a player's choice, with the flood forecast: %+v", b)
	}
}
