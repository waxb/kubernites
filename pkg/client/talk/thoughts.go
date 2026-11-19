package talk

import (
	"fmt"
	"sort"

	"github.com/waxb/kubernites/pkg/client/model"
)

const (
	HapKindled  = "kindled"
	HapPulled   = "pulled"
	HapFirstBox = "first-box"
	HapHungry   = "hungry"
	HapFed      = "fed"
	HapCramped  = "cramped"
	HapRelit    = "relit"
	HapBackOff  = "backoff"
	HapJoined   = "joined"
	HapLeft     = "left"
	HapRollout  = "rollout"
	HapRites    = "rites"
	HapKilled   = "killed"
	MaxThoughts = 5
)

type Happening struct {
	At     float64
	Kind   string
	Reason string
	Exit   int32
	Box    string
}

type Thought struct {
	At   float64
	Text string
}

func FromStatus(f Facts) []Happening {
	out := []Happening{}
	w := f.Worker
	if w.State.Phase == model.ContainerRunning && w.State.Started > 0 {
		out = append(out, Happening{At: w.State.Started, Kind: HapKindled})
	}
	if w.Restarts > 0 && w.Last.Phase == model.ContainerTerminated && w.Last.Finished > 0 {
		out = append(out, Happening{At: w.Last.Finished, Kind: HapRelit, Reason: w.Last.Reason, Exit: w.Last.ExitCode})
	}
	return out
}

func Thoughts(f Facts, haps []Happening, n int) []Thought {
	sorted := append([]Happening(nil), haps...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].At > sorted[j].At })
	out := []Thought{}
	seen := map[string]bool{}
	for _, h := range sorted {
		text := thoughtText(f, h)
		key := fmt.Sprintf("%s@%.0f", h.Kind, h.At)
		if text == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Thought{At: h.At, Text: text})
		if len(out) == n {
			break
		}
	}
	return out
}

func thoughtText(f Facts, h Happening) string {
	if f.Names {
		return namesThought(f, h)
	}
	return storyThought(f, h)
}

func storyThought(f Facts, h Happening) string {
	switch h.Kind {
	case HapKindled:
		return "Kindled. Here we go."
	case HapPulled:
		return "My trade was already at the Keep. Quick start."
	case HapFirstBox:
		if h.Box == "" {
			return "Carried my first box!"
		}
		return "Carried my first " + h.Box + "!"
	case HapHungry:
		return "Hungry. Everything slowed down."
	case HapFed:
		return "Fed again. Much better."
	case HapCramped:
		if f.Known["binding"] {
			return "Cramped, close to my binding."
		}
		return "Cramped. Hardly any room."
	case HapRelit:
		if h.Reason == "OOMKilled" && f.Known["voidtaken"] {
			return "Voidtaken, then relit."
		}
		return fmt.Sprintf("Stopped (exit %d), then relit.", h.Exit)
	case HapBackOff:
		return "Stopped again. The relighting waits longer now."
	case HapJoined:
		return known(f, "roll", "Our hut joined the roll.")
	case HapLeft:
		return known(f, "roll", "Our hut left the roll.")
	case HapRollout:
		return known(f, "crew", "Our crew began a new class.")
	case HapRites:
		return "Our hut is at Last Rites."
	case HapKilled:
		if f.Known["bell"] {
			return "The bell rang, and I was put out."
		}
		return "I was put out."
	}
	return ""
}

func namesThought(f Facts, h Happening) string {
	switch h.Kind {
	case HapKindled:
		return "Started."
	case HapPulled:
		return "Image already present on machine."
	case HapFirstBox:
		return "Served my first request."
	case HapHungry:
		if f.Worker.Resources.CPULimit > 0 && h.Reason == "limit" {
			return "Throttled at my CPU limit."
		}
		return "Short of CPU."
	case HapFed:
		return "CPU enough again."
	case HapCramped:
		return "Working set near my memory limit."
	case HapRelit:
		return fmt.Sprintf("Restarted after %s, exit code %d.", h.Reason, h.Exit)
	case HapBackOff:
		return "Back-off restarting failed container."
	case HapJoined:
		return known(f, "depot", "Our Pod joined the Service's endpoints.")
	case HapLeft:
		return known(f, "depot", "Our Pod left the Service's endpoints.")
	case HapRollout:
		return known(f, "crew", "Our Deployment began a rollout.")
	case HapRites:
		return "Our Pod is being deleted: SIGTERM."
	case HapKilled:
		return "Killed: liveness probe failed."
	}
	return ""
}

func known(f Facts, concept, text string) string {
	if !f.Known[concept] {
		return ""
	}
	return text
}
