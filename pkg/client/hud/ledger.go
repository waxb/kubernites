package hud

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayLedger   = "ledger"
	LedgerButton    = "top:ledger"
	neutralSamples  = 20
	sloFloor        = .6
	sloWeight       = .4
	nodeCrews       = "crews"
	NodeShips       = "ships"
	NodeTier        = "tier"
	NodeGold        = "gold"
	NodeHands       = "hands"
	nodeSpend       = "spend"
	nodeNines       = "nines"
	nodeRenown      = "renown"
	nodeWords       = "words"
	nodeStudy       = "study"
	NodeWisdom      = "wisdom"
	nodeHandbook    = "handbook"
	ledgerTitleText = "HOW THE REALM WORKS"
)

type LedgerNode struct {
	ID    string
	Title string
	Lines []string
	Col   int
	Row   int
}

type LedgerEdge struct {
	From, To string
	Label    string
}

var LedgerEdgesOf = []LedgerEdge{
	{NodeShips, nodeCrews, "carried by"}, {nodeCrews, NodeGold, "pay"}, {NodeGold, nodeSpend, "buys"},
	{nodeSpend, NodeHands, "add food"}, {NodeHands, nodeCrews, "each needs food"},
	{nodeCrews, nodeNines, "fresh in time"}, {nodeNines, NodeShips, "draw"}, {nodeRenown, NodeShips, "draw"},
	{nodeCrews, NodeTier, "fresh value"}, {NodeTier, NodeGold, "x each box"},
	{nodeWords, NodeWisdom, "teach"}, {NodeWisdom, nodeHandbook, "+ gold"},
	{nodeCrews, nodeStudy, "deliver"}, {nodeStudy, NodeWisdom, "teach"},
}

func burning(pods []model.PodView) int {
	n := 0
	for _, p := range pods {
		if !p.System && p.State == model.HearthReady {
			n++
		}
	}
	return n
}

func SLOFactor(f Frame) float64 {
	if f.State.SLO.Samples < neutralSamples {
		return 1
	}
	return sloFloor + sloWeight*min(max(f.State.SLO.Current, 0), 1)
}

func pagesLearned(f Frame) (int, int) {
	learned, total := 0, 0
	for _, p := range f.State.Grimoire {
		if p.State == "hidden" {
			continue
		}
		total++
		if p.Rank > 0 || p.State == "unlocked" {
			learned++
		}
	}
	return learned, total
}

func nodeShown(id string, f Frame) bool {
	switch id {
	case nodeCrews:
		return (f.Revealed("door") && f.Shown("cmd:door")) || (f.Revealed("summon") && f.Shown("cmd:summon")) || burning(f.Pods) > 0
	case NodeTier:
		return f.Revealed("freight") && f.Shown(SurfaceFreight)
	case NodeHands:
		return f.Revealed("hands") && f.Shown(MeterHands)
	case nodeSpend:
		return f.Revealed("build") && f.Shown("cmd:build")
	case nodeNines:
		return f.Revealed("slo") && f.Shown(MeterNines)
	case nodeRenown:
		return f.Revealed("renown") && f.Shown(MeterRenown)
	case nodeWords:
		return f.Revealed("stone") && f.Shown(SurfaceStone)
	case NodeWisdom, nodeStudy:
		return f.Revealed("wisdom") && f.Shown(MeterWisdom)
	case nodeHandbook:
		return f.Revealed("grimoire") && f.Shown("cmd:grimoire")
	}
	return true
}

