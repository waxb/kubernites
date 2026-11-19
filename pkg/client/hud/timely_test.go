package hud

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

const gildedNews = "news:gilded"

func beatOrder(h *HUD, keys ...string) []int {
	out := make([]int, len(keys))
	for i, k := range keys {
		out[i] = slices.IndexFunc(h.Beats(), func(b Beat) bool { return b.Key == k })
	}
	return out
}

func TestATimelyNoticeTakesItsTurnBeforeTheNextStepButAfterTheBoxBeingRead(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	if !shownStep("raise-keep", false)(h.Stage(f)) {
		t.Fatal("the first step never came")
	}
	tick(h, &f, 2)
	lesson := "Your shift now belongs to a crew: change the crew's class and a new shift takes over, hut by hut, with nothing stopped. The old huts carry on until the new ones stand, so not one request is dropped while the class rolls out across the realm."
	f.State.Objectives = []view.ObjectiveView{{ID: "raise-keep", Text: "Raise a Keep", Done: true, Lesson: lesson},
		{ID: "door-hearth", Text: "Set a porter at the Lone Tower's door", Progress: "ready: press DOOR", Target: "cmd:door"}}
	f.Guide.Target = "cmd:door"
	f.Notices = []Notice{{ID: gildedNews, Kind: NoticeNews, Icon: "icon/chronicle", Title: "A traffic spike reaches the harbor",
		Detail: "A traffic spike reaches the harbor: 3 waves of requests wait at its gate. Answer one, and the rest fade fast.", Since: f.Clock, Expires: f.Clock + 600, Timely: true}}
	waitUntil(t, h, &f, paceLimit, "the next step", shownStep("door-hearth", false))
	order := beatOrder(h, actDone+"raise-keep", ActNotice+gildedNews, ActStep+"door-hearth")
	if order[0] < 0 || order[1] < order[0] || order[2] < order[1] {
		t.Fatalf("beats %v: want the step's done box, then the spike's notice (its requests fade in two minutes), then the next step; a step can wait, the spike cannot", beatKeys(h.Beats()))
	}
	noAttentionFaults(t, h)
}

func TestATimelyCardThatMissesItsMomentIsDroppedUnreadAndTheNextOneTellsIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	h.Result("CPU, RATE AND OVERHEAD", "The number is your CPU. The +/S beside it is throughput: the requests your huts answer each second.")
	waitUntil(t, h, &f, paceLimit, "the card being read", func(Frame) bool { return h.Director.Stage.Has(actLesson + "CPU, RATE AND OVERHEAD") })
	docked := true
	const title = "A TRAFFIC SPIKE ARRIVED"
	h.ResultOnceWhile(GildedTaught, title, "Three waves of requests wait at the gate.", nil, func() bool { return docked })
	tick(h, &f, 5)
	docked = false
	tick(h, &f, 1)
	clickOn(t, h, f, LessonAck)
	for range 120 {
		tick(h, &f, .5)
	}
	if order := beatOrder(h, actLesson+title); order[0] >= 0 {
		t.Fatalf("the spike's card came after the spike had left: %v", beatKeys(h.Beats()))
	}
	if slices.Contains(h.Taught(), GildedTaught) {
		t.Fatal("the card was never shown, yet the spike counts as taught: the next spike would come without its card")
	}
	docked = true
	h.ResultOnceWhile(GildedTaught, title, "Three waves of requests wait at the gate.", nil, func() bool { return docked })
	waitUntil(t, h, &f, BoxFloor+CalmGap+1, "the next spike's card", func(Frame) bool { return h.Director.Stage.Has(actLesson + title) })
	if !slices.Contains(h.Taught(), GildedTaught) {
		t.Fatal("the card is on screen but the spike is not marked taught")
	}
	noAttentionFaults(t, h)
}

