package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	VoiceIntro     = "intro"
	VoiceObjective = "objective"
	VoiceNext      = "next"
	readGrace      = 12.0
	bannerHalfW    = 120
	bannerHalfH    = 20
	causeGap       = 24
	causeRows      = 3
)

type director struct {
	intro    string
	waitSeen string
	TrayOpen bool `json:"-"`
	inbox    []announcement
	nextOut  float64
	Banners  []bannerItem `json:"-"`
	window   string
	windowAt float64
	step     string
	Stage    stageState `json:"-"`
}

func (h *HUD) direct(f Frame) {
	id := ""
	if f.Guide.Intro != nil {
		id = f.Guide.Intro.ID
	}
	if h.Director.intro != "" && id != h.Director.intro {
		h.Director.TrayOpen = false
	}
	if id != "" && id != h.Director.intro {
		h.bringIntroHome(f)
	}
	h.Director.intro = id
	if h.Director.window != h.OverlayID {
		h.Director.windowAt = h.Now
	}
	h.Director.window = h.OverlayID
	if f.State.Calm <= 0 {
		h.releaseAnnouncement()
	}
	if !h.Director.Stage.Primed && !h.BannerActive() {
		h.showBanner()
	}
	h.pruneShelved(f.RawFrame())
	h.manageStage(f)
}

func (h *HUD) bringIntroHome(f Frame) {
	if _, main := MainHomes[f.Guide.Intro.Target]; main && h.PageID != PageMain && f.Placing == "" {
		h.PageID = PageMain
	}
}

func (h *HUD) Voice(f Frame) string {
	v := h.candidateVoice(f)
	if v != "" && h.leavesWindow(f, h.voiceGoal(f, v)) {
		return ""
	}
	return v
}

func (h *HUD) candidateVoice(f Frame) string {
	switch {
	case f.Unguided || placingGuideTarget(f):
		return ""
	case f.Introducing() && h.OverlayID == OverlayWelcome:
		return ""
	case f.Introducing():
		return VoiceIntro
	case f.Guide.Held != "":
		return ""
	}
	if _, ok := f.current(); ok {
		return VoiceObjective
	}
	if f.State.Soaking {
		return ""
	}
	if NextSpeaks(f) {
		return VoiceNext
	}
	return ""
}

func (h *HUD) voiceGoal(f Frame, v string) string {
	switch v {
	case VoiceIntro:
		return gridGoal(h.chooserGoal(f.Guide.Intro.Target, f), f)
	case VoiceObjective:
		return gridGoal(h.chooserGoal(ObjectiveGoal(f), f), f)
	}
	return gridGoal(NextGoal(f), f)
}

func (h *HUD) noteReading(in Input) {
	if h.OverlayID != OverlayNone && (in.Click || in.Wheel != 0 || len(in.Keys) > 0 || len(in.Chars) > 0) {
		h.Director.windowAt = h.Now
	}
}

func (h *HUD) reading() bool {
	return h.OverlayID != OverlayNone && h.OverlayID != OverlayWelcome && h.Director.window == h.OverlayID && h.Now-h.Director.windowAt < readGrace
}

func (h *HUD) leavesWindow(f Frame, goal string) bool {
	if !h.reading() || goal == "" {
		return false
	}
	path := NavPath(goal, h.NavState(f))
	return len(path) > 0 && path[0] == closeID(h.OverlayID)
}

func NextSpeaks(f Frame) bool {
	_, objective := f.current()
	return !objective && f.Guide.Held == "" && !f.Introducing() && NextShown(f) && BoardShown(f) && f.State.Next.Ready && nextOnScreen(f, NextGoal(f))
}

func nextOnScreen(f Frame, goal string) bool {
	switch {
	case goal == "":
		return false
	case strings.HasPrefix(goal, "build:"):
		return f.Shown("cmd:build")
	case strings.HasPrefix(goal, "cmd:") || strings.HasPrefix(goal, "surface:") || strings.HasPrefix(goal, "meter:"):
		return f.Shown(goal)
	}
	return true
}

func NextGoal(f Frame) string {
	n := f.State.Next
	switch {
	case n == nil || n.Label == "":
		return ""
	case n.Command != "":
		return nextRow
	case n.Build != "":
		return "build:" + n.Build
	}
	return n.Target
}

func (h *HUD) NextQuiet(f Frame) bool { return h.Voice(f) != VoiceNext }

func (h *HUD) BannerRect(f Frame) (ui.Rect, bool) {
	_, ok := BannerOffset(h.Now - h.Juice.BannerAt)
	if !ok || h.Juice.Banner == "" {
		return ui.Rect{}, false
	}
	cx, top := h.keptBannerSpot(f)
	return h.bannerAt(cx, top), true
}

func (h *HUD) keptBannerSpot(f Frame) (int, int) {
	if h.Juice.spotSet && h.Juice.spotFor == h.Juice.BannerAt && !OverlapsAny(h.bannerAt(h.Juice.spot[0], h.Juice.spot[1]), h.bannerPanels(f)) {
		return h.Juice.spot[0], h.Juice.spot[1]
	}
	return h.bannerSpot(f)
}

