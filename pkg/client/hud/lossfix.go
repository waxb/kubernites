package hud

import (
	"fmt"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	familiarsUnlock = "familiars"
	lossLineChars   = (BoardW - 2*BoardPad) / 6
)

func cardFix(st view.GameState, kind, box, fallback string) string {
	for _, c := range st.Calamities {
		if c.ID == kind+":"+box && len(c.Remedies) > 0 {
			return fadedShortFix(st, c.Remedies[0])
		}
	}
	return fallback
}

func ShortFix(rem view.RemedyView) string {
	a := rem.Action
	switch {
	case a == nil && strings.HasPrefix(rem.Command, "kubectl delete pod"):
		return "raze a hut"
	case a == nil:
		return shortLabel(rem.Label)
	}
	switch a.Action {
	case view.ActScaleCrew:
		return fmt.Sprintf("scale %s to %d", a.Name, a.Replicas)
	case view.ActCrewImage:
		return "roll " + a.Image + " out"
	case view.ActRolloutCrew:
		return "roll " + a.Name + " back"
	case view.ActRetuneHorn:
		return "raise the Horn's Ceiling"
	case view.ActReseatDoor:
		return "switch the door porter"
	case view.ActMoveWorker:
		return "raze a hut"
	case "buyKeep":
		return "raise a Keep"
	case view.ActCloseBerth:
		return "close a berth"
	case "buySoulstone":
		return "endow at the Academy"
	case "raiseCrew":
		return "raise " + a.Name
	case "openDepot":
		if d, ok := soulstone.DepotFor(a.Kind); ok {
			return ui.Bridge("open the "+d.Service, "expose "+d.Crew)
		}
	}
	return shortLabel(rem.Label)
}

func shortLabel(label string) string {
	head, _, _ := strings.Cut(label, ": ")
	head, _, _ = strings.Cut(head, " (")
	if head == "" {
		return head
	}
	return strings.ToLower(head[:1]) + head[1:]
}
