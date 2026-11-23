package hud

import (
	"fmt"
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	ToneText = iota
	ToneDim
	ToneGood
	ToneWarn
	ToneBad
	ToneGold
)

const (
	busiestHearths = 3
	keepRows       = 9
	guildRows      = 6
	tributeRows    = 5
	NoValue        = "--"
)

type Cell struct {
	Text string
	Tone int
}

type Section struct {
	Title string
	Cols  []int
	Head  []string
	Rows  [][]Cell
	Empty string
}

type CouncilPage struct {
	Line     string
	Sections []Section
}

func cell(text string, tone int) Cell { return Cell{Text: text, Tone: tone} }

func capRows(rows [][]Cell, n int) [][]Cell {
	if len(rows) <= n {
		return rows
	}
	more := []Cell{cell(fmt.Sprintf("+%d more", len(rows)-(n-1)), ToneDim)}
	return append(append([][]Cell(nil), rows[:n-1]...), more)
}

func ratioTone(r float64) int {
	switch {
	case r >= 1:
		return ToneBad
	case r >= supplyWarn:
		return ToneWarn
	}
	return ToneGood
}

func gaugeTone(level int) int {
	return [...]int{ToneGood, ToneWarn, ToneBad}[min(max(level, 0), 2)]
}

func keepStructures(site view.Site) []view.Structure {
	out := []view.Structure{}
	for _, s := range site.Structures {
		if s.Kind == view.StructKeep && s.Object.Name != "" {
			out = append(out, s)
		}
	}
	return out
}

func nodeLabel(site view.Site, node string) string {
	for _, s := range keepStructures(site) {
		if s.Object.Name == node && s.Label != "" {
			return s.Label
		}
	}
	return node
}

