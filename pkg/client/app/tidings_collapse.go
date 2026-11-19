package app

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	collapseWindow = 3.0
	podSuffixLen   = 5
	hashMinLen     = 6
	hashMaxLen     = 10
	randomAlphabet = "bcdfghjklmnpqrstvwxz2456789"
	podKind        = "Pod"
)

var pluralVerbs = map[string]string{"is": "are", "was": "were", "has": "have", "cannot": "cannot"}

var pluralPhrases = map[string]string{
	"Scheduled": "are placed on Keeps by the Quartermaster", "Pulling": "wait for their trade to arrive from overseas",
	"Pulled": "have received their trade", "Created": "get their worker", "Started": "are Burning: up and ready for boxes",
	"Killing": "are shutting down (Last Rites)", "Terminating": "are shutting down (Last Rites)",
	"BackOff": "are Guttering: their workers stop and are relit again and again", "Unhealthy": "are Unfit for Duty: not ready for boxes",
	"Tremor": "were toppled by the Simian Imp", "OOMKilling": "are Voidtaken: killed for using too much housing",
	"OOMKilled": "are Voidtaken: killed for using too much housing", "Evicted": "are Banished: turned out of a crowded Keep",
	"Completed": "have finished their Quest", "Preempted": "are pushed out to make room for huts of higher Rank",
}

var causeReasons = map[string]bool{
	"FailedScheduling": true, "BackOff": true, "Unhealthy": true, "OOMKilling": true, "OOMKilled": true, "Evicted": true, "Preempted": true,
}

var addonKinds = map[string]bool{"DaemonSet": true, "Deployment": true}

func citadelAddon(obj corev1.ObjectReference) bool {
	return obj.Namespace == systemNS && addonKinds[obj.Kind]
}

type tidingCluster struct {
	start  float64
	events []*corev1.Event
}

func eventTidings(events []*corev1.Event) []hud.Tiding {
	sorted := make([]*corev1.Event, 0, len(events))
	for _, e := range events {
		if !allClear[e.Reason] && !citadelAddon(e.InvolvedObject) {
			sorted = append(sorted, e)
		}
	}
	sort.SliceStable(sorted, func(i, j int) bool { return EventTime(sorted[i]) < EventTime(sorted[j]) })
	out := []hud.Tiding{}
	open := map[string]*tidingCluster{}
	for _, e := range sorted {
		key, ok := collapseKey(e)
		if !ok {
			out = append(out, TidingOf(e))
			continue
		}
		at := EventTime(e)
		if c := open[key]; c != nil && at-c.start <= collapseWindow {
			c.events = append(c.events, e)
			continue
		} else if c != nil {
			out = append(out, c.tiding())
		}
		open[key] = &tidingCluster{start: at, events: []*corev1.Event{e}}
	}
	keys := make([]string, 0, len(open))
	for k := range open {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, open[k].tiding())
	}
	return out
}

func collapseKey(e *corev1.Event) (string, bool) {
	if e.InvolvedObject.Kind != podKind {
		return "", false
	}
	if causeReasons[e.Reason] {
		return e.Reason + "|", true
	}
	crew, ok := crewOfPod(e.InvolvedObject.Name)
	if !ok {
		return "", false
	}
	return e.Reason + "|" + crew, true
}

func crewOfPod(name string) (string, bool) {
	parts := strings.Split(name, "-")
	n := len(parts)
	if n < 2 || !randomPart(parts[n-1], podSuffixLen, podSuffixLen) {
		return "", false
	}
	if n >= 3 && randomPart(parts[n-2], hashMinLen, hashMaxLen) {
		return strings.Join(parts[:n-2], "-"), true
	}
	return strings.Join(parts[:n-1], "-"), true
}

func randomPart(s string, lo, hi int) bool {
	if len(s) < lo || len(s) > hi {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune(randomAlphabet, r) {
			return false
		}
	}
	return true
}

func (c *tidingCluster) tiding() hud.Tiding {
	last := c.events[len(c.events)-1]
	if len(c.events) == 1 {
		return TidingOf(last)
	}
	t := TidingOf(last)
	crew, _ := crewOfPod(last.InvolvedObject.Name)
	subject := fmt.Sprintf("%d %s %ss", len(c.events), crewTrade(crew), strings.ToLower(gameName(podKind)))
	if causeReasons[last.Reason] {
		subject = fmt.Sprintf("%d %ss", len(c.events), strings.ToLower(gameName(podKind)))
	}
	if phrase, ok := pluralPhrases[last.Reason]; ok {
		t.Text = subject + " " + phrase
	} else if phrase, ok := tidingPhrases[last.Reason]; ok {
		t.Text = subject + " " + pluralPhrase(phrase)
	} else if story, ok := storyPhrase(last); ok {
		t.Text = ui.Bridge(subject+" "+pluralPhrase(story), fmt.Sprintf("%d %s Pods: %s", len(c.events), crewTrade(crew), last.Message))
		if causeReasons[last.Reason] {
			t.Text = ui.Bridge(subject+" "+pluralPhrase(story), fmt.Sprintf("%d Pods: %s", len(c.events), last.Message))
		}
	} else {
		t.Text = subject + ": " + last.Message
	}
	for _, e := range c.events {
		t.Warning = t.Warning || e.Type == corev1.EventTypeWarning
	}
	return t
}

func pluralPhrase(phrase string) string {
	words := strings.Fields(phrase)
	if len(words) == 0 {
		return phrase
	}
	if v, ok := pluralVerbs[words[0]]; ok {
		words[0] = v
	} else {
		words[0] = strings.TrimSuffix(words[0], "s")
	}
	for i, w := range words {
		if w == "its" {
			words[i] = "their"
		}
	}
	return strings.Join(words, " ")
}

func crewTrade(crew string) string {
	if len(crew) > 3 && strings.HasSuffix(crew, "s") {
		return strings.TrimSuffix(crew, "s")
	}
	return crew
}
