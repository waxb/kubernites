package hud

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
)

const (
	BeatGap   = 2.5
	BeatQuiet = 4.0
	SoakGap   = 15.0

	BeatCard   = "card"
	BeatStep   = "step"
	BeatDone   = "done"
	BeatBanner = "banner"
	BeatToast  = "toast"
	BeatReveal = "reveal"
	BeatNotice = "notice"
	BeatGone   = "gone"
	BeatPanel  = "panel"
	BeatShut   = "shut"

	RuleBeatGap   = "two new things less than 2.5 s apart"
	RuleBeatQuiet = "something new inside the 4 s of quiet after an achievement"
	RuleBeatWaits = "something new while a card waits for GOT IT or X"
	RuleBeatFlash = "a waiting card set aside and shown again within the quiet"
	RuleBeatFeat  = "an achievement over a card that waits for GOT IT or X"
	RuleBeatPanel = "something new over an open Chronicle, Grimoire, Academy or AGE panel"
	RuleBeatSoak  = "a new idea inside the 15 s of quiet after the last one"
)

var readingPanels = []string{OverlayChronicles, OverlayGrimoire, OverlaySoulvault, OverlayAge}

type Beat struct {
	At     float64
	Realm  float64
	Kind   string
	Key    string
	Waits  bool
	Feat   bool
	Part   bool
	Idea   bool
	Urgent bool
	Asked  bool
}

type BeatFault struct {
	Rule  string
	At    float64
	Realm float64
	Keys  []string
}

func (b BeatFault) String() string {
	sec := int(b.Realm)
	return fmt.Sprintf("%02d:%02d (wall %.1f s) %s: %s", sec/60, sec%60, b.At, b.Rule, strings.Join(b.Keys, ", "))
}

const (
	beatGrace = 0.3
	maxBeats  = 20000
)

type beatLedger struct {
	primed   bool
	Shown    map[string]Beat `json:"-"`
	missing  map[string]float64
	revealed map[string]bool
	drawn    map[string]bool
	Beats    []Beat `json:"-"`
}

func (h *HUD) Beats() []Beat { return slices.Clone(h.Ledger.Beats) }

func (l beatLedger) clone() beatLedger {
	out := l
	out.Shown, out.missing, out.revealed, out.drawn = maps.Clone(l.Shown), maps.Clone(l.missing), maps.Clone(l.revealed), maps.Clone(l.drawn)
	out.Beats = slices.Clone(l.Beats)
	return out
}

func (h *HUD) noteBeats(f Frame) {
	now := h.screenBeats(f)
	h.noteDrawn(now)
	reveals := h.RevealKeys(f)
	if !h.Ledger.primed {
		h.Ledger = beatLedger{primed: true, Shown: now, missing: map[string]float64{}, revealed: map[string]bool{}, drawn: h.Ledger.drawn}
		for _, k := range reveals {
			h.Ledger.revealed[k] = true
		}
		return
	}
	if !h.CardsDrawn(f) || h.QuestCardShown(f) {
		now = h.coveredBeats(now)
	}
	fresh := []Beat{}
	for key, b := range now {
		if _, was := h.Ledger.Shown[key]; !was {
			fresh = append(fresh, b)
		}
	}
	for _, k := range reveals {
		if !h.Ledger.revealed[k] {
			h.Ledger.revealed[k] = true
			fresh = append(fresh, Beat{Kind: BeatReveal, Key: "reveal:" + k, Part: h.namedReveal(k, f)})
		}
	}
	gone := []Beat{}
	missing := map[string]float64{}
	for key, b := range h.Ledger.Shown {
		if _, still := now[key]; still {
			continue
		}
		since, ok := h.Ledger.missing[key]
		if !ok {
			since = h.Now
		}
		if h.Now-since < beatGrace {
			missing[key], now[key] = since, b
			continue
		}
		switch {
		case b.Waits:
			gone = append(gone, Beat{Kind: BeatGone, Key: key, At: since})
		case b.Kind == BeatPanel:
			gone = append(gone, Beat{Kind: BeatShut, Key: key, At: since})
		}
	}
	h.Ledger.missing = missing
	sortBeats(fresh)
	sortBeats(gone)
	fresh = h.RidingParts(revealsAsParts(fresh))
	h.quietAfterDrawnFeats(fresh)
	next := slices.Clone(h.Ledger.Beats)
	for _, b := range gone {
		b.Realm = f.State.RealmTime
		next = append(next, b)
	}
	for _, b := range fresh {
		b.At, b.Realm, b.Asked = h.Now, f.State.RealmTime, h.askedKey(b.Key)
		next = append(next, b)
	}
	h.Ledger.Beats, h.Ledger.Shown = next[max(len(next)-maxBeats, 0):], now
}

func (h *HUD) noteDrawn(now map[string]Beat) {
	for key, b := range now {
		if b.Idea && !h.Ledger.drawn[key] {
			h.Ledger.drawn = withKey(h.Ledger.drawn, key)
		}
	}
}