func Unhoused(pods []model.PodView) []model.PodView {
	out := []model.PodView{}
	for _, p := range pods {
		if !p.System && p.State == model.HearthPending {
			out = append(out, p)
		}
	}
	return out
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func keepRow(s view.Structure, f Frame) []Cell {
	m, ok := f.Metrics.Nodes[s.Object.Name]
	if !ok {
		return []Cell{cell(s.Label, ToneText), cell(NoValue, ToneDim), cell(NoValue, ToneDim), cell(NoValue, ToneDim), cell(NoValue, ToneDim)}
	}
	hands := Supply{Used: float64(m.CPURequestedMilli) / 1000, Total: float64(m.CPUAllocatableMilli) / 1000}
	lore := Supply{Used: float64(m.MemoryRequestedBytes) / gib, Total: float64(m.MemoryAllocatableBytes) / gib}
	used := cell(NoValue, ToneDim)
	if f.Live {
		used = cell("", ToneDim)
	}
	if f.Metrics.Augury && (m.Measured || !f.Live) {
		used = cell(foodAmount(m.CPUMilli/1000), ratioTone(m.CPUMilli/1000/max(hands.Total, 1e-9)))
	}
	pods := fmt.Sprint(m.Pods)
	if m.PodsAllocatable > 0 {
		pods = fmt.Sprintf("%d/%d", m.Pods, m.PodsAllocatable)
	}
	return []Cell{cell(s.Label, ToneText), cell(foodRatio(hands), ratioTone(hands.Ratio())),
		used, cell(housingRatio(lore), ratioTone(lore.Ratio())), cell(pods, ToneText)}
}

func foodAmount(cores float64) string {
	return ui.Bridge(foodCount(cores*view.RationsPerCPU), fmt.Sprintf("%.2f", cores))
}

func foodRatio(s Supply) string {
	return ui.Bridge(foodCount(s.Used*view.RationsPerCPU)+" / "+foodCount(s.Total*view.RationsPerCPU), fmt.Sprintf("%.1f / %.1f", s.Used, s.Total))
}

func housingRatio(s Supply) string {
	return ui.Bridge(housingCount(s.Used*gib)+" / "+housingCount(s.Total*gib), fmt.Sprintf("%.1fGi / %.1fGi", s.Used, s.Total))
}

func QuartermasterPage(f Frame) CouncilPage {
	keeps := Section{Title: "KEEPS", Cols: []int{0, 180, 310, 420, 560}, Head: []string{"KEEP", "FOOD PLEDGED", ui.Bridge("FOOD EATEN", "CPU USED"), "HOUSING PLEDGED", "HUTS"},
		Empty: "No Keep answers the Chancery yet."}
	for _, s := range keepStructures(f.Site) {
		keeps.Rows = append(keeps.Rows, keepRow(s, f))
	}
	keeps.Rows = capRows(keeps.Rows, keepRows)
	waiting := Unhoused(f.Pods)
	names := []string{}
	for _, p := range waiting[:min(len(waiting), listShown)] {
		names = append(names, p.Name)
	}
	homeless := Section{Title: "UNHOUSED", Cols: []int{0, 180}, Rows: [][]Cell{{cell(plural(len(waiting), "hut", "huts"), unhousedTone(len(waiting))),
		cell(strings.Join(names, ", "), ToneDim)}}}
	return CouncilPage{Line: quartermasterLine(f, len(waiting)), Sections: []Section{keeps, homeless}}
}

const listShown = 3

func unhousedTone(n int) int {
	if n > 0 {
		return ToneBad
	}
	return ToneGood
}

func quartermasterLine(f Frame, unhoused int) string {
	if unhoused > 0 {
		verb := "are"
		if unhoused == 1 {
			verb = "is"
		}
		return fmt.Sprintf("%s %s Unhoused: no Keep has room for their Pledge.", plural(unhoused, "hut", "huts"), verb)
	}
	for _, s := range keepStructures(f.Site) {
		m := f.Metrics.Nodes[s.Object.Name]
		if m.CPUAllocatableMilli > 0 && m.CPURequestedMilli >= m.CPUAllocatableMilli {
			return s.Label + " is fully pledged: the next hut must live elsewhere."
		}
	}
	hands, _ := SupplyOf(f.Metrics.Nodes)
	if hands.Total <= 0 {
		return "No Keep answers the Chancery yet."
	}
	return fmt.Sprintf("%s of %s are pledged across your Keeps. Every hut has a room.", foodCount(hands.Used*view.RationsPerCPU), foodOfCores(hands.Total))
}

func hornText(name string, horns []model.Horn) string {
	for _, h := range horns {
		if h.Target == name {
			return fmt.Sprintf("%d-%d, %d -> %d", h.Min, h.Max, h.Current, h.Desired)
		}
	}
	return "none"
}

var RankReachable = func() bool { return view.Content.RuneReachable(view.RankRune) }

func StewardsPage(f Frame) CouncilPage {
	ranked := RankReachable()
	s := Section{Title: "CREWS", Cols: stewardCols[:3], Head: []string{"CREW", "HUTS", "WAR HORN"}, Empty: "No crew stands yet."}
	line := "A War Horn grows and shrinks a crew's huts with the load."
	if ranked {
		s.Cols, s.Head = stewardCols, []string{"CREW", "HUTS", "RANK", "WAR HORN"}
		line = "A Rank reaches a crew's huts through its rollout. When quarters run out, a higher Rank may displace a lower one."
	}
	for _, c := range rankedCrews(f) {
		tone := ToneGood
		if c.Ready < c.Desired {
			tone = ToneWarn
		}
		row := []Cell{cell(CrewName(c), ToneText), cell(fmt.Sprintf("%d/%d", c.Ready, c.Desired), tone)}
		if ranked {
			row = append(row, cell("", ToneDim))
		}
		s.Rows = append(s.Rows, append(row, cell(hornText(CrewName(c), f.Horns), ToneDim)))
	}
	return CouncilPage{Line: line, Sections: []Section{s}}
}

func rankedCrews(f Frame) []model.CrewChain {
	return f.Crews[:min(len(f.Crews), crewRows)]
}

func CrewName(c model.CrewChain) string {
	if c.Name == "" && len(c.Foremen) > 0 {
		return c.Foremen[0].Name
	}
	return c.Name
}

var (
	stewardCols = []int{0, 150, 230, 420}
	tierNames   = []string{"JETTY", "PIER", "WHARF", "PORT"}
)

const crewRows = 9

func rollTone(r model.GuildRoll) int {
	switch {
	case r.Ready == 0:
		return ToneBad
	case r.Ready < r.Endpoints:
		return ToneWarn
	}
	return ToneGood
}

func WayfinderPage(f Frame, berths func(view.Haven) int) CouncilPage {
	guilds := Section{Title: "GUILDHALLS", Cols: []int{0, 180}, Head: []string{"GUILDHALL", "GUILD ROLL"}, Empty: "No Guildhall stands yet."}
	for _, r := range f.Rolls {
		guilds.Rows = append(guilds.Rows, []Cell{cell(r.Service, ToneText), cell(fmt.Sprintf("%d of %d ready", r.Ready, r.Endpoints), rollTone(r))})
	}
	guilds.Rows = capRows(guilds.Rows, guildRows)
	harbors := Section{Title: "HARBORS", Cols: []int{0, 150, 240, 330, 420, 540}, Head: []string{"HARBOR", "STATE", "TIER", "SHARE", "BOXES/S", "BERTHS"},
		Empty: "No harbor is charted."}
	open, rate := 0, f.State.Pilgrims.ArrivalRate
	for _, h := range f.Site.Havens {
		state, tone := "CLOSED", ToneDim
		if h.Open {
			state, tone = "OPEN", ToneGood
			open++
		}
		harbors.Rows = append(harbors.Rows, []Cell{cell(h.Name, ToneText), cell(state, tone), cell(tierNames[min(int(h.Tier), len(tierNames)-1)], ToneText),
			cell(fmt.Sprintf("%d%%", int(h.Share*100+.5)), ToneText), cell(fmt.Sprintf("%.2f", rate*h.Share), ToneText), cell(fmt.Sprint(berths(h)), ToneText)})
	}
	line := fmt.Sprintf("%s open; %.2f boxes land each second.", plural(open, "harbor", "harbors"), rate)
	return CouncilPage{Line: line, Sections: []Section{guilds, harbors}}
}

func Upkeep(f Frame) (float64, int) {
	total := 0.0
	for _, k := range f.State.Keeps {
		total += k.Upkeep
	}
	return total, len(f.State.Keeps)
}

func Rate(v float64) string {
	return view.RateText(v)
}

func Signed(v float64) string {
	r := Rate(v)
	if v > 0 && r != "0" {
		return "+" + r
	}
	return r
}

func TreasurerPage(f Frame) CouncilPage {
	upkeep, keeps := Upkeep(f)
	net := f.State.GoldPerSec
	netTone := ToneGood
	if net < 0 {
		netTone = ToneBad
	}
	short := !view.RentDrains(f.State.Age) && upkeep > 0 && net < .005 && f.State.Economy.Income < upkeep
	earned := net + upkeep
	if short {
		earned = f.State.Economy.Income
	}
	ledger := Section{Title: "THE TREASURY", Cols: []int{0, 260}, Rows: [][]Cell{
		{cell("GOLD", ToneText), cell(Compact(f.State.Gold), ToneGold)},
		{cell("EARNED PER SECOND", ToneText), cell(Signed(earned), ToneText)},
		{cell("UPKEEP PER SECOND", ToneText), cell(UpkeepCell(upkeep, keeps), ToneWarn)},
		{cell("NET PER SECOND", ToneText), cell(Signed(net), netTone)},
	}}
	tributes := Section{Title: "HELD CONTRACTS", Cols: []int{0, 300, 420}, Head: []string{"CONTRACT", "DUE IN", "REWARD"}, Empty: "No contract held."}
	for _, r := range ContractRows(f) {
		if r.State == StateActive {
			tributes.Rows = append(tributes.Rows, []Cell{cell(r.Title, ToneText), cell(r.Countdown(), ToneWarn), cell(r.Reward, ToneGold)})
		}
	}
	tributes.Rows = capRows(tributes.Rows, tributeRows)
	line := "Your Keeps cost " + Rate(upkeep) + " gold each second in upkeep."
	if upkeep <= 0 {
		line = "Your Keeps cost no upkeep yet: only greater Keeps must be paid for."
	}
	switch {
	case net < 0:
		line = "Upkeep outruns what the realm earns: gold drains each second."
	case short:
		line = "Upkeep eats everything the realm earns: no gold comes in until the bill shrinks."
	}
	return CouncilPage{Line: line, Sections: []Section{ledger, tributes}}
}

type podLoad struct {
	name string
	m    view.PodMetrics
}

func busiest(f Frame) []podLoad {
	out := []podLoad{}
	for _, p := range f.Pods {
		if m, ok := f.Metrics.Pods[p.Namespace+"/"+p.Name]; ok && !p.System {
			out = append(out, podLoad{name: p.Name, m: m})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].m.CPUMilli != out[j].m.CPUMilli {
			return out[i].m.CPUMilli > out[j].m.CPUMilli
		}
		return out[i].name < out[j].name
	})
	return out[:min(len(out), busiestHearths)]
}

func UpkeepCell(upkeep float64, keeps int) string {
	if upkeep <= 0 {
		return "none yet"
	}
	return "-" + Rate(upkeep) + "  (" + plural(keeps, "Keep", "Keeps") + ")"
}
