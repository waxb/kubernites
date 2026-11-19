package hud

import "testing"

func TestAQuietAlertIsDrawnAsANotice(t *testing.T) {
	loud := Alert{Kind: AlertFire}
	quiet := Alert{Kind: AlertFire, Quiet: true}
	if SeverityOf(loud) != LevelDanger || SeverityOf(quiet) != LevelNotice {
		t.Fatalf("loud %d, quiet %d", SeverityOf(loud), SeverityOf(quiet))
	}
	arrows := bySeverity([]Alert{quiet, {Kind: AlertKeep}, loud})
	if arrows[0].Quiet || arrows[0].Kind != AlertFire || !arrows[2].Quiet {
		t.Fatalf("order = %+v", arrows)
	}
}
