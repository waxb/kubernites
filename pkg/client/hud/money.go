package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func Gold(v float64) string { return Money(Compact(v)) }

func Money(amount string) string {
	if rest, ok := strings.CutPrefix(amount, "-"); ok {
		return "-" + Money(rest)
	}
	return ui.Bridge(ui.IconGold+amount, "$"+amount)
}

func Wisdom(v float64) string {
	amount := Compact(v)
	return ui.Bridge(ui.IconWisdom+amount, amount+" KN")
}

func GoldWord(v float64) string {
	amount := Compact(v)
	return ui.Bridge(amount+" GOLD", "$"+amount)
}

func GoldRate(v float64) string {
	amount := Compact(v)
	return ui.Bridge("+"+amount+" GOLD/S", "+$"+amount+"/S")
}
