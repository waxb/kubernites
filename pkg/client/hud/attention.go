package hud

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	BoxReadMin     = 4.0
	BoxReadBase    = 1.0
	BoxReadPerWord = 0.4
	BoxFloor       = 3.0
	EnjoyMin       = 20.0
	EnjoyTime      = 25.0
	IdeaGapDark    = 90.0
	IdeaGapFeudal  = 60.0
	FirstMinutes   = 15 * 60.0
	BusyWindow     = 60.0
	BusyMost       = 4
	CounterTick    = 1.0

	BoxCard    = "card"
	BoxQuest   = "quest"
	BoxLesson  = "lesson"
	BoxNews    = "news"
	BoxTrouble = "trouble"
	BoxToast   = "toast"
	BoxBanner  = "banner"
	BoxStep    = "step"
	BoxDone    = "done"
	BoxBoard   = "board"
	BoxReveal  = "reveal"
	BoxModal   = "modal"
	BoxCallout = "callout"

	SlotTop     = "top"
	SlotConsole = "console"
	SlotStep    = "step"
	SlotHarbor  = "harbor"
	SlotRail    = "rail"
	SlotBox     = "box"
	SlotWorker  = "worker"
	SlotPanel   = "panel"
	SlotTip     = "tip"
	SlotPointer = "pointer"
	SlotWorld   = "world"
	SlotJuice   = "juice"

	EventNextStep = "next-step"
	EventAction   = "action"
	EventUsed     = "used"
	EventStepDone = "step-done"

	RuleTwoAtOnce   = "two boxes on screen at once"
	RuleUnread      = "a box before the last one had its reading time"
	RuleFloor       = "two boxes less than 3 s apart"
	RuleTextChanged = "a box whose text changes while it is read"
	RuleCounterFast = "a counter that ticks more than once a second"
	RuleResized     = "a box that resizes while it is shown"
	RuleShownTwice  = "a card shown twice"
	RuleIdeaGap     = "ideas closer than 90 s in the Dark Age or 60 s in the Feudal Age"
	RuleBusyMinute  = "more than 4 unasked boxes a minute in the first 15 minutes"
	RuleEnjoy       = "a step before its enjoyment time without NEXT STEP"
	RuleUntaught    = "an untaught trouble while a lesson box is open"
	RuleOverlap     = "two surfaces that overlap"
)

var cardKinds = map[string]bool{BoxCard: true, BoxQuest: true, BoxLesson: true}

type Surface struct {
	Slot    string  `json:"slot"`
	Kind    string  `json:"kind,omitempty"`
	Key     string  `json:"key"`
	Text    string  `json:"text,omitempty"`
	Status  string  `json:"status,omitempty"`
	Rect    ui.Rect `json:"rect"`
	Box     bool    `json:"box,omitempty"`
	Asked   bool    `json:"asked,omitempty"`
	Modal   bool    `json:"modal,omitempty"`
	Face    bool    `json:"face,omitempty"`
	Waits   bool    `json:"waits,omitempty"`
	Idea    bool    `json:"idea,omitempty"`
	Urgent  bool    `json:"urgent,omitempty"`
	Taught  bool    `json:"taught,omitempty"`
	Owner   string  `json:"owner,omitempty"`
	Target  string  `json:"target,omitempty"`
	Line    [4]int  `json:"line,omitempty"`
	Ring    bool    `json:"ring,omitempty"`
	Layer   string  `json:"layer,omitempty"`
	Tray    bool    `json:"tray,omitempty"`
	Crowded bool    `json:"crowded,omitempty"`
}

func (s Surface) pointer() bool { return s.Line != [4]int{} || s.Ring }

type AttnText struct {
	At      float64 `json:"at"`
	Text    string  `json:"text"`
	Status  string  `json:"status,omitempty"`
	Rect    ui.Rect `json:"rect"`
	Pressed bool    `json:"pressed,omitempty"`
}

