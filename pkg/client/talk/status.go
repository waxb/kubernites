package talk

import (
	"fmt"
	"math"

	"github.com/waxb/kubernites/pkg/client/model"
)

type Need struct {
	Key   string
	Label string
	Value string
	Share float64
	Mark  string
}

const (
	NeedFood   = "food"
	NeedRoom   = "room"
	NeedBusy   = "busy"
	NeedShaken = "shaken"
	NeedAge    = "age"
	freshAge   = 120.0
	mebi       = 1 << 20
)

func Ago(seconds float64) string {
	switch {
	case seconds < 0:
		return ""
	case seconds < 60:
		return fmt.Sprintf("%ds", int(seconds))
	case seconds < 3600:
		return fmt.Sprintf("%dm", int(seconds/60))
	case seconds < 86400:
		return fmt.Sprintf("%dh", int(seconds/3600))
	}
	return fmt.Sprintf("%dd", int(seconds/86400))
}

func Status(f Facts) string {
	if f.Names {
		return namesStatus(f)
	}
	return storyStatus(f)
}

func storyStatus(f Facts) string {
	w := f.Worker.State
	switch {
	case f.NeverArrived():
		return "Never arrived: the Academy sent no one."
	case f.Guttering():
		return "Guttering: stopped, and will be relit."
	case w.Phase == model.ContainerTerminated:
		return storyEnd(w)
	case f.Kindling():
		return "Being kindled."
	case f.Hut.Rites:
		return "Still burning; our hut is at Last Rites."
	case f.Hut.Frozen && f.Known["bell"]:
		return "Frozen: not answering the bell."
	case f.Hut.Frozen:
		return "Frozen: not answering."
	case f.Shuttered() && f.Known["shutters"]:
		return "Shutters down: takes no boxes yet."
	case f.Shuttered():
		return "Not taking boxes yet."
	}
	doing := storyDoing(f)
	if f.Hungry() {
		return "Hungry and slowed. " + doing
	}
	return doing
}

func storyEnd(w model.ContainerState) string {
	switch w.Reason {
	case "OOMKilled":
		return "Voidtaken."
	case "Completed":
		return "Finished its work."
	}
	return fmt.Sprintf("Stopped (exit %d).", w.ExitCode)
}

func storyDoing(f Facts) string {
	switch f.Doing.Activity {
	case DoCarrying:
		if f.Doing.Load == "" {
			return "Carrying a box."
		}
		if f.Hut.Depot != "" && f.Known["depot"] {
			return fmt.Sprintf("Carrying a %s to the %s depot.", f.Doing.Load, f.Hut.Depot)
		}
		return fmt.Sprintf("Carrying a %s.", f.Doing.Load)
	case DoReturning:
		return "Walking back to the quay."
	case DoWaiting:
		return "Waiting for boxes."
	case DoResting:
		return "Waiting for boxes by the hut."
	case DoServing:
		return "Serving at the door."
	}
	if f.Waiting() {
		return "Waiting for boxes."
	}
	return "At work."
}

func namesStatus(f Facts) string {
	w := f.Worker.State
	switch w.Phase {
	case "":
		return "Waiting: not started yet"
	case model.ContainerWaiting:
		if w.Reason == "" {
			return "Waiting"
		}
		return fmt.Sprintf("Waiting (Reason: %s)", w.Reason)
	case model.ContainerTerminated:
		return fmt.Sprintf("Terminated (Reason: %s, Exit Code: %d)", w.Reason, w.ExitCode)
	}
	if f.Hut.Rites {
		return "Running; stopping after SIGTERM within its grace period"
	}
	since := ""
	if up := f.Uptime(); up >= 0 {
		since = ", started " + Ago(up) + " ago"
	}
	return fmt.Sprintf("Running%s, Ready %v, Restart Count %d", since, f.Worker.Ready, f.Worker.Restarts)
}

func LastState(f Facts) string {
	l := f.Worker.Last
	if l.Phase != model.ContainerTerminated {
		return ""
	}
	if f.Names {
		return fmt.Sprintf("Last State: Terminated (Reason: %s, Exit Code: %d)", l.Reason, l.ExitCode)
	}
	if l.Reason == "OOMKilled" {
		return "Last stopped: voidtaken."
	}
	return fmt.Sprintf("Last stopped: exit %d.", l.ExitCode)
}

func share(used, of float64) float64 {
	if of <= 0 {
		return -1
	}
	return math.Min(used/of, 1)
}

func Needs(f Facts) []Need {
	out := []Need{}
	if f.Known["rations"] && f.Use.Measured && f.running() {
		if f.Use.Alone {
			out = append(out, food(f), room(f))
		} else {
			out = append(out, Need{Key: NeedFood, Label: label(f, "FOOD", "CPU"), Value: pick(f, "counted for the whole hut", "measured for the whole Pod"), Share: -1})
		}
	}
	if f.Use.Measured && f.Hut.Workers == 1 && f.running() {
		rate := rateText(f.Use.Served)
		value := pick(f, rate+" boxes a second", rate+" requests/s")
		if f.Use.Served < 0.01 {
			value = pick(f, "no boxes reach it", "no requests")
		}
		out = append(out, Need{Key: NeedBusy, Label: label(f, "BOXES", "SERVED"), Value: value, Share: -1})
	}
	out = append(out, shaken(f))
	if up := f.Uptime(); up >= 0 {
		word := pick(f, "SEASONED", "STARTED")
		value := pick(f, "burning "+Ago(up), Ago(up)+" ago")
		if up < freshAge && !f.Names {
			word = "FRESH"
		}
		out = append(out, Need{Key: NeedAge, Label: word, Value: value, Share: -1})
	}
	return out
}

