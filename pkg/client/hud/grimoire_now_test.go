package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

const musterNow = "Your 4 porters are kept by hand: a ReplicaSet keeps them whole."

func nowFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	pages := append([]view.GrimoirePage(nil), f.State.Grimoire...)
	pages[0].Now = musterNow
	pages = append(pages, view.GrimoirePage{ID: "augury", Name: "Augury", K8s: "metrics-server", State: "available", MaxRank: 1,
		Cost: view.Cost{Gold: 80, Wisdom: 100}, Now: "The metrics page reads what every worker really uses."})
	f.State.Grimoire = pages
	return f
}

func TestAPageCardSaysWhatItWouldDoForThisRealm(t *testing.T) {
	f := nowFrame(t)
	l := LayoutKnowledge(f.State.Grimoire, KmapRowsTiles)
	p, ok := l.Find("muster")
	if !ok {
		t.Fatal("no ReplicaSet plaque")
	}
	rows := KnowledgeCard(p, f)
	found := false
	for i, r := range rows {
		if r.Kind == RowNow {
			found = strings.Contains(r.Text, musterNow) && i <= 3
		}
	}
	if !found {
		t.Fatalf("card rows = %+v", rows)
	}
	if !PageCalls(p) {
		t.Fatal("a learnable page with work to do does not call")
	}
}

func TestTheWisdomTipShowsTheWayToTheNextPage(t *testing.T) {
	f := nowFrame(t)
	f.State.Gold, f.State.Wisdom = 30, 4
	tip := WisdomTip(f)
	if !strings.Contains(tip, "Next page: Replicas I: Novice: 4/10 wisdom, 30/60 gold") || !strings.Contains(tip, musterNow) {
		t.Fatalf("points tip = %q", tip)
	}
	f.State.Gold, f.State.Wisdom = 100, 20
	if tip := WisdomTip(f); !strings.Contains(tip, "Replicas I: Novice is yours to learn") {
		t.Fatalf("a ready page's tip = %q", tip)
	}
}

func TestNextNamesWhatAPageWouldDo(t *testing.T) {
	f := nowFrame(t)
	n := &view.NextView{Kind: view.NextPage, ID: "muster", Label: "Replicas", Gold: 60, Wisdom: 10}
	if tip := NextTip(n, f.State); !strings.Contains(tip, "For your realm: "+musterNow) {
		t.Fatalf("NEXT page tip = %q", tip)
	}
}

func TestTheGrimoireOpensWithThePageThatAnswersANeedInSight(t *testing.T) {
	f := nowFrame(t)
	f.State.Age = view.AgeEmpire
	h := New(nil, Backend{})
	h.Open(OverlayGrimoire)
	l := h.Knowledge(f)
	p, _ := l.Find("muster")
	if _, seen := clipRect(h.ScreenOf(p.Rect()), KnowledgeView); !seen || h.Grimoire != "" {
		t.Fatalf("the calling page is out of sight (scroll %v) or its card forced open (%q)", h.KnowledgeScroll(), h.Grimoire)
	}
}

func TestNextOpensTheGrimoireOnItsPage(t *testing.T) {
	f := nowFrame(t)
	f.State.Next = &view.NextView{Kind: view.NextPage, ID: "augury", Label: "Augury", Gold: 80, Wisdom: 100, Target: "cmd:grimoire"}
	h := New(nil, Backend{})
	h.PressNext(f)
	h.Knowledge(f)
	if h.Grimoire != "augury" {
		t.Fatalf("NEXT opened the manual on %q", h.Grimoire)
	}
}

func grimoireCell(t *testing.T, f Frame) Button {
	t.Helper()
	for _, e := range mainEntries {
		if e.button.ID == "cmd:grimoire" {
			return mainButton(e.button, f)
		}
	}
	t.Fatal("no manual button")
	return Button{}
}

func TestTheGrimoireButtonShowsTheWayToTheNextPage(t *testing.T) {
	f := nowFrame(t)
	f.State.Gold, f.State.Wisdom = 30, 4
	b := grimoireCell(t, f)
	if b.Note != "40%" || b.Cost != "" || !strings.Contains(b.Tip, "Next page: Replicas I: Novice") || !strings.Contains(b.Tip, musterNow) || !strings.Contains(b.Tip, "40%: you hold 40% of its price") {
		t.Fatalf("on the way: note %q cost %q tip %q", b.Note, b.Cost, b.Tip)
	}
	f.State.Gold, f.State.Wisdom = 100, 20
	if b := grimoireCell(t, f); b.Cost != "!" || b.Note != "" {
		t.Fatalf("an entry that answers a need is ready: note %q cost %q", b.Note, b.Cost)
	}
}