func (h *HUD) settleBanner(f Frame) (int, int) {
	cx, top := h.keptBannerSpot(f)
	h.Juice.spot, h.Juice.spotFor, h.Juice.spotSet = [2]int{cx, top}, h.Juice.BannerAt, true
	return cx, top
}

func (h *HUD) bannerAt(cx, y int) ui.Rect {
	w := h.bannerWidth()
	r := ui.Rect{X: cx - w/2, Y: y - bannerHalfH, W: w, H: 2 * bannerHalfH}
	if h.Juice.Cause == "" {
		return r
	}
	rows := h.Font.WrapLines(h.Juice.Cause, causeW-16, 1, causeRows)
	cause := ui.Rect{X: cx - causeW/2, Y: y + causeGap, W: causeW, H: len(rows)*ui.LineHeight + 8}
	return unionRect(r, cause, true)
}

var (
	bannerTops   = []int{BannerY, BannerY + 90, BannerY + 180, BannerY + 250}
	bannerShifts = []int{0, -160, 160, -280, 280}
)

type bannerPlace struct{ cx, y int }

func bannerPlaces(half int) []bannerPlace {
	out := make([]bannerPlace, 0, len(bannerShifts)*len(bannerTops))
	for _, dx := range bannerShifts {
		cx := min(max(Width/2+dx, half), Width-half)
		for _, y := range bannerTops {
			out = append(out, bannerPlace{cx, y})
		}
	}
	return out
}

func (h *HUD) BannerBlocked(f Frame) []ui.Rect {
	return append(inflated(h.GuideTargets(f), targetClear), h.bannerPanels(f)...)
}

func (h *HUD) bannerPanels(f Frame) []ui.Rect {
	blocked := []ui.Rect{}
	if h.IntroShown(f) {
		blocked = append(blocked, h.IntroCard(f))
	}
	if h.CardsDrawn(f) && h.ObjectiveShown(f) {
		blocked = append(blocked, h.ObjectiveRectOf(f))
	}
	blocked = append(blocked, FreightRects(f)...)
	return inflated(append(blocked, h.bannerObstacles(f)...), TipClear)
}

func (h *HUD) bannerSpot(f Frame) (int, int) {
	blocked := h.BannerBlocked(f)
	places := bannerPlaces(max(causeW, h.bannerWidth()) / 2)
	spots := make([]ui.Rect, 0, len(places))
	for _, p := range places {
		spots = append(spots, h.bannerAt(p.cx, p.y))
	}
	best := FirstClear(spots, blocked)
	for i, s := range spots {
		if s == best {
			return places[i].cx, places[i].y
		}
	}
	return Width / 2, BannerY
}

func (h *HUD) bannerObstacles(f Frame) []ui.Rect {
	out := []ui.Rect{}
	if h.OverlayID != OverlayNone {
		out = append(out, Shadowed(h.OverlayRect()))
	}
	if r, ok := h.CalamityRect(f); ok {
		out = append(out, r)
	}
	if h.LessonShown(f) {
		out = append(out, h.LessonRect(f))
	}
	if r, ok := h.TrayRect(f); ok {
		out = append(out, r)
	}
	for _, id := range h.Win.Stacked() {
		if r, ok := WindowSpecs[id].Rect(h, f); ok {
			out = append(out, r)
		}
	}
	return out
}

func (h *HUD) Voices(f Frame) []string {
	f = h.Stage(f)
	out := []string{}
	if h.IntroShown(f) {
		out = append(out, VoiceIntro+":"+f.Guide.Intro.ID)
	}
	if o, ok := f.current(); ok && f.Guide.Held == "" && h.CardsDrawn(f) && h.ObjectiveShown(f) && !f.Introducing() {
		out = append(out, VoiceObjective+":"+o.ID)
	}
	if NextShown(f) && BoardShown(f) && !h.NextQuiet(f) {
		out = append(out, VoiceNext+":"+f.State.Next.Label)
	}
	if line := NextLine(f); line != "" && CardShown(f) && !f.cardSelection() {
		out = append(out, "advice:"+line)
	}
	if t := h.LessonTitle(); t != "" && h.LessonShown(f) {
		out = append(out, "lesson:"+t)
	}
	if v, _, _, ok := h.CurrentCalamity(f); ok {
		out = append(out, "calamity:"+v.ID)
	}
	return append(out, h.noticeVoices(f)...)
}

func (h *HUD) noticeVoices(f Frame) []string {
	out := []string{}
	for _, t := range h.ToastList {
		out = append(out, "toast:"+t.Text)
	}
	if _, ok := h.BannerRect(f); ok {
		out = append(out, "banner:"+h.Juice.Banner)
	}
	return out
}

func objectiveOf(f Frame, id string) (view.ObjectiveView, bool) {
	for _, o := range f.State.Objectives {
		if o.ID == id {
			return o, true
		}
	}
	return view.ObjectiveView{}, false
}