func rateText(v float64) string {
	if v < 0.1 {
		return fmt.Sprintf("%.2f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

func food(f Facts) Need {
	r := f.Worker.Resources
	n := Need{Key: NeedFood, Label: label(f, "FOOD", "CPU"), Share: share(f.Use.CPUMilli, float64(r.CPURequest))}
	switch {
	case f.Names && r.CPULimit > 0:
		n.Value = fmt.Sprintf("%.0fm of %dm req, limit %dm", f.Use.CPUMilli, r.CPURequest, r.CPULimit)
	case f.Names:
		n.Value = fmt.Sprintf("%.0fm of %dm requested", f.Use.CPUMilli, r.CPURequest)
	case r.CPURequest > 0 && f.Known["pledge"]:
		n.Value = fmt.Sprintf("%.0f of %d rations pledged", f.Use.CPUMilli, r.CPURequest)
	default:
		n.Value = fmt.Sprintf("%.0f rations", f.Use.CPUMilli)
	}
	switch {
	case f.AtCPULimit() && f.Names:
		n.Mark = "THROTTLED"
	case f.Hungry() && f.Names:
		n.Mark = "CPU-STARVED"
	case f.Hungry():
		n.Mark = "HUNGRY"
	}
	return n
}

func room(f Facts) Need {
	r := f.Worker.Resources
	used := f.Use.MemBytes / mebi
	n := Need{Key: NeedRoom, Label: label(f, "ROOMS", "MEMORY"), Share: share(f.Use.MemBytes, float64(r.MemLimit))}
	if n.Share < 0 {
		n.Share = share(f.Use.MemBytes, float64(r.MemRequest))
	}
	switch {
	case f.Names && r.MemLimit > 0:
		n.Value = fmt.Sprintf("%.0fMi of %dMi req, limit %dMi", used, r.MemRequest/mebi, r.MemLimit/mebi)
	case f.Names:
		n.Value = fmt.Sprintf("%.0fMi of %dMi requested", used, r.MemRequest/mebi)
	case r.MemLimit > 0 && f.Known["binding"]:
		n.Value = fmt.Sprintf("%.0f, binding %d", used, r.MemLimit/mebi)
	case r.MemRequest > 0 && f.Known["pledge"]:
		n.Value = fmt.Sprintf("%.0f of %d pledged", used, r.MemRequest/mebi)
	default:
		n.Value = fmt.Sprintf("%.0f rooms", used)
	}
	if f.Cramped() {
		n.Mark = pick(f, "CRAMPED", "NEAR LIMIT")
	}
	return n
}

func shaken(f Facts) Need {
	n := Need{Key: NeedShaken, Label: label(f, "RELIT", "RESTARTS"), Share: -1}
	switch {
	case f.Worker.Restarts == 0:
		n.Value = pick(f, "never", "0")
	case f.RelitAgo() >= 0:
		n.Value = fmt.Sprintf("%d, last %s ago", f.Worker.Restarts, Ago(f.RelitAgo()))
	default:
		n.Value = fmt.Sprint(f.Worker.Restarts)
	}
	if f.Guttering() {
		n.Mark = pick(f, "GUTTERING", "CrashLoopBackOff")
	}
	return n
}

func label(f Facts, story, names string) string {
	return pick(f, story, names)
}

func pick(f Facts, story, names string) string {
	if f.Names {
		return names
	}
	return story
}

func Mood(f Facts) string {
	if f.Names {
		return health(f)
	}
	switch {
	case f.Guttering() || f.Relit():
		return "dizzy"
	case f.Hut.Frozen:
		return "dazed"
	case f.Hungry():
		return "peckish"
	case f.Cramped():
		return "cramped"
	case f.Kindling() || f.Shuttered() || f.NeverArrived():
		return "unsure"
	case f.Hut.Rites:
		return "wistful"
	case f.Waiting():
		return "lonely"
	}
	return "content"
}

func health(f Facts) string {
	switch {
	case f.Guttering():
		return "Restarting"
	case f.NeverArrived():
		return "Image not pulled"
	case f.Kindling():
		return "Not started"
	case f.Hut.Rites:
		return "Stopping"
	case f.Hut.Frozen:
		return "Liveness failing"
	case f.AtCPULimit():
		return "Throttled"
	case f.Hungry():
		return "Short of CPU"
	case f.Cramped():
		return "Near its memory limit"
	case f.Shuttered():
		return "Not Ready"
	case f.Relit():
		return "Restarted recently"
	}
	return "Ready"
}

func Thinks(id Identity, f Facts) string {
	l, ok := Current(id, f)
	if !ok {
		return ""
	}
	if l.Then != "" {
		return l.Say(f) + " " + l.Rest(f)
	}
	return l.Say(f)
}