func LedgerNodes(f Frame) []LedgerNode {
	s := f.State
	hands, _ := SupplyOf(f.Metrics.Nodes)
	upkeep, _ := Upkeep(f)
	learned, total := pagesLearned(f)
	all := []LedgerNode{
		{ID: nodeNines, Title: "THE NINES", Col: 1, Lines: []string{Nines(s.SLO.Current) + " delivered fresh", fmt.Sprintf("ships x%.2f", SLOFactor(f))}},
		{ID: nodeRenown, Title: "RENOWN", Lines: []string{ui.Bridge("grows with each box delivered", "grows per request served"), fmt.Sprintf("ships x%.2f", s.Renown)}},
		{ID: NodeTier, Title: "HARBOR TIER", Col: 2, Lines: tierLines(s.Freight)},
		{ID: NodeShips, Title: "SHIPS AND THE QUAY", Row: 1, Lines: quayLines(s.Freight)},
		{ID: nodeCrews, Title: "DEPOTS AND CREWS", Col: 1, Row: 1, Lines: crewLines(f)},
		{ID: NodeGold, Title: "GOLD", Col: 2, Row: 1, Lines: []string{Compact(s.Gold) + " in the treasury", Signed(s.GoldPerSec) + " each second"}},
		{ID: NodeHands, Title: "FOOD ON KEEPS", Col: 1, Row: 2, Lines: []string{foodCount(hands.Used*view.RationsPerCPU) + " of " + foodCount(hands.Total*view.RationsPerCPU) + " pledged", "none left: huts Unhoused"}},
		{ID: nodeSpend, Title: "KEEPS AND BUILDINGS", Col: 2, Row: 2, Lines: []string{"and Academy classes", upkeepLineIn(f, upkeep)}},
		{ID: nodeWords, Title: ui.Bridge("SPOKEN WORDS", "KUBECTL WORDS"), Row: 3, Lines: []string{fmt.Sprintf("%d spoken, +%s a word", len(s.Fluency), Wisdom(view.FluencyWisdom)), "Chronicles +" + Wisdom(view.ChronicleWisdom)}},
		{ID: nodeStudy, Title: "BOXES AND OBJECTIVES", Row: 2, Lines: []string{fmt.Sprintf("+1 wisdom / %.0f delivered", s.Study.Every), fmt.Sprintf("+%g per objective met", s.Study.Objective)}},
		{ID: NodeWisdom, Title: "WISDOM", Col: 1, Row: 3, Lines: []string{Compact(s.Wisdom) + " gathered", StudyLeft(f) + " boxes to the next"}},
		{ID: nodeHandbook, Title: "GRIMOIRE PAGES", Col: 2, Row: 3, Lines: []string{fmt.Sprintf("%d of %d learned", learned, total), "teach new kubectl powers"}},
	}
	out := []LedgerNode{}
	for _, n := range all {
		if nodeShown(n.ID, f) {
			out = append(out, n)
		}
	}
	return out
}

func tierLines(fv view.FreightView) []string {
	first := fmt.Sprintf("deepest port level %d", max(fv.Tier, 1))
	if fv.NextTier <= 0 {
		return []string{first, "the top of the ladder"}
	}
	return []string{first, fmt.Sprintf("%s gold to %ss", Compact(fv.NextTier), fv.NextShip)}
}

func quayLines(fv view.FreightView) []string {
	rate := Rate(arrivals(fv))
	return []string{ui.Bridge(rate+" boxes land a second", rate+" requests arrive/s"), fmt.Sprintf("%d of %d wait on the quay", int(fv.Waiting+.5), int(fv.QuayCap))}
}

func arrivals(fv view.FreightView) float64 {
	total := 0.0
	for _, k := range fv.Kinds {
		total += k.ArrivalRate
	}
	return total
}

func crewLines(f Frame) []string {
	depots := 0
	for _, k := range f.State.Freight.Kinds {
		if k.DepotOpen {
			depots++
		}
	}
	if f.State.Freight.Seals.DepotOpen {
		depots++
	}
	return []string{fmt.Sprintf("%s, %s ready", plural(depots, "depot", "depots"), plural(burning(f.Pods), "hut", "huts")),
		ui.Bridge(Rate(f.State.Freight.FreshPerSec)+" fresh boxes/s", Rate(f.State.Freight.FreshPerSec)+" on-time requests/s")}
}

func LedgerEdges(nodes []LedgerNode) []LedgerEdge {
	has := map[string]bool{}
	for _, n := range nodes {
		has[n.ID] = true
	}
	out := []LedgerEdge{}
	for _, e := range LedgerEdgesOf {
		if has[e.From] && has[e.To] {
			out = append(out, e)
		}
	}
	return out
}

func tipLines(lines ...string) string {
	return strings.Join(lines, "\n")
}

func GoldTip(f Frame) string {
	upkeep, keeps := Upkeep(f)
	e := EconomyOf(f.State)
	out := []string{"GOLD " + Compact(f.State.Gold) + ", " + Signed(NetGoldRate(f.State)) + " each second",
		"+/S is what your crews' deliveries earn each second, a recent average, less upkeep; a box you carry by hand pays at once instead."}
	switch {
	case NetFloored(f.State):
		out = append(out, "= income "+Rate(e.Income)+" - upkeep "+Rate(RentTaken(e))+" paid each second, a recent average: that is the profit.",
			"The "+view.Content.AgeTitle(f.State.Age)+" floors the net at 0: it takes upkeep only from what the realm earns, never from your gold ("+Rate(e.Upkeep)+" due, "+Rate(RentTaken(e))+" paid each second).")
	case e.Upkeep != 0:
		out = append(out, "= income "+Rate(e.Income)+" - upkeep "+Rate(RentTaken(e))+" each second, a recent average: that is the profit.")
	case e.Income != 0:
		out = append(out, "= income "+Rate(e.Income)+" each second, a recent average: that is the profit.")
	}
	out = append(out, deliveredLine(e), "Spent on: Keeps, buildings and Academy classes.")
	if upkeep > 0 {
		out = append(out, "Upkeep now: -"+Rate(upkeep)+" each second for "+plural(keeps, "Keep", "Keeps")+".")
	}
	return tipLines(append(out, "F1: how the realm works")...)
}