func (h *HUD) RidingParts(fresh []Beat) []Beat {
	riding := slices.ContainsFunc(h.Ledger.Beats, func(b Beat) bool { return b.At == h.Now && !b.Part && !closing(b) && b.Kind != BeatPanel })
	if !riding {
		return fresh
	}
	out := slices.Clone(fresh)
	for i := range out {
		if out[i].Kind == BeatReveal {
			out[i].Part = true
		}
	}
	return out
}

func revealsAsParts(fresh []Beat) []Beat {
	out := slices.Clone(fresh)
	lead := slices.ContainsFunc(out, func(b Beat) bool { return b.Kind != BeatReveal && !b.Part })
	for i := range out {
		if out[i].Kind != BeatReveal || out[i].Part {
			continue
		}
		if lead {
			out[i].Part = true
			continue
		}
		lead = true
	}
	return out
}

func (h *HUD) coveredBeats(now map[string]Beat) map[string]Beat {
	out := map[string]Beat{}
	for k, b := range now {
		out[k] = b
	}
	for k, b := range h.Ledger.Shown {
		if _, ok := out[k]; !ok && b.Kind != BeatPanel {
			out[k] = b
		}
	}
	return out
}

func sortBeats(list []Beat) {
	sort.SliceStable(list, func(i, j int) bool { return list[i].Key < list[j].Key })
}

func (h *HUD) screenBeats(f Frame) map[string]Beat {
	out := map[string]Beat{}
	add := func(b Beat) { out[b.Key] = b }
	if slices.Contains(readingPanels, h.OverlayID) {
		add(Beat{Kind: BeatPanel, Key: "panel:" + h.OverlayID})
	}
	for _, b := range h.cardBeats(f) {
		add(b)
	}
	if h.CardsDrawn(f) && h.ObjectiveShown(f) {
		if o, ok := f.ShownObjective(); ok && !h.trialShown() {
			if o.Done {
				add(Beat{Kind: BeatDone, Key: "done:" + o.ID, Feat: true})
			} else {
				add(Beat{Kind: BeatStep, Key: "step:" + o.ID})
			}
		}
	}
	if _, ok := h.BannerRect(f); ok && (h.CardsDrawn(f) || h.Juice.feat && h.OverlayID == OverlayAge) {
		add(Beat{Kind: BeatBanner, Key: "banner:" + h.Juice.Banner, Feat: h.Juice.feat})
	}
	if !h.toastsHeld(f) {
		for _, t := range h.ToastList {
			if t.News {
				add(Beat{Kind: BeatToast, Key: "toast:" + t.Text, Part: t.Fanfare})
			}
		}
	}
	for _, n := range f.Notices {
		if !n.Quiet {
			add(Beat{Kind: BeatNotice, Key: "notice:" + n.ID, Idea: n.Kind == NoticeDecision})
		}
	}
	return out
}

func (h *HUD) cardBeats(f Frame) []Beat {
	out := []Beat{}
	if h.IntroShown(f) {
		out = append(out, Beat{Kind: BeatCard, Key: "intro:" + f.Guide.Intro.ID, Waits: true, Idea: true})
	}
	if c, ok := h.QuestCard(f); ok {
		out = append(out, Beat{Kind: BeatCard, Key: "quest:" + c.key, Waits: true, Idea: true})
	}
	if t := h.LessonTitle(); t != "" && h.LessonShown(f) {
		out = append(out, Beat{Kind: BeatCard, Key: "lesson:" + t, Waits: true, Idea: true, Urgent: h.LessonCard.timely})
	}
	if v, ok := h.ShownCalamity(f); ok && h.CardsDrawn(f) {
		if v.Mended != "" {
			out = append(out, Beat{Kind: BeatCard, Key: "mended:" + v.ID, Feat: true})
		} else {
			out = append(out, Beat{Kind: BeatCard, Key: "trouble:" + v.ID, Waits: true, Idea: true, Urgent: urgentTrouble(v.Kind)})
		}
	}
	if h.DawnLinesShown(f) {
		out = append(out, Beat{Kind: BeatCard, Key: "dawn:" + f.State.Age, Part: true})
	}
	return out
}

