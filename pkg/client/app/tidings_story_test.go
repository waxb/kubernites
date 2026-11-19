package app

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func rawEvent(kind, name, reason, message string) *corev1.Event {
	return &corev1.Event{Reason: reason, Message: message, InvolvedObject: corev1.ObjectReference{Kind: kind, Name: name}}
}

func TestRawControllerTidingsSpeakTheRealmsWordsInStoryMode(t *testing.T) {
	for _, c := range []struct {
		e     *corev1.Event
		story string
	}{
		{rawEvent("Node", "stone-keep-1", "NodeHasInsufficientMemory", "Node stone-keep-1 status is now: NodeHasInsufficientMemory"),
			"Keep stone-keep-1 is Crowded: its housing runs short"},
		{rawEvent("Pod", "porters-7db9946f9d-gmmmt", "TaintManagerEviction", "Cancelling deletion of Pod default/porters-7db9946f9d-gmmmt"),
			"Hut porters-7db9946f9d-gmmmt is spared: its removal by the Ward is cancelled"},
		{rawEvent("HorizontalPodAutoscaler", "porters", "SuccessfulRescale", "New size: 16; reason: All metrics below target"),
			"HorizontalPodAutoscaler porters sets its crew's count to 16: they eat below the mark"},
		{rawEvent("HorizontalPodAutoscaler", "porters", "SuccessfulRescale", "New size: 20; reason: cpu resource utilization (percentage of request) above target"),
			"HorizontalPodAutoscaler porters sets its crew's count to 20: they eat past the mark"},
		{rawEvent("HorizontalPodAutoscaler", "porters", "FailedComputeMetricsReplicas", "invalid metrics (1 invalid out of 1), first error is: failed to get cpu resource metric value"),
			"HorizontalPodAutoscaler porters has no Omens to reckon by, so it keeps the crew as it is"},
		{rawEvent("HorizontalPodAutoscaler", "porters", "FailedGetResourceMetric", "failed to get cpu utilization: unable to get metrics for resource cpu"),
			"HorizontalPodAutoscaler porters cannot read the Augur's Omens"},
	} {
		if got := TidingOf(c.e).Text; got != c.story {
			t.Errorf("story tiding = %q, want %q", got, c.story)
		}
		ui.SetNamer(func(s string) string { return s })
		if got, want := TidingOf(c.e).Text, c.e.InvolvedObject.Kind+" "+c.e.InvolvedObject.Name+": "+c.e.Message; got != want {
			t.Errorf("Kubernetes tiding = %q, want %q", got, want)
		}
		ui.SetNamer(nil)
	}
}

func TestACollapsedRawTidingSpeaksThePluralInStoryMode(t *testing.T) {
	events := []*corev1.Event{
		rawEvent("Pod", "porters-7db9946f9d-gmmmt", "TaintManagerEviction", "Cancelling deletion of Pod default/porters-7db9946f9d-gmmmt"),
		rawEvent("Pod", "porters-7db9946f9d-kxzzq", "TaintManagerEviction", "Cancelling deletion of Pod default/porters-7db9946f9d-kxzzq"),
	}
	got := eventTidings(events)
	if len(got) != 1 || got[0].Text != "2 porter huts are spared: their removal by the Ward is cancelled" {
		t.Fatalf("collapsed = %+v", got)
	}
}
