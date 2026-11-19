package hud

import (
	"fmt"

	"github.com/waxb/kubernites/pkg/view"
)

func SeasonLine(c view.CrewView) string {
	step := 2.0
	if c.Multiplier > 0 && c.NextMultiplier > 0 {
		step = c.NextMultiplier / c.Multiplier
	}
	switch {
	case c.Milestone > 0 && c.Next == 0:
		return fmt.Sprintf("x%s SEASONED, every milestone won", Compact(c.Multiplier))
	case c.Milestone > 0:
		return fmt.Sprintf("x%s SEASONED, next x%s at %d", Compact(c.Multiplier), Compact(step), c.Next)
	case c.Next > 0:
		return fmt.Sprintf("next x%s at %d ready", Compact(step), c.Next)
	}
	return ""
}

func CrewOf(crews []view.CrewView, depot, name string) (view.CrewView, bool) {
	for _, c := range crews {
		if depot != "" && c.Depot == depot || name != "" && c.Name == name {
			return c, true
		}
	}
	return view.CrewView{}, false
}
