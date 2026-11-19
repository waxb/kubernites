package hud

import (
	"fmt"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheTechSummaryCountsPagesTheWayTheMapDoes(t *testing.T) {
	pages := []view.GrimoirePage{
		{ID: "a", State: "unlocked", Rank: 1}, {ID: "b", State: "available", Rank: 1},
		{ID: "c", State: "available"}, {ID: "d", State: "hidden"}, {ID: "e", State: "locked"},
	}
	learned, shown := learnedCount(pages)
	want := fmt.Sprintf("PAGES LEARNED %d/%d", learned, shown)
	if got := techSummaryTitle(pages); got != want || learned != 1 || shown != 4 {
		t.Errorf("summary %q, map %d/%d", got, learned, shown)
	}
}

func TestALessonInTheEventLogWrapsWholeInsteadOfClipping(t *testing.T) {
	h := &HUD{}
	lesson := Tiding{Kind: "lesson", Detail: "more", Text: "What just happened: A fallen worker is relit by its Reeve, but a razed hut is gone: only a shift raises another."}
	f := Frame{Events: []Tiding{lesson, {Kind: "event", Text: "Hut porters-abcde is Burning"}}}
	rows := h.tidingRows(f, WellBody())
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	last := rows[0].lines[len(rows[0].lines)-1]
	if strings.HasSuffix(last, "...") || !strings.HasSuffix(last, "another.") {
		t.Errorf("the hint is cut: %q", rows[0].lines)
	}
}

func TestALongProclamationInTheTidingsIsWholeOrReachable(t *testing.T) {
	h := &HUD{}
	long := "The Chancery opens its doors (kubeadm init: the API server answers at 172.18.0.2:6443). The Lone Tower swears fealty (it registers as a Node), and its Reeve shows the hut at the door to the Chancery (a mirror Pod: kubectl get pods lists porter-lone-tower)."
	f := Frame{Events: []Tiding{{Kind: "info", Text: long}, {Kind: "event", Text: "Hut porters-abcde is Burning"}}}
	rows := h.tidingRows(f, WellBody())
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	if last := rows[0].lines[len(rows[0].lines)-1]; strings.HasSuffix(last, "...") || !strings.HasSuffix(last, "porter-lone-tower).") {
		t.Errorf("the proclamation is cut: %q", rows[0].lines)
	}
	endless := Tiding{Kind: "info", Text: strings.Repeat(long+" ", 4)}
	f = Frame{Events: []Tiding{endless}}
	rows = h.tidingRows(f, WellBody())
	if len(rows) != 1 {
		t.Fatalf("a proclamation longer than the well shows %d rows", len(rows))
	}
	reachable := false
	for _, b := range h.tidingButtons(f) {
		reachable = reachable || b.Tip == Label(endless.Text)
	}
	if !reachable {
		t.Error("a proclamation cut to fit the well cannot be read whole")
	}
}
