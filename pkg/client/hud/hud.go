package hud

import (
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	PageMain   = "main"
	PageBuild  = "build"
	PageSummon = "summon"

	TabEvents = "EVT"
	TabTech   = "TECH"
	TabKube   = "KUBE"
	TabYAML   = "YAML"

	OverlayNone        = ""
	OverlayDetails     = "details"
	OverlayStone       = "stone"
	OverlayScriptorium = "scriptorium"
	OverlayGrimoire    = "grimoire"
	OverlaySoulvault   = "soulvault"
	OverlayChronicles  = "chronicles"
	OverlayAge         = "age"
	OverlayReforge     = "reforge"
	OverlayMenu        = "menu"

	ToastLife = 4.0
	maxToasts = 4
)

type Runner func(command string) view.CommandResult

type Backend struct {
	Kubectl Runner
	Inspect Runner
	Apply   func(yaml string) (string, bool)
	Start   func(command string) (StoneJob, bool)
	Send    func(edit LiveEdit, text string, force bool) (StoneJob, bool)
	Reveal  func(edit LiveEdit, key string) (string, bool)
}

type Toast struct {
	Text    string
	Bad     bool
	Start   float64
	Count   int
	News    bool
	Fanfare bool
	Spot    ui.Rect
	Opens   string
}

type HUD struct {
	claims        map[string]Intent
	live          *LiveCard
	liveReads     map[string]bool
	lw            liveWrites
	BoardOpen     bool `json:"-"`
	Lib           *gfx.Library
	backend       Backend
	Font          ui.Font `json:"-"`
	PageID        string  `json:"-"`
	TabID         string  `json:"-"`
	OverlayID     string  `json:"-"`
	DetailsCard   Details `json:"-"`
	Stone         Stone   `json:"-"`
	kube          *stoneRun
	stoneMode     string
	liveEdit      *editState
	fetching      *stoneRun
	Script        Script `json:"-"`
	Grimoire      string `json:"-"`
	list          int
	ageOpen       bool
	HintOpen      string `json:"-"`
	pauseAsks     []PauseAsk
	pauseFor      string
	loomRow       int
	scroll        ui.Scroll
	ToastList     []Toast     `json:"-"`
	Hover         Button      `json:"-"`
	Hovering      bool        `json:"-"`
	Plain         *plainWatch `json:"-"`
	Now           float64     `json:"-"`
	events        float64
	minimap       minimapCache
	vocab         Vocabulary
	Reduce        bool `json:"-"`
	gate          gateWatch
	Juice         juice      `json:"-"`
	LessonCard    LessonCard `json:"-"`
	roadBack      string
	roadSel       string
	alertCursor   int
	ping          ping
	voyageList    int
	retire        retireArm
	edgeScroll    bool
	Coins         purse  `json:"-"`
	CouncilTabID  string `json:"-"`
	lens          string
	deltas        []delta
	unguidedAck   string
	welcomePage   string
	academy       academyState
	questSeen     map[string]bool
	folded        bool
	appear        appearances
	scratch       *ebiten.Image
	fading        bool
	consumed      bool
	Ops           opsState  `json:"-"`
	Book          bookState `json:"-"`
	Pin           tipPin    `json:"-"`
	help          bool
	MouseX        int           `json:"-"`
	MouseY        int           `json:"-"`
	Kmap          kmapState     `json:"-"`
	Crew          crewCardState `json:"-"`
	pending       pendingMode
	Board         boardState     `json:"-"`
	View          viewToggles    `json:"-"`
	CrewsList     CrewsListState `json:"-"`
	KindCard      string         `json:"-"`
	Director      director       `json:"-"`
	Ledger        beatLedger     `json:"-"`
	Woe           WoeState       `json:"-"`
	teach         teachHint
	taught        map[string]bool
	Results       []LessonCard `json:"-"`
	taughtDirty   bool
	terms         *termWatch
	Dawn          ageDawn `json:"-"`
	termAt        float64
	Trial         trialState `json:"-"`
	Yard          teachState `json:"-"`
	rateSince     float64
	Win           windowState `json:"-"`
	grid          gridState
	Kube          kubeLayer `json:"-"`
	drawn         []Surface
	attn          attnTracker
	Steady        steadyPanel `json:"-"`
	steadyTrouble steadyCard
	top           steadyTop
}