type AttnBox struct {
	Kind     string     `json:"kind"`
	Key      string     `json:"key"`
	Text     string     `json:"text"`
	Status   string     `json:"status,omitempty"`
	Slot     string     `json:"slot,omitempty"`
	Rect     ui.Rect    `json:"rect"`
	From     float64    `json:"from"`
	To       float64    `json:"to"`
	Realm    float64    `json:"realm"`
	RealmTo  float64    `json:"realmTo"`
	Age      string     `json:"age,omitempty"`
	Waits    bool       `json:"waits,omitempty"`
	Face     bool       `json:"face,omitempty"`
	Idea     bool       `json:"idea,omitempty"`
	Urgent   bool       `json:"urgent,omitempty"`
	Taught   bool       `json:"taught,omitempty"`
	Tray     bool       `json:"tray,omitempty"`
	Aside    bool       `json:"aside,omitempty"`
	Open     bool       `json:"open,omitempty"`
	Answered float64    `json:"answered,omitempty"`
	ByPlayer bool       `json:"byPlayer,omitempty"`
	Entry    bool       `json:"entry,omitempty"`
	Asked    bool       `json:"asked,omitempty"`
	Due      bool       `json:"due,omitempty"`
	Texts    []AttnText `json:"texts,omitempty"`
}

func (b AttnBox) Read() float64 { return ReadTime(b.Text) }

func (b AttnBox) end() float64 {
	if b.Face && b.Answered > 0 {
		return min(b.From+b.Read(), b.Answered)
	}
	if b.Face {
		return b.From + b.Read()
	}
	return b.To
}

func (b AttnBox) readBy(at float64) bool {
	return at >= b.From+b.Read() || b.Answered > 0 && at >= b.Answered || (b.ByPlayer || b.Kind == BoxTrouble) && !b.Open && at >= b.To
}

type AttnFrame struct {
	At       float64   `json:"at"`
	Realm    float64   `json:"realm"`
	Surfaces []Surface `json:"surfaces"`
	Drawn    bool      `json:"drawn,omitempty"`
}

type AttnEvent struct {
	At    float64 `json:"at"`
	Realm float64 `json:"realm"`
	Kind  string  `json:"kind"`
	Key   string  `json:"key,omitempty"`
}

type AttnLog struct {
	Boxes  []AttnBox   `json:"boxes"`
	Frames []AttnFrame `json:"frames,omitempty"`
	Events []AttnEvent `json:"events,omitempty"`
	Marks  []PaceMark  `json:"marks,omitempty"`
}

type AttnRules struct {
	OverlapOnly bool
}

type AttnFault struct {
	Rule  string
	At    float64
	Realm float64
	Keys  []string
}

func (f AttnFault) String() string {
	sec := int(f.Realm)
	return fmt.Sprintf("%02d:%02d (at %.1f s) %s: %s", sec/60, sec%60, f.At, f.Rule, strings.Join(f.Keys, ", "))
}

func ReadTime(text string) float64 {
	return max(BoxReadMin, BoxReadBase+BoxReadPerWord*float64(len(strings.Fields(text))))
}

func CheckAttention(log AttnLog, rules AttnRules) []AttnFault {
	out := checkOverlaps(log.Frames)
	if rules.OverlapOnly {
		return out
	}
	boxes := slices.DeleteFunc(slices.Clone(log.Boxes), func(b AttnBox) bool { return b.Entry })
	sort.SliceStable(boxes, func(i, j int) bool { return boxes[i].From < boxes[j].From })
	out = append(out, checkTurns(boxes)...)
	out = append(out, checkTexts(log.Boxes)...)
	out = append(out, checkOnce(boxes)...)
	out = append(out, checkIdeas(boxes)...)
	out = append(out, checkBusy(boxes)...)
	out = append(out, checkEnjoy(boxes, log.Events)...)
	out = append(out, checkUntaught(boxes)...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].At < out[j].At })
	return out
}

func boxFault(rule string, b AttnBox, with ...string) AttnFault {
	return AttnFault{Rule: rule, At: b.From, Realm: b.Realm, Keys: append([]string{b.Key}, with...)}
}

