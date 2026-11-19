package hud

import (
	"fmt"
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	SurfaceFreight = "surface:freight"
	SurfaceLosses  = "surface:losses"
	minLoss        = .01
	FlashFor       = 6.0
)

type KindChip struct {
	Kind    string
	Name    string
	Waiting int
	Fresh   float64
	Level   int
	Served  float64
	Lost    float64
	Tip     string
}

type FreightHead struct {
	Tier     int
	Progress float64
	Count    string
	Goal     string
	Chips    []KindChip
}

type LossLine struct {
	Cause string
	Rate  float64
	Text  string
}

func FreightHeadOf(fv view.FreightView, spoils bool, crews ...view.CrewView) FreightHead {
	h := FreightHead{Tier: max(fv.Tier, 1), Progress: fv.TierProgress, Goal: TierGoal(fv)}
	h.Count = Money(Compact(fv.TierProgress*fv.NextTier)) + "/" + Compact(fv.NextTier)
	if fv.NextTier <= 0 {
		h.Count = Compact(fv.Fresh) + " fresh"
	}
	for _, k := range fv.Kinds {
		if k.Open || k.Waiting >= .5 {
			h.Chips = append(h.Chips, ChipOf(k, spoils, crews...))
		}
	}
	return h
}

func TierGoal(fv view.FreightView) string {
	tier := max(fv.Tier, 1)
	switch {
	case fv.NextWait != "":
		return fmt.Sprintf("Tier %d: %s", tier+1, fv.NextWait)
	case fv.NextTier > 0:
		next := view.HarborTierOf(max(fv.NextLevel, 2))
		cost := Gold(fv.NextTier)
		return ui.Bridge(fmt.Sprintf("%s: %s, %s", fv.NextPortName, next.Ships, cost),
			fmt.Sprintf("%s: %s, %s", ui.Name(fv.NextPortName), listenerGoal(next), cost))
	case tier >= view.MaxTier:
		return fmt.Sprintf("Tier %d: the top of the harbor ladder", tier)
	}
	return fmt.Sprintf("Tier %d: no port upgrade on offer now", tier)
}

func listenerGoal(level view.HarborTierDef) string {
	if level.Listener == "" {
		return fmt.Sprintf("level %d", level.Tier)
	}
	return "+" + level.Listener
}

func tierBoxes(tier int, opens string) string {
	name := ""
	for _, k := range view.BoxKinds {
		if k.Opens == tier && !k.Staged {
			name = k.Name
		}
	}
	if name != "" {
		return name
	}
	parts := strings.Split(strings.ReplaceAll(opens, " and ", ", "), ", ")
	return parts[len(parts)-1]
}

func OldestFresh(k view.KindView) float64 {
	fresh := 1.0
	for _, c := range k.Cohorts {
		if c.TTL > 0 && c.Count >= .5 {
			fresh = min(fresh, c.Remaining/c.TTL)
		}
	}
	return fresh
}

func ChipOf(k view.KindView, spoils bool, crews ...view.CrewView) KindChip {
	fresh := OldestFresh(k)
	c := KindChip{Kind: k.Kind, Name: strings.ToUpper(k.Name), Waiting: int(k.Waiting + .5), Fresh: fresh, Level: gfx.FreshLevel(fresh),
		Served: k.ServedRate, Tip: ChipTip(k, fresh, spoils, crews...)}
	u, ok := model.UnservedOf(k)
	return unservedChip(c, u, ok)
}

func ChipTip(k view.KindView, fresh float64, spoils bool, crews ...view.CrewView) string {
	lines := []string{fmt.Sprintf("%s: %d waiting on the quay, %s carried each second.", strings.ToUpper(k.Name), int(k.Waiting+.5), Rate(k.ServedRate))}
	if k.Waiting >= .5 {
		lines = append(lines, freshLine(k, fresh, spoils))
	}
	lines = append(lines, fmt.Sprintf("Each pays %s gold fresh, less as it ages.", Compact(k.Value*1.25)), crewState(k))
	if c, ok := CrewOf(crews, k.Depot, k.CrewName); ok && SeasonLine(c) != "" {
		lines = append(lines, "The "+k.CrewName+": "+SeasonLine(c))
	}
	return strings.Join(lines, "\n")
}

func freshLine(k view.KindView, fresh float64, spoils bool) string {
	box := strings.TrimSuffix(k.Name, "s")
	if !spoils {
		return fmt.Sprintf("The oldest keeps %.0f%% of its freshness; no %s spoils yet: it waits until carried.", fresh*100, box)
	}
	return fmt.Sprintf("The oldest keeps %.0f%% of its freshness; a %s spoils after %.0fs.", fresh*100, box, k.TTL)
}

