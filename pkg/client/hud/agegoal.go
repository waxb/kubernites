package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const ageGoalRows = 2

type ageAsk struct {
	label string
	met   bool
}

func ageAskOf(r view.Requirement) (ageAsk, bool) {
	switch r.Kind {
	case view.RequireNine:
		if r.Latched {
			return ageAsk{"One Nine (held)", r.Met}, true
		}
		return ageAsk{"One Nine", r.Met}, true
	case view.RequirePort:
		return ageAsk{ui.Bridge("a caravel port", "a medium-burst port"), r.Met}, true
	case view.RequireGold:
		if r.Set {
			return ageAsk{view.Grouped(int64(r.Gold)) + " gold (set)", r.Met}, true
		}
		return ageAsk{fmt.Sprintf("%.0f minutes of your income (now %s gold)", view.AgeUpIncomeMinutes, view.Grouped(int64(r.Gold))), r.Met}, true
	case view.RequireWisdom:
		return ageAsk{"wisdom", r.Met}, true
	case view.RequireChronicle:
		return ageAsk{"a Chronicle", r.Met}, true
	}
	return ageAsk{}, false
}

func ageAsks(f Frame) []ageAsk {
	out := []ageAsk{}
	for _, r := range f.State.AgeUp.Requirements {
		if a, ok := ageAskOf(r); ok {
			out = append(out, a)
		}
	}
	return out
}

func (h *HUD) ageGoalLines(f Frame, w int) []Line {
	asks := ageAsks(f)
	if f.State.Age != view.AgeDark || len(asks) == 0 || !f.Shown(MeterGold) {
		return nil
	}
	have, need := []string{}, []string{}
	for _, a := range asks {
		if a.met {
			have = append(have, a.label)
		} else {
			need = append(need, a.label)
		}
	}
	out := []Line{{Text: "TO REACH THE FEUDAL AGE", Clr: ui.Palette.Gold}}
	for _, part := range []struct {
		head  string
		items []string
		clr   color.RGBA
	}{{"done: ", have, ui.Palette.Green}, {"to do: ", need, ui.Palette.TextDim}} {
		if len(part.items) == 0 {
			continue
		}
		for _, row := range h.Font.WrapLines(part.head+strings.Join(part.items, ", "), w, 1, ageGoalRows) {
			out = append(out, Line{Text: row, Clr: part.clr})
		}
	}
	return out
}
