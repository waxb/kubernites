package hud

import "strings"

const (
	CoinFlightsAll = "all"
	CoinFlightsFew = "few"
	CoinFlightsOff = "off"
	coinOpening    = 30
	coinNotable    = 10.0
	coinBeat       = 2.0
)

type CoinPolicy struct {
	Mode   string
	Flown  int
	Income float64
}

func CoinFlightMode(mode string) string {
	switch mode {
	case CoinFlightsAll, CoinFlightsOff:
		return mode
	}
	return CoinFlightsFew
}

func NextCoinFlights(mode string) string {
	switch CoinFlightMode(mode) {
	case CoinFlightsAll:
		return CoinFlightsFew
	case CoinFlightsFew:
		return CoinFlightsOff
	}
	return CoinFlightsAll
}

func CoinFlightsLabel(mode string) string {
	return "COIN FLIGHTS: " + strings.ToUpper(CoinFlightMode(mode))
}

func (p CoinPolicy) opening() bool { return p.Flown < coinOpening }

func (p CoinPolicy) Flies(value float64, golden bool) bool {
	switch CoinFlightMode(p.Mode) {
	case CoinFlightsAll:
		return true
	case CoinFlightsOff:
		return false
	}
	return p.opening() || golden || value >= coinNotable*max(p.Income, 0)
}

func (p CoinPolicy) TreasuryGap() float64 {
	mode := CoinFlightMode(p.Mode)
	if mode == CoinFlightsAll || mode == CoinFlightsFew && p.opening() {
		return coinGap
	}
	return coinBeat
}

func PurseLabel(v float64, trueNames bool) string {
	if trueNames {
		return "+$" + Compact(v)
	}
	return "+" + Compact(v)
}

func (h *HUD) SetCoinFlights(mode string) { h.Coins.mode = CoinFlightMode(mode) }

func (h *HUD) CoinFlights() string { return CoinFlightMode(h.Coins.mode) }

func (h *HUD) TreasuryGap() float64 { return h.coinPolicy().TreasuryGap() }

func (h *HUD) coinPolicy() CoinPolicy {
	return CoinPolicy{Mode: h.Coins.mode, Flown: h.Coins.flown, Income: h.Coins.income}
}
