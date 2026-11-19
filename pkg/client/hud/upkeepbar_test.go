package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheTopBarShowsUpkeepBesideIncomeOnceAKeepCostsRent(t *testing.T) {
	f := staged(fullFrame(t))
	f.State.Economy = view.EconomyView{Income: 2.5, Profit: 2.5}
	if st := GoldStat(f); st.Upkeep != "" || st.Rate != "+2.5/S" {
		t.Fatalf("with no rent the bar shows %q and %q", st.Rate, st.Upkeep)
	}
	f.State.Economy = view.EconomyView{Income: 2.5, Upkeep: 0.4, Profit: 2.1}
	st := GoldStat(f)
	if st.Rate != "+2.5/S" || st.Upkeep != "-0.4/S" || st.UpkeepClr != ui.Palette.RedText {
		t.Fatalf("with rent the bar shows %q and %q", st.Rate, st.Upkeep)
	}
	h := New(nil, (&fakeRunner{}).backend())
	calm := f
	calm.State.Economy = view.EconomyView{Income: 2.5, Profit: 2.5}
	if h.StatWidth(st) <= h.StatWidth(GoldStat(calm)) {
		t.Fatal("the overhead takes no room in the bar")
	}
	dst := ebiten.NewImage(Width, TopH)
	defer dst.Deallocate()
	runs := recordRuns(func() { h.drawStat(dst, st) })
	if !hasRun(runs, "-0.4/S") {
		t.Fatalf("overhead not drawn: %+v", runs)
	}
}

func TestTheKeepsBuildCardNamesItsRent(t *testing.T) {
	f := staged(fullFrame(t))
	f.State.KeepShop = []view.KeepOffer{{KeepType: view.KeepTypeStone, Name: "Stone Keep", Cost: 900, UpkeepPerSec: 0.35}}
	for _, b := range f.State.Buildables {
		if b.Kind == "keep" {
			if tip := KeepButton(b, f).Tip; !strings.Contains(tip, "0.35") || !strings.Contains(strings.ToLower(tip), "rent") {
				t.Fatalf("the keep's build card tip: %q", tip)
			}
			return
		}
	}
	t.Skip("no keep on the build page in this frame")
}
