package hud

import (
	"math"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	SliderButton    = CrewPrefix + ":slider"
	BuyKeepButton   = CrewPrefix + ":buykeep"
	ScaleUpButton   = CrewPrefix + ":scale:up"
	ScaleDownButton = CrewPrefix + ":scale:down"
	StepUpButton    = CrewPrefix + ":step:up"
	StepDownButton  = CrewPrefix + ":step:down"
	RetuneButton    = CrewPrefix + ":horn:retune"
	CrewBoardButton = CrewPrefix + ":board"
	DragKnob        = "knob"
	DragMin         = "min"
	DragMax         = "max"
	SliderFloor     = 6
	sliderRoom      = 2
	knobGlide       = 12.0
	glideMaxDt      = 0.1
	glideSnap       = 0.02
	PopTime         = 0.35
)

type boardState struct {
	Crew     string `json:"-"`
	Drag     string `json:"-"`
	Pick     int    `json:"-"`
	Top      int    `json:"-"`
	shown    float64
	marker   float64
	clock    float64
	born     []float64
	hornBase HornPick
}

func SliderValue(track ui.Rect, top, x int) int {
	if track.W <= 0 || top <= 0 {
		return 0
	}
	v := int(math.Round(float64(x-track.X) * float64(top) / float64(track.W)))
	return min(max(v, 0), top)
}

func SliderX(track ui.Rect, top int, v float64) int {
	if top <= 0 {
		return track.X
	}
	return track.X + int(math.Round(v*float64(track.W)/float64(top)))
}

func SliderTop(base, fits int) int {
	room := min(max(fits, 0), max(2*sliderRoom, base/2))
	return min(max(base+room+sliderRoom, SliderFloor), MaxCrewSize)
}

func bandTop(c model.CrewChain, p HornPick) int {
	now := 0
	if c.Horn != nil {
		now = c.Horn.Current
	}
	return min(max(p.Max+sliderRoom, now+sliderRoom, SliderFloor), hornCeiling)
}

func ScaleTo(c model.CrewChain, n int) view.Action {
	return view.Action{Action: view.ActScaleCrew, Name: c.Name, Replicas: min(max(n, 0), MaxCrewSize), Crew: view.CrewOrder{Container: c.Container, Keeper: chainKeeper(c)}}
}

func hornNow(c model.CrewChain) float64 {
	if c.Horn == nil || c.Horn.Current <= 0 {
		return float64(c.Desired)
	}
	return float64(c.Horn.Current)
}

func (h *HUD) SliderPoint(f Frame, n int) (int, int, bool) {
	v, ok := h.BoardView(f)
	if !ok {
		return 0, 0, false
	}
	return SliderX(v.Track, v.Top, float64(n)), v.Track.Y + v.Track.H/2, true
}

func glideTo(cur, target, dt float64, reduce bool) float64 {
	if reduce || math.Abs(target-cur) < glideSnap {
		return target
	}
	return cur + (target-cur)*(1-math.Exp(-knobGlide*dt))
}

func bornTimes(born []float64, n int, now float64) []float64 {
	out := make([]float64, n)
	copy(out, born)
	for i := len(born); i < n; i++ {
		out[i] = now
	}
	return out
}

func (s boardState) follow(v BoardView, now float64, reduce bool) boardState {
	next := s
	if s.Crew != v.Chain.ID() {
		next = boardState{Crew: v.Chain.ID(), shown: float64(v.Target), marker: hornNow(v.Chain), clock: now, hornBase: s.hornBase,
			born: bornTimes(nil, len(v.Slots), now-whistleTime)}
	}
	dt := min(max(now-next.clock, 0), glideMaxDt)
	next.clock = now
	switch {
	case next.Drag == DragKnob:
		next.shown = float64(v.Target)
	default:
		next.shown = glideTo(next.shown, float64(v.Chain.Desired), dt, reduce)
	}
	next.marker = glideTo(next.marker, hornNow(v.Chain), dt, reduce)
	next.born = bornTimes(next.born, len(v.Slots), now)
	return next
}

func (h *HUD) stepBoard(in Input, f Frame) ([]Intent, bool) {
	v, ok := h.BoardView(f)
	if !ok {
		h.Board = boardState{}
		return nil, false
	}
	h.syncHorn(v.Chain)
	h.Board = h.Board.follow(v, f.Clock, h.Reduce)
	switch {
	case in.Click && h.Hovering && h.Hover.ID == SliderButton:
		h.Hover.Tip = ""
		return h.grabSlider(v, in.MX), true
	case h.Board.Drag != "" && in.Held:
		h.Hover.Tip = ""
		return h.dragSlider(v, in.MX), true
	case h.Board.Drag != "":
		return h.releaseSlider(v), true
	}
	return nil, false
}

func (h *HUD) syncHorn(c model.CrewChain) {
	base := FirstHorn(c)
	if h.Crew.id == c.ID() && h.Board.hornBase == base {
		return
	}
	h.Crew = crewCardState{id: c.ID(), Mode: h.CrewMode(c), Res: c.Resources, Horn: base}
	h.Board.hornBase = base
}

func (h *HUD) grabSlider(v BoardView, x int) []Intent {
	next := h.Board
	next.Top = v.Top
	val := SliderValue(v.Track, v.Top, x)
	if v.Band {
		next.Drag = NearerHandle(v, h.Crew.Horn, x)
		h.Crew.Horn = BandPick(h.Crew.Horn, next.Drag, val)
	} else {
		next.Drag, next.Pick = DragKnob, val
	}
	h.Board = next
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) dragSlider(v BoardView, x int) []Intent {
	val := SliderValue(v.Track, h.Board.Top, x)
	if h.Board.Drag != DragKnob {
		before := h.Crew.Horn
		h.Crew.Horn = BandPick(before, h.Board.Drag, val)
		return sliderTick(before != h.Crew.Horn)
	}
	if val == h.Board.Pick {
		return nil
	}
	next := h.Board
	next.Pick = val
	h.Board = next
	return sliderTick(true)
}

func sliderTick(moved bool) []Intent {
	if !moved {
		return nil
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) releaseSlider(v BoardView) []Intent {
	drag, pick := h.Board.Drag, h.Board.Pick
	next := h.Board
	next.Drag = ""
	h.Board = next
	if drag != DragKnob || pick == v.Chain.Desired {
		return nil
	}
	return []Intent{ActFrom(ScaleTo(v.Chain, pick), FromCrewCard)}
}

func NearerHandle(v BoardView, p HornPick, x int) string {
	lo, hi := SliderX(v.Track, v.Top, float64(p.Min)), SliderX(v.Track, v.Top, float64(p.Max))
	if iabs(x-hi) < iabs(x-lo) || x > hi {
		return DragMax
	}
	return DragMin
}

func BandPick(p HornPick, handle string, v int) HornPick {
	next := p
	switch handle {
	case DragMin:
		next.Min = min(max(v, 1), p.Max)
	case DragMax:
		next.Max = min(max(v, p.Min, 1), hornCeiling)
	}
	return next
}
