package hud

import (
	"fmt"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func statTexts(stats []Stat) []string {
	out := []string{}
	for _, st := range stats {
		out = append(out, st.Text+st.Rate)
	}
	return out
}

func crowdedTopFrame(t *testing.T) Frame {
	t.Helper()
	f := flowFrame(t)
	f.State.Reveals = append(append([]string(nil), allReveals...), "hands", "lore", CodexReveal)
	f.State.Age = "feudal"
	f.Caesar.Codex = model.Codex{QuotaBytes: 100, SizeBytes: 40}
	f.Site.Structures = append(f.Site.Structures, view.Structure{ID: "codex-1", Kind: view.StructCodex, State: view.StateReady})
	return f
}

type topSample struct {
	gold, rate, wisdom, renown, slo float64
	cpu, mem                        [2]int64
}

var topSamples = []topSample{
	{0, 0, 0, 1, 1, [2]int64{0, 1900}, [2]int64{0, 3 << 30}},
	{7, .02, 12, 1.25, .752, [2]int64{1550, 1900}, [2]int64{420 << 20, 3500 << 20}},
	{37.04, -.5, 531, 6, .999, [2]int64{7150, 48900}, [2]int64{7300 << 20, 58500 << 20}},
	{999, 11.7, 9999, 12.5, 1, [2]int64{48900, 48900}, [2]int64{58500 << 20, 58500 << 20}},
	{23640, -190, 1.2e4, 99.9, .5, [2]int64{123450, 800000}, [2]int64{900 << 30, 4000 << 30}},
	{999999, 12345, 3.4e6, 1.05, .9999, [2]int64{999990, 999990}, [2]int64{9999 << 30, 9999 << 30}},
	{4.5e9, -98765, 7.7e9, 2, .0001, [2]int64{7, 12000000}, [2]int64{1, 40000 << 30}},
}

func withSample(f Frame, s topSample) Frame {
	next := f
	next.State.Gold, next.State.GoldPerSec, next.State.Wisdom, next.State.Renown = s.gold, s.rate, s.wisdom, s.renown
	next.State.Economy.Income, next.State.Economy.Upkeep, next.State.Economy.Profit = max(s.rate, 0)+1, 1, s.rate
	next.State.SLO.Current = s.slo
	next.Metrics.Nodes = map[string]view.NodeMetrics{"lone-tower": {CPURequestedMilli: s.cpu[0], CPUAllocatableMilli: s.cpu[1],
		MemoryRequestedBytes: s.mem[0], MemoryAllocatableBytes: s.mem[1]}}
	next.State.RealmTime = s.gold
	return next
}

type topLayout struct {
	xs      []int
	widths  []int
	buttons []ui.Rect
	clock   ui.Rect
}

func layoutOf(h *HUD, f Frame) topLayout {
	out := topLayout{}
	for _, st := range h.TopStats(f) {
		out.xs = append(out.xs, st.X)
		out.widths = append(out.widths, h.StatWidth(st))
	}
	for _, b := range h.TopButtons(f) {
		out.buttons = append(out.buttons, b.Rect)
	}
	_, out.clock = h.ClockSlot(f)
	return out
}

func TestTopBarElementsHoldStillWhateverTheValues(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, mode := range []string{"story", "names"} {
		ui.SetNamer(nil)
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		h := New(nil, Backend{})
		base := crowdedTopFrame(t)
		first := layoutOf(h, withSample(base, topSamples[0]))
		if len(first.xs) != 7 {
			t.Fatalf("[%s] %d stats, want the seven stats in order", mode, len(first.xs))
		}
		for _, s := range topSamples {
			f := withSample(base, s)
			if got := layoutOf(h, f); fmt.Sprint(got) != fmt.Sprint(first) {
				t.Errorf("[%s] %+v moved the bar:\n%+v\nwas\n%+v", mode, s, got, first)
			}
			assertStatsFitTheirSlots(t, h, f, mode)
		}
	}
}

func assertStatsFitTheirSlots(t *testing.T, h *HUD, f Frame, mode string) {
	t.Helper()
	stats := h.TopStats(f)
	for i, st := range stats {
		end := st.X + h.StatWidth(st)
		if used := h.StatUsed(st); used > h.StatWidth(st) {
			t.Errorf("[%s] %q is %d wide in a %d slot", mode, st.Text+st.Rate, used, h.StatWidth(st))
		}
		if i+1 < len(stats) && end > stats[i+1].X {
			t.Errorf("[%s] %q runs into %q", mode, st.Text, stats[i+1].Text)
		}
	}
	last := stats[len(stats)-1]
	text, clock := h.ClockSlot(f)
	if text != "" && clock.X < last.X+h.StatWidth(last) {
		t.Errorf("[%s] the clock %q at %+v overlaps the stats ending at %d", mode, text, clock, last.X+h.StatWidth(last))
	}
	if w := h.Font.Width(text, 1); w > clock.W {
		t.Errorf("[%s] the clock %q is %d wide in a %d slot", mode, text, w, clock.W)
	}
	buttons := h.TopButtons(f)
	if clock.X+clock.W > buttons[len(buttons)-1].Rect.X {
		t.Errorf("[%s] the clock %+v runs under the buttons", mode, clock)
	}
}

func TestTheTopBarDropsProfitAndKeepsGoldWithItsRate(t *testing.T) {
	h := New(nil, Backend{})
	f := flowFrame(t)
	f.State.Gold = 1500
	for _, st := range h.TopStats(f) {
		if strings.Contains(st.Text, "PROFIT") || strings.Contains(st.Text, "NET") || st.Id == "info:profit" {
			t.Fatalf("the top bar still shows a net line: %q", statTexts(h.TopStats(f)))
		}
	}
	gold := h.TopStats(f)[0]
	if gold.Text != "1.5K" || gold.Rate != "+4.4/S" || gold.RateClr != ui.Palette.GoldBright || gold.Upkeep != "-0.5/S" {
		t.Errorf("first stat = %q %q %v, overhead %q", gold.Text, gold.Rate, gold.RateClr, gold.Upkeep)
	}
	for _, part := range []string{"3.9", "income 4.4", "upkeep 0.5"} {
		if !strings.Contains(gold.Tip, part) {
			t.Errorf("the first stat's tip does not spell out %q: %q", part, gold.Tip)
		}
	}
	f.State.Economy.Profit, f.State.Economy.Upkeep = -7.9, 12.3
	if gold := h.TopStats(f)[0]; gold.Upkeep != "-12.3/S" || gold.UpkeepClr != ui.Palette.RedText {
		t.Errorf("a loss shows its rent red beside the rate: %q %v", gold.Upkeep, gold.UpkeepClr)
	}
	f.State.Economy = view.EconomyView{}
	f.State.GoldPerSec = 1.5
	if gold := h.TopStats(f)[0]; gold.Rate != "+1.5/S" {
		t.Errorf("before the economy ticks the rate stands in: %q", gold.Rate)
	}
	f.State.GoldPerSec = 0
	if gold := h.TopStats(f)[0]; gold.Rate != "" {
		t.Errorf("no rate, no rate text: %q", gold.Rate)
	}
}

func TestTheTopBarFitsInBothNamesModes(t *testing.T) {
	h := New(nil, Backend{})
	f := crowdedTopFrame(t)
	f.State.Economy.Profit = -1234.5
	assertTextFits(t, h, f, textPanel{name: "top bar", bounds: fixed(TopBar), draw: (*HUD).DrawTopBar, buttons: (*HUD).TopButtons})
}

func TestTopBarNumbersKeepTheirWidth(t *testing.T) {
	for v, want := range map[float64]string{.5: "+0.5/S", .02: "+0.02/S", -1234.5: "-1.2K/S", -.001: "+0/S", 190: "+190/S", 11.7: "+11.7/S"} {
		if got := FitRate(v); got != want {
			t.Errorf("fitRate(%v) = %q, want %q", v, got, want)
		}
	}
	for _, v := range []float64{0, .5, 37.04, 999.9, 999999, 1e6 - 1, 4.56e9, 9.99e11, 1.2e14} {
		if got := FitAmount(v, GoldChars); len(got) > GoldChars {
			t.Errorf("stat %v reads %q, wider than %d", v, got, GoldChars)
		}
	}
	for v, want := range map[float64]string{1.9: "1.9", 1.95: "1.95", 17.6: "17.6", 123.45: "123", 48900: "49K"} {
		if got := SupplyAmount(v); got != want {
			t.Errorf("supplyAmount(%v) = %q, want %q", v, got, want)
		}
	}
}

func TestTheCrowdedTopBarKeepsTheNetIncomeAndTheDay(t *testing.T) {
	h := New(nil, Backend{})
	for i, s := range topSamples[:len(topSamples)-1] {
		f := withSample(crowdedTopFrame(t), s)
		f.State.Economy.Upkeep = 11.1
		f.State.Economy.Profit = s.rate
		gold := h.TopStats(f)[0]
		net := gold.Net == "="+FitRate(s.rate) || (gold.Upkeep == "" && gold.Rate == FitRate(s.rate))
		if !net {
			t.Errorf("sample %d: the top bar hides the net rate: rate %q overhead %q net %q", i, gold.Rate, gold.Upkeep, gold.Net)
		}
		if clock, _ := h.ClockSlot(f); !strings.Contains(clock, "DAY") {
			t.Errorf("sample %d: the clock lost its DAY: %q", i, clock)
		}
	}
}

func TestTheGoldBreakdownAddsUpOrSaysTheFeudalAgeFloorsIt(t *testing.T) {
	f := flowFrame(t)
	f.State.Age = view.AgeFeudal
	f.State.Economy = view.EconomyView{Income: 308, Upkeep: 125, Profit: 172}
	gold := GoldStat(f)
	if gold.Rate != "+308/S" || gold.Upkeep != "-136/S" || gold.Net != "=+172/S" || gold.Floor != "" {
		t.Errorf("the first line reads %q %q %q %q, want the overhead shown in the same window as the rate and the net", gold.Rate, gold.Upkeep, gold.Net, gold.Floor)
	}
	f.State.Economy = view.EconomyView{Income: 161, Upkeep: 922, Profit: 0}
	gold = GoldStat(f)
	if gold.Upkeep != "-922/S" || gold.Net != "=+0/S" || gold.Floor == "" {
		t.Errorf("the first line reads %q %q %q %q and does not say the second tier floors the net at 0", gold.Rate, gold.Upkeep, gold.Net, gold.Floor)
	}
	if tip := GoldTip(f); !strings.Contains(tip, "floors the net at 0") || !strings.Contains(tip, "922") || !strings.Contains(tip, "161") {
		t.Errorf("the first stat's tip does not say the second tier floors the net: %q", tip)
	}
	f.State.Age = view.AgeEmpire
	f.State.Economy = view.EconomyView{Income: 161, Upkeep: 922, Profit: -761}
	if gold := GoldStat(f); gold.Floor != "" || gold.Net != "=-761/S" {
		t.Errorf("the fourth tier drains the pool, and the first line reads %q %q", gold.Net, gold.Floor)
	}
}

func TestTheTopBarKeepsSayingTheFeudalAgeFloorsTheNetWhenItSqueezes(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	h := New(nil, Backend{})
	f := flowFrame(t)
	f.State.Age = view.AgeFeudal
	f.State.Economy = view.EconomyView{Income: 161, Upkeep: 922, Profit: 0}
	if gold := h.TopStats(f)[0]; gold.Floor == "" {
		t.Errorf("the first line reads %q %q %q and does not say the second tier floors the net (after the squeeze: \"+0/S\")", gold.Rate, gold.Upkeep, gold.Net)
	}
	crowded := crowdedTopFrame(t)
	crowded.State.Age, crowded.State.Economy = view.AgeFeudal, f.State.Economy
	assertTextFits(t, h, crowded, textPanel{name: "top bar", bounds: fixed(TopBar), draw: (*HUD).DrawTopBar, buttons: (*HUD).TopButtons})
}
