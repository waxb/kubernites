package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheTitlePageOffersEachModeByItsNumberAndOneName(t *testing.T) {
	title := &Title{HasSave: true}
	want := []string{}
	for _, c := range title.choices() {
		want = append(want, c.label)
	}
	if len(want) != 5 || !strings.HasPrefix(want[0], "1  ") || want[1] != "CONTINUE" || !strings.HasPrefix(want[2], "2  ") || want[3] != "3  LIVE CLUSTER" || want[4] != "QUIT" {
		t.Fatalf("title labels = %v", want)
	}
	got := []string{}
	for _, b := range title.Buttons() {
		got = append(got, b.Label)
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("title choices = %v, want %v", got, want)
	}
	for key, kind := range map[ebiten.Key]string{ebiten.Key1: IntentNew, ebiten.Key2: IntentTutorial, ebiten.Key3: IntentLive} {
		if _, ok := hasIntent(title.Update(Input{Keys: []ebiten.Key{key}}), kind); !ok {
			t.Errorf("key %v does not choose %s", key, kind)
		}
	}
	for _, c := range title.choices() {
		if c.label != "QUIT" && c.caption == "" {
			t.Errorf("%s has no caption", c.label)
		}
	}
	learn, _ := HitID(title.Buttons(), TitleLearn)
	intents := title.Update(Input{MX: learn.Rect.X + 2, MY: learn.Rect.Y + 2, Click: true})
	if len(intents) != 1 || intents[0].Kind != IntentTutorial || intents[0].Command != TutorialOpen {
		t.Fatalf("the second mode does not open its tutorial map: %+v", intents)
	}
}

func TestTheMenuOffersLearnKubernetesBesideTrials(t *testing.T) {
	h := New(nil, Backend{})
	f := Frame{}
	buttons := h.MenuButtons(f)
	trials, learn := -1, -1
	for i, b := range buttons {
		switch b.ID {
		case TrialsButton:
			trials = i
		case TutorialButton:
			learn = i
		}
	}
	if trials < 0 || learn != trials+1 {
		t.Fatalf("the tutorial entry at %d, the challenge entry at %d", learn, trials)
	}
	if b := buttons[learn]; b.Label != ui.Settle(learnModeLabel) || b.Rect.Y+b.Rect.H > MenuBox.Y+MenuBox.H {
		t.Fatalf("the entry: %+v", b)
	}
}

func titleText(title *Title) string {
	var out []string
	stop := ui.RecordText(func(r ui.TextRun) { out = append(out, r.Text) })
	title.Draw(ebiten.NewImage(Width, Height))
	stop()
	return strings.Join(out, " ")
}

func TestTheTitleChoicesStandInOneColumnAndTellTheirCaptionAfterTwoSeconds(t *testing.T) {
	title := &Title{HasSave: true}
	buttons := title.Buttons()
	for _, b := range buttons {
		if b.Rect.X != buttons[0].Rect.X || b.Rect.W != buttons[0].Rect.W || b.Rect.X+b.Rect.W/2 != Width/2 {
			t.Errorf("%s at x %d, width %d: every choice stands in one centered column", b.ID, b.Rect.X, b.Rect.W)
		}
	}
	caption := ""
	for _, c := range title.choices() {
		if c.id == TitleLearn {
			caption = c.caption
		}
	}
	if caption == "" {
		t.Fatal("the tutorial choice has no caption")
	}
	if strings.Contains(titleText(title), caption) {
		t.Fatal("a caption shows before anyone hovers")
	}
	learn, _ := HitID(buttons, TitleLearn)
	at := Input{MX: learn.Rect.X + learn.Rect.W/2, MY: learn.Rect.Y + learn.Rect.H/2}
	for range TitleTipTicks - 1 {
		title.Update(at)
	}
	if strings.Contains(titleText(title), caption) {
		t.Fatal("the caption shows before two seconds of hovering")
	}
	title.Update(at)
	if text := titleText(title); !strings.Contains(text, caption) {
		t.Fatalf("two seconds on the tutorial choice show its caption:\n%s", text)
	}
	title.Update(Input{MX: -10, MY: -10})
	if strings.Contains(titleText(title), caption) {
		t.Fatal("the caption stays after the pointer leaves")
	}
}
