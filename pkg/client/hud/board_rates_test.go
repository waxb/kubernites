package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheHarborBoardShowsTheRatesAsTheyChange(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.Clock = 1
	f.State.Economy = view.EconomyView{}
	h.Update(Input{MX: -5, MY: -5}, h.Stage(f))
	for i := 2; i <= 40; i++ {
		f.Clock = float64(i)
		f.State.RealmTime = float64(i)
		f.State.Economy = view.EconomyView{Arriving: 1.5, Carried: 1.2, Lost: .1, Income: 4.1}
		f.State.Freight.Fresh = .9
		h.Update(Input{MX: -5, MY: -5}, h.Stage(f))
	}
	staged := h.Stage(f)
	e := staged.State.Economy
	if e.Arriving != 1.5 || e.Carried != 1.2 || e.Income != 4.1 || staged.State.Freight.Fresh != .9 {
		t.Errorf("the board's rates read zero beside +4.1/S on every read: the board holds %+v while the session hands Update a frame it already staged", e)
	}
}
