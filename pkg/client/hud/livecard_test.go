package hud

import (
	"errors"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func cardLines(c LiveCard) (string, string) {
	var texts []string
	var tip string
	for _, l := range c.lines() {
		texts = append(texts, l.text)
		if l.id == liveRefusedID {
			tip = l.tip
		}
	}
	return strings.Join(texts, "\n"), tip
}

func TestTheLiveCardSaysWhoYouAreAndWhatYouMayNotSee(t *testing.T) {
	c := LiveCard{Context: "lab-west", Server: "https://lab-west.example.internal:6443", Who: "reader@example.com", Keeps: 3, Huts: 4, Fiefs: 1, Isles: 1,
		Refused: []string{"pods", "services"}, Namespace: "team-a", Denied: []string{"nodes"}}
	defer ui.SetNamer(nil)
	ui.SetNamer(lexicon.TrueNames)
	all, tip := cardLines(c)
	if !strings.Contains(all, "AS reader@example.com") || !strings.Contains(all, "ONLY team-a: pods, services, nodes") {
		t.Errorf("card lines:\n%s", all)
	}
	for _, want := range []string{"reader@example.com may not list pods, services across the cluster", "only namespace team-a was read", "may not list nodes at all"} {
		if !strings.Contains(tip, want) {
			t.Errorf("the refusal tip %q does not say %q", tip, want)
		}
	}
	ui.SetNamer(nil)
	all, tip = cardLines(c)
	for _, text := range []string{all, tip} {
		text = strings.NewReplacer(c.Context, "", c.Server, "").Replace(text)
		if leak := lexicon.KubernetesWords.FindString(text); leak != "" {
			t.Errorf("story mode names %q in %q", leak, text)
		}
	}
	if !strings.Contains(all, "ONLY THE FIEF team-a: huts, depots, keeps") || !strings.Contains(tip, "reader@example.com may not see the huts, depots of every fief") {
		t.Errorf("story card:\n%s\n%s", all, tip)
	}
	if !LiveReadButton(liveRefusedID) || !LiveReadButton(liveIdleID) {
		t.Error("the card's tips are gated in a live realm")
	}
	if got := 6 + ui.LineHeight + 2 + len(c.lines())*ui.LineHeight; got > CardBox.H {
		t.Errorf("the card needs %d pixels, it has %d", got, CardBox.H)
	}
}

func TestIdleShrinesAreListedOnTheLiveCard(t *testing.T) {
	c := LiveCard{Idle: []string{"kube-system/a", "kube-system/b", "kube-system/c", "kube-system/d", "kube-system/e", "kube-system/f", "kube-system/g"}, Trouble: 1,
		Refused: []string{"pods"}, Namespace: "team-a", Who: "x"}
	var idle liveLine
	for _, l := range c.lines() {
		if l.id == liveIdleID {
			idle = l
		}
	}
	if idle.text != "7 SHRINES STAND NOWHERE" || !strings.Contains(idle.tip, "kube-system/a") || !strings.Contains(idle.tip, "and 1 more") {
		t.Errorf("idle line %+v", idle)
	}
	if got := 6 + ui.LineHeight + 2 + len(c.lines())*ui.LineHeight; got > CardBox.H {
		t.Errorf("a full card needs %d pixels, it has %d", got, CardBox.H)
	}
}

func TestThePickerShowsWhoTheClusterSaysYouAre(t *testing.T) {
	p := LivePicker{}
	p.Open(nil)
	p.ShowContexts([]LiveContext{{Name: "a"}, {Name: "b"}}, nil)
	p.Chosen = 1
	intents := p.press(liveChoose)
	if p.Step != LiveStepConfirm || len(intents) != 2 || intents[1].Kind != IntentLiveWho || intents[1].Target != "b" {
		t.Fatalf("CHOOSE: step %q, intents %+v", p.Step, intents)
	}
	if line, _ := p.Who.line(); !strings.Contains(line, "asking") {
		t.Errorf("while asking: %q", line)
	}
	p.Answered("a", "stale@example.com", nil, nil)
	if !p.Who.Asked {
		t.Error("an answer for another context was taken")
	}
	p.Answered("b", "", nil, errors.New("the server could not find the requested resource"))
	if line, _ := p.Who.line(); !strings.Contains(line, "did not say") || p.Who.Asked {
		t.Errorf("after a refusal: %q", line)
	}
	p.press(liveBack)
	p.press(liveChoose)
	p.Answered("b", "player@example.com", []string{"system:authenticated"}, nil)
	if line, _ := p.Who.line(); line != "player@example.com" {
		t.Errorf("after the answer: %q", line)
	}
}

func TestTheLiveCardSaysHowManyOfItsHutsStandAtShrines(t *testing.T) {
	c := LiveCard{Keeps: 9, Huts: 58, Fiefs: 9, Isles: 2, Shrines: 21}
	defer ui.SetNamer(nil)
	for _, mode := range []struct {
		namer func(string) string
		want  string
	}{{nil, "58 HUTS IN 9 FIEFS, 21 AT SHRINES"}, {lexicon.TrueNames, "58 PODS IN 9 NAMESPACES, 21 OF DAEMONSETS"}} {
		ui.SetNamer(mode.namer)
		found := false
		for _, l := range c.lines() {
			text := ui.Name(l.text)
			found = found || text == mode.want
			if w := (ui.Font{}).Width(text, 1); w > CardBox.W-16 {
				t.Errorf("%q is %d pixels wide, the card has %d", text, w, CardBox.W-16)
			}
		}
		if !found {
			all, _ := cardLines(c)
			t.Errorf("the card does not say %q:\n%s", mode.want, ui.Name(all))
		}
	}
	ui.SetNamer(nil)
	c.Shrines = 0
	if all, _ := cardLines(c); !strings.Contains(all, "58 HUTS IN 9 FIEFS\n") && !strings.HasSuffix(all, "58 HUTS IN 9 FIEFS") || strings.Contains(all, "SHRINES") {
		t.Errorf("a realm with no shrines still counts them:\n%s", all)
	}
}
