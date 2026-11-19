package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func directorFrame(t *testing.T) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "raise-keep", Text: "Raise a Keep", Progress: "ready: press BUILD", Build: "keep"}}
	f.Guide.Target = "build:keep"
	f.State.Next = &view.NextView{Label: "Stone Keep", Progress: 1, Ready: true, Build: "keep"}
	return f
}

func speakingVoices(voices []string) []string {
	out := []string{}
	for _, v := range voices {
		kind, _, _ := strings.Cut(v, ":")
		if kind == VoiceIntro || kind == VoiceObjective || kind == VoiceNext || kind == "advice" || kind == "lesson" {
			out = append(out, v)
		}
	}
	return out
}

func tick(h *HUD, f *Frame, seconds float64) {
	f.Clock += seconds
	f.State.RealmTime += seconds
	f.EventClock = f.Clock
	h.Update(Input{MX: -10, MY: -10}, *f)
}

func TestOneVoiceSpeaksInTheOrderIntroObjectiveNext(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.Guide.Intro = &Intro{ID: "build", Title: "Build", Try: "Press BUILD.", Target: "cmd:build", Via: ViaOpen}
	if got := speakingVoices(h.Voices(f)); len(got) != 1 || h.Voice(f) != VoiceIntro {
		t.Fatalf("with an intro: voice %q, speaking %v", h.Voice(f), got)
	}
	f.Guide.Intro = nil
	if got := speakingVoices(h.Voices(f)); len(got) != 1 || h.Voice(f) != VoiceObjective || !h.NextQuiet(f) {
		t.Fatalf("with a step: voice %q, speaking %v", h.Voice(f), got)
	}
	f.State.Objectives = nil
	if h.Voice(f) != VoiceNext || h.GuideGoal(f) != "build:keep" || NextLine(f) != "" {
		t.Fatalf("with only NEXT: voice %q goal %q advice %q", h.Voice(f), h.GuideGoal(f), NextLine(f))
	}
	if got := h.StepID(f); got != "cmd:build" {
		t.Fatalf("NEXT walks its path from BUILD, got %q", got)
	}
	f.State.Next.Ready = false
	if h.Voice(f) != "" || h.GuideGoal(f) != "" {
		t.Fatalf("a NEXT not yet in reach stays quiet: %q", h.Voice(f))
	}
}

func TestTheNextIntroWaitsTheIdeaGapAfterTheLastCard(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f.Guide.Intro = &Intro{ID: "build", Title: "Build", Try: "Press BUILD.", Target: "cmd:build", Via: ViaOpen}
	tick(h, &f, 1)
	first := waitUntil(t, h, &f, paceLimit, "the first card", func(s Frame) bool { return h.IntroShown(s) })
	if h.Voice(h.Stage(f)) != VoiceIntro {
		t.Fatal("the first card does not speak")
	}
	f.Guide.Intro = &Intro{ID: "door", Title: "Door", Try: "Press it.", Target: "cmd:door", Via: ViaAct}
	tick(h, &f, .1)
	if s := h.Stage(f); h.IntroShown(s) || !h.ObjectiveShown(s) {
		t.Fatal("the next card took the place of the one just answered")
	}
	f.State.Objectives[0].Progress = "being built"
	next := waitUntil(t, h, &f, paceLimit, "the next card", func(s Frame) bool { return h.IntroShown(s) })
	if next-first < IdeaGapDark {
		t.Fatalf("the next card came %.1f s after the last, inside the first window's %.0f s between ideas", next-first, IdeaGapDark)
	}
	if h.Voice(h.Stage(f)) != VoiceIntro {
		t.Fatal("once it comes, the next card speaks")
	}
	f.Guide.Intro = nil
	tick(h, &f, .1)
	if s := h.Stage(f); !h.ObjectiveShown(s) || h.Voice(s) != VoiceObjective {
		t.Fatal("once the card is answered the step speaks again")
	}
	noAttentionFaults(t, h)
}

func TestAnnouncementsComeOneAtATimeAndRoutineNewsStaysInTheLog(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetAttention(true)
	f := directorFrame(t)
	h.SetTaught([]string{TeachTopBar})
	tick(h, &f, 1)
	h.Announce(AnnounceBuilt, "The Codex stands complete", false)
	h.Announce(AnnounceCalamity, "Tremor!", true)
	h.Announce(AnnounceSeasoned, "The porters are seasoned", false)
	h.Announce(AnnounceSeasoned, "The clerks are seasoned", false)
	h.Announce(AnnounceGate, "The crews carry every box", false)
	seen := []string{}
	for range 4 * 60 {
		tick(h, &f, .25)
		for _, toast := range h.Toasts() {
			if !slices.Contains(seen, toast.Text) {
				seen = append(seen, toast.Text)
			}
		}
	}
	if want := []string{"The crews carry every box"}; !slices.Equal(seen, want) {
		t.Fatalf("toasts %v, want %v: buildings, troubles and seasoned crews are log lines", seen, want)
	}
	noAttentionFaults(t, h)
}

