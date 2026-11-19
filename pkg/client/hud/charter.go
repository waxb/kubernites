package hud

import (
	"fmt"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func charterCrossed(at, from, to int) bool {
	return at > 0 && from < at && to >= at
}

func charterLine(at int) string {
	return fmt.Sprintf(ui.Bridge("At %d porters a merchant's charter sails in: more boxes than %d can carry.", "At %d replicas a customer's charter arrives: more requests than %d Pods can serve."), at, at)
}

func scaleCharter(f Frame, a view.Action) string {
	cv, ok := crewByName(f, a.Name)
	if !ok || a.Action != view.ActScaleCrew || !charterCrossed(cv.CharterAt, cv.Replicas, a.Replicas) {
		return ""
	}
	return charterLine(cv.CharterAt)
}

func nextCharter(n *view.NextView, st view.GameState) string {
	if n.Kind != view.NextMilestone {
		return ""
	}
	for _, c := range st.Crews {
		if c.Name == n.ID && charterCrossed(c.CharterAt, c.Replicas, c.Next) {
			return charterLine(c.CharterAt)
		}
	}
	return ""
}