func New(lib *gfx.Library, b Backend) *HUD {
	h := &HUD{Lib: lib, backend: b, PageID: PageMain, TabID: TabEvents, Stone: NewStone(), Script: NewScript(), Juice: newJuice(), terms: &termWatch{}, Plain: &plainWatch{}}
	h.Script.Live = h.liveCrewText
	return h
}

func (h *HUD) Page() string    { return h.PageID }
func (h *HUD) Tab() string     { return h.TabID }
func (h *HUD) Overlay() string { return h.OverlayID }
func (h *HUD) Toasts() []Toast { return h.ToastList }

func (h *HUD) Open(overlay string) {
	h.OverlayID = overlay
}

func (h *HUD) SetEdgeScroll(on bool) { h.edgeScroll = on }

func (h *HUD) TextFocus() bool {
	return h.stoneTyping() || h.OverlayID == OverlayScriptorium || h.OverlayID == OverlayLiveOrder
}

func (h *HUD) stoneTyping() bool {
	return h.OverlayID == OverlayStone && (h.Stone.Tall || !h.Stone.away || h.StoneTerminal())
}

func (h *HUD) FocusStone() { h.Stone.away = false }

func (h *HUD) trackStoneFocus(in Input) {
	if h.OverlayID == OverlayStone && in.Click {
		h.Stone.away = !h.StoneRect().Contains(in.MX, in.MY)
	}
}

func (h *HUD) Toast(text string, bad bool) {
	h.PushToast(text, bad)
}

func (h *HUD) Consumed() bool { return h.consumed }

func (h *HUD) Captures(x, y int, f Frame) bool {
	f = h.Stage(f)
	if TopBar.Contains(x, y) || h.consoleCaptures(x, y, f) {
		return true
	}
	if h.LessonShown(f) && h.LessonRect(f).Contains(x, y) {
		return true
	}
	if h.OverlayID != OverlayNone && h.OverlayRect().Contains(x, y) {
		return true
	}
	if _, ok := HitButton(alertButtons(f, h.Arrows(f)), x, y); ok {
		return true
	}
	if h.RailRect(f).Contains(x, y) || containsAny(FreightRects(f), x, y) {
		return true
	}
	if r, ok := h.CalamityRect(f); ok && r.Contains(x, y) {
		return true
	}
	if h.IntroShown(f) && h.IntroCard(f).Contains(x, y) {
		return true
	}
	if c, ok := h.QuestCard(f); ok && h.questRect(c).Contains(x, y) {
		return true
	}
	if r, ok := h.TrayRect(f); ok && r.Contains(x, y) {
		return true
	}
	if _, _, ok := h.windowAt(f, x, y); ok {
		return true
	}
	return h.ObjectiveShown(f) && h.ObjectiveRectOf(f).Contains(x, y)
}

func (h *HUD) Update(in Input, f Frame) []Intent {
	h.holdTerminal()
	h.settleLive(f)
	if in.Click || in.RightClick || len(in.Keys) > 0 {
		h.attn.pressed = f.Clock
		h.pressed(f.Clock)
	}
	intents := append(h.update(in, f), h.pumpKubectl()...)
	h.noteSteady(f)
	f = h.Stage(f)
	h.afterUpdate(f)
	h.noteBeats(f)
	h.noteAttention(f)
	return h.liveFiltered(append(append(intents, h.readClosedAge(f)...), h.OpenEnding(f)...))
}