func TestFeedbackIsImmediateAndTheNextAnnouncementTakesItsTurn(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	tick(h, &f, 5)
	h.Toast("Not enough memory", true)
	h.Announce(AnnounceGate, "The crews carry every box", false)
	tick(h, &f, .2)
	if got := h.Toasts(); len(got) != 1 || got[0].Text != "Not enough memory" {
		t.Fatalf("the click's answer is not at once, or the announcement came with it: %+v", got)
	}
	waitUntil(t, h, &f, paceLimit, "the announcement", func(Frame) bool {
		return slices.ContainsFunc(h.Toasts(), func(t Toast) bool { return t.Text == "The crews carry every box" })
	})
}

func TestMergedAnnouncementsCountTheirKind(t *testing.T) {
	cases := []struct {
		kind  string
		texts []string
		want  string
	}{
		{AnnounceBuilt, []string{"A stands complete"}, "A stands complete"},
		{AnnounceSeasoned, []string{"a", "b"}, "2 crews are seasoned: they carry faster"},
		{"other", []string{"first", "second", "third"}, "first (and 2 more)"},
		{"other", nil, ""},
	}
	for _, c := range cases {
		if got, _ := MergeAnnouncements(c.kind, c.texts, false); got != c.want {
			t.Errorf("%s %v = %q, want %q", c.kind, c.texts, got, c.want)
		}
	}
}

func TestBannersQueueAndStepAsideFromTheStep(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	tick(h, &f, 1)
	h.Banner("KEEP NODE-A CORDONED")
	h.BannerCause("HARBOR TIER 3", "Carracks dock now")
	waitUntil(t, h, &f, paceLimit, "the first banner", func(Frame) bool { return h.Juice.Banner == "KEEP NODE-A CORDONED" && h.BannerActive() })
	if len(h.Director.Banners) != 1 {
		t.Fatalf("the second banner does not wait its turn: %d waiting", len(h.Director.Banners))
	}
	waitUntil(t, h, &f, paceLimit, "the second banner", func(Frame) bool { return h.Juice.Banner == "HARBOR TIER 3" && h.BannerActive() })
	f.State.Objectives = []view.ObjectiveView{{ID: "o", Text: "Look at the pod", Target: "pod:default/x"}}
	f.Guide.Target = "pod:default/x"
	f.Beacons = []ui.Rect{{X: Width/2 - 10, Y: BannerY - 10, W: 20, H: 20}}
	r, ok := h.BannerRect(f)
	if !ok || Overlaps(r, f.Beacons[0]) {
		t.Fatalf("the banner %+v covers the beacon %+v", r, f.Beacons[0])
	}
}

func TestTheTrayListsWhatWaitsInOrder(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.Guide.Intro = &Intro{ID: "build", Title: "Build", Try: "Press BUILD.", Target: "cmd:build", Via: ViaOpen}
	f.Guide.Queue = []string{"Porter at the Door", "The harbor tier"}
	tick(h, &f, 1)
	b, ok := HitID(h.Buttons(f), IntroTray)
	if !ok || b.Label != TrayLabel(2) {
		t.Fatalf("tray button = %+v %v", b, ok)
	}
	if _, open := h.TrayRect(f); open {
		t.Fatal("the tray starts closed")
	}
	clickOn(t, h, f, IntroTray)
	r, open := h.TrayRect(f)
	if !open || Overlaps(r, h.IntroCard(f)) {
		t.Fatalf("the tray opens beside the card: %+v", r)
	}
	for _, target := range h.GuideTargets(f) {
		if Overlaps(r, target) {
			t.Fatalf("the tray %+v covers the step %+v", r, target)
		}
	}
	lines := h.TrayLines(f)
	if !slices.Contains(lines, "1. Porter at the Door") || !slices.Contains(lines, "2. The harbor tier") {
		t.Fatalf("tray lines = %v", lines)
	}
	f.Guide.Intro = &Intro{ID: "door", Title: "Door", Try: "Press it.", Target: "cmd:door", Via: ViaAct}
	tick(h, &f, .1)
	if h.Director.TrayOpen {
		t.Error("the tray closes when the next intro comes")
	}
}