func (h *HUD) RevealKeys(f Frame) []string {
	out := []string{}
	for key, on := range map[string]bool{keyStrip: h.StripShown(f), KeyMinimap: MinimapShown(f), KeyCard: f.Shown(SurfaceGate) && !f.State.Tutorial, keyWell: h.WellShown(f)} {
		if on {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	for _, st := range h.TopStats(f) {
		if st.Meter != "" {
			out = append(out, st.Meter)
		}
	}
	if BoardShown(f) {
		out = append(out, SurfaceFreight)
	}
	for _, t := range h.visibleTabs(f) {
		out = append(out, "tab:"+t)
	}
	main, more := h.MainCard(f)
	for _, b := range append(main, more...) {
		if b.ID != CmdMore {
			out = append(out, b.ID)
		}
	}
	return out
}

func (h *HUD) namedReveal(key string, f Frame) bool {
	keys := []string{key}
	for _, t := range wellTabs {
		if key == "tab:"+t.tab {
			keys = append(keys, t.surface)
		}
	}
	for _, goal := range []string{introTarget(f), ObjectiveGoal(f), f.Guide.Button} {
		if goal == "" {
			continue
		}
		path := NavPath(goal, NavState{Page: PageMain})
		for _, k := range keys {
			if goal == k || NavAliases[goal] == k || slices.Contains(path, k) {
				return true
			}
		}
	}
	return false
}

func CheckBeats(list []Beat) []BeatFault {
	beats := withoutInstantPanels(list)
	sort.SliceStable(beats, func(i, j int) bool {
		if beats[i].At != beats[j].At {
			return beats[i].At < beats[j].At
		}
		return closing(beats[i]) && !closing(beats[j])
	})
	out := []BeatFault{}
	var last *Beat
	quietUntil, quietFrom := -1.0, ""
	soakUntil, soakFrom := -1.0, ""
	ideas := map[string]bool{}
	waiting := map[string]float64{}
	gone := map[string]float64{}
	panels := map[string]bool{}
	for i := range beats {
		b := beats[i]
		switch b.Kind {
		case BeatGone:
			delete(waiting, b.Key)
			gone[b.Key] = b.At
			if ideas[b.Key] && b.At+SoakGap > soakUntil {
				soakUntil, soakFrom = b.At+SoakGap, b.Key
			}
			continue
		case BeatShut:
			delete(panels, b.Key)
			continue
		case BeatPanel:
			panels[b.Key] = true
			continue
		}
		if b.Part {
			continue
		}
		if b.Asked {
			if b.Feat {
				quietUntil, quietFrom = b.At+BeatQuiet, b.Key
			}
			if b.Waits {
				waiting[b.Key] = b.At
			}
			last = &beats[i]
			continue
		}
		if last != nil && b.At-last.At < BeatGap && !b.Feat {
			out = append(out, fault(RuleBeatGap, b, last.Key))
		}
		if b.At < quietUntil && !b.Feat {
			out = append(out, fault(RuleBeatQuiet, b, quietFrom))
		}
		if b.Idea && !b.Urgent && !ideas[b.Key] && b.At < soakUntil {
			out = append(out, fault(RuleBeatSoak, b, soakFrom))
		}
		if others := waitingOthers(waiting, b.Key); len(others) > 0 && !b.Feat && b.Kind != BeatStep {
			out = append(out, fault(RuleBeatWaits, b, others...))
		}
		if since, was := gone[b.Key]; was && b.Waits && !strings.HasPrefix(b.Key, "trouble:") && b.At-since < BeatQuiet+BeatGap {
			out = append(out, fault(RuleBeatFlash, b))
		}
		if others := nonTrouble(waitingOthers(waiting, b.Key)); len(others) > 0 && b.Feat && b.Kind != BeatDone {
			out = append(out, fault(RuleBeatFeat, b, others...))
		}
		if len(panels) > 0 && b.Kind != BeatReveal && !b.Feat && !strings.HasPrefix(b.Key, "dawn:") {
			out = append(out, fault(RuleBeatPanel, b, panelKeys(panels)...))
		}
		if b.Feat {
			quietUntil, quietFrom = b.At+BeatQuiet, b.Key
		}
		if b.Waits {
			waiting[b.Key] = b.At
		}
		if b.Idea && !b.Urgent {
			ideas[b.Key] = true
		}
		if b.Idea && !b.Urgent && !b.Waits && b.At+SoakGap > soakUntil {
			soakUntil, soakFrom = b.At+SoakGap, b.Key
		}
		last = &beats[i]
	}
	return out
}

func withoutInstantPanels(list []Beat) []Beat {
	seen := map[string]map[string]bool{}
	at := func(b Beat) string { return fmt.Sprintf("%s@%v", b.Key, b.At) }
	for _, b := range list {
		if b.Kind == BeatPanel || b.Kind == BeatShut {
			if seen[at(b)] == nil {
				seen[at(b)] = map[string]bool{}
			}
			seen[at(b)][b.Kind] = true
		}
	}
	out := []Beat{}
	for _, b := range list {
		if (b.Kind == BeatPanel || b.Kind == BeatShut) && seen[at(b)][BeatPanel] && seen[at(b)][BeatShut] {
			continue
		}
		out = append(out, b)
	}
	return out
}

func closing(b Beat) bool { return b.Kind == BeatGone || b.Kind == BeatShut }

func waitingOthers(waiting map[string]float64, key string) []string {
	out := []string{}
	for k := range waiting {
		if k != key {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func panelKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func fault(rule string, b Beat, with ...string) BeatFault {
	return BeatFault{Rule: rule, At: b.At, Realm: b.Realm, Keys: append([]string{b.Key}, with...)}
}

func (h *HUD) quietAfterDrawnFeats(fresh []Beat) {
	s := &h.Director.Stage
	if !s.Primed {
		return
	}
	for _, b := range fresh {
		if b.Feat && !b.Part {
			s.quiet = max(s.quiet, h.Now+BeatQuiet)
		}
	}
}

func nonTrouble(keys []string) []string {
	out := []string{}
	for _, k := range keys {
		if !strings.HasPrefix(k, "trouble:") {
			out = append(out, k)
		}
	}
	return out
}