func (h *HUD) update(in Input, f Frame) []Intent {
	before := h.Now
	h.Now, h.events, h.vocab, h.Reduce = f.Clock, f.GameClock(), f.Vocabulary, f.Reduce
	h.Yard.place = f.placed(PanelObjective, Objective)
	h.consumed = false
	h.trackStoneFocus(in)
	h.holdToasts(f, h.Now-before)
	h.expireToasts()
	h.pruneBanners()
	h.pruneResults()
	h.deltas = h.LiveDeltas()
	h.stepCoins(f)
	h.watch(f)
	f = h.withUnreadTrouble(f)
	h.noteLiveTrouble(f)
	h.Ops.Calamity = h.Ops.Calamity.Step(f.State.Calamities, f.Clock, h.calamityGate(f))
	h.direct(f)
	f = h.Stage(f)
	h.noteWaitSeen(in, f)
	h.noteReading(in)
	if intents := h.passUnguided(f); len(intents) > 0 {
		return intents
	}
	h.syncChooser(f)
	h.syncViews(f)
	h.forgetCrewsList(f)
	h.syncGrid(f)
	h.returnToGoal(f)
	h.syncScript()
	h.stepKnowledge(in, f)
	if h.Script.Pending {
		return h.ApplyScript()
	}
	h.stepSpot(f)
	buttons := h.Buttons(f)
	h.Hover, h.Hovering = HitButton(buttons, in.MX, in.MY)
	h.MouseX, h.MouseY = in.MX, in.MY
	h.stepPin(in, f, buttons)
	if intents, done := h.stepWindows(in, f); done {
		return intents
	}
	if intents, done := h.stepBoard(in, f); done {
		return intents
	}
	if intents, done := h.keys(in, f); done {
		return intents
	}
	if in.Wheel != 0 {
		h.wheel(in, f)
	}
	if in.RightClick {
		if intents, ok := h.dismissNotice(); ok {
			return intents
		}
		return h.back(f)
	}
	if !in.Click {
		return nil
	}
	if intents, ok := h.helpClick(in, f); ok {
		return intents
	}
	if h.Hovering {
		h.consumed = true
		if h.Hover.Disabled {
			h.Toast(h.Hover.Tip, true)
			return []Intent{{Kind: IntentSound, Command: "error"}}
		}
		return append(h.Press(h.Hover.ID, f), h.IntroTried(f, h.Hover.ID)...)
	}
	if h.dismissOverlay(in) {
		return []Intent{{Kind: IntentSound, Command: "click"}}
	}
	if Minimap.Contains(in.MX, in.MY) {
		return h.MinimapJump(in.MX, in.MY, f)
	}
	return nil
}

var ModalOverlays = []string{OverlayStone, OverlayScriptorium, OverlayReforge, OverlaySiege}

func (h *HUD) dismissOverlay(in Input) bool {
	if h.OverlayID == OverlayNone || slices.Contains(ModalOverlays, h.OverlayID) || h.OverlayRect().Contains(in.MX, in.MY) {
		return false
	}
	h.OverlayID, h.consumed = OverlayNone, true
	return true
}

func (h *HUD) expireToasts() {
	kept := make([]Toast, 0, len(h.ToastList))
	for _, t := range h.ToastList {
		if h.Now-t.Start < ToastLife && h.Now >= t.Start {
			kept = append(kept, t)
		}
	}
	h.ToastList = kept
}

func (h *HUD) back(f Frame) []Intent {
	switch {
	case h.OverlayID == OverlayReforge:
		h.OverlayID = OverlayAge
	case h.OverlayID == OverlayLiveOrder:
		return h.scrollIntent(ScrollCancel)
	case h.OverlayID == OverlayNone && h.lw.page > 0:
		h.lw.page = 0
	case h.OverlayID == OverlayRoad && h.roadBack != "":
		h.OverlayID, h.roadBack = h.roadBack, ""
	case f.Placing != "":
		return []Intent{{Kind: IntentCancel}}
	case h.OverlayID != OverlayNone:
		h.OverlayID = OverlayNone
	case h.CrewsList.Open:
		h.CrewsList = CrewsListState{}
	case h.PageID != PageMain:
		h.PageID = pageParent(h.PageID)
	case h.objectShown(f) && h.crewPage(f):
		h.Crew.Mode = CrewModeChain
	case !f.Selection.Empty():
		return h.leaveObject(f)
	case h.lens != LensNone:
		h.lens = LensNone
	}
	return nil
}

func pageParent(page string) string {
	if parent, ok := pageParents[page]; ok {
		return parent
	}
	return PageMain
}

func (h *HUD) wheel(in Input, f Frame) {
	step := -int(in.Wheel * 3)
	if step == 0 {
		return
	}
	switch {
	case h.OverlayID == OverlayBook:
		h.bookWheel(in, f, step)
	case h.OverlayID == OverlayDetails:
		h.DetailsCard.scrollBy(step)
	case h.OverlayID == OverlayStone:
		h.scrollStone(step)
	case h.OverlayID == OverlayScriptorium:
		h.Script.Editor.ScrollBy(step, scriptVisible())
	case h.OverlayID == OverlayVoyage:
		h.voyageList = min(max(h.voyageList+step, 0), max(len(VoyageRows(f))-voyageVisible(), 0))
	case h.OverlayID == OverlaySoulvault:
		h.academyWheel(in, f, step)
	case InfoWell.Contains(in.MX, in.MY) && h.activeTab(f) == TabEvents:
		h.scroll = h.scroll.By(step, len(f.Events), 1)
	}
}

