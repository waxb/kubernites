package hud

import (
	"slices"
	"testing"
)

func TestSilenceCountsOnlyAChaptersSecondsWithNoCardAndNothingStarted(t *testing.T) {
	marks := []PaceMark{
		{At: 0, Chapter: true, Card: true},
		{At: 10, Chapter: true},
		{At: 13, Chapter: true, Busy: BusyWatch},
		{At: 20, Chapter: true, Card: true},
		{At: 30},
		{At: 90, Card: true},
	}
	spans := silences(marks, 100)
	if len(spans) != 1 || spans[0].Seconds() != 3 {
		t.Fatalf("silences %v: want one of 3 s (the watch and the marked beats do not count)", spans)
	}
}

func TestPressToCardIsTheTimeToTheNextCardOtherThanTheOneAnswered(t *testing.T) {
	log := AttnLog{
		Boxes: []AttnBox{
			{Key: "done:a", From: 0, To: 30},
			{Key: "step:b", From: 30, Open: true},
			{Key: "toast:x", From: 5, To: 9},
		},
		Events: []AttnEvent{{Kind: EventAsk, Key: "done:a", At: 4}},
	}
	p := pressSpans(log, 60)
	if len(p) != 1 || p[0].Seconds() != 26 {
		t.Fatalf("press spans %v: want 26 s to the next step, the toast is no card", p)
	}
	log.Events = []AttnEvent{{Kind: EventAsk, Key: "", At: 40}}
	if p := pressSpans(log, 60); len(p) != 1 || p[0].Seconds() != 0 {
		t.Fatalf("a press that keeps its card on screen measured %v", p)
	}
	log.Events = []AttnEvent{{Kind: EventAsk, Key: "step:b", At: 40}}
	if p := pressSpans(log, 60); len(p) != 1 || !p[0].Open || p[0].Seconds() != 20 {
		t.Fatalf("a press whose card never came measured %v", p)
	}
}

func TestMetToCreditedPairsEachMetWithItsCredit(t *testing.T) {
	events := []AttnEvent{
		{Kind: EventMet, Key: "step:a", At: 2},
		{Kind: EventMet, Key: "step:a", At: 3},
		{Kind: EventCredited, Key: "step:a", At: 2.5},
		{Kind: EventMet, Key: "step:b", At: 10},
	}
	spans := creditSpans(events, 50)
	if len(spans) != 2 || spans[0].Seconds() != .5 || !spans[1].Open || spans[1].Seconds() != 40 {
		t.Fatalf("credit spans %v", spans)
	}
}

func TestNothingToDoCountsTheCampaignsIdleMarks(t *testing.T) {
	marks := []PaceMark{{At: 0}, {At: 5, Idle: true}, {At: 70}, {At: 80, Idle: true}}
	spans := idleSpans(marks, 100)
	if len(spans) != 2 || spans[0].Seconds() != 65 || !spans[1].Open {
		t.Fatalf("idle spans %v", spans)
	}
	if faults := (Pace{Idles: spans}).Faults(); len(faults) != 1 || faults[0].Rule != RuleIdle {
		t.Fatalf("faults %v: want one over a minute", faults)
	}
}

func TestAnAskedCardStillMayNotShareTheScreenWithABoxItDidNotAnswer(t *testing.T) {
	toast := AttnBox{Kind: BoxToast, Key: "toast:x", Text: "The harbor is quiet.", From: 10, To: 30}
	asked := AttnBox{Kind: BoxStep, Key: "step:b", Text: "Raise a Keep", From: 12, Open: true, Asked: true}
	if faults := checkTurns([]AttnBox{toast, asked}); !slices.ContainsFunc(faults, func(f AttnFault) bool { return f.Rule == RuleTwoAtOnce }) {
		t.Fatalf("an asked card over a toast still on screen passed the one-box rule: %v", faults)
	}
	done := AttnBox{Kind: BoxDone, Key: "done:a", Text: "STEP DONE Raise a hut", From: 10, Face: true, Answered: 12}
	if faults := checkTurns([]AttnBox{done, asked}); len(faults) > 0 {
		t.Fatalf("the card the press asked for, replacing the box it answered, faulted: %v", faults)
	}
}

func TestRepeatedPressesOnACardThatDoesNotTurnAreOneWaitFromTheFirstPress(t *testing.T) {
	log := AttnLog{
		Boxes: []AttnBox{
			{Key: "done:a", From: 0, To: 30},
			{Key: "step:b", From: 30, Open: true},
		},
		Events: []AttnEvent{{Kind: EventAsk, Key: "done:a", At: 5}, {Kind: EventAsk, Key: "done:a", At: 6}, {Kind: EventAsk, Key: "done:a", At: 7}},
	}
	p := pressSpans(log, 60)
	if len(p) != 1 || p[0].From != 5 || p[0].Seconds() != 25 {
		t.Fatalf("press spans %v: want one wait of 25 s from the first press", p)
	}
}

func TestGotItOnACardOverTheStepCardIsAnsweredByTheStepCardStillOnScreen(t *testing.T) {
	log := AttnLog{
		Boxes: []AttnBox{
			{Key: "step:a", From: 0, To: 100},
			{Key: "intro:map", From: 40, To: 40},
			{Key: "done:a", From: 100, To: 120},
		},
		Events: []AttnEvent{{Kind: EventAsk, Key: "intro:map", At: 40}},
	}
	if p := pressSpans(log, 130); len(p) != 1 || p[0].Seconds() != 0 {
		t.Fatalf("GOT IT with the step card on screen measured %v, want 0 s", p)
	}
}

func TestADoneCardIsSomethingToWatchOnlyForItsEnjoyment(t *testing.T) {
	h := New(nil, Backend{})
	done := []Surface{{Box: true, Key: actDone + "door-hearth"}}
	if h.nothingToDo(done, true) {
		t.Fatal("a STEP DONE in its enjoyment counted as nothing to do or watch")
	}
	if !h.nothingToDo(done, false) {
		t.Fatal("a STEP DONE past its enjoyment, with nothing pressed, counted as something to do")
	}
}

func TestAStepWhoseDoneShowsBeforeItsMetReachesTheLogIsCreditedAtOnce(t *testing.T) {
	h := New(nil, Backend{})
	h.SetAttention(true)
	h.Now = 10
	h.noteCredited(ActStep + "pledges")
	h.Now = 10.5
	h.noteMet(ActStep + "pledges")
	spans := creditSpans(h.Attention().Events, 100)
	if len(spans) != 1 || spans[0].Open || spans[0].Seconds() != 0 {
		t.Fatalf("a STEP DONE on screen before the realm's met reached the log measured %v", spans)
	}
}
