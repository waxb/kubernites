package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	WhyMaxChars = 11
	whyHeight   = 9
	whySingleY  = 12
	whyDoubleY  = 4
	whyKeyClear = 7
)

var whyRules = []struct{ has, short string }{
	{"Carry the shipment", "CARRY FIRST"},
	{"holds two Keeps", "TWO KEEPS"},
	{"later step", "LATER STEP"},
	{"Feudal", "FEUDAL AGE"},
	{"Already raised", "RAISED"},
	{"already open", "OPEN"},
	{"Endow", "ENDOW FIRST"},
	{"in the Grimoire", "GRIMOIRE"},
	{"open it first", "OPEN FIRST"},
	{"top level", "TOP LEVEL"},
	{"no berth", "NO BERTH"},
	{"Rush Tide", "AFTER TIDE"},
	{"quay holds", "UPGRADE"},
	{"realm calls them", "ONE NINE"},
	{"masters", "SHIPS SOON"},
}

var whyNames = whyNamesK8s()

func whyNamesK8s() map[string]string {
	return map[string]string{
		"UPGRADE": "UPGRADE", "NO BERTH": "NO RAMP", "AFTER TIDE": "AFTER TEST", "SHIPS SOON": "BURSTS SOON", "CHANCERY": "APISERVER",
		"AUGURY": "METRICS", "CHARTER": "NODES", "CHARTERS": "ZONE", "WAR HORNS": "AUTOSCALERS", "MASONRY": "INSTANCE", "BINDINGS": "LIMITS",
		"SHRINES": "DAEMONSETS", "VAULTS": "VOLUMES", "SPEAKING": "KUBECTL", "SHUTTERS": "READINESS", "WATCHMAN'S": "LIVENESS",
	}
}

var romanRanks = map[string]bool{"I": true, "II": true, "III": true, "IV": true, "V": true}

func WhyOf(reason string) string {
	short := storyWhy(reason)
	if k8s, ok := whyNames[short]; ok {
		return ui.Bridge(short, k8s)
	}
	return short
}

func storyWhy(reason string) string {
	reason = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(reason), "."))
	if reason == "" {
		return ""
	}
	if short, ok := gateWhy(reason); ok {
		return short
	}
	lower := strings.ToLower(reason)
	for _, r := range whyRules {
		if strings.Contains(lower, strings.ToLower(r.has)) {
			return r.short
		}
	}
	if short, ok := needsWhy(reason); ok {
		return short
	}
	if strings.Contains(lower, "gold") && (strings.Contains(lower, "needs") || strings.Contains(lower, "costs")) {
		return "NEEDS GOLD"
	}
	return strings.ToUpper(reason)
}

var needsNames = map[string]string{"QUARTERMASTER": "QMASTER", "STEWARDS' HALL": "STEWARDS"}

func needsWhy(reason string) (string, bool) {
	rest, ok := strings.CutPrefix(reason, "Needs ")
	if !ok || strings.ContainsAny(rest, "0123456789") || strings.Contains(strings.ToLower(rest), "gold") {
		return "", false
	}
	rest = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(rest, "the "), "The "))
	rest = strings.TrimPrefix(strings.TrimPrefix(rest, "a "), "A ")
	name := strings.ToUpper(rest)
	if short, ok := needsNames[name]; ok {
		name = short
	}
	switch {
	case name == "" || strings.ContainsAny(name, ",;:") || len(strings.Fields(name)) > 2:
		return "", false
	case len("NEEDS ")+len(name) <= WhyMaxChars:
		return "NEEDS " + name, true
	case len(name) <= WhyMaxChars:
		return name, true
	}
	return "", false
}

func gateWhy(reason string) (string, bool) {
	if strings.Contains(reason, "hold the Nine") {
		if strings.Contains(reason, "99%") {
			return "TWO NINES", true
		}
		return "ONE NINE", true
	}
	page, ok := strings.CutPrefix(reason, "Learn ")
	if !ok {
		return "", false
	}
	page, ok = strings.CutSuffix(page, " in the Grimoire")
	if !ok {
		return "", false
	}
	return pageShort(page)
}

func pageShort(page string) (string, bool) {
	base, _, _ := strings.Cut(page, ":")
	words := strings.Fields(base)
	if len(words) > 0 && strings.EqualFold(words[0], "the") {
		words = words[1:]
	}
	if n := len(words); n > 1 && romanRanks[words[n-1]] {
		words = words[:n-1]
	}
	if len(words) == 0 {
		return "", false
	}
	switch whole := strings.ToUpper(strings.Join(words, " ")); {
	case len([]rune(whole)) <= WhyMaxChars:
		return whole, true
	case len([]rune(words[0])) <= WhyMaxChars:
		return strings.ToUpper(words[0]), true
	}
	return "", false
}

func whyUnless(ok bool, reason string) string {
	if ok {
		return ""
	}
	return WhyOf(reason)
}

func cellWhy(b Button) string {
	switch {
	case !b.Disabled:
		return ""
	case b.Why == "" && b.Tip != "" && !strings.Contains(b.Tip, "\n"):
		return WhyOf(b.Tip)
	}
	return b.Why
}

func (h *HUD) drawWhy(dst *ebiten.Image, b Button) {
	why := cellWhy(b)
	if why == "" {
		return
	}
	strip := WhyStrip(b)
	ui.Fill(dst, strip, ui.Tint(ui.Palette.ConsoleEdge, 245))
	h.Font.DrawCentered(dst, ui.Clip(why, ui.MaxChars(strip.W-2, 1)), strip, 1, ui.Palette.TextDim)
}

func cellShowsIcon(b Button) bool {
	return cellWhy(b) == "" || !whyOnTop(b)
}

func whyOnTop(b Button) bool {
	return len(CellLabel(b.Label, ui.MaxChars(b.Rect.W-4, 1))) > 1
}

func WhyStrip(b Button) ui.Rect {
	r := b.Rect
	if whyOnTop(b) {
		return ui.Rect{X: r.X + 2 + whyKeyClear, Y: r.Y + whyDoubleY, W: r.W - 4 - whyKeyClear, H: whyHeight}
	}
	return ui.Rect{X: r.X + 2, Y: r.Y + whySingleY, W: r.W - 4, H: whyHeight}
}

func gatedPrice(b view.Buildable) string {
	if b.Locked && !b.Available {
		return ""
	}
	return Compact(b.Cost)
}