func (h *HUD) keys(in Input, f Frame) ([]Intent, bool) {
	if sc := h.kube.screen(); sc != nil && h.kube.ask == nil {
		h.OverlayID = OverlayStone
		return h.terminalKeys(sc, in), len(in.Keys) > 0 || len(in.Chars) > 0
	}
	if h.help && in.Pressed(ebiten.KeyEscape) {
		h.help = false
		return nil, true
	}
	if h.OverlayID == OverlayLiveOrder {
		return h.orderKeys(in)
	}
	if h.OverlayID != OverlayStone && h.OverlayID != OverlayScriptorium {
		if intents, done := h.liveChord(in, f); done || h.live != nil && in.Control {
			return intents, done || len(in.Keys) > 0
		}
	}
	switch h.OverlayID {
	case OverlayStone:
		if h.stoneTyping() {
			return h.StoneKeys(in), len(in.Keys) > 0 || len(in.Chars) > 0
		}
	case OverlayScriptorium:
		if h.liveEdit != nil {
			return h.liveEditKeys(in), len(in.Keys) > 0 || len(in.Chars) > 0
		}
		return h.scriptKeys(in, f), len(in.Keys) > 0 || len(in.Chars) > 0
	case OverlayReforge:
		if in.Pressed(ebiten.KeyEscape) {
			h.OverlayID = OverlayAge
		}
		return nil, len(in.Keys) > 0
	case OverlaySiege:
		if in.Pressed(ebiten.KeyEscape) {
			h.OverlayID = OverlayNone
		}
		return nil, len(in.Keys) > 0
	case OverlayWelcome:
		return h.welcomeKeys(in, f)
	case OverlayChapters:
		return h.chapterKeys(in, f)
	case OverlayDebrief:
		if in.Pressed(ebiten.KeyEscape) {
			h.OverlayID = OverlayNone
			return nil, true
		}
	case OverlayBook:
		return h.bookKeys(in, f)
	case OverlayDetails:
		if h.detailKeys(in) {
			return nil, true
		}
	case OverlayGrimoire:
		if in.Pressed(ebiten.KeyEscape) && h.closePageCard() {
			return nil, true
		}
	case OverlayAge:
		if intents, done := h.loomKeys(in, f); done {
			return intents, true
		}
	}
	return h.globalKeys(in, f)
}

const (
	stoneSleeps = "Nothing answers yet: the Speaking Stone sleeps."
	StoneLater  = "Borg hands you kubectl when a step asks for it."
)

func (h *HUD) cardAsksForStone() bool {
	v := h.Yard.View
	if v == nil || !v.running() || v.walking() {
		return false
	}
	if v.Chip != "" {
		return true
	}
	s, ok := v.Current()
	body, _ := v.bodyText()
	return ok && (s.Speaks() || strings.Contains(body, "Enter") || strings.Contains(s.Try, "Enter"))
}