func BoxesSpoil(gs view.GameState) bool {
	if gs.Age != view.AgeDark {
		return true
	}
	for _, o := range gs.Objectives {
		switch {
		case o.Done && (o.ID == view.SpoilObjective || o.ID == view.HandPortersObjective):
			return true
		case !o.Done:
			return o.ID == view.SpoilObjective
		}
	}
	return false
}

func crewState(k view.KindView) string {
	switch {
	case !k.DepotOpen && k.Door:
		return fmt.Sprintf("The door porter carries them; the %s depot is not open.", k.Depot)
	case !k.DepotOpen:
		return fmt.Sprintf("No %s depot: open it, then raise %s.", k.Depot, k.CrewName)
	case k.Crew == 0:
		return fmt.Sprintf("The %s depot stands empty: raise %s.", k.Depot, k.CrewName)
	}
	return fmt.Sprintf("%d of the %s carry them to the %s depot.", k.Crew, k.CrewName, k.Depot)
}

func Flashing(fv view.FreightView, realmTime float64) bool {
	return fv.Tier > 1 && fv.TierAt > 0 && realmTime >= fv.TierAt && realmTime-fv.TierAt < FlashFor
}

type lossSource struct {
	name, depot, crew, kind string
	losses                  view.LossView
}

func lossSources(fv view.FreightView) []lossSource {
	out := make([]lossSource, 0, len(fv.Kinds)+1)
	for _, k := range fv.Kinds {
		out = append(out, lossSource{name: k.Name, depot: k.Depot, crew: k.CrewName, kind: k.Kind, losses: k.Losses})
	}
	s := fv.Seals
	return append(out, lossSource{name: "sealed boxes", depot: s.Depot, crew: s.CrewName, kind: "sealed", losses: s.Losses})
}

func worst(fv view.FreightView, pick func(view.LossView) float64) lossSource {
	best := lossSource{}
	top := -1.0
	for _, s := range lossSources(fv) {
		if v := pick(s.losses); v > top {
			best, top = s, v
		}
	}
	return best
}

var lossCauses = []struct {
	cause string
	pick  func(view.LossView) float64
	fix   func(lossSource, view.GameState) string
}{
	{"Spoiled", func(l view.LossView) float64 { return l.Spoiled }, func(s lossSource, st view.GameState) string {
		return cardFix(st, view.CalamitySpoiling, s.kind, "carried too slowly")
	}},
	{"Quay full", func(l view.LossView) float64 { return l.QuayFull }, func(s lossSource, st view.GameState) string {
		return cardFix(st, view.CalamitySpoiling, s.kind, "the quay overflows")
	}},
	{"No depot", func(l view.LossView) float64 { return l.NoDepot }, func(s lossSource, st view.GameState) string {
		return cardFix(st, view.CalamityNoDepot, s.kind, ui.Bridge("open "+s.depot, "expose "+s.crew))
	}},
	{"No crew", func(l view.LossView) float64 { return l.NoCrew }, func(s lossSource, st view.GameState) string {
		return cardFix(st, view.CalamityNoCrew, s.kind, "raise "+s.crew)
	}},
	{"Lost in flight", func(l view.LossView) float64 { return l.Blocked }, func(_ lossSource, st view.GameState) string {
		if st.Unlocks[familiarsUnlock] {
			return ui.Bridge("bind a Hound", "add a proxy sidecar")
		}
		return "worker stopped"
	}},
}

var lossLabelK8s = map[string]string{"Lost in flight": "Dropped"}

func lossLabel(cause string) string {
	if k8s, ok := lossLabelK8s[cause]; ok {
		return ui.Bridge(cause, k8s)
	}
	return cause
}

func LossLines(fv view.FreightView) []LossLine {
	return StateLossLines(view.GameState{Freight: fv})
}

func StateLossLines(st view.GameState) []LossLine {
	out := []LossLine{}
	fv := st.Freight
	for _, c := range lossCauses {
		rate := c.pick(fv.Losses)
		if rate < minLoss {
			continue
		}
		head := fmt.Sprintf("%s %s/s: ", lossLabel(c.cause), Rate(rate))
		fix := ui.Clip(c.fix(worst(fv, c.pick), st), max(lossLineChars-len(head), 8))
		out = append(out, LossLine{Cause: c.cause, Rate: rate, Text: head + fix})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Rate > out[j].Rate })
	return out
}

func LossTotal(fv view.FreightView) float64 {
	l := fv.Losses
	return l.Spoiled + l.QuayFull + l.NoDepot + l.NoCrew + l.Blocked
}
