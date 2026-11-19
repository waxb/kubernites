package world

import (
	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
)

var tradeGaits = map[string]float64{"porter": 1, "clerk": .85, "hauler": .7, "courier": 1.4, "sealwright": .92}

func TradeGait(role string) float64 {
	if g, ok := tradeGaits[role]; ok {
		return g
	}
	return 1
}

func LoadKind(role string, kind int) string {
	if gfx.IsTrade(role) && gfx.TradeKind(role) == "" {
		return art.SealedCarry
	}
	return KindName(kind)
}