func deliveredLine(e view.EconomyView) string {
	if e.Carried < .005 || e.Income <= 0 {
		return "From: boxes your crews deliver, worth more fresh and at a higher harbor tier."
	}
	return "From: " + Rate(e.Carried) + " boxes delivered each second, about " + Rate(e.Income/e.Carried) + " gold each: worth more fresh and at a higher harbor tier."
}

func WisdomTip(f Frame) string {
	s := f.State.Study
	return tipLines("WISDOM "+Compact(f.State.Wisdom),
		fmt.Sprintf("From: +%g for each new word spoken at the Stone, +%g for a Chronicle answered well,", view.FluencyWisdom, view.ChronicleWisdom),
		fmt.Sprintf("+%g for each objective met, +1 every %.0f boxes delivered (%s to go).", s.Objective, s.Every, StudyLeft(f)),
		"Spent on: Grimoire pages, together with gold.", tipLines(NextPageLines(f)...))
}

func StudyLeft(f Frame) string {
	s := f.State.Study
	return fmt.Sprintf("%.0f", max(s.Every-s.Progress, 0))
}

func RenownTip(f Frame) string {
	return tipLines(fmt.Sprintf("RENOWN x%.2f", f.State.Renown), "Grows with every box ever delivered.",
		"Used: it multiplies the boxes the ships bring.")
}

func NinesTip(f Frame) string {
	slo := f.State.SLO
	window := "the last few minutes"
	if slo.WindowSeconds > 0 {
		window = fmt.Sprintf("the last %s", minutesText(slo.WindowSeconds))
	}
	head := "THE NINES " + Nines(slo.Current) + " of boxes delivered fresh in " + window
	lines := []string{head, "Every box counts once, so a busy noon weighs more than a quiet night, and the tide can move it fast."}
	if slo.Target > 0 {
		lines = append(lines, fmt.Sprintf("The Age asks %s.", Nines(slo.Target)))
	}
	return tipLines(append(lines, fmt.Sprintf("Used: arrivals x%.2f. Spoiled and turned-away boxes lower it.", SLOFactor(f)))...)
}

func minutesText(seconds int) string {
	if seconds%60 != 0 || seconds < 120 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	return fmt.Sprintf("%d minutes", seconds/60)
}

func HandsTip(f Frame) string {
	hands, _ := SupplyOf(f.Metrics.Nodes)
	cpu, _ := systemShare(f)
	return tipLines(fmt.Sprintf("FOOD %s of %s pledged (CPU requests %dm of %dm, as kubectl describe node counts them)",
		foodCount(hands.Used*view.RationsPerCPU), foodOfCores(hands.Total), milli(hands.Used), milli(hands.Total)),
		"From: Keeps. Used: every worker's Pledge"+citadelPart(ui.Bridge(foodOf(float64(cpu)), fmt.Sprintf("%dm", cpu)), cpu > 0)+".", supplyTip)
}

func LoreTip(f Frame) string {
	_, lore := SupplyOf(f.Metrics.Nodes)
	_, mem := systemShare(f)
	return tipLines(fmt.Sprintf("HOUSING %s of %s pledged (memory requests %dMi of %dMi)",
		housingCount(lore.Used*gib), housingOfGiB(lore.Total), int64(lore.Used*gib)/mebi, int64(lore.Total*gib)/mebi),
		"From: Keeps. Used: every worker's Pledge"+citadelPart(ui.Bridge(housingOf(float64(mem)), MemQuantity(mem)), mem > 0)+".", supplyTip)
}

func citadelPart(amount string, held bool) string {
	if !held {
		return ""
	}
	return ", and the Citadel's own huts (kube-system) hold " + amount + " of it"
}

func upkeepLineIn(f Frame, upkeep float64) string {
	if upkeep <= 0 && f.State.Age == view.AgeDark {
		return "upkeep: none in the Dark Age"
	}
	return UpkeepLine(upkeep)
}

func UpkeepLine(upkeep float64) string {
	if upkeep <= 0 {
		return "upkeep: none yet"
	}
	return "upkeep -" + Rate(upkeep) + " each second"
}