func TestANoticeTakesNoTurnWhileTheRailIsNotOnScreen(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f.Guide.Staged = true
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	const storm = "news:storm"
	f.Notices = []Notice{{ID: storm, Kind: NoticeNews, Icon: "icon/alert", Title: "A storm",
		Detail: "A storm rolls over the realm: ships wait outside the harbor.", Since: f.Clock, Expires: f.Clock + 600}}
	for range 120 {
		tick(h, &f, .5)
		if h.Director.Stage.Has(ActNotice + storm) {
			t.Fatalf("the notice took its turn while the rail is not on screen: a turn no one can see (%s)", h.StageDebug())
		}
	}
	f.Guide.Introduced = append(f.Guide.Introduced, SurfaceRail)
	waitUntil(t, h, &f, BoxFloor+CalmGap+1, "the notice once the rail shows", func(Frame) bool { return h.Director.Stage.Has(ActNotice + storm) })
	noAttentionFaults(t, h)
}

func TestAGildedShipsNoticeBringsTheRailBeforeItsOwnCardAndShowsAloneOnIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f.Guide.Staged = true
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	const storm = "news:storm"
	f.Notices = []Notice{{ID: storm, Kind: NoticeNews, Icon: "icon/alert", Title: "A storm",
		Detail: "A storm rolls over the realm: ships wait outside the harbor.", Since: f.Clock, Expires: f.Clock + 600},
		{ID: gildedNews, Kind: NoticeNews, Icon: "icon/chronicle", Title: "A traffic spike reaches the harbor",
			Detail: "A traffic spike reaches the harbor: 3 waves of requests wait at its gate. Answer one, and the rest fade fast.", Since: f.Clock, Expires: f.Clock + 600, Timely: true}}
	waitUntil(t, h, &f, BoxFloor+CalmGap+1, "the spike's notice", func(Frame) bool { return h.Director.Stage.Has(ActNotice + gildedNews) })
	staged := h.Stage(f)
	if !h.railShown(staged) {
		t.Fatal("the spike's notice took its turn, but the rail is not on screen: the gleam hint points at nothing")
	}
	ids := []string{}
	for _, b := range h.RailButtons(staged) {
		ids = append(ids, b.ID)
	}
	if !slices.Equal(ids, []string{noticePrefix + gildedNews}) {
		t.Fatalf("before the rail's own card the rail shows %v: only the spike's notice belongs there yet", ids)
	}
	for range 60 {
		tick(h, &f, .5)
		if h.Director.Stage.Has(ActNotice + storm) {
			t.Fatal("the storm's notice took a turn on a rail not yet introduced")
		}
	}
	noAttentionFaults(t, h)
}

func TestAGildedShipsNoticeIsNotHeldBackByAnIntroStillWaitingItsTurn(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetTaught([]string{TeachTopBar})
	h.SetAttention(true)
	f := directorFrame(t)
	f = h.SettleStage(Input{MX: -10, MY: -10}, f, settleLimit)
	f.Guide.Intro = &Intro{ID: "hands", Title: "Hands", Purpose: "Your hands still serve.", Try: "Click the pile.", Target: "surface:hands", Via: ViaAck}
	f.Notices = []Notice{{ID: gildedNews, Kind: NoticeNews, Icon: "icon/chronicle", Title: "A traffic spike reaches the harbor",
		Detail: "A traffic spike reaches the harbor: 3 waves of requests wait at its gate. Answer one, and the rest fade fast.", Since: f.Clock, Expires: f.Clock + 600, Timely: true}}
	waitUntil(t, h, &f, BoxFloor+CalmGap+1, "the spike's notice", func(Frame) bool { return h.Director.Stage.Has(ActNotice + gildedNews) })
	if h.Director.Stage.Has(actIntro + "hands") {
		t.Fatal("the intro came first: the spike's requests fade, the intro can wait")
	}
	if !h.railShown(h.Stage(f)) {
		t.Fatal("the spike notice has its turn but the rail is hidden")
	}
}
