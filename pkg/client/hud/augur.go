package hud

import (
	"fmt"
	"sort"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	BlindAugur    = "The Augur sees nothing until the Observatory (metrics-server) stands."
	BlindOmens    = "No omens yet."
	CodexUnknown  = "The Codex reports no size yet."
	MiddenUnknown = "No Reeve reports its ground yet."
	DisarmCommand = "etcdctl alarm disarm"
)

func AugurPage(f Frame) CouncilPage {
	top := Section{Title: "BUSIEST HUTS", Cols: []int{0, 220, 320, 420}, Head: []string{"HUT", "FOOD", "HOUSING", "SERVED/S"}, Empty: BlindOmens}
	if f.Metrics.Augury {
		top.Empty = "No hut burns yet."
		for _, p := range busiest(f) {
			top.Rows = append(top.Rows, []Cell{cell(p.name, ToneText), cell(Hands(p.m.CPUMilli), ToneText),
				cell(fmt.Sprintf("%dMi", int(p.m.MemoryBytes/(1<<20))), ToneText), cell(fmt.Sprintf("%.1f", p.m.ServedRate), ToneText)})
		}
	}
	return CouncilPage{Line: augurLine(f), Sections: []Section{top, MiddenSection(f), CodexSection(f.Caesar.Codex)}}
}

const middenRows = 3

func middenNodes(f Frame) []string {
	out := []string{}
	for node, m := range f.Caesar.Middens {
		if m.CapacityBytes > 0 {
			out = append(out, node)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := f.Caesar.Middens[out[i]], f.Caesar.Middens[out[j]]
		if a.Pressure != b.Pressure {
			return a.Pressure
		}
		if a.Fill() != b.Fill() {
			return a.Fill() > b.Fill()
		}
		return out[i] < out[j]
	})
	return out
}

func MiddenState(m model.Midden) (string, int) {
	switch level := model.GaugeLevel(m.Fill(), m.Pressure); {
	case m.Pressure:
		return ui.Bridge("CELLAR WARD", "DISK PRESSURE"), ToneBad
	case level == model.GaugeDanger:
		return "NEAR THE LINE", ToneBad
	case level == model.GaugeWarn:
		return "FILLING", ToneWarn
	}
	return "TENDED", ToneGood
}

func MiddenSection(f Frame) Section {
	s := Section{Title: "THE CELLARS", Cols: []int{0, 220, 420}, Head: []string{"KEEP", "GROUND", "STATE"}, Empty: MiddenUnknown}
	nodes := middenNodes(f)
	for _, node := range nodes[:min(len(nodes), middenRows)] {
		m := f.Caesar.Middens[node]
		state, tone := MiddenState(m)
		s.Rows = append(s.Rows, []Cell{cell(nodeLabel(f.Site, node), ToneText),
			cell(groundUse(m), tone), cell(state, tone)})
	}
	return s
}

func percent(v float64) int { return int(v*100 + .5) }

func groundUse(m model.Midden) string {
	story := fmt.Sprintf("%s of %s (%d%%)", view.Grouped(int64(m.UsedBytes/Mib)), view.Barrels(m.CapacityBytes/Mib), percent(m.Fill()))
	return ui.Bridge(story, fmt.Sprintf("%s of %s (%d%%)", model.Bytes(m.UsedBytes), model.Bytes(m.CapacityBytes), percent(m.Fill())))
}

func CodexSection(c model.Codex) Section {
	s := Section{Title: "THE CODEX", Cols: []int{0, 220}, Empty: CodexUnknown}
	if !c.Known() {
		return s
	}
	tone := gaugeTone(model.GaugeLevel(c.Fill(), c.Alarm))
	alarm, alarmTone := "quiet", ToneGood
	if c.Alarm {
		alarm, alarmTone = "NOSPACE: every write fails", ToneBad
	}
	s.Rows = [][]Cell{
		{cell("SIZE", ToneText), cell(fmt.Sprintf("%s of %s quota (%d%%)", model.Bytes(c.SizeBytes), model.Bytes(c.QuotaBytes), percent(c.Fill())), tone)},
		{cell("IN USE", ToneText), cell(fmt.Sprintf("%s, %s to reclaim", model.Bytes(c.InUseBytes), model.Bytes(c.Reclaimable())), ToneText)},
		{cell("ALARM", ToneText), cell(alarm, alarmTone)},
	}
	return s
}

func augurLine(f Frame) string {
	if f.Caesar.Codex.Alarm {
		return "The Codex is full: the Chancery refuses every write. Rebind the Codex, then disarm the alarm."
	}
	for _, node := range middenNodes(f) {
		if f.Caesar.Middens[node].Pressure {
			return nodeLabel(f.Site, node) + " has DiskPressure: its Reeve evicts huts to clear its ground."
		}
	}
	if f.Caesar.Codex.Large() {
		return "The Codex swells. Rebind it before it reaches its quota."
	}
	if !f.Metrics.Augury {
		return BlindAugur
	}
	if top := busiest(f); len(top) > 0 {
		return fmt.Sprintf("%s works hardest, eating %s.", top[0].name, ui.Bridge(foodOf(top[0].m.CPUMilli), Hands(top[0].m.CPUMilli)+" of CPU"))
	}
	return "The omens are calm."
}
