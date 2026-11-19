package sound

import (
	"cmp"
	"maps"
	"slices"
	"time"
)

const (
	ChimeGap   = 3 * time.Second
	FamilyGap  = 9 * time.Second
	AlertGap   = 5 * time.Second
	FanfareGap = 8 * time.Second
)

type cueClass int

const (
	cueFeedback cueClass = iota
	cueChime
	cueAlert
	cueFanfare
)

type cueKind struct {
	class  cueClass
	family string
	ducks  bool
}

var cueClasses = map[SFX]cueKind{
	SFXClick: {class: cueFeedback}, SFXCoin: {class: cueFeedback}, SFXBuild: {class: cueFeedback},
	SFXError: {class: cueFeedback}, SFXRollout: {class: cueFeedback}, SFXChronicle: {class: cueFeedback},
	SFXBoat: {class: cueChime, family: "hearth"}, SFXPodStart: {class: cueChime, family: "hearth"},
	SFXBuilt: {class: cueChime, family: "works"}, SFXHorn: {class: cueChime, family: "works"},
	SFXReveal: {class: cueChime, family: "news", ducks: true}, SFXUnlock: {class: cueChime, family: "news", ducks: true},
	SFXReady:     {class: cueChime, family: "news"},
	SFXPage:      {class: cueFeedback},
	SFXGleamLast: {class: cueFeedback},
	SFXCalamity:  {class: cueAlert, family: "alert", ducks: true}, SFXTremor: {class: cueAlert, family: "alert", ducks: true},
	SFXDrums:   {class: cueAlert, family: "alert", ducks: true},
	SFXVictory: {class: cueFanfare, family: "fanfare", ducks: true},
}

type Budget struct {
	last     time.Time
	families map[string]time.Time
	hush     time.Time
	calm     time.Time
}

func NewBudget() Budget {
	return Budget{families: map[string]time.Time{}}
}

func Ducks(id SFX) bool {
	return cueClasses[id].ducks
}

func within(then, now time.Time, gap time.Duration) bool {
	return !then.IsZero() && now.Sub(then) < gap
}

func (b Budget) Hushed(now, until time.Time) Budget {
	next := b
	if !b.hushing(now) {
		next.calm = now
	}
	next.hush = until
	return next
}

func (b Budget) hushing(now time.Time) bool {
	return now.Before(b.hush)
}

func (b Budget) Admit(id SFX, now time.Time) (Budget, bool) {
	kind := cueClasses[id]
	if kind.class == cueFeedback {
		return b, true
	}
	if !b.open(kind, now) {
		return b, false
	}
	return b.heard(kind, now), true
}

func (b Budget) open(kind cueKind, now time.Time) bool {
	switch {
	case kind.class == cueFanfare:
		return !within(b.families[kind.family], now, FanfareGap) || b.heralds(kind, now)
	case b.hushing(now):
		return false
	case kind.class == cueAlert:
		return !within(b.families[kind.family], now, AlertGap)
	}
	return !within(b.last, now, ChimeGap) && !within(b.families[kind.family], now, FamilyGap)
}

func (b Budget) heralds(kind cueKind, now time.Time) bool {
	return b.hushing(now) && b.families[kind.family].Before(b.calm)
}

func (b Budget) heard(kind cueKind, now time.Time) Budget {
	next := b
	next.families = maps.Clone(b.families)
	if next.families == nil {
		next.families = map[string]time.Time{}
	}
	next.families[kind.family] = now
	next.last = now
	return next
}

func (b Budget) Pick(ids []SFX, now time.Time) (Budget, []SFX) {
	order := slices.Clone(ids)
	slices.SortStableFunc(order, func(x, y SFX) int {
		kx, ky := cueClasses[x], cueClasses[y]
		if c := cmp.Compare(ky.class, kx.class); c != 0 {
			return c
		}
		return cmp.Compare(priorityOf(y), priorityOf(x))
	})
	next, out := b, []SFX{}
	for _, id := range order {
		var ok bool
		if next, ok = next.Admit(id, now); ok {
			out = append(out, id)
		}
	}
	return next, out
}

func Feedback(id SFX) bool {
	return cueClasses[id].class == cueFeedback
}
