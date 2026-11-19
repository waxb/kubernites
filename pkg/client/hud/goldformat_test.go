package hud

import (
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

var numberRun = regexp.MustCompile(`[0-9][0-9.]*`)

func significantDigits(run string) int {
	digits := strings.TrimLeft(strings.ReplaceAll(run, ".", ""), "0")
	return len(digits)
}

func goldFormats() map[string]func(float64) string {
	return map[string]func(float64) string{
		"Compact": Compact, "Gold": Gold, "GoldWord": GoldWord, "GoldRate": GoldRate, "Rate": Rate, "BoxGold": BoxGold,
		"goldAmount": goldAmount, "GoldPerSecond": GoldPerSecond, "SignedMoney": SignedMoney, "Wisdom": Wisdom,
		"Money":     func(v float64) string { return Money(Compact(v)) },
		"fitAmount": func(v float64) string { return FitAmount(v, GoldChars) }, "fitRate": FitRate, "rateAmount": rateAmount,
		"GainLabel": func(v float64) string { return GainLabel(v, false) }, "GainLabelNames": func(v float64) string { return GainLabel(v, true) },
		"negative": func(v float64) string { return Gold(-v) },
	}
}

func goldSamples() []float64 {
	out := []float64{}
	for e := -3; e <= 14; e++ {
		for _, m := range []float64{1, 1.234, 5.5, 9.994, 9.999} {
			out = append(out, m*math.Pow(10, float64(e)))
		}
	}
	return append(out, 999.5, 999.9, 999999, 999999.9, 9.99e8, 9.999e11, 1e12, 3.7e12)
}

func TestNoGoldAmountPrintsMoreThanFourSignificantDigits(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, k8s := range []bool{false, true} {
		ui.SetNamer(nil)
		if k8s {
			ui.SetNamer(names.Kubernetes(64))
		}
		for name, format := range goldFormats() {
			for _, v := range goldSamples() {
				got := format(v)
				for _, run := range numberRun.FindAllString(got, -1) {
					if significantDigits(run) > 4 {
						t.Errorf("%s(%g) = %q prints %s (k8s=%v)", name, v, got, run, k8s)
					}
				}
			}
		}
	}
}

func TestCompactReadsKMBT(t *testing.T) {
	cases := map[float64]string{
		999: "999", 1000: "1K", 1234: "1.23K", 999999: "1M", 1.5e6: "1.5M", 999.996e6: "1B", 2.34e9: "2.34B", 1e12: "1T", 3.7e12: "3.7T", 4.567e14: "456.7T",
	}
	for v, want := range cases {
		if got := Compact(v); got != want {
			t.Errorf("Compact(%g) = %q, want %q", v, got, want)
		}
	}
}