func (h *HUD) globalKeys(in Input, f Frame) ([]Intent, bool) {
	if intents, done := h.gridKeys(in, f); done {
		return intents, true
	}
	switch {
	case in.Pressed(ebiten.KeyEscape):
		return h.back(f), true
	case in.Pressed(ebiten.KeyEnter) && in.Shift && f.Revealed("scriptorium") && f.Shown(SurfaceScriptorium):
		h.OverlayID = OverlayScriptorium
		return nil, true
	case in.Pressed(ebiten.KeyEnter) && h.cardAsksForStone():
		h.OpenStone(h.Yard.View.Chip)
		return h.IntroTried(f, SurfaceStone), true
	case in.Pressed(ebiten.KeyEnter) && f.Revealed("stone") && f.Shown(SurfaceStone):
		h.OpenStone(f.SpeakCommand())
		return h.IntroTried(f, SurfaceStone), true
	case in.Pressed(ebiten.KeyEnter) && !f.Revealed("stone"):
		return h.StoneAsleep(), true
	case in.Pressed(ebiten.KeyF2) && len(f.State.Almanac) > 0:
		return h.OpenBook("", f), true
	case in.Pressed(ebiten.KeyG) && siegeKnown(f):
		return h.OpenSiege(f), true
	case in.Pressed(ebiten.KeyN) && NextLine(f) != "":
		return h.FollowNext(f), true
	case in.Pressed(ebiten.KeyN) && NextShown(f):
		return h.PressNext(f), true
	case in.Pressed(ebiten.KeyR) && roadKnown(f):
		h.toggleRoad()
		return []Intent{{Kind: IntentSound, Command: "click"}}, true
	case in.Pressed(ebiten.KeyV) && voyageOpen(f):
		h.toggleVoyage()
		return []Intent{{Kind: IntentSound, Command: "click"}}, true
	case in.Pressed(ebiten.KeyF1) || in.Pressed(ebiten.KeyL):
		return h.toggleLedger(), true
	case in.Pressed(ebiten.KeyC) && CouncilOpen(f):
		return h.toggleCouncil(), true
	case in.Pressed(ebiten.KeyO):
		return h.cycleLens(f), true
	case in.Pressed(ebiten.KeyTab) && len(f.Alerts) > 0:
		return h.nextAlert(f), true
	case in.Pressed(ebiten.KeyP) && f.Site.Width > 0:
		return []Intent{{Kind: IntentGo, Target: view.ParcelPile}}, true
	case in.Pressed(ebiten.KeyH) && f.Site.Width > 0:
		return h.focus(HomeOf(f.Site)), true
	case in.Pressed(ebiten.KeyB) && f.Revealed("build") && f.Shown("cmd:build"):
		return h.openBuild(f), true
	case in.Pressed(ebiten.KeySpace):
		return []Intent{{Kind: IntentPause}}, true
	}
	return nil, false
}

type hudLayer struct {
	Name     string                             `json:"-"`
	Draw     func(*HUD, *ebiten.Image, Frame)   `json:"-"`
	Surfaces func(*HUD, Frame) []paintedSurface `json:"-"`
}

const (
	GuideLayer    = "guide"
	overlayLayer  = "overlay"
	PanelRegion   = "panel:"
	WindowRegion  = "window:"
	ConsoleRegion = "console:"
)

var HudLayers = []hudLayer{
	{"vignette", func(h *HUD, d *ebiten.Image, _ Frame) { h.drawVignette(d) }, vignetteSurfaces},
	{"dim", (*HUD).drawWorldDim, dimSurfaces},
	{"topbar", (*HUD).DrawTopBar, topBarSurfaces},
	{"coins", func(h *HUD, d *ebiten.Image, _ Frame) { h.drawCoins(d) }, coinSurfaces},
	{"deltas", (*HUD).drawDeltas, deltaSurfaces},
	{"siege", (*HUD).DrawSiegeStrip, siegeSurfaces},
	{"alerts", (*HUD).drawAlerts, alertSurfaces},
	{"windows", (*HUD).drawWindows, WindowSurfaces},
	{"purse", (*HUD).DrawPurseTab, purseSurfaces},
	{"survey", (*HUD).drawSurvey, surveySurfaces},
	{"legend", (*HUD).drawLegend, legendSurfaces},
	{"console", (*HUD).drawConsole, consoleSurfaces},
	{"lens", (*HUD).drawLensButton, lensSurfaces},
	{overlayLayer, (*HUD).drawOverlay, OverlaySurfaces},
	{"intro", (*HUD).drawIntro, introSurfaces},
	{questLayer, (*HUD).drawQuestCard, questSurfaces},
	{"spot", (*HUD).drawObjectiveSpot, spotSurfaces},
	{"banner", (*HUD).drawBanner, bannerSurfaces},
	{"toasts", (*HUD).DrawToasts, toastSurfaces},
	{"pinned", (*HUD).drawPinned, pinnedSurfaces},
	{"hint", (*HUD).drawHint, hintSurfaces},
	{TeacherRegion, (*HUD).drawTeacherBeacons, teacherBeaconSurfaces},
	{GuideLayer, (*HUD).drawTargetsOnTop, guideSurfaces},
	{"tooltip", (*HUD).drawTooltip, tooltipSurfaces},
	{"worldtip", (*HUD).drawWorldTip, worldTipSurfaces},
}

func (h *HUD) Draw(dst *ebiten.Image, f Frame) {
	h.settleLive(f)
	f = h.Stage(f)
	h.Plain.Reset()
	if !h.toastsHeld(f) {
		h.settleToasts(f)
	}
	if _, ok := h.BannerRect(f); ok {
		h.settleBanner(f)
	}
	h.paintAll(dst, f)
}
