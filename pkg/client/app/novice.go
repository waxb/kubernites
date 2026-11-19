package app

import (
	"regexp"
	"slices"
	"strings"
)

type Seat interface {
	Look(shot bool) Look
	Click(id string) (Changes, error)
	Wait(seconds float64) Changes
	Key(name string) (Changes, error)
}

const (
	noviceRepeats  = 3
	noviceBurst    = 8
	novicePanTries = 8
	noviceBreath   = 1.0
	noviceRead     = 2.0
	ObjectiveWin   = "window:objective"
	placeHint      = "Click to build"
	pausedNote     = "paused"
	plotSuggested  = "plot:suggested"
	pileKind       = "pile"
	nextStepButton = "obj:next"
	stoneEnter     = "Enter"
)

var (
	noviceAcks     = []string{"welcome:begin", "quest:ack", "intro:ack", "lesson:ack"}
	noviceArrows   = regexp.MustCompile(`Arrow[A-Za-z]+`)
	noviceClickOn  = regexp.MustCompile(`(?:(?:ready: |then )click (?:the )?|ready: open one at (?:the )?)([^,:(]+?)(?:,| then|:|\(|$)`)
	noviceKeptBack = []string{"calamity:close", "calamity:shelve", "calamity:show", "calamity:almanac", "calamity:ok", "calamity:next", "calamity:prev"}
)

type Novice struct {
	Hasty   bool
	seat    Seat
	Log     []string
	last    string
	repeats int
	picked  string
	read    map[string]bool
	step    string
	shown   string
	burst   int
	fixed   map[string]bool
	lit     string
}

func NewNovice(s Seat) *Novice {
	return &Novice{seat: s, read: map[string]bool{}, fixed: map[string]bool{}}
}

func (n *Novice) Play(done func(Look) bool, until float64) (Look, bool) {
	for {
		l := n.seat.Look(false)
		if done(l) {
			return l, true
		}
		if l.Seconds >= until {
			return l, false
		}
		n.Turn(l)
	}
}

func (n *Novice) Turn(l Look) {
	if obj := RegionOf(l, ObjectiveWin); len(obj) > 1 && obj[1] != n.step {
		n.step = obj[1]
		n.note(l, "== "+n.step)
	}
	if l.Lit != n.lit {
		n.lit = l.Lit
		if l.Lit != "" && !n.Hasty {
			n.seat.Wait(noviceRead)
			return
		}
	}
	id := n.choose(l)
	if card := cardText(l, id); card != "" && !n.read[card] && !n.Hasty {
		n.read[card] = true
		n.seat.Wait(noviceRead)
		return
	}
	if id == "" || id == n.last && n.repeats >= noviceRepeats || n.burst >= noviceBurst {
		n.breathe()
		return
	}
	n.burst++
	n.press(l, id)
}

func (n *Novice) breathe() {
	ch := n.seat.Wait(noviceBreath)
	n.last, n.repeats, n.burst = "", 0, 0
	if strings.Contains(ch.Note, pausedNote) {
		n.seat.Click("top:pause")
	}
}

func (n *Novice) press(l Look, id string) {
	if id == n.last {
		n.repeats++
	} else {
		n.last, n.repeats = id, 1
	}
	if _, err := n.seat.Click(id); err != nil {
		n.note(l, "refused "+id+": "+err.Error())
		n.breathe()
		return
	}
	if !strings.HasPrefix(id, pileKind+":") {
		n.note(l, id)
	}
	now := n.seat.Look(false)
	if strings.Contains(strings.Join(RegionOf(now, worldRegion), " "), placeHint) {
		n.place()
	}
	if id == "obj:speak" || strings.HasSuffix(id, ":speak") {
		n.seat.Key(stoneEnter)
		n.note(l, "key "+stoneEnter)
	}
}

func (n *Novice) note(l Look, what string) {
	n.Log = append(n.Log, l.Time+" "+what)
}

func (n *Novice) place() {
	for range novicePanTries {
		plot, ok := lookObject(n.seat.Look(false), plotSuggested)
		if !ok {
			return
		}
		keys := noviceArrows.FindAllString(plot.Label, -1)
		if len(keys) == 0 {
			break
		}
		for _, k := range keys {
			n.seat.Key(k)
		}
	}
	if _, err := n.seat.Click(plotSuggested); err == nil {
		n.Log = append(n.Log, "  "+plotSuggested)
	}
	if strings.Contains(strings.Join(RegionOf(n.seat.Look(false), worldRegion), " "), placeHint) {
		n.seat.Key("Escape")
	}
}