func checkTurns(boxes []AttnBox) []AttnFault {
	out := []AttnFault{}
	var last *AttnBox
	for i := range boxes {
		b := boxes[i]
		if last != nil {
			aside := last.Aside && last.To <= b.From
			switch {
			case b.From < last.end() && !aside:
				out = append(out, boxFault(RuleTwoAtOnce, b, last.Key))
			case b.Asked:
			case !last.readBy(b.From) && !aside:
				out = append(out, boxFault(RuleUnread, b, last.Key))
			case b.From < last.end()+BoxFloor && !aside:
				out = append(out, boxFault(RuleFloor, b, last.Key))
			}
		}
		if last == nil || b.end() > last.end() {
			last = &boxes[i]
		}
	}
	return out
}

var digitsRun = regexp.MustCompile(`[0-9]+([.,][0-9]+)*`)

func masked(text string) string { return digitsRun.ReplaceAllString(text, "#") }

func checkTexts(boxes []AttnBox) []AttnFault {
	out := []AttnFault{}
	for _, b := range boxes {
		text, status, rect, tick := b.Text, b.Status, b.Rect, -CounterTick
		for _, t := range b.Texts {
			reading := t.At < b.From+b.Read() && !(b.Face && b.Answered > 0 && t.At >= b.Answered)
			ticked := t.Status != status || masked(t.Text) == masked(text) && t.Text != text
			switch {
			case masked(t.Text) != masked(text) && reading && !t.Pressed:
				out = append(out, AttnFault{Rule: RuleTextChanged, At: t.At, Realm: b.Realm, Keys: []string{b.Key, text, t.Text}})
			case ticked && t.At-tick < CounterTick && !t.Pressed:
				out = append(out, AttnFault{Rule: RuleCounterFast, At: t.At, Realm: b.Realm, Keys: []string{b.Key, text + " " + status, t.Text + " " + t.Status}})
			}
			if ticked {
				tick = t.At
			}
			same := masked(t.Text) == masked(text) && masked(t.Status) == masked(status)
			if (t.Rect.W != rect.W || t.Rect.H != rect.H) && (reading || same) && !t.Pressed {
				out = append(out, AttnFault{Rule: RuleResized, At: t.At, Realm: b.Realm, Keys: []string{b.Key, fmt.Sprintf("%dx%d to %dx%d", rect.W, rect.H, t.Rect.W, t.Rect.H)}})
			}
			text, status, rect = t.Text, t.Status, t.Rect
		}
	}
	return out
}

func checkOnce(boxes []AttnBox) []AttnFault {
	out := []AttnFault{}
	seen := map[string]bool{}
	for _, b := range boxes {
		if !cardKinds[b.Kind] {
			continue
		}
		if seen[b.Key] && !b.Tray {
			out = append(out, boxFault(RuleShownTwice, b))
		}
		seen[b.Key] = true
	}
	return out
}

func ideaGap(age string) float64 {
	if age == "" || age == "dark" {
		return IdeaGapDark
	}
	return IdeaGapFeudal
}

func checkIdeas(boxes []AttnBox) []AttnFault {
	out := []AttnFault{}
	var last *AttnBox
	for i := range boxes {
		b := boxes[i]
		if !b.Idea || b.Tray || b.Asked {
			continue
		}
		if last != nil && b.From-last.From < ideaGap(b.Age) && !b.Due {
			out = append(out, boxFault(RuleIdeaGap, b, last.Key))
		}
		last = &boxes[i]
	}
	return out
}

func checkBusy(boxes []AttnBox) []AttnFault {
	out := []AttnFault{}
	early := []AttnBox{}
	for _, b := range boxes {
		if b.From < FirstMinutes && !b.Asked {
			early = append(early, b)
		}
	}
	for i, b := range early {
		n := 0
		for _, c := range early[i:] {
			if c.From-b.From < BusyWindow {
				n++
			}
		}
		if n > BusyMost {
			out = append(out, boxFault(RuleBusyMinute, b, fmt.Sprintf("%d boxes from here within %.0f s", n, BusyWindow)))
		}
	}
	return out
}

