package talk

import (
	"fmt"
	"regexp"
	"strings"
)

var carrierWords = regexp.MustCompile(`(?i)\b(?:box|boxes|carr(?:y|ies|ied|ying)|parcels?|crates?|quay|ledgers?|seals?|cart|ships?|gilded|lift)\b`)

const (
	PoolStory = iota
	PoolNames
	PoolBoth
)

type Line struct {
	Text  string
	When  Situation
	Tone  Tone
	Pool  int
	Needs []string
	If    func(Facts) bool
	Then  string
	Trade string
}

func (l Line) speaks(f Facts) bool {
	switch {
	case l.Pool == PoolStory && f.Names, l.Pool == PoolNames && !f.Names:
		return false
	case l.Trade != "" && l.Trade != f.Trade:
		return false
	case l.Trade == "" && !f.Carrier && carrierWords.MatchString(l.Text+" "+l.Then):
		return false
	case !f.knows(l.Needs):
		return false
	case l.When != SitAnnoyed && !holds(l.When, f):
		return false
	case l.If != nil && !l.If(f):
		return false
	}
	return true
}

func (l Line) Say(f Facts) string {
	return fill(l.Text, f)
}

func (l Line) Rest(f Facts) string {
	return fill(l.Then, f)
}

func fill(text string, f Facts) string {
	if !strings.Contains(text, "{") {
		return text
	}
	r := strings.NewReplacer(
		"{rations}", fmt.Sprint(f.Worker.Resources.CPURequest),
		"{cpu}", fmt.Sprintf("%dm", f.Worker.Resources.CPURequest),
		"{app}", f.Hut.App,
		"{trade}", strings.ToLower(f.Trade),
		"{exit}", fmt.Sprint(f.Worker.Last.ExitCode),
		"{depot}", f.Hut.Depot,
	)
	return r.Replace(text)
}

func pledged(f Facts) bool { return f.Worker.Resources.CPURequest > 0 }

func hasApp(f Facts) bool { return f.Hut.App != "" }

func onRollApp(f Facts) bool { return f.Hut.OnRoll && f.Hut.App != "" }

func scheduled(f Facts) bool { return !f.Hut.Static && f.Hut.Node != "" }

func drainable(f Facts) bool { return f.ReplicaSet() }

func busy(f Facts) bool { return f.Use.Served > 0 || f.Doing.Activity == DoCarrying }

func shortlived(f Facts) bool {
	up := f.Uptime()
	return up >= 0 && up < 300
}

func oomed(f Facts) bool {
	return f.Worker.Last.Reason == "OOMKilled" && f.Worker.Resources.MemLimit > 0
}

func roomy(f Facts) bool {
	limit := f.Worker.Resources.MemLimit
	return limit > 0 && f.Use.Measured && f.Use.Alone && f.Use.MemBytes < 0.5*float64(limit) && f.Worker.Last.Reason != "OOMKilled"
}

func alive(f Facts) bool { return f.Worker.Liveness && f.running() && !f.Hut.Frozen }

func readyProbe(f Facts) bool { return f.Worker.Readiness && f.Worker.Ready }

func restarts(f Facts) bool { return f.Restarts() }

func impReplaced(f Facts) bool { return f.Around.ImpReplaced }

func retiring(f Facts) bool { return f.Hut.Rollout && f.Hut.OldClass && f.ReplicaSet() }

func again(f Facts) bool { return f.Worker.Restarts >= 2 }

func daytime(f Facts) bool { return !f.Around.Night }

func stateless(f Facts) bool { return f.Hut.Claims == 0 }

func imageReady(f Facts) bool { return f.Hut.ImageReady }

func newAge(f Facts) bool { return recent(f.Around.AgeAgo) }

func firstClick(f Facts) bool { return f.FirstClick }