func (n *Novice) choose(l Look) string {
	on := enabledIDs(l)
	if n.Hasty && on[nextStepButton] {
		return nextStepButton
	}
	for _, ack := range noviceAcks {
		if on[ack] {
			return ack
		}
	}
	if coveredAck(l) && on["calamity:shelve"] {
		return "calamity:shelve"
	}
	if fix := troubleFix(l); fix != "" {
		key := fix + "|" + cardText(l, fix)
		if !n.fixed[key] {
			n.fixed[key] = true
			return fix
		}
		if on["calamity:close"] {
			return "calamity:close"
		}
	}
	if on["calamity:ok"] {
		return "calamity:ok"
	}
	world, lit := worldIDs(l), l.Lit
	if lit != "" && world[lit] {
		return lit
	}
	obj := strings.Join(RegionOf(l, ObjectiveWin), " ")
	if hit := n.namedTarget(l, obj); hit != "" {
		return hit
	}
	switch {
	case lit != "" && on[lit]:
		return lit
	case strings.HasPrefix(lit, "build:") && on["cmd:build"]:
		return "cmd:build"
	case strings.Contains(obj, "ready: press SHOW ME") && on["obj:goal"]:
		return "obj:goal"
	}
	if on["obj:speak"] {
		return "obj:speak"
	}
	if strings.Contains(obj, "ready: open one at") || portStep(l, obj) {
		haven, named := lookHaven(l, obj)
		switch {
		case named, haven != "" && !portStep(l, obj):
			return haven
		case on["obj:goal"]:
			return "obj:goal"
		}
	}
	if on["obj:goal"] && n.shown != n.step {
		n.shown = n.step
		return "obj:goal"
	}
	if pile := pileOf(l); pile != "" && strings.Contains(strings.ToLower(obj), "carry") {
		return pile
	}
	for _, b := range l.Buttons {
		if b.Enabled && strings.HasPrefix(b.ID, "chron:") {
			return b.ID
		}
	}
	if l.Panel != "" && on["overlay:close"] {
		return "overlay:close"
	}
	return ""
}

var noviceAnyHut = regexp.MustCompile(`click an? ([a-z]+) hut`)

func anyHut(l Look, obj string) string {
	m := noviceAnyHut.FindStringSubmatch(obj)
	if m == nil {
		return ""
	}
	for _, o := range l.World {
		if (o.Kind == "hearth" || o.Kind == "walker") && strings.HasPrefix(strings.ToLower(o.Label), m[1]) {
			return o.ID
		}
	}
	return ""
}

func (n *Novice) namedTarget(l Look, obj string) string {
	m := noviceClickOn.FindStringSubmatch(obj)
	if m == nil {
		if hut := anyHut(l, obj); hut != "" && n.picked != hut+"|"+obj {
			n.picked = hut + "|" + obj
			return hut
		}
		return ""
	}
	want := strings.ToLower(strings.TrimSpace(m[1]))
	for _, o := range l.World {
		if o.ID != "hearth:"+want && !(strings.Contains(strings.ToLower(o.Label), want) && (o.Kind == "structure" || o.Kind == "hearth" || o.Kind == "haven")) {
			continue
		}
		if key := o.ID + "|" + obj; n.picked != key {
			n.picked = key
			return o.ID
		}
	}
	return ""
}

func coveredAck(l Look) bool {
	for _, b := range l.Buttons {
		if b.Enabled && b.CoveredBy != "" && slices.Contains(noviceAcks, b.ID) {
			return true
		}
	}
	return false
}

func cardText(l Look, id string) string {
	region := ""
	for _, b := range l.Buttons {
		if b.ID == id {
			region = b.Region
		}
	}
	if region == "" || !slices.Contains(noviceAcks, id) && region != "window:calamity" {
		return ""
	}
	lines := RegionOf(l, region)
	if len(lines) == 0 {
		return region
	}
	return region + "|" + lines[0]
}

func troubleFix(l Look) string {
	for _, b := range l.Buttons {
		if !b.Enabled || b.Region != "window:calamity" || slices.Contains(noviceKeptBack, b.ID) {
			continue
		}
		if strings.HasPrefix(b.ID, "line:cal:") || strings.HasPrefix(b.ID, "calamity:") && !strings.HasPrefix(b.ID, "calamity:more") {
			return b.ID
		}
	}
	return ""
}

func enabledIDs(l Look) map[string]bool {
	out := map[string]bool{}
	for _, b := range l.Buttons {
		if b.Enabled && b.CoveredBy == "" {
			out[b.ID] = true
		}
	}
	return out
}

func worldIDs(l Look) map[string]bool {
	out := map[string]bool{}
	for _, o := range l.World {
		out[o.ID] = true
	}
	return out
}

func portStep(l Look, obj string) bool {
	return l.Lit == "" && (strings.Contains(obj, "ready: open ") || strings.Contains(obj, "ready: upgrade "))
}

func lookHaven(l Look, obj string) (string, bool) {
	first := ""
	for _, o := range l.World {
		if o.Kind != "haven" {
			continue
		}
		if name, _, ok := strings.Cut(o.Label, ":"); ok && strings.Contains(obj, name) {
			return o.ID, true
		}
		if first == "" {
			first = o.ID
		}
	}
	return first, false
}

func pileOf(l Look) string {
	for _, o := range l.World {
		if o.Kind == pileKind {
			return o.ID
		}
	}
	return ""
}

func lookObject(l Look, id string) (LookObject, bool) {
	for _, o := range l.World {
		if o.ID == id {
			return o, true
		}
	}
	return LookObject{}, false
}

func RegionOf(l Look, region string) []string {
	for _, r := range l.Regions {
		if r.Region == region {
			return r.Lines
		}
	}
	return nil
}
