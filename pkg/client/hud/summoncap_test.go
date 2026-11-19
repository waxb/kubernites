package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func summonFrame(t *testing.T, waiting int) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Summon = view.SummonView{Running: 3, Waiting: waiting}
	tower := view.Structure{ID: "keep-1", Kind: view.StructKeep, Label: "Lone Tower", State: view.StateReady, Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}}
	f.Site.Structures = append(append([]view.Structure(nil), f.Site.Structures...), tower)
	return f
}

func buttonFor(t *testing.T, h *HUD, f Frame, id string) Button {
	t.Helper()
	b, ok := HitID(h.Buttons(f), id)
	if !ok {
		t.Fatalf("button %s not found", id)
	}
	return b
}

func TestSummonTipsCountRunningAndWaitingHearths(t *testing.T) {
	f := summonFrame(t, 2)
	h := New(nil, Backend{})
	want := "Pods: 3 running, 2 waiting for room"
	if tip := buttonFor(t, h, f, "cmd:summon").Tip; !strings.Contains(tip, want) || !strings.Contains(tip, "free") {
		t.Fatalf("SUMMON tip = %q", tip)
	}
	h.PageID = PageSummon
	b := buttonFor(t, h, f, "summon:porter:1.0")
	if b.Disabled || !strings.Contains(b.Tip, want) || !strings.Contains(b.Tip, "free") {
		t.Fatalf("summon button = %+v", b)
	}
	lines := strings.Join(SummonLines(f), "\n")
	if !strings.Contains(lines, want) || !strings.Contains(lines, "free") {
		t.Fatalf("summon card = %q", lines)
	}
}

func TestSummonIsNeverRefused(t *testing.T) {
	f := summonFrame(t, 40)
	h := New(nil, Backend{})
	h.PageID = PageSummon
	if b := buttonFor(t, h, f, "summon:porter:1.0"); b.Disabled {
		t.Fatalf("summon button with 40 waiting = %+v", b)
	}
	if _, acted := hasIntent(clickOn(t, h, f, "summon:porter:1.0"), IntentAction); !acted {
		t.Fatal("a summon with many waiting never reached the realm")
	}
	if lines := strings.Join(SummonLines(f), "\n"); !strings.Contains(lines, summonFull) {
		t.Fatalf("summon card = %q", lines)
	}
}