func TestVoicesNameEveryElementOnScreen(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := directorFrame(t)
	f.State.Next = nil
	f.State.Objectives = nil
	h.Now = 10
	h.Lesson("Cordon Keep node-a", "A Keep is a Node.")
	h.PushToast("The Codex stands complete", false)
	h.Juice.Banner, h.Juice.BannerAt = "KEEP NODE-A CORDONED", h.Now
	voices := h.Voices(f)
	for _, want := range []string{"advice:" + f.State.Advice, "lesson:Cordon Keep node-a", "toast:The Codex stands complete", "banner:KEEP NODE-A CORDONED"} {
		if !slices.Contains(voices, want) {
			t.Errorf("voices %v lack %q", voices, want)
		}
	}
}

func TestTheTrayAndMapButtonDraw(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := feudalIntroFrame(t, "overlays", SurfaceOverlays, ViaAck)
	f.Guide.Queue = []string{"a", "b", "c", "d", "e", "f", "g"}
	h.Director.TrayOpen = true
	if lines := h.TrayLines(f); len(lines) != TrayRows+2 || lines[len(lines)-1] != "and 2 more" {
		t.Fatalf("a long queue is cut short: %v", lines)
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	if _, ok := h.TrayRect(f); !ok {
		t.Fatal("the open tray is drawn")
	}
	if !h.Captures(LensButtonRect().X+1, LensButtonRect().Y+1, f) {
		t.Error("the map button takes its click")
	}
}

func TestTheObjectiveCardStaysClearOfTheOpenConsole(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "o", Text: "Look at the pod", Target: "pod:default/x"}}
	f.Guide.Target = "pod:default/x"
	f.Beacons = []ui.Rect{{X: 20, Y: TopH + 20, W: 30, H: 30}, {X: Width - 360, Y: TopH + 20, W: 30, H: 30}}
	h.OpenStone("")
	if h.Stone.Tall || !h.CardsDrawn(f) {
		t.Fatal("the short console leaves the cards drawn")
	}
	r := h.ObjectiveRectOf(f)
	if Overlaps(r, h.StoneRect()) {
		t.Fatalf("the step card %+v sits on the console %+v", r, h.StoneRect())
	}
	if r.X != Objective.X || r.Y != Objective.Y {
		t.Fatalf("the step card %+v left its home %+v for a beacon", r, Objective)
	}
	h.Lesson("A hint", "Something happened.")
	if l := h.LessonRect(f); Overlaps(l, h.StoneRect()) {
		t.Fatalf("the hint card %+v sits on the console", l)
	}
}

func TestTheArrivalCalmHoldsAnnouncementsUntilItEnds(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	f := directorFrame(t)
	f.State.Calm = 30
	tick(h, &f, 5)
	h.Announce(AnnounceBuilt, "Stone Keep 2 stands complete", false)
	h.Announce(AnnounceGate, "The crews carry every box", false)
	h.Toast("Not enough memory", true)
	tick(h, &f, .2)
	if got := h.Toasts(); len(got) != 1 || got[0].Text != "Not enough memory" || h.Pending() != 1 {
		t.Fatalf("during the calm: toasts %+v, pending %d; want the hand's feedback alone", got, h.Pending())
	}
	tick(h, &f, 10)
	if h.Pending() != 1 {
		t.Fatalf("the calm let %d announcements through", 1-h.Pending())
	}
	f.State.Calm = 0
	waitUntil(t, h, &f, paceLimit, "the held announcement after the calm", func(Frame) bool {
		got := h.Toasts()
		return len(got) > 0 && got[len(got)-1].Text == "The crews carry every box"
	})
}

func TestABannerWaitsWhileACalamityCardIsOpen(t *testing.T) {
	h, f := calamityCase(t)
	h.FeatBanner("PORTERS: CLASS 2.0", "3 huts rebuilt one by one: +3 requests a second more than before")
	tick(h, &f, .3)
	if _, ok := h.CalamityRect(f); !ok {
		t.Fatal("the calamity card is open")
	}
	for range 4 * 30 {
		tick(h, &f, .25)
		if _, shown := h.BannerRect(h.Stage(f)); shown {
			if card, open := h.CalamityRect(h.Stage(f)); open {
				t.Fatalf("the banner came beside the open calamity card %+v: one box at a time", card)
			}
		}
	}
}

func TestNextStaysQuietWhileAFinishedStepSinksIn(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	f.State.Objectives = nil
	if h.Voice(f) != VoiceNext {
		t.Fatalf("with only NEXT the voice is %q", h.Voice(f))
	}
	f.State.Soaking = true
	if h.Voice(f) != "" {
		t.Fatalf("NEXT spoke during the soak after a finished step: %q", h.Voice(f))
	}
}
