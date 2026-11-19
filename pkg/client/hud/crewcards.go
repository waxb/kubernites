package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/soulstone"
)

const ForemanPortrait = "portrait/tallyman"

func CrewCard(f Frame) Card {
	c, ok := ChainOf(f)
	if !ok {
		return Card{progress: -1}
	}
	if f.Selection.Kind == SelectForeman {
		if i := foremanOf(c, f.Selection.ID); i >= 0 {
			return foremanCardIn(c, c.Foremen[i], f)
		}
	}
	if c.Bare && len(c.Foremen) > 0 {
		return foremanCardIn(c, c.Foremen[0], f)
	}
	return DeploymentCard(c)
}

func DeploymentCard(c model.CrewChain) Card {
	_, sprite := soulstone.ProfessionOf(c.Image)
	out := Card{Portrait: "portrait/" + sprite, Title: upperBridge("Crew "+c.Name, "Deployment "+c.Name), K8s: "Deployment/" + c.Name, progress: -1}
	out.Lines = append(out.Lines,
		countLine(c),
		Line{Text: ui.Bridge("TRADE ", "IMAGE ") + c.Image, Clr: ui.Palette.Text},
		Line{Text: StrategyLine(c), Clr: ui.Palette.TextDim},
		Line{Text: fmt.Sprintf("REVISION %d  HISTORY %d OF %d", c.Revision, len(c.History()), c.HistoryLimit), Clr: ui.Palette.TextDim})
	if c.Paused {
		out.Lines = append(out.Lines, Line{Text: ui.Bridge("PAUSED: template changes wait for RESUME", "PAUSED: kubectl rollout resume to continue"), Clr: ui.Palette.RedText})
	}
	out.Lines = append(out.Lines, Line{Text: ui.Bridge("One shift per revision; the newest keeps the huts.",
		"One ReplicaSet per revision; the newest scales up."), Clr: ui.Palette.Cream, Optional: true})
	return out
}

func upperBridge(story, k8s string) string {
	return ui.Bridge(strings.ToUpper(story), strings.ToUpper(k8s))
}

func StrategyLine(c model.CrewChain) string {
	if c.MaxSurge == "" {
		return strings.ToUpper(c.Strategy)
	}
	return fmt.Sprintf(ui.Bridge("EXTRA HUTS %s  HUTS MAY BE DOWN %s", "SURGE %s  UNAVAILABLE %s"), c.MaxSurge, c.MaxUnavailable)
}

func readyTone(ready, desired int) color.RGBA {
	if ready >= desired {
		return ui.Palette.Green
	}
	return ui.Palette.GoldBright
}

func ForemanCard(c model.CrewChain, fm model.Foreman) Card {
	return foremanCard(c, fm, ForemanRole(c, fm))
}

func foremanCardIn(c model.CrewChain, fm model.Foreman, f Frame) Card {
	if c.Bare && foremenOnly(f) {
		return foremanCard(c, fm, ui.Bridge("It keeps its count: a hut that is razed, it raises again.", "A ReplicaSet keeps its replicas: it replaces every Pod that goes."))
	}
	return ForemanCard(c, fm)
}

func foremanCard(c model.CrewChain, fm model.Foreman, role string) Card {
	out := Card{Portrait: ForemanPortrait, Title: upperBridge("Shift "+fm.Name, "ReplicaSet "+fm.Name), K8s: "ReplicaSet/" + fm.Name, progress: -1}
	out.Lines = append(out.Lines,
		Line{Text: fmt.Sprintf(ui.Bridge("KEEPS %d  READY %d  HUTS %d", "REPLICAS %d  READY %d  PODS %d"), fm.Desired, fm.Ready, len(fm.Pods)), Clr: readyTone(fm.Ready, fm.Desired)})
	if text := revisionText(fm); text != "" {
		out.Lines = append(out.Lines, Line{Text: text, Clr: ui.Palette.TextDim})
	}
	out.Lines = append(out.Lines, Line{Text: ui.Bridge("TRADE ", "IMAGE ") + fm.Image, Clr: ui.Palette.Text},
		Line{Text: role, Clr: ui.Palette.Cream})
	return out
}

func revisionText(fm model.Foreman) string {
	switch {
	case fm.Hash != "":
		return fmt.Sprintf("REVISION %d  pod-template-hash %s", fm.Revision, fm.Hash)
	case fm.Revision > 0:
		return fmt.Sprintf("REVISION %d", fm.Revision)
	}
	return ""
}

func ForemanRole(c model.CrewChain, fm model.Foreman) string {
	switch {
	case c.Bare:
		return ui.Bridge("No crew stands behind it: a bare ReplicaSet. It still raises a hut for every one that falls.",
			"No Deployment owns it: a bare ReplicaSet still replaces every Pod that goes.")
	case fm.Desired == 0 && !fm.Newest:
		return ui.Bridge("An old revision, kept for ROLLOUT UNDO. It keeps no huts now.",
			"An old revision with 0 replicas, kept for kubectl rollout undo.")
	}
	return ui.Bridge("Replaces any hut that is razed. Scale the crew, not the shift: the Deployment sets its count.",
		"Replaces any Pod that goes. Scale the Deployment: it sets this ReplicaSet's replicas.")
}

func crewDetailTarget(sel Selection) (target, bool) {
	ns, name, ok := strings.Cut(sel.ID, "/")
	if !ok {
		return target{}, false
	}
	kind := "deployment"
	if sel.Kind == SelectForeman {
		kind = "replicaset"
	}
	return target{kind: kind, namespace: ns, name: name}, true
}

func crewSelectionTitle(sel Selection) string {
	_, name, _ := strings.Cut(sel.ID, "/")
	if sel.Kind == SelectForeman {
		return ui.Bridge("Shift", "ReplicaSet") + " - " + name
	}
	return ui.Bridge("Crew", "Deployment") + " - " + name
}

const uncountedLabel = "COUNTING ITS HUTS"

func uncountedText() string {
	return ui.Bridge(uncountedLabel, "STATUS NOT YET OBSERVED")
}

func countLine(c model.CrewChain) Line {
	if c.Uncounted {
		return Line{Text: fmt.Sprintf("WANTS %d  ", c.Desired) + uncountedText(), Clr: ui.Palette.TextDim}
	}
	return Line{Text: fmt.Sprintf("WANTS %d  READY %d  UP-TO-DATE %d", c.Desired, c.Ready, c.Updated), Clr: readyTone(c.Ready, c.Desired)}
}
