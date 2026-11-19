package app

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podEventAt(reason, pod string, at float64) *corev1.Event {
	return &corev1.Event{Reason: reason, Type: corev1.EventTypeWarning, InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: pod},
		Message: "0/2 nodes are available", EventTime: metav1.NewMicroTime(time.Unix(0, int64(at*float64(time.Second))))}
}

func TestTidingsCollapseEveryHutOfOneCauseIntoOneLine(t *testing.T) {
	events := []*corev1.Event{
		podEventAt("FailedScheduling", "haulers-dfcf76554-ght52", 100),
		podEventAt("FailedScheduling", "haulers-dfcf76554-rdgwl", 100.4),
		podEventAt("FailedScheduling", "haulers-dfcf76554-x7k2p", 101),
		podEventAt("FailedScheduling", "porters-5dfdd7746f-m6fc2", 101),
		podEventAt("BackOff", "haulers-dfcf76554-ght52", 101),
		podEventAt("FailedScheduling", "porter-lone-tower", 101),
		podEventAt("FailedScheduling", "porter-by-hand", 101),
		podEventAt("FailedScheduling", "haulers-dfcf76554-zz9qq", 140),
	}
	got := map[string]bool{}
	for _, td := range Tidings(events, nil, true, 0) {
		got[td.Text] = true
	}
	for _, want := range []string{
		"6 huts are Unhoused: no Keep has room",
		"Hut haulers-dfcf76554-ght52 is Guttering: its worker stops and is relit again and again",
		"Hut haulers-dfcf76554-zz9qq is Unhoused: no Keep has room",
	} {
		if !got[want] {
			t.Errorf("missing %q in %v", want, got)
		}
	}
	if len(got) != 3 {
		t.Fatalf("tidings = %v", got)
	}
}

func TestACollapsedLineSpeaksInThePlural(t *testing.T) {
	for phrase, want := range map[string]string{
		"is Unhoused: no Keep has room":               "are Unhoused: no Keep has room",
		"gets its worker":                             "get their worker",
		"is shutting down (Last Rites)":               "are shutting down (Last Rites)",
		"was evicted by a drain":                      "were evicted by a drain",
		"has finished its run":                        "have finished their run",
		"cannot get started":                          "cannot get started",
		"waits for its trade to arrive from overseas": "wait for their trade to arrive from overseas",
	} {
		if got := pluralPhrase(phrase); got != want {
			t.Errorf("pluralPhrase(%q) = %q, want %q", phrase, got, want)
		}
	}
}

func TestAScalingTidingSpeaksTheRealmsWordsInStoryMode(t *testing.T) {
	e := &corev1.Event{Reason: "ScalingReplicaSet", InvolvedObject: corev1.ObjectReference{Kind: "Deployment", Name: "porters"},
		Message: "Scaled up replica set porters-59fcdcb77f from 0 to 3"}
	if got := TidingOf(e).Text; got != "Deployment porters bids its shift porters-59fcdcb77f keep 3 huts, up from 0" {
		t.Fatalf("story scaling tiding = %q", got)
	}
	e.Message = "Scaled down replica set porters-59fcdcb77f from 3 to 1"
	if got := TidingOf(e).Text; got != "Deployment porters bids its shift porters-59fcdcb77f keep 1 hut, down from 3" {
		t.Fatalf("story scaling down = %q", got)
	}
}
