package app

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/view"
)

var tidingPhrases = map[string]string{
	"Scheduled": "is placed on a Keep by the Quartermaster", "FailedScheduling": "is Unhoused: no Keep has room",
	"Pulling": "waits for its trade to arrive from overseas", "Pulled": "has received its trade",
	"Created": "gets its worker", "Started": "is Burning: up and ready for boxes", "Killing": "is shutting down (Last Rites)",
	"Terminating": "is shutting down (Last Rites)", "BackOff": "is Guttering: its worker stops and is relit again and again",
	"Unhealthy": "is Unfit for Duty: not ready for boxes", "Tremor": "was toppled by the Simian Imp",
	"OOMKilling": "is Voidtaken: killed for using too much housing", "OOMKilled": "is Voidtaken: killed for using too much housing",
	"Evicted": "is Banished: turned out of a crowded Keep", "NodeReady": "is ready to host huts",
	"NodeNotReady": "has gone dark: it hosts no new huts", "RegisteredNode": "is raised", "EvictionThresholdMet": "runs short of housing",
	"SuccessfulCreate": "raises a hut", "SuccessfulDelete": "sends a hut home",
	"FailedCreatePodSandBox": "cannot get started", "Completed": "has finished its Quest",
	"Preempted": "is pushed out to make room for a hut of higher Rank",
}

const (
	neverArrived  = "The Graduate Never Arrived"
	scalingReason = "ScalingReplicaSet"
)

var scalingLine = regexp.MustCompile(`^Scaled (up|down) replica set (\S+) from (\d+) to (\d+)$`)

func scalingStory(message string) string {
	m := scalingLine.FindStringSubmatch(message)
	huts := "huts"
	if m[4] == "1" {
		huts = "hut"
	}
	return fmt.Sprintf("bids its shift %s keep %s %s, %s from %s", m[2], m[4], huts, m[1], m[3])
}

var gameNames = map[string]string{
	"Pod": "Hut", "Node": "Keep", "ReplicaSet": "Shift", "Service": "Guildhall", "ConfigMap": "Scroll", "EndpointSlice": "Guild Roll",
}

var vocabularyKinds = map[string]string{
	"pods": store.KindPod, "nodes": store.KindNode, "replicasets": store.KindReplicaSet, "services": store.KindService,
	"configmaps": store.KindConfigMap, "namespaces": store.KindNamespace, "endpointslices": store.KindEndpointSlice,
}

func gameName(kind string) string {
	if name, ok := gameNames[kind]; ok {
		return name
	}
	return kind
}

func TidingOf(e *corev1.Event) hud.Tiding {
	subject := gameName(e.InvolvedObject.Kind) + " " + e.InvolvedObject.Name
	text := subject + ": " + e.Message
	switch phrase, ok := tidingPhrases[e.Reason]; {
	case ok:
		text = subject + " " + phrase
	case storyReasons[e.Reason] != nil:
		story, _ := storyPhrase(e)
		text = ui.Bridge(subject+" "+story, kubeLine(e))
	case e.Reason == scalingReason && scalingLine.MatchString(e.Message):
		text = ui.Bridge(subject+" "+scalingStory(e.Message), text)
	case e.Reason == "Failed" && strings.Contains(strings.ToLower(e.Message), "pull"):
		text = subject + ": " + neverArrived
	}
	return hud.Tiding{Time: EventTime(e), Kind: "event", Text: text, Reason: e.Reason, Warning: e.Type == corev1.EventTypeWarning}
}

var allClear = map[string]bool{"NodeHasSufficientMemory": true, "NodeHasNoDiskPressure": true, "NodeHasSufficientPID": true}

func Tidings(events []*corev1.Event, procs []view.Proclamation, withEvents bool, epoch float64) []hud.Tiding {
	out := []hud.Tiding{}
	folded := stepLessons(procs)
	for i, p := range procs {
		if _, gone := folded[i]; p.Text == "" || gone {
			continue
		}
		t := hud.Tiding{Time: epoch + p.T, Kind: p.Kind, Text: p.Text, Warning: p.Kind == "calamity"}
		if p.Kind == "lesson" {
			title, text := LessonOf(p)
			t.Text, t.Title, t.Detail = LessonLine(title, text), title, text
			if done, ok := doneLineOf(procs, folded, i); ok {
				t.Text = done + ". " + firstSentence(text)
			}
		}
		out = append(out, t)
	}
	if withEvents {
		out = append(out, eventTidings(events)...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time > out[j].Time })
	out = MergeTidings(out)
	return out[:min(len(out), MaxTidings)]
}

const (
	lessonLead    = "What just happened: "
	fulfilledLead = "Objective fulfilled: "
)

func stepLessons(procs []view.Proclamation) map[int]int {
	out := map[int]int{}
	for i, p := range procs {
		if p.Kind != "lesson" || p.Title == "" {
			continue
		}
		for j, q := range procs {
			if q.Kind == "info" && q.T == p.T && strings.HasPrefix(q.Text, fulfilledLead+p.Title) {
				out[j] = i
				break
			}
		}
	}
	return out
}

func doneLineOf(procs []view.Proclamation, folded map[int]int, lesson int) (string, bool) {
	for info, i := range folded {
		if i == lesson {
			return strings.TrimSuffix(strings.TrimSpace(procs[info].Text), "."), true
		}
	}
	return "", false
}

func firstSentence(text string) string {
	first := strings.TrimSpace(text)
	if i := strings.Index(first, lessonStop); i > 0 {
		first = first[:i+1]
	}
	return first
}

func WithOmen(tidings []hud.Tiding, imp *view.ImpView) []hud.Tiding {
	if imp == nil {
		return tidings
	}
	omen := hud.Tiding{Kind: "omen", Text: "The Simian Imp is running at " + imp.Pod + "!", Reason: "Omen"}
	return append([]hud.Tiding{omen}, tidings...)
}

var spacedKinds = []string{"pods", "replicasets", "services", "configmaps", "endpointslices"}

func Vocab(c *model.Cache) hud.Vocabulary {
	names := make(map[string][]string, len(vocabularyKinds))
	for plural, kind := range vocabularyKinds {
		names[plural] = c.Names(kind)
	}
	spaced := make(map[string]map[string][]string, len(spacedKinds))
	for _, plural := range spacedKinds {
		spaced[plural] = c.SpacedNames(vocabularyKinds[plural])
	}
	return hud.Vocabulary{Names: names, Spaced: spaced}
}

const MaxEchoes = 20

func Echo(echoes []hud.Tiding, it hud.Intent, at float64) []hud.Tiding {
	text := "$ " + hud.EchoLabel(it.Command)
	if it.Target != "" {
		text += ": " + it.Target
	}
	next := append(append([]hud.Tiding(nil), echoes...), hud.Tiding{Time: at, Kind: "kubectl", Text: text, Warning: it.Tab == "error"})
	return next[max(len(next)-MaxEchoes, 0):]
}

func Chronicled(echoes []hud.Tiding, text string, at float64) []hud.Tiding {
	next := append(append([]hud.Tiding(nil), echoes...), hud.Tiding{Time: at, Kind: "info", Text: text})
	return next[max(len(next)-MaxEchoes, 0):]
}

func WithEchoes(tidings, echoes []hud.Tiding) []hud.Tiding {
	if len(echoes) == 0 {
		return tidings
	}
	out := append(append([]hud.Tiding(nil), tidings...), echoes...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time > out[j].Time })
	return out[:min(len(out), MaxTidings)]
}
