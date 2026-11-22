package write

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestEveryChangeADrainSendsReachesItsReport(t *testing.T) {
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) { return http.StatusCreated, `{}` }}
	g := bound(t, Writes, f, Destroy)
	d, err := PlanDrain(fixtureNode(false), drainFixture())
	if err != nil {
		t.Fatal(err)
	}
	g.IssueDrain(d)
	var reports []DrainReport
	if err := g.RunDrain(context.Background(), d, time.Millisecond, func(r DrainReport) { reports = append(reports, r) }); err != nil {
		t.Fatal(err)
	}
	var changes []string
	for _, l := range f.lines() {
		if !strings.HasPrefix(l, "GET ") {
			changes = append(changes, l)
		}
	}
	if len(reports) < len(changes) {
		t.Errorf("the drain sent %d changes but reported %d, and the app writes the ledger and the card only from these reports: %v reported as %+v", len(changes), len(reports), changes, reports)
	}
}
