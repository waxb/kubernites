package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func ratedFrame(t *testing.T) Frame {
	f := fullFrame(t)
	f.Caesar.Rated = true
	f.Caesar.Ratings = model.Ratings{Reliability: 80, Efficiency: 60, Resilience: 40, Favor: 100}
	f.State.Ratings = view.RatingsView{Reliability: 80, Efficiency: 60, Resilience: 40, Favor: 100}
	return f
}

func TestTheFavorDialNamesTheShipsItDraws(t *testing.T) {
	defer ui.SetNamer(nil)
	dials := RatingDials(ratedFrame(t))
	if tip := dials[3].Tip; !strings.Contains(tip, "x1.15 ships now") {
		t.Fatalf("favor tip = %q", tip)
	}
	ui.SetNamer(names.Kubernetes(64))
	if tip := RatingDials(ratedFrame(t))[3].Tip; !strings.Contains(tip, "x1.15") || strings.Contains(tip, "ships") {
		t.Fatalf("names favor tip = %q", tip)
	}
}

func TestTheRatingsNameTheirReforgeThreadsOnceTheReforgeIsKnown(t *testing.T) {
	f := ratedFrame(t)
	f.State.Reveals = slices.DeleteFunc(slices.Clone(f.State.Reveals), func(id string) bool { return id == "reforge" })
	for _, d := range RatingDials(f) {
		if strings.Contains(d.Tip, "Golden Threads") {
			t.Fatalf("a rating speaks of its bonus before it is known: %q", d.Tip)
		}
	}
	f.State.Reveals = append(append([]string(nil), f.State.Reveals...), "reforge")
	for _, d := range RatingDials(f) {
		if !strings.Contains(d.Tip, "+14% Golden Threads") {
			t.Fatalf("%s tip = %q", d.Label, d.Tip)
		}
	}
}

func TestTheCouncilShowsTheFavorDialsEffect(t *testing.T) {
	h := New(nil, Backend{})
	for _, b := range h.CouncilButtons(ratedFrame(t)) {
		if b.ID == "info:dial:FAVOR" && strings.Contains(b.Tip, "x1.15 ships now") {
			return
		}
	}
	t.Fatal("the Council's Favor dial hides the ships it draws")
}
