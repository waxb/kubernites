package hud

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	Width    = 960
	Height   = 540
	TopH     = 18
	ConsoleH = 132
	ConsoleY = Height - ConsoleH
)

var (
	TopBar     = ui.Rect{X: 0, Y: 0, W: Width, H: TopH}
	ConsoleBox = ui.Rect{X: 0, Y: ConsoleY, W: Width, H: ConsoleH}
	Minimap    = ui.Rect{X: 6, Y: ConsoleY + 8, W: 140, H: ConsoleH - 14}
	CardBox    = ui.Rect{X: 152, Y: ConsoleY + 8, W: 270, H: ConsoleH - 14}
	InfoWell   = ui.Rect{X: 428, Y: ConsoleY + 8, W: 290, H: ConsoleH - 14}
	Command    = ui.Rect{X: 724, Y: ConsoleY + 8, W: 230, H: ConsoleH - 14}
	Overlay    = ui.Rect{X: 120, Y: TopH + 12, W: 720, H: ConsoleY - TopH - 22}
	Objective  = ui.Rect{X: 6, Y: TopH + 6 + purseTabH, W: 340, H: 74}
)

const (
	IntentAction      = "action"
	IntentKubectl     = "kubectl"
	IntentDetails     = "details"
	IntentPlace       = "place"
	IntentCancel      = "cancel"
	IntentPause       = "pause"
	IntentJump        = "jump"
	IntentNew         = "new"
	IntentLoad        = "continue"
	IntentQuit        = "quit"
	IntentSave        = "save"
	IntentMute        = "mute"
	IntentTitle       = "title"
	IntentSelect      = "select"
	IntentSound       = "sound"
	IntentReforge     = "reforge"
	IntentReduce      = "reduce"
	IntentFocus       = "focus"
	IntentEdge        = "edgescroll"
	IntentNames       = "names"
	IntentStone       = "stone"
	IntentFullscreen  = "fullscreen"
	IntentCoinFlights = "coinflights"
	IntentPausePolicy = "pausepolicy"
	IntentVolume      = "volume"
)

type Intent struct {
	Kind    string
	Action  view.Action
	Command string
	Tab     string
	Target  string
	From    string
	X, Y    float64
}

type Input struct {
	MX, MY      int
	Click       bool
	DoubleClick bool
	RightClick  bool
	Wheel       float64
	WheelX      float64
	Chars       []rune
	Keys        []ebiten.Key
	Held        bool
	Shift       bool
	Ctrl        bool
	Control     bool
	Alt         bool
	Layout      func(ebiten.Key) string
}

func (in Input) Pressed(k ebiten.Key) bool {
	return slices.Contains(in.Keys, k)
}

func (in Input) chordIs(k ebiten.Key, letter rune) bool {
	r, ok := in.ChordRune(k)
	return ok && r == letter
}

func (in Input) PressedLetter(letter rune) bool {
	for _, k := range in.Keys {
		if r, ok := in.ChordRune(k); ok && r == letter {
			return true
		}
	}
	return false
}

type Selection struct {
	Kind      string
	ID        string
	Structure view.Structure
	Pod       model.PodView
	Gone      bool
}

func (s Selection) Empty() bool { return s.Kind == "" }

type Frame struct {
	State         view.GameState
	Metrics       view.MetricsSnapshot
	Pods          []model.PodView
	Events        []Tiding
	Selection     Selection
	Worker        *WorkerView
	Placing       string
	Paused        bool
	Muted         bool
	Clock         float64
	EventClock    float64
	Site          view.Site
	View          [4][2]float64
	Vocabulary    Vocabulary
	Reduce        bool
	TrueNames     bool
	Unguided      bool
	Saved         bool
	SavedAgo      float64
	SaveError     string
	Version       string
	Fullscreen    bool
	Volumes       Volumes
	Horns         []model.Horn
	KeepsReady    int
	Joining       bool
	Alerts        []Alert
	Notices       []Notice
	Caesar        model.Caesar
	Rolls         []model.GuildRoll
	Guide         Guide
	AutonomyToday int
	Nodes         map[string]model.NodeView
	GoalAt        *[2]float64
	GoalOfNext    bool
	Beacons       []ui.Rect
	WorldBoxes    []ui.Rect
	Survey        *SurveyPanel
	Causes        []CauseTip
	Places        map[string][2]float64
	Tip           string
	TipAt         [2]int
	Crews         []model.CrewChain
	Land          bool
	Panels        map[string]PanelPin
	BoardCompact  bool
	BoardForced   bool
	Live          bool
	LiveCouncil   view.LiveCouncil
	raw           *rawParts
	liveBoard     *steadyBoard
}

type Alert struct {
	Kind    string
	At      [2]float64
	Text    string
	Key     string
	Quiet   bool
	Routine bool
}

type Tiding struct {
	Time    float64
	Kind    string
	Text    string
	Reason  string
	Warning bool
	Title   string
	Detail  string
}

func (f Frame) Revealed(id string) bool {
	return slices.Contains(f.State.Reveals, id)
}

type Button struct {
	ID       string
	Rect     ui.Rect
	Face     ui.Rect
	Label    string
	Icon     string
	Tip      string
	Cost     string
	Note     string
	Disabled bool
	Selected bool
	Pips     int
	Menu     bool
	Page     string
	Why      string
	Speaks   string
}

func HitButton(buttons []Button, x, y int) (Button, bool) {
	for i := len(buttons) - 1; i >= 0; i-- {
		if buttons[i].Rect.Contains(x, y) {
			return buttons[i], true
		}
	}
	return Button{}, false
}

const (
	CloseFace  = 16
	ClosePad   = 4
	closeInset = 2
	closeReach = CloseFace + 2*ClosePad + closeInset
)

func (f Frame) GameClock() float64 {
	if f.EventClock == 0 {
		return f.Clock
	}
	return f.EventClock
}
