package hud

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func withRemedyEffect(l CommandLine, f Frame) CommandLine {
	next := l
	switch {
	case l.Blocked != "":
		next.Effect, next.EffectBad = CapitalFirst(l.Blocked)+".", true
	case l.Action == nil:
	case l.Action.Action == view.ActScaleCrew:
		next.Effect, next.EffectBad = scaleEffect(f, *l.Action)
		if charter := scaleCharter(f, *l.Action); charter != "" {
			next.Effect, next.EffectBad = strings.TrimSpace(next.Effect+" "+charter), true
		}
	case l.Action.Action == view.ActCrewResources:
		next.Effect, next.EffectBad = pledgeEffect(f, *l.Action)
	case l.Action.Action == "buyKeep":
		next.Effect = CapitalFirst(buyPayoff(f, l.Action))
	}
	return next
}

func scaleEffect(f Frame, a view.Action) (string, bool) {
	cv, ok := crewByName(f, a.Name)
	if !ok || a.Replicas <= cv.Replicas || cv.Room == (view.RoomFit{}) {
		return "", false
	}
	step := a.Replicas - cv.Replicas
	avail := cv.Room.Seated - cv.Room.Short
	fed := min(cv.Room.Fed, cv.Room.Seated) - cv.Room.Short
	upkeep := ""
	if up := cv.PerReplica.UpkeepShare * float64(step); up >= .005 {
		upkeep = ui.Bridge(", upkeep +"+Rate(up)+" gold/s", ", +$"+Rate(up)+"/s")
	}
	switch {
	case step > max(avail, 0):
		return fmt.Sprintf(ui.Bridge("%d of the %d new workers wait Unhoused: no Keep has room.", "%d of %d new Pods stay Pending: no Node fits them."), step-max(avail, 0), step), true
	case step > max(fed, 0) && cv.Room.Limit == view.FlowMemory:
		return fmt.Sprintf(ui.Bridge("All are placed, but %d crowd a Keep at what each really takes.", "All schedule, but %d overcommit memory at real use."), step-max(fed, 0)), true
	case step > max(fed, 0):
		return fmt.Sprintf(ui.Bridge("All are placed, but %d go hungry at what each really eats.", "All schedule, but %d overcommit CPU at real use."), step-max(fed, 0)), true
	}
	return fmt.Sprintf(ui.Bridge("All %d new workers fit%s.", "All %d new Pods fit%s."), step, upkeep), false
}

func pledgeEffect(f Frame, a view.Action) (string, bool) {
	cv, ok := crewByName(f, a.Name)
	if !ok || len(f.State.Room.Keeps) == 0 || a.Crew.Requests == "" {
		return "", false
	}
	old := view.ResourcePair{CPU: cv.PerReplica.CPU, Memory: cv.PerReplica.Memory}
	asks := repledgeOf(old, a.Crew.Requests)
	fit := f.State.Room.Repledged(a.Name, cv.Replicas, old, asks, cv.Use, cv.Work)
	return repledgeLine(fit, cv.Replicas), fit.Seated < 0 || fit.Fed < fit.Seated
}

func repledgeOf(old view.ResourcePair, requests string) view.ResourcePair {
	out := old
	for _, part := range strings.Split(requests, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		q, err := resource.ParseQuantity(value)
		if err != nil {
			continue
		}
		switch key {
		case "cpu":
			out.CPU = q.MilliValue()
		case "memory":
			out.Memory = q.Value()
		}
	}
	return out
}
