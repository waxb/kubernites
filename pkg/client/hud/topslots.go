package hud

import (
	"math"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	statGap     = 8
	clockGap    = 8
	GoldChars   = 6
	rateChars   = 4
	countChars  = 5
	renownChars = 4
	supplyChars = 4
	slotDigit   = "8"
	clockDay    = "DAY 99 88:88"
	clockHour   = "88:88"
	ageSpacer   = " AGE  "
)

type slotUnit struct {
	at     float64
	suffix string
}

var slotUnits = []slotUnit{{1, ""}, {1e3, "K"}, {1e6, "M"}, {1e9, "B"}, {1e12, "T"}}

func IncomeRate(st view.GameState) float64 {
	raw := st.Economy
	if raw.Income == 0 && raw.Upkeep == 0 {
		return max(st.GoldPerSec-raw.Hand, 0)
	}
	return EconomyOf(st).Income
}

func NetGoldRate(st view.GameState) float64 {
	raw := st.Economy
	if raw.Income == 0 && raw.Upkeep == 0 {
		return st.GoldPerSec - raw.Hand
	}
	return EconomyOf(st).Profit
}

func digits(n int) string { return strings.Repeat(slotDigit, n) }

func trimFloat(v float64, decimals int) string {
	return trimZeros(strconv.FormatFloat(v, 'f', decimals, 64))
}

func naturalUnit(v float64) int {
	i := 0
	for i+1 < len(slotUnits) && v >= slotUnits[i+1].at {
		i++
	}
	return i
}

func FitAmount(v float64, n int) string {
	if s := Compact(v); len(s) <= n {
		return s
	}
	for i := max(naturalUnit(v), 1); i < len(slotUnits); i++ {
		for d := 2; d >= 0; d-- {
			if s := trimFloat(v/slotUnits[i].at, d) + slotUnits[i].suffix; len(s) <= n {
				return s
			}
		}
	}
	return Compact(v)
}

func rateAmount(v float64) string {
	if v < 1 {
		return trimZeros(Rate(v))
	}
	return FitAmount(v, rateChars)
}

func trimZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	return strings.TrimSuffix(strings.TrimRight(s, "0"), ".")
}

func FitRate(v float64) string {
	sign := "+"
	amount := rateAmount(math.Abs(v))
	if v < 0 && amount != "0" {
		sign = "-"
	}
	return sign + amount + "/S"
}

func SupplyAmount(v float64) string {
	for _, s := range []string{Amount(v), strconv.FormatFloat(v, 'f', 1, 64), strconv.FormatFloat(v, 'f', 0, 64)} {
		if len(s) <= supplyChars {
			return s
		}
	}
	return FitAmount(v, supplyChars)
}

func GoldStat(f Frame) Stat {
	st := Stat{icon: "gold", Text: FitAmount(f.State.Gold, GoldChars), clr: ui.Palette.GoldBright, Tip: GoldTip(f), Meter: MeterGold,
		slot: digits(GoldChars), rateSlot: "+" + digits(rateChars) + "/S", RateClr: ui.Palette.GoldBright}
	if rate := NetGoldRate(f.State); rate != 0 {
		st.Rate = FitRate(rate)
	}
	if e := EconomyOf(f.State); e.Upkeep > 0 {
		paid := RentTaken(e)
		if NetFloored(f.State) {
			paid, st.Floor = e.Upkeep, netFloorTag
		}
		st.Rate, st.Upkeep, st.UpkeepClr = FitRate(e.Income), FitRate(-paid), ui.Palette.RedText
		st.Net, st.netClr = "="+FitRate(e.Profit), ui.Palette.Cream
	}
	if strings.HasPrefix(st.Rate, "-") {
		st.RateClr = ui.Palette.RedText
	}
	return st
}

const netFloorTag = "FLOORED"

func RentTaken(e view.EconomyView) float64 {
	return e.Income - e.Profit
}

func NetFloored(st view.GameState) bool {
	e := EconomyOf(st)
	margin := max(0.01, 0.02*e.Upkeep)
	return !view.RentDrains(st.Age) && e.Upkeep-RentTaken(e) > margin && e.Profit < margin
}

func (h *HUD) StatWidth(st Stat) int {
	w := h.Font.Width(st.slot, 1)
	if st.icon != "" {
		w += statIconW
	}
	if st.rateSlot != "" {
		w += ui.Advance + h.Font.Width(st.rateSlot, 1)
	}
	if st.Upkeep != "" {
		w += ui.Advance + h.Font.Width(st.rateSlot, 1)
	}
	if st.Net != "" {
		w += 2*ui.Advance + h.Font.Width(st.rateSlot, 1)
	}
	if st.Floor != "" {
		w += ui.Advance + h.Font.Width(st.Floor, 1)
	}
	if st.gauge {
		w += gaugeGap + gaugeW
	}
	return w
}

func (h *HUD) StatUsed(st Stat) int {
	used := h.StatWidth(st)
	for _, part := range [][2]string{{st.Text, st.slot}, {st.Rate, st.rateSlot}} {
		used += max(h.Font.Width(part[0], 1)-h.Font.Width(part[1], 1), 0)
	}
	return used
}

func (h *HUD) rawClockSlot(f Frame) (string, ui.Rect) {
	buttons := h.TopButtons(f)
	right := buttons[len(buttons)-1].Rect.X - clockGap
	left := goldX
	if stats := h.TopStats(f); len(stats) > 0 {
		last := stats[len(stats)-1]
		left = last.X + h.StatWidth(last) + clockGap
	}
	slot := ui.Rect{X: left, Y: 1, W: max(right-left, 0), H: TopH - 3}
	return h.clockFor(f, slot.W), slot
}

func (h *HUD) clockFor(f Frame, room int) string {
	if h.live != nil {
		return h.liveClock()
	}
	clock := Clock(f.State.RealmTime, f.State.Pilgrims.DayPhase)
	age := strings.ToUpper(f.State.Age) + ageSpacer
	dayOnly := strings.TrimSpace(clock[:len(clock)-len(clockHour)])
	short := strings.ToUpper(f.State.Age) + " "
	switch {
	case f.Revealed("ageup") && h.Font.Width(age+clockDay, 1) <= room && h.Font.Width(age+clock, 1) <= room:
		return age + clock
	case f.Revealed("ageup") && h.Font.Width(age+dayOnly, 1) <= room:
		return age + dayOnly
	case h.Font.Width(clockDay, 1) <= room && h.Font.Width(clock, 1) <= room:
		return clock
	case f.Revealed("ageup") && h.Font.Width(age+clockHour, 1) <= room:
		return age + clock[len(clock)-len(clockHour):]
	case f.Revealed("ageup") && h.Font.Width(short+clockHour, 1) <= room:
		return short + clock[len(clock)-len(clockHour):]
	case h.Font.Width(clockHour, 1) <= room:
		return clock[len(clock)-len(clockHour):]
	}
	return ""
}
