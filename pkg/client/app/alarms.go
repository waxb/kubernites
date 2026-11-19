package app

import (
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
)

const (
	AlarmCooldown = 600.0
	AlarmLinger   = 8.0
	DangerPrefix  = "danger:"
	repeatGrowth  = 2
	repeatSlack   = 3
)

var RepeatWait = map[string]float64{hud.AlertPending: 420}

var SustainFor = map[string]float64{
	hud.AlertStorm: 0, hud.AlertKeep: 20, hud.AlertGhost: 20, hud.AlertPending: 45, hud.AlertFire: 60, hud.AlertDark: 60,
}

type alarmKind struct {
	rung    bool
	rang    float64
	last    hud.Alert
	until   float64
	voiced  bool
	live    bool
	cleared float64
	peak    int
}

func (k alarmKind) hushed(kind string, realmNow float64, count int) bool {
	wait, ok := RepeatWait[kind]
	if !ok || !k.voiced || k.live || realmNow-k.cleared >= wait {
		return false
	}
	return count < k.peak*repeatGrowth || count < k.peak+repeatSlack
}

type Alarms struct {
	troubled  map[string]float64
	escalated map[string]bool
	kinds     map[string]alarmKind
	danger    []hud.Alert
}

func AlertKey(a hud.Alert) string {
	if a.Key != "" {
		return a.Kind + "|" + a.Key
	}
	return a.Kind + "|" + a.Text
}

func escalates(a hud.Alert, troubled float64) bool {
	need, ok := SustainFor[a.Kind]
	return ok && !a.Routine && troubled >= need
}

func (a Alarms) Step(realmNow, now float64, alerts []hud.Alert) Alarms {
	next := Alarms{troubled: make(map[string]float64, len(alerts)), escalated: map[string]bool{}, kinds: make(map[string]alarmKind, len(a.kinds))}
	for k, v := range a.kinds {
		next.kinds[k] = v
	}
	live, counts := map[string]bool{}, map[string]int{}
	for _, al := range alerts {
		counts[al.Kind]++
	}
	for _, al := range alerts {
		key := AlertKey(al)
		since, ok := a.troubled[key]
		if !ok {
			since = realmNow
		}
		next.troubled[key] = since
		if !a.escalated[key] && (!escalates(al, realmNow-since) || a.kinds[al.Kind].hushed(al.Kind, realmNow, counts[al.Kind])) {
			continue
		}
		next.escalated[key] = true
		next.danger = append(next.danger, al)
		if !live[al.Kind] {
			live[al.Kind] = true
			k := next.kinds[al.Kind]
			k.last, k.until, k.voiced = al, now+AlarmLinger, true
			next.kinds[al.Kind] = k
		}
	}
	return next.withVoices(live, counts, realmNow).withLingering(live, now)
}

func (a Alarms) withVoices(live map[string]bool, counts map[string]int, realmNow float64) Alarms {
	for kind, k := range a.kinds {
		switch {
		case live[kind]:
			k.live, k.peak = true, max(k.peak, counts[kind])
		case k.live:
			k.live, k.cleared = false, realmNow
		}
		a.kinds[kind] = k
	}
	return a
}

func (a Alarms) withLingering(live map[string]bool, now float64) Alarms {
	kinds := make([]string, 0, len(a.kinds))
	for kind := range a.kinds {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		if k := a.kinds[kind]; !live[kind] && now < k.until {
			a.danger = append(a.danger, k.last)
		}
	}
	return a
}

func (a Alarms) Dangers() []hud.Alert {
	return append([]hud.Alert(nil), a.danger...)
}

func (a Alarms) Grade(alerts []hud.Alert) []hud.Alert {
	loud := make(map[string]bool, len(a.danger))
	for _, d := range a.danger {
		loud[AlertKey(d)] = true
	}
	out := make([]hud.Alert, len(alerts))
	for i, al := range alerts {
		al.Quiet = !loud[AlertKey(al)]
		out[i] = al
	}
	return out
}

func (a Alarms) Rung() map[string]float64 {
	out := map[string]float64{}
	for kind, k := range a.kinds {
		if k.rung {
			out[kind] = k.rang
		}
	}
	return out
}

func NewlyRung(before, after map[string]float64) []string {
	out := []string{}
	for kind, at := range after {
		if prev, ok := before[kind]; !ok || prev != at {
			out = append(out, kind)
		}
	}
	sort.Strings(out)
	return out
}

func (a Alarms) loud(kind string, now float64) bool {
	k := a.kinds[kind]
	return !k.rung || now-k.rang >= AlarmCooldown
}

func (a Alarms) rung(kind string, now float64) Alarms {
	kinds := make(map[string]alarmKind, len(a.kinds))
	for k, v := range a.kinds {
		kinds[k] = v
	}
	k := kinds[kind]
	k.rung, k.rang = true, now
	kinds[kind] = k
	a.kinds = kinds
	return a
}

func (a Alarms) arrive(n hud.Notice, now float64) (Alarms, bool) {
	kind, danger := strings.CutPrefix(n.ID, DangerPrefix)
	if !danger {
		return a, true
	}
	if !a.loud(kind, now) {
		return a, false
	}
	return a.rung(kind, now), true
}
