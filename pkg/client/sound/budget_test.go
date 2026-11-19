package sound

import (
	"slices"
	"testing"
	"time"
)

var budgetStart = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func at(d time.Duration) time.Time { return budgetStart.Add(d) }

func admitted(b Budget, id SFX, when time.Duration) (Budget, bool) {
	return b.Admit(id, at(when))
}

func TestFeedbackAlwaysAnswersTheHand(t *testing.T) {
	b := NewBudget()
	b, _ = admitted(b, SFXCalamity, 0)
	b, _ = admitted(b, SFXReveal, 100*time.Millisecond)
	for i, id := range []SFX{SFXClick, SFXCoin, SFXBuild, SFXError, SFXRollout, SFXChronicle} {
		var ok bool
		if b, ok = admitted(b, id, time.Duration(200+i)*time.Millisecond); !ok {
			t.Errorf("the hand's %s was held back by the budget", id)
		}
	}
}

func TestAtMostOneChimeEveryFewSeconds(t *testing.T) {
	if ChimeGap < 2*time.Second || ChimeGap > 5*time.Second {
		t.Fatalf("chimes are spaced %v, want a few seconds", ChimeGap)
	}
	b, ok := admitted(NewBudget(), SFXReveal, 0)
	if !ok {
		t.Fatal("the first chime is heard")
	}
	if _, ok = admitted(b, SFXBuilt, ChimeGap-time.Millisecond); ok {
		t.Fatal("a second chime within the gap was heard")
	}
	if _, ok = admitted(b, SFXBuilt, ChimeGap); !ok {
		t.Fatal("a chime after the gap was held back")
	}
}

func TestSimilarCuesAreMerged(t *testing.T) {
	b, _ := admitted(NewBudget(), SFXBoat, 0)
	if _, ok := admitted(b, SFXPodStart, ChimeGap); ok {
		t.Fatal("a hut cue soon after another hut cue was heard, not merged")
	}
	if _, ok := admitted(b, SFXHorn, ChimeGap); !ok {
		t.Fatal("a different kind of chime after the gap was held back")
	}
	if _, ok := admitted(b, SFXPodStart, FamilyGap); !ok {
		t.Fatal("a hut cue after the merge window was held back")
	}
}

func TestAlertsTakePriority(t *testing.T) {
	b, _ := admitted(NewBudget(), SFXReveal, 0)
	b, ok := admitted(b, SFXCalamity, 500*time.Millisecond)
	if !ok {
		t.Fatal("an alert waited behind a chime")
	}
	if _, ok := admitted(b, SFXUnlock, 500*time.Millisecond+ChimeGap-time.Millisecond); ok {
		t.Fatal("a chime crowded in right after an alert")
	}
	if _, ok := admitted(b, SFXTremor, 2*time.Second); ok {
		t.Fatal("two alerts two seconds apart were not merged")
	}
	if _, ok := admitted(b, SFXTremor, 500*time.Millisecond+AlertGap); !ok {
		t.Fatal("an alert after the alert window was held back")
	}
}

func TestABatchLeadsWithItsMostImportantCue(t *testing.T) {
	_, got := NewBudget().Pick([]SFX{SFXBoat, SFXReveal, SFXCalamity, SFXClick, SFXBoat}, at(0))
	want := []SFX{SFXCalamity, SFXClick}
	if !slices.Equal(got, want) {
		t.Fatalf("heard %v, want %v", got, want)
	}
	_, got = NewBudget().Pick([]SFX{SFXBoat, SFXPodStart, SFXUnlock, SFXCoin}, at(0))
	if want := []SFX{SFXUnlock, SFXCoin}; !slices.Equal(got, want) {
		t.Fatalf("heard %v, want %v", got, want)
	}
}

func TestOnePromotionGetsOneFanfare(t *testing.T) {
	b, got := NewBudget().Pick([]SFX{SFXVictory, SFXVictory, SFXReveal}, at(0))
	if !slices.Equal(got, []SFX{SFXVictory}) {
		t.Fatalf("heard %v, want one fanfare", got)
	}
	if _, ok := admitted(b, SFXVictory, 3*time.Second); ok {
		t.Fatal("a second fanfare three seconds later was heard")
	}
}

func TestTheHushLetsOnlyTheHandAndTheFanfareThrough(t *testing.T) {
	b := NewBudget().Hushed(at(0), at(time.Minute))
	for _, id := range []SFX{SFXCalamity, SFXTremor, SFXReveal, SFXBoat, SFXBuilt, SFXHorn} {
		if _, ok := admitted(b, id, 10*time.Second); ok {
			t.Errorf("%s was heard during the hush", id)
		}
	}
	for _, id := range []SFX{SFXVictory, SFXClick, SFXCoin} {
		if _, ok := admitted(b, id, 10*time.Second); !ok {
			t.Errorf("%s was held back by the hush", id)
		}
	}
	if _, ok := admitted(b, SFXReveal, time.Minute); !ok {
		t.Error("the hush outlasted its end")
	}
}

func TestTheArrivalKeepsItsFanfare(t *testing.T) {
	b, _ := admitted(NewBudget(), SFXVictory, 0)
	b = b.Hushed(at(5*time.Second), at(80*time.Second))
	b, ok := admitted(b, SFXVictory, 5*time.Second)
	if !ok {
		t.Fatal("the promotion's fanfare was swallowed by a fanfare five seconds before the calm")
	}
	b = b.Hushed(at(5*time.Second+100*time.Millisecond), at(80*time.Second))
	if _, ok := admitted(b, SFXVictory, 6*time.Second); ok {
		t.Fatal("a second fanfare in the same calm was heard")
	}
}

