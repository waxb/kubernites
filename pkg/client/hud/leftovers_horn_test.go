package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheHornPanelSaysWhenHousingRunsOutBeforeFood(t *testing.T) {
	v := BoardView{Chain: model.CrewChain{Horn: &model.Horn{Goal: 70, Measured: true, Utilization: 20}}, horn: HornPick{Min: 30, Max: 40}}
	v.Facts = CrewFacts{Fed: 12, Seats: 12, Limit: view.FlowMemory, Per: ReplicaFacts{CPU: 200, Memory: 64}}
	text := func() string {
		out := []string{}
		for _, p := range bandPieces(v) {
			out = append(out, p.Text)
		}
		return strings.Join(out, " | ")
	}
	if !strings.Contains(text(), "HOUSING FILLS BEFORE FOOD") {
		t.Fatalf("a Horn on a housing-bound crew says nothing: %s", text())
	}
	v.Facts.Limit = view.FlowCPU
	if strings.Contains(text(), "HOUSING FILLS BEFORE FOOD") {
		t.Fatalf("a food-bound crew is told about housing: %s", text())
	}
}

func TestAWarningPieceOutlivesTheQuietOnesWhenTheRowIsFull(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	pieces := []deltaPiece{{"WAR HORN", ui.Palette.Cream}, {"FLOOR 2 CEILING 6", ui.Palette.Blue}, {"NOW 6, FOOD 253 OF 50%", ui.Palette.TextDim}, {"THE KEEPS HOLD 5: PAST IT, HUTS WAIT", ui.Palette.RedText}}
	width := h.Font.Width(pieces[0].Text, 1) + h.Font.Width(pieces[1].Text, 1) + h.Font.Width(pieces[3].Text, 1) + 3*pieceGap
	got := h.fitPieces(pieces, width)
	if len(got) != 3 || got[2].Text != pieces[3].Text || got[1].Text != pieces[1].Text {
		t.Fatalf("a full row dropped the warning: %+v", got)
	}
}
