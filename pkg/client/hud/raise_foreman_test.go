package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func foremanFrame(t *testing.T) Frame {
	t.Helper()
	f := chooserFrame(t)
	f.State.Crews, f.Crews = nil, nil
	f.State.RaiseKeeper = view.KeeperReplicaSet
	return f
}

func TestBeforeCrewsAreLearnedTheChooserRaisesAForeman(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := foremanFrame(t)
	clickOn(t, h, f, CmdCrew)
	clickOn(t, h, f, "raise:pick:porters")
	confirm, ok := HitID(h.Buttons(f), RaiseConfirm)
	if !ok || confirm.Disabled || !strings.Contains(confirm.Tip, "ReplicaSet") || strings.Contains(confirm.Tip, "deployment") {
		t.Fatalf("confirm = %+v", confirm)
	}
	a := actionIntent(t, clickOn(t, h, f, RaiseConfirm))
	if a.Action != "raiseMuster" || a.Name != "porters" || !strings.HasPrefix(a.Soulstone, "porter:") || a.Replicas != foremanWorkers {
		t.Fatalf("confirm raised %+v", a)
	}
	rows := RaiseRows(f)
	if line := CrewChooserLine(rows[0], 3); !strings.Contains(line, "shift") || strings.Contains(line, "crew") {
		t.Fatalf("chooser line = %q", line)
	}
	if !strings.Contains(RaiseRowText(rows[0]), "nobody keeps") || !strings.Contains(RaiseHeadline(rows), "shift") {
		t.Fatalf("row %q headline %q", RaiseRowText(rows[0]), RaiseHeadline(rows))
	}
	if raiseTitle(f) != "RAISE A SHIFT" {
		t.Fatalf("title = %q", raiseTitle(f))
	}
}

func TestTheCrewButtonNamesTheForemanItRaises(t *testing.T) {
	defer ui.SetNamer(nil)
	h := New(nil, (&fakeRunner{}).backend())
	f := foremanFrame(t)
	b, ok := HitID(h.CommandButtons(f), CmdCrew)
	if !ok || b.Label != "SHIFT" || !strings.Contains(b.Tip, "RAISE A SHIFT") {
		t.Fatalf("on the shift's rung the button reads %q (%q)", b.Label, b.Tip)
	}
	ui.SetNamer(names.Kubernetes(64))
	if b, _ := HitID(h.CommandButtons(f), CmdCrew); ui.Name(b.Label) != "REPLICASET" || !strings.Contains(ui.Name(b.Tip), "CREATE A REPLICASET") {
		t.Fatalf("in NAMES mode the button reads %q (%q)", ui.Name(b.Label), ui.Name(b.Tip))
	}
	ui.SetNamer(nil)
	f.State.RaiseKeeper = view.KeeperDeployment
	if b, _ := HitID(h.CommandButtons(f), CmdCrew); b.Label != "CREW" {
		t.Fatalf("once Crews are learned the button reads %q", b.Label)
	}
}
