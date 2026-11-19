package kubectl

import (
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *fakeCluster) event(t *testing.T, name, kind, obj, typ, reason, msg string, ago time.Duration, newAPI bool) {
	t.Helper()
	at := f.clk.Now().Add(-ago)
	e := &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: name, Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{Kind: kind, Namespace: "default", Name: obj},
		Type:           typ, Reason: reason, Message: msg,
	}
	if newAPI {
		e.EventTime = metav1.NewMicroTime(at)
		e.Series = &corev1.EventSeries{Count: 4, LastObservedTime: metav1.NewMicroTime(at)}
	} else {
		e.FirstTimestamp, e.LastTimestamp, e.Count = metav1.NewTime(at), metav1.NewTime(at), 1
	}
	if _, err := f.st.Create(e); err != nil {
		t.Fatal(err)
	}
}

func TestEventsListsTheNamespaceOldestFirst(t *testing.T) {
	f := newCluster(t)
	f.clk.Advance(600)
	f.event(t, "a", "Pod", "web-1", "Warning", "BackOff", "Back-off restarting failed container", 2*time.Minute, true)
	f.event(t, "b", "Pod", "web-2", "Normal", "Pulled", "Container image already present", 5*time.Minute, false)
	res := Execute(f, "kubectl events")
	if !res.OK {
		t.Fatalf("events failed: %s", res.Output)
	}
	lines := strings.Split(strings.TrimSpace(res.Output), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "LAST SEEN") || !strings.Contains(lines[1], "Pulled") || !strings.Contains(lines[2], "BackOff") {
		t.Fatalf("events output:\n%s", res.Output)
	}
	if !strings.HasPrefix(lines[2], "2m (x4 over 2m)") || !strings.Contains(lines[2], "Pod/web-1") {
		t.Errorf("an event with only eventTime and series reads like kubectl's interval and Kind/name:\n%s", res.Output)
	}
	if all := Execute(f, "kubectl events -A").Output; !strings.HasPrefix(all, "NAMESPACE") {
		t.Errorf("events -A has no NAMESPACE column:\n%s", all)
	}
}

func TestEventsFiltersByTypeAndObject(t *testing.T) {
	f := newCluster(t)
	f.clk.Advance(600)
	f.event(t, "a", "Pod", "web-1", "Warning", "BackOff", "Back-off", time.Minute, false)
	f.event(t, "b", "Pod", "web-2", "Normal", "Pulled", "pulled", time.Minute, false)
	if out := Execute(f, "kubectl events --types=Warning").Output; strings.Contains(out, "Pulled") || !strings.Contains(out, "BackOff") {
		t.Errorf("--types=Warning:\n%s", out)
	}
	if out := Execute(f, "kubectl events --for pod/web-2").Output; strings.Contains(out, "BackOff") || !strings.Contains(out, "Pulled") {
		t.Errorf("--for pod/web-2:\n%s", out)
	}
	if out := Execute(f, "kubectl events -n quiet").Output; strings.TrimSpace(out) != "No events found in quiet namespace." {
		t.Errorf("an empty namespace:\n%s", out)
	}
}
