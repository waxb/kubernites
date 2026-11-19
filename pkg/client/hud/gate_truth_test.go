package hud

import "testing"

func handsToast(h *HUD) bool {
	for _, toast := range h.Toasts() {
		if toast.Text == HandsNeededToast {
			return true
		}
	}
	return false
}

func TestNoCrewToastOnlyWhenNoCrewCarriesForAWhile(t *testing.T) {
	h := New(nil, Backend{})
	h.SetStaging(false)
	f := fullFrame(t)
	crewCarries(&f, true)
	h.Update(Input{}, f)
	crewCarries(&f, false)
	f.Clock += NoticeGap
	f.State.RealmTime++
	h.Update(Input{}, f)
	crewCarries(&f, true)
	f.Clock += NoticeGap
	f.State.RealmTime++
	h.Update(Input{}, f)
	if handsToast(h) {
		t.Fatal("'No crew carries the parcels!' while the porters carried: the depot's roll was rewritten for a moment")
	}
	crewCarries(&f, false)
	for range 4 {
		f.Clock += NoticeGap
		f.State.RealmTime += 2
		h.Update(Input{}, f)
	}
	if !handsToast(h) || !h.GateReopened() {
		t.Error("once no crew has carried for a while, the toast must still come")
	}
}
