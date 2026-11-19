package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func cutRows(h *HUD, lines []Line, w, room int) []string {
	cut := []string{}
	for _, l := range h.FitLines(lines, w, room) {
		if room <= 0 {
			break
		}
		rows := h.Font.WrapLines(l.Text, w, 1, room)
		if l.Bar {
			rows = []string{l.Text}
		}
		if last := rows[len(rows)-1]; strings.HasSuffix(last, "...") {
			cut = append(cut, last)
		}
		room -= len(rows)
	}
	return cut
}

func TestACrewCardNeverEndsOnACutLine(t *testing.T) {
	h := New(nil, Backend{})
	box := PortraitBox()
	w := CardBox.X + CardBox.W - box.X - box.W - 14
	c := model.CrewChain{Name: "gate-controller", Image: "registry.example/gate-controller:1.4.2", Desired: 1, Ready: 1, MaxSurge: "25%", MaxUnavailable: "25%", HistoryLimit: 10}
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		for room := 6; room <= 9; room++ {
			if cut := cutRows(h, DeploymentCard(c).Lines, w, room); len(cut) > 0 {
				t.Errorf("%s, %d rows: the card ends on %q", mode, room, cut)
			}
		}
		ui.SetNamer(nil)
	}
}
