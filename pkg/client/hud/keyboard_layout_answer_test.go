package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheYThatAnswersAnAskIsTheKeyLabelledYOnEveryLayout(t *testing.T) {
	for name, labels := range keyboardLayouts {
		layout := layoutTable(labels)
		yes := labelledKey(t, layout, 'y')
		var others []ebiten.Key
		if yes != ebiten.KeyY {
			others = append(others, ebiten.KeyY)
		}
		check := func(what string, answered func(Input) bool) {
			if !answered(Input{Keys: []ebiten.Key{yes}, Layout: layout}) {
				t.Errorf("%s: the key labelled y (Ebitengine's %v) does not answer %s", name, yes, what)
			}
			for _, k := range others {
				if answered(Input{Keys: []ebiten.Key{k}, Layout: layout}) {
					t.Errorf("%s: the key labelled %q (Ebitengine's %v) answers %s", name, layout(k), k, what)
				}
			}
		}
		check("the Stone's ask", func(in Input) bool {
			r := &stoneRun{ask: &StoneAsk{Confirm: true}, askFrames: AskSeenFrames}
			return r.yes(in)
		})
		check("a card's CONFIRM", func(in Input) bool {
			h, _ := liveHUD(t, true)
			h.OpenOrder(OrderForm{ID: 1, Title: "DELETE", Go: "DELETE [Y]", GoKey: ebiten.KeyY})
			for range AskSeenFrames {
				h.seeOrder()
			}
			intents, _ := h.orderKeys(in)
			for _, it := range intents {
				if it.Kind == IntentLiveScroll && it.Command == ScrollGo {
					return true
				}
			}
			return false
		})
		check("the picker's WRITES warning", func(in Input) bool {
			p := confirmPicker(nil, false)
			p.press(LiveBoxID)
			for range AskSeenFrames {
				p.Update(Input{})
			}
			p.Update(in)
			return p.Step == LiveStepConfirm
		})
	}
}
