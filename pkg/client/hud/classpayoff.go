package hud

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	classGainFloor = .01
	classBetter    = 1.05
)

func crewOfTrade(st view.GameState, p soulstone.Profile) (view.CrewView, soulstone.Profile, bool) {
	for _, c := range st.Crews {
		cur, ok := soulstone.Lookup(c.Image)
		if ok && cur.Repository == p.Repository {
			return c, cur, true
		}
	}
	return view.CrewView{}, soulstone.Profile{}, false
}

func boxCost(p soulstone.Profile, mix [soulstone.SizeClasses]float64) float64 {
	cost, share := 0.0, 0.0
	for s, w := range mix {
		cost += w * p.BoxMilliS[s]
		share += w
	}
	if share <= 0 {
		return p.BoxMilliS[0]
	}
	return cost / share
}

func ratioText(r float64) string {
	return strconv.FormatFloat(math.Round(r*10)/10, 'f', -1, 64)
}

func classPayoffLine(st view.GameState, image string) string {
	p, ok := soulstone.Lookup(image)
	if !ok || p.Cursed || p.Trade == "" {
		return ""
	}
	c, cur, ok := crewOfTrade(st, p)
	if !ok || cur.Image == p.Image {
		return ""
	}
	parts := []string{}
	if k, ok := kindOf(st.Freight, c.Kind); ok {
		if line := carryPayoff(k, boxCost(cur, k.SizeMix)/max(boxCost(p, k.SizeMix), 1), classPrice(st, p)); line != "" {
			parts = append(parts, line)
		}
	}
	if cur.MTBFSeconds > 0 && p.MTBFSeconds >= classBetter*cur.MTBFSeconds {
		times := ratioText(p.MTBFSeconds / cur.MTBFSeconds)
		parts = append(parts, ui.Bridge("FALLS "+times+"x LESS OFTEN", "RESTARTS "+times+"x LESS OFTEN"))
	}
	return strings.Join(parts, "; ")
}

func classPrice(st view.GameState, p soulstone.Profile) float64 {
	for _, s := range st.Soulstones {
		if s.Image == p.Image {
			return s.Cost
		}
	}
	return p.Cost
}

func withLabel(label, line string) string {
	if line == "" {
		return ""
	}
	return label + line
}

func ClassPayoff(st view.GameState, image string) string {
	return withLabel(ui.Bridge("ENDOWING BRINGS: ", "BUYING BRINGS: "), classPayoffLine(st, image))
}

func rollOutPayoff(st view.GameState, image string) string {
	return withLabel("ROLLING OUT BRINGS: ", classPayoffLine(st, image))
}

func carryPayoff(k view.KindView, r, price float64) string {
	if r < classBetter {
		return ""
	}
	name := strings.ToUpper(k.Name)
	head := ui.Bridge("x"+ratioText(r)+" "+name+" PER RATION", "x"+ratioText(r)+" "+ui.Name(name)+" PER CPU")
	gain := min(model.UnservedRate(k), k.ServedRate*(r-1)) * k.Value
	lighter := ui.Bridge("THE SAME LOAD ON "+lessFood(r, "FOOD"), "THE SAME LOAD ON "+lessFood(r, "CPUS"))
	if gain < classGainFloor {
		return head + ": " + lighter
	}
	amount := goldAmount(gain)
	out := head + ": " + ui.Bridge("+"+amount+" GOLD/S NOW", "+$"+amount+"/S NOW")
	if back := price / gain; price > 0 && back > slowPayback {
		out += ", " + paybackText(back) + "; OR " + lighter
	}
	return out
}

const slowPayback = 1800.0

func lessFood(r float64, what string) string {
	if math.Abs(r-2) <= .05 {
		return "HALF THE " + what
	}
	if what == "CPUS" {
		return fmt.Sprintf("%.0f%% FEWER CPUS", (1-1/r)*100)
	}
	return fmt.Sprintf("%.0f%% LESS %s", (1-1/r)*100, what)
}

func paybackText(seconds float64) string {
	if hours := seconds / 3600; hours >= 2 {
		return fmt.Sprintf("PAYS BACK IN %.0f H", math.Ceil(hours))
	}
	return fmt.Sprintf("PAYS BACK IN %.0f MIN", math.Ceil(seconds/60))
}

func withPayoffLine(tip, payoff string) string {
	if payoff == "" {
		return tip
	}
	return tip + "\n" + payoff
}

func AcademyPayoff(f Frame, s view.SoulstoneEntry) string {
	if s.Owned {
		return ""
	}
	return ClassPayoff(f.State, s.Image)
}

func nextPayoff(n *view.NextView, st view.GameState) string {
	switch n.Kind {
	case view.NextClass:
		return ClassPayoff(st, n.ID)
	case view.NextPage:
		if now := pageNowOf(st, n.ID); now != "" {
			return nowLead + now
		}
	}
	return ""
}