func checkEnjoy(boxes []AttnBox, events []AttnEvent) []AttnFault {
	out := []AttnFault{}
	var done *AttnBox
	for i := range boxes {
		b := boxes[i]
		switch b.Kind {
		case BoxDone:
			done = &boxes[i]
		case BoxStep:
			if done != nil && b.Realm-done.Realm < EnjoyMin && !pressedBetween(events, done.From, b.From) {
				out = append(out, boxFault(RuleEnjoy, b, done.Key))
			}
			done = nil
		}
	}
	return out
}

func pressedBetween(events []AttnEvent, from, to float64) bool {
	return slices.ContainsFunc(events, func(e AttnEvent) bool { return e.Kind == EventNextStep && e.At >= from && e.At <= to })
}

func checkUntaught(boxes []AttnBox) []AttnFault {
	out := []AttnFault{}
	for _, b := range boxes {
		if b.Kind != BoxTrouble || b.Taught {
			continue
		}
		for _, c := range boxes {
			if (c.Kind == BoxCard || c.Kind == BoxLesson || c.Kind == BoxQuest) && c.From <= b.From && b.From < c.To {
				out = append(out, boxFault(RuleUntaught, b, c.Key))
			}
		}
	}
	return out
}

func checkOverlaps(frames []AttnFrame) []AttnFault {
	out := []AttnFault{}
	seen := map[string]bool{}
	for _, f := range frames {
		for i, a := range f.Surfaces {
			for _, b := range f.Surfaces[i+1:] {
				if !surfacesClash(a, b) {
					continue
				}
				pair := a.Key + "|" + b.Key
				if seen[pair] {
					continue
				}
				seen[pair] = true
				out = append(out, AttnFault{Rule: RuleOverlap, At: f.At, Realm: f.Realm, Keys: []string{describeSurface(a), describeSurface(b)}})
			}
		}
	}
	return out
}

func describeSurface(s Surface) string {
	return fmt.Sprintf("%s %s at %d,%d %dx%d", s.Slot, s.Key, s.Rect.X, s.Rect.Y, s.Rect.W, s.Rect.H)
}

func surfacesClash(a, b Surface) bool {
	switch {
	case a.Modal || b.Modal || a.Slot == SlotJuice || b.Slot == SlotJuice || a.Crowded || b.Crowded:
		return false
	case a.Owner != "" && a.Owner == b.Key || b.Owner != "" && b.Owner == a.Key:
		return false
	case a.pointer() && b.pointer():
		return false
	case a.pointer():
		return pointerCrosses(a, b)
	case b.pointer():
		return pointerCrosses(b, a)
	case a.Rect.W <= 0 || a.Rect.H <= 0 || b.Rect.W <= 0 || b.Rect.H <= 0:
		return false
	}
	return Overlaps(a.Rect, b.Rect)
}

func pointerCrosses(p, s Surface) bool {
	if s.Key == p.Owner || s.Key == p.Target || s.Owner == p.Owner && p.Owner != "" || s.Rect.W <= 0 || s.Rect.H <= 0 || s.Slot == SlotJuice {
		return false
	}
	x, y := p.Rect.X+p.Rect.W/2, p.Rect.Y+p.Rect.H/2
	if !p.Ring {
		x, y = p.Line[2], p.Line[3]
	}
	if s.Rect.Inset(-HolePad-pointerReach).Contains(x, y) {
		return false
	}
	if p.Ring {
		return Overlaps(p.Rect, s.Rect)
	}
	return segmentHits(p.Line, s.Rect)
}

func segmentHits(line [4]int, r ui.Rect) bool {
	x0, y0, x1, y1 := float64(line[0]), float64(line[1]), float64(line[2]), float64(line[3])
	steps := int(max(absf(x1-x0), absf(y1-y0))) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		if r.Contains(int(x0+(x1-x0)*t), int(y0+(y1-y0)*t)) {
			return true
		}
	}
	return false
}

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
