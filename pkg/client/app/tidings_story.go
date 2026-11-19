package app

import (
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

var storyReasons = map[string]func(message string) string{
	"NodeHasInsufficientMemory":    said("is Crowded: its housing runs short"),
	"NodeHasDiskPressure":          said("is choked: its ground runs short"),
	"NodeHasInsufficientPID":       said("is Thronged: too many fires burn in it"),
	"TaintManagerEviction":         wardEviction,
	"SuccessfulRescale":            hornRescale,
	"FailedRescale":                hornRescale,
	"FailedComputeMetricsReplicas": said("has no Omens to reckon by, so it keeps the crew as it is"),
	"FailedGetResourceMetric":      said("cannot read the Augur's Omens"),
	"InvalidMetricSourceType":      said("is told to read an Omen it does not know"),
	"FailedGetScale":               said("cannot find the crew it sounds for"),
	"FailedUpdateStatus":           said("cannot write its reading in the Codex"),
}

func said(phrase string) func(string) string {
	return func(string) string { return phrase }
}

func wardEviction(message string) string {
	if strings.HasPrefix(message, "Cancelling") {
		return "is spared: its removal by the Ward is cancelled"
	}
	return "is marked to be turned out by a Ward"
}

var rescaleLine = regexp.MustCompile(`^New size: (\d+); reason: ([^;]*)`)

func hornRescale(message string) string {
	m := rescaleLine.FindStringSubmatch(message)
	if m == nil {
		return "sounds, but its call goes unheard"
	}
	verb := "sets its crew's count to"
	if strings.Contains(message, "; error:") {
		verb = "cannot set its crew's count to"
	}
	return fmt.Sprintf("%s %s: %s", verb, m[1], hornReason(m[2]))
}

func hornReason(reason string) string {
	switch {
	case strings.HasSuffix(reason, "above Spec.MaxReplicas"):
		return "no more than its ceiling"
	case strings.HasSuffix(reason, "below Spec.MinReplicas"):
		return "no fewer than its floor"
	case strings.HasSuffix(reason, "above target"):
		return "they eat past the mark"
	}
	return "they eat below the mark"
}

func storyPhrase(e *corev1.Event) (string, bool) {
	f, ok := storyReasons[e.Reason]
	if !ok {
		return "", false
	}
	return f(e.Message), true
}

func kubeLine(e *corev1.Event) string {
	return e.InvolvedObject.Kind + " " + e.InvolvedObject.Name + ": " + e.Message
}
