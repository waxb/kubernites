package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func pendingDecreeFrame(t *testing.T) Frame {
	t.Helper()
	f := questFrame(t, founding(0), view.ObjectiveView{ID: "part-The Codex", Text: "Check the control plane", Quest: view.BorgQuest, Progress: "CPU 3/20"})
	f.State.Objectives = append([]view.ObjectiveView{{ID: "door-hearth", Text: "Place a porter at the door", Done: true, Lesson: "A static Pod."}}, f.State.Objectives...)
	f.State.Reveals = append(f.State.Reveals, view.BorgIntro)
	f.Guide = Guide{Staged: true, Introduced: []string{"serve"}}
	return f
}

func linesText(lines []Line) string {
	out := ""
	for _, l := range lines {
		out += l.Text + "\n"
	}
	return out
}

func TestTheTrackerWaitsForTheScrollAndTheLastStepStaysReadable(t *testing.T) {
	h := New(nil, Backend{})
	f := pendingDecreeFrame(t)
	text := linesText(h.ObjectiveLines(f))
	if strings.Contains(text, "Codex") || !strings.Contains(text, "Place a porter at the door") {
		t.Fatalf("before the scroll the card shows %q", text)
	}
	if title, _ := h.CardTitle(f); strings.Contains(title, "DECREE") {
		t.Errorf("the panel title shows before its intro: %q", title)
	}
	f.Guide.Introduced = append(f.Guide.Introduced, view.BorgIntro)
	if text := linesText(h.ObjectiveLines(f)); !strings.Contains(text, "The Codex") {
		t.Fatalf("after the scroll the tracker is missing: %q", text)
	}
}

func TestTheScrollHoldsRealmTimeWhileItIsOpen(t *testing.T) {
	h := New(nil, Backend{})
	f := pendingDecreeFrame(t)
	if h.ScrollOpen(f) {
		t.Fatal("no scroll yet")
	}
	f.Guide.Intro = &Intro{ID: view.BorgIntro, Title: "Operator brief", Target: "surface:borg", Via: ViaAck}
	if !h.ScrollOpen(f) {
		t.Fatal("the scroll is open and time must be held")
	}
	f.Guide.Intro = nil
	f.Guide.Introduced = append(f.Guide.Introduced, view.BorgIntro)
	if h.ScrollOpen(f) {
		t.Fatal("time still held after the scroll was read")
	}
}

func TestTheTrackerFoldsToOneHeaderAndKeepsTheStepReadable(t *testing.T) {
	h := New(nil, Backend{})
	f := pendingDecreeFrame(t)
	f.Guide.Introduced = append(f.Guide.Introduced, view.BorgIntro)
	f.Guide.Why = "so what you declare is written"
	open := len(h.ObjectiveLines(f))
	btn, ok := HitID(h.Buttons(f), FoldID)
	if !ok {
		t.Fatal("no fold control on the tracker header")
	}
	if title, _ := h.CardTitle(f); title != "BORG'S DECREE 0/4" {
		t.Errorf("header %q", title)
	}
	h.Update(Input{MX: btn.Rect.X + 3, MY: btn.Rect.Y + 3, Click: true}, f)
	if !h.TrackerFolded() {
		t.Fatal("clicking the header did not fold the tracker")
	}
	folded := linesText(h.ObjectiveLines(f))
	for _, want := range []string{"Check the control plane", "so what you declare", "CPU 3/20"} {
		if !strings.Contains(folded, want) {
			t.Errorf("folded tracker misses %q: %q", want, folded)
		}
	}
	if strings.Contains(folded, "The Chancery") || len(h.ObjectiveLines(f)) >= open {
		t.Errorf("folded tracker still lists the halls: %q", folded)
	}
	h.SetTrackerFolded(false)
	if len(h.ObjectiveLines(f)) != open {
		t.Error("unfolding did not restore the tracker")
	}
}

func TestAHalfDonePartShowsItsCount(t *testing.T) {
	if got := PartCount(view.QuestPartView{Steps: 2, StepsDone: 1}); got != "  1/2" {
		t.Fatalf("a raised Chancery still to be asked reads %q", got)
	}
	if got := PartCount(view.QuestPartView{Steps: 1}); got != "" {
		t.Fatalf("a one-step part reads %q", got)
	}
}
