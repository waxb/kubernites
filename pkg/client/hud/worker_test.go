package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func workerFrame(t *testing.T) Frame {
	f := fullFrame(t)
	p := f.Pods[0].Person(0)
	f.Selection = Selection{Kind: "walker", ID: p.Namespace + "/" + p.Name, Pod: p}
	f.Worker = &WorkerView{
		Title: "PORTER", Line: KuberniteLine(p), Status: "Carrying a parcel to the parcels depot.", Mood: "content",
		Thought: "Box in, box out. That's the whole song.",
		Needs:   []WorkerNeed{{Label: "FOOD", Value: "30 of 200 rations pledged"}, {Label: "RELIT", Value: "2, last 3m ago", Mark: "GUTTERING", Alarm: true}},
		Recent:  []WorkerThought{{Ago: "1m ago", Text: "Answered my first request!"}, {Ago: "5m ago", Text: "Kindled. Here we go."}},
	}
	return f
}

func drawnRuns(h *HUD, f Frame) []ui.TextRun {
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	var runs []ui.TextRun
	stop := ui.RecordText(func(r ui.TextRun) { runs = append(runs, r) })
	h.Draw(dst, f)
	stop()
	return runs
}

func regionText(runs []ui.TextRun, region string) string {
	out := []string{}
	for _, r := range runs {
		if r.Region == region {
			out = append(out, r.Text)
		}
	}
	return strings.Join(out, " | ")
}

func TestWorkerCardShowsTheWorkersStatusBesideItsPortrait(t *testing.T) {
	h := New(nil, Backend{})
	f := workerFrame(t)
	found := false
	for _, l := range h.CardFor(f).Lines {
		if l.Text == f.Worker.Status {
			found = !l.Optional
		}
	}
	if !found {
		t.Fatalf("the card lists the live status: %+v", h.CardFor(f).Lines)
	}
	f.Worker.View = ebiten.NewImage(48, 56)
	defer f.Worker.View.Deallocate()
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	if !h.DrawWorkerView(dst, PortraitBox().Inset(4), f) {
		t.Error("the live view takes the portrait's place")
	}
}

func TestWorkerWindowHoldsNeedsMoodThoughtAndRecentThoughts(t *testing.T) {
	h := New(nil, Backend{})
	f := workerFrame(t)
	r, ok := WorkerWindow(h, f)
	if !ok || r.Y+r.H > ConsoleY || r.X+r.W > Width {
		t.Fatalf("worker window %v shown %v", r, ok)
	}
	for _, run := range drawnRuns(h, f) {
		if run.Region == WindowRegion+PanelWorker && run.Rect.Y+run.Rect.H > r.Y+r.H {
			t.Errorf("%q spills below the window %v", run.Text, r)
		}
	}
	text := regionText(drawnRuns(h, f), WindowRegion+PanelWorker)
	for _, want := range []string{"PORTER", "MOOD", "content", "THOUGHT", "FOOD", "30 of 200 rations pledged", "GUTTERING", "RECENT THOUGHTS", "1m ago", "Answered my first request!"} {
		if !strings.Contains(text, want) {
			t.Errorf("the worker window lacks %q: %s", want, text)
		}
	}
	f.Selection = Selection{}
	if _, ok := WorkerWindow(h, f); ok {
		t.Error("no worker selected, no worker window")
	}
}

func TestALongRecentThoughtWrapsWholeInsideTheWorkerWindow(t *testing.T) {
	h := New(nil, Backend{})
	f := workerFrame(t)
	long := "My trade was already at the Keep. Quick start."
	f.Worker.Recent = []WorkerThought{{Ago: "2m ago", Text: long}, {Ago: "5m ago", Text: "Kindled. Here we go."}}
	r, _ := WorkerWindow(h, f)
	words := []string{}
	for _, run := range drawnRuns(h, f) {
		if run.Region != WindowRegion+PanelWorker {
			continue
		}
		if run.Rect.Y+run.Rect.H > r.Y+r.H {
			t.Errorf("%q spills below the window %v", run.Text, r)
		}
		if strings.Contains(run.Text, "...") {
			t.Errorf("%q is clipped", run.Text)
		}
		words = append(words, run.Text)
	}
	if text := strings.Join(words, " "); !strings.Contains(text, "already at the Keep. Quick start.") || !strings.Contains(text, "Kindled. Here we go.") {
		t.Errorf("the recent thoughts read whole: %s", text)
	}
}

func TestWorkerWindowCloseDeselects(t *testing.T) {
	h := New(nil, Backend{})
	f := workerFrame(t)
	intents := h.PressWorker("close", f)
	if len(intents) == 0 || intents[0].Kind != IntentSelect {
		t.Errorf("close = %+v", intents)
	}
}

func TestGoneWorkerHoldsWithoutActions(t *testing.T) {
	h := New(nil, Backend{})
	f := workerFrame(t)
	f.Selection.Gone = true
	f.Worker.Status = "Its hut is gone. The view holds where it stood."
	c := h.CardFor(f)
	if len(c.Lines) == 0 || c.Lines[len(c.Lines)-1].Text != f.Worker.Status {
		t.Errorf("the card says the hut is gone: %+v", c.Lines)
	}
	if acts := SelectionActions(f); len(acts) != 0 {
		t.Errorf("a gone hut offers no actions: %+v", acts)
	}
	if b := h.SelectionButtons(f); len(b) != 1 {
		t.Errorf("only the close button: %+v", b)
	}
}

func TestWorkerWindowStepsAsideFromOtherCards(t *testing.T) {
	h := New(nil, Backend{})
	f := workerFrame(t)
	home, _ := WorkerWindow(h, f)
	others := []ui.Rect{home}
	if OverlapsAny(home, h.OtherWindows(f)) {
		t.Fatalf("the worker window %v covers another card %v", home, h.OtherWindows(f))
	}
	if !OverlapsAny(home, others) {
		t.Fatal("overlap check")
	}
}
