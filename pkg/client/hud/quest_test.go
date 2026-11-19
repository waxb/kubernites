package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func founding(done int) view.QuestView {
	q := view.QuestView{ID: view.BorgQuest, Title: "Borg's Decree", Giver: "Borg, the old builder of realms", Intro: "Borg decrees: found the control plane.", Current: done}
	for i, title := range []string{"The Codex", "The Chancery", "The Quartermaster", "The Stewards"} {
		q.Parts = append(q.Parts, view.QuestPartView{Objective: "part-" + title, Title: title, Purpose: "what " + title + " does", Gained: "gained " + title, Done: i < done})
	}
	q.Fulfilled = done == len(q.Parts)
	if q.Fulfilled {
		q.Recap = "The four halls stand."
	}
	return q
}

func questFrame(t *testing.T, q view.QuestView, step view.ObjectiveView) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Quests = []view.QuestView{q}
	f.State.Objectives = []view.ObjectiveView{step}
	return f
}

func TestBorgsDecreeOpensWithItsCardThenTracksItsParts(t *testing.T) {
	h := New(nil, Backend{})
	f := questFrame(t, founding(0), view.ObjectiveView{ID: "part-The Codex", Text: "Check the control plane", Quest: view.BorgQuest})
	f.Guide.Intro = &Intro{ID: view.BorgIntro, Title: "Operator brief", Purpose: "The operator briefs.", Try: "Read the brief.", Target: "surface:borg", Via: ViaAck}
	c, ok := h.QuestCard(f)
	if !ok || c.Recap || c.Body == "" || len(c.Parts) != 4 {
		t.Fatalf("the brief card does not open the task: %+v", c)
	}
	if h.ObjectiveShown(f) || h.IntroShown(f) {
		t.Error("the brief speaks alone")
	}
	intents := h.PressQuest("ack", f)
	if len(intents) == 0 || intents[0].Action != Ack(view.BorgIntro) {
		t.Fatalf("I WILL RAISE THEM does not acknowledge the brief: %+v", intents)
	}
	f.Guide.Intro = nil
	if h.QuestCardShown(f) || !h.ObjectiveShown(f) {
		t.Fatal("after the brief, the tracker takes the task card's place")
	}
	if title, decree := h.CardTitle(f); !decree || title != "BORG'S DECREE 0/4" {
		t.Errorf("the tracker is not titled as the brief: %q", title)
	}
	text := ""
	for _, l := range h.ObjectiveLines(f) {
		text += l.Text + "\n"
	}
	for _, want := range []string{MarkNow + "The Codex", "what The Codex does", "Check the control plane", MarkLater + "The Stewards"} {
		if !strings.Contains(text, want) {
			t.Errorf("the tracker misses %q:\n%s", want, text)
		}
	}
}

func TestAFulfilledDecreeEndsWithItsRecapAndOtherStepsReadNextStep(t *testing.T) {
	h := New(nil, Backend{})
	f := questFrame(t, founding(4), view.ObjectiveView{ID: "spoil", Text: "Watch a box spoil", Purpose: "A box left waiting is lost"})
	c, ok := h.QuestCard(f)
	if !ok || !c.Recap || !strings.Contains(c.Body, "four halls") {
		t.Fatalf("no recap card for a fulfilled brief: %+v", c)
	}
	intents := h.PressQuest("ack", f)
	if h.QuestCardShown(f) {
		t.Fatal("the recap stays after ONWARD")
	}
	if len(intents) == 0 || intents[0].Action != (view.Action{Action: view.ActCloseQuest, ID: view.BorgQuest}) {
		t.Fatalf("ONWARD does not tell the realm the recap is read: %+v", intents)
	}
	closed := founding(4)
	closed.Closed = true
	if New(nil, Backend{}).QuestCardShown(questFrame(t, closed, view.ObjectiveView{ID: "spoil"})) {
		t.Fatal("a recap the realm has closed comes back in a new session")
	}
	if title, decree := h.CardTitle(f); decree || title != NextStepTitle {
		t.Errorf("a step outside the task is titled %q", title)
	}
	found := false
	for _, l := range h.ObjectiveLines(f) {
		found = found || strings.HasPrefix(l.Text, ForMark+"A box left waiting")
	}
	if !found {
		t.Error("the step card does not say what the step is for")
	}
}

func TestTheGuideFindsTheDecreeScrollsButton(t *testing.T) {
	h := New(nil, Backend{})
	f := questFrame(t, founding(0), view.ObjectiveView{ID: "part-The Codex", Text: "Check the control plane", Quest: view.BorgQuest})
	f.Guide.Intro = &Intro{ID: view.BorgIntro, Title: "Operator brief", Purpose: "The operator briefs.", Try: "Read the brief.", Target: SurfaceDecree, Via: ViaAck}
	if w := h.WalkGuide(f); !w.OK() || len(w.Steps) != 1 || w.Steps[0] != SurfaceDecree {
		t.Fatalf("the brief's guide walk: %+v", w)
	}
	if r, ok := h.TargetRect(SurfaceDecree, f); !ok || r.W == 0 {
		t.Fatalf("the brief scroll's button has no place: %+v", r)
	}
}