func TestTheManagerGivesTheCalmItsFanfare(t *testing.T) {
	m, out, clk := fakeManager()
	m.PlaySFX(SFXVictory)
	clk.advance(5 * time.Second)
	m.Hush(75 * time.Second)
	m.PlaySFX(SFXVictory)
	if _, shots := out.counts(); shots != 2 {
		t.Fatalf("%d fanfares started, want the earlier one and the promotion's", shots)
	}
	clk.advance(time.Second)
	m.Hush(74 * time.Second)
	m.PlaySFX(SFXVictory)
	if _, shots := out.counts(); shots != 2 {
		t.Fatal("the calm heard a second fanfare")
	}
}

func TestImportantCuesDuck(t *testing.T) {
	for _, id := range []SFX{SFXCalamity, SFXVictory, SFXReveal, SFXUnlock} {
		if !Ducks(id) {
			t.Errorf("%s does not duck the music", id)
		}
	}
	for _, id := range []SFX{SFXClick, SFXCoin, SFXBoat, SFXPodStart} {
		if Ducks(id) {
			t.Errorf("%s ducks the music", id)
		}
	}
}

func TestEveryEffectHasACueClass(t *testing.T) {
	for _, id := range AllSFX() {
		if _, ok := cueClasses[id]; !ok {
			t.Errorf("%s has no cue class", id)
		}
	}
}

func TestTheManagerSpendsTheBudget(t *testing.T) {
	m, out, clk := fakeManager()
	m.PlaySFX(SFXBoat)
	clk.advance(time.Second)
	m.PlaySFX(SFXPodStart)
	m.PlaySFX(SFXReveal)
	if _, shots := out.counts(); shots != 1 {
		t.Fatalf("%d effects started, want the first chime only", shots)
	}
	m.PlaySFX(SFXClick)
	clk.advance(ChimeGap)
	m.PlaySFX(SFXReveal)
	if _, shots := out.counts(); shots != 3 {
		t.Fatalf("%d effects started, want the chime, the click and the later reveal", shots)
	}
}

func TestPlayAllLeadsWithTheAlert(t *testing.T) {
	m, out, _ := fakeManager()
	m.PlayAll(SFXBoat, SFXBuilt, SFXCalamity)
	if _, shots := out.counts(); shots != 1 || m.effects[0].id != SFXCalamity {
		t.Fatalf("%d effects started, first %v; want the alert alone", shots, m.effects)
	}
}

func TestTheHushKeepsTheWorldQuiet(t *testing.T) {
	m, out, clk := fakeManager()
	m.Hush(time.Minute)
	m.PlayAll(SFXVictory, SFXBoat, SFXCalamity, SFXClick)
	if _, shots := out.counts(); shots != 2 {
		t.Fatalf("%d effects started under the hush, want the fanfare and the click", shots)
	}
	clk.advance(time.Minute)
	m.Hush(0)
	m.PlaySFX(SFXCalamity)
	if _, shots := out.counts(); shots != 3 {
		t.Fatal("the alert was held after the hush ended")
	}
}

func TestMusicDucksUnderImportantCues(t *testing.T) {
	m, out, clk := fakeManager()
	m.PlayMusic(MusicDark)
	clk.advance(crossfade)
	m.Update()
	music := out.loopAt(0)
	m.PlaySFX(SFXBoat)
	clk.advance(DuckAttack)
	m.Update()
	if vol, _, _ := music.state(); !near(vol, defaultMusicVolume) {
		t.Fatalf("a hut chime ducked the music to %v", vol)
	}
	clk.advance(ChimeGap)
	m.PlaySFX(SFXCalamity)
	clk.advance(DuckAttack)
	m.Update()
	if vol, _, _ := music.state(); !near(vol, defaultMusicVolume*DuckDepth) {
		t.Fatalf("under an alert the music plays at %v, want %v", vol, defaultMusicVolume*DuckDepth)
	}
	clk.advance(cueLength(SFXCalamity) + DuckRelease)
	m.Update()
	if vol, _, _ := music.state(); !near(vol, defaultMusicVolume) {
		t.Fatalf("after the alert the music stays at %v", vol)
	}
}

func TestFeedbackNamesTheHandsOwnCues(t *testing.T) {
	if !Feedback(SFXClick) || !Feedback(SFXCoin) || Feedback(SFXBoat) || Feedback(SFXCalamity) || Feedback(SFXVictory) {
		t.Fatal("feedback cues are the hand's own: click and coin, not chimes, alerts or the fanfare")
	}
}

func TestTheManagerReportsItsHush(t *testing.T) {
	m, _, clk := fakeManager()
	if m.Hushed() {
		t.Fatal("a new manager is hushed")
	}
	m.Hush(10 * time.Second)
	if !m.Hushed() {
		t.Fatal("the hush is not reported")
	}
	clk.advance(10 * time.Second)
	if m.Hushed() {
		t.Fatal("the hush outlived its span")
	}
	var none *Manager
	none.Hush(time.Second)
	if none.Hushed() {
		t.Fatal("a nil manager reports a hush")
	}
}
