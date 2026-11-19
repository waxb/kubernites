package view

import (
	"fmt"
)

func (p CrewRoomPlan) StarvingLine(crew string) string {
	what := fmt.Sprintf("scaling the %s down to %d", crew, p.To)
	if p.Horned {
		what = fmt.Sprintf("holding the %s Horn's Ceiling at %d", crew, p.To)
	}
	return fmt.Sprintf("%s leaves fewer than the %d the ships need at noon", what, p.Starves)
}

func (p CrewRoomPlan) How(crew string) string {
	if p.Horned {
		return fmt.Sprintf("the %s Horn's Ceiling held at %d", crew, p.To)
	}
	return fmt.Sprintf("down by %d, to %d", p.Shrink, p.To)
}
