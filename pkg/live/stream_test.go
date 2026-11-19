package live

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func hutsAt(w *World) map[types.UID]HutPlot {
	out := map[types.UID]HutPlot{}
	for _, h := range w.Huts {
		out[h.UID] = h
	}
	return out
}

func TestAddingAndRemovingHutsOnAnEventStreamMovesNoStandingHut(t *testing.T) {
	s := kindSnapshot(t)
	r, err := OpenRecorded(s)
	if err != nil {
		t.Fatal(err)
	}
	w := r.World()
	born := s.TakenAt.Add(time.Hour)
	workers := []string{w.Keeps[1].Node, w.Keeps[2].Node}
	var added []corev1.Pod
	for i := 0; i < 100; i++ {
		p := syntheticPod(i, workers[i%2], born)
		added = append(added, p)
		stream(t, r, "pods", watchAdded, p)
		if i%7 == 0 {
			r.Pump(PumpEvery)
		}
	}
	r.Pump(PumpEvery)
	before := hutsAt(r.World())
	if len(before) != len(s.Pods)+100 {
		t.Fatalf("%d huts after 100 streamed in, %d pods before", len(before), len(s.Pods))
	}
	for i, p := range added {
		if i%3 == 1 {
			stream(t, r, "pods", watchDeleted, p)
		}
	}
	for i := 100; i < 200; i++ {
		stream(t, r, "pods", watchAdded, syntheticPod(i, workers[i%2], born))
		if i%11 == 0 {
			r.Pump(PumpEvery)
		}
	}
	r.Pump(PumpEvery)
	if r.World() != w {
		t.Fatal("huts coming and going redrew the realm")
	}
	moved, stayed := 0, 0
	for _, h := range r.World().Huts {
		if was, ok := before[h.UID]; ok {
			stayed++
			if was.X != h.X || was.Y != h.Y || was.Placed != h.Placed {
				moved++
				t.Errorf("hut %s moved from %d,%d to %d,%d", h.Name, was.X, was.Y, h.X, h.Y)
			}
		}
	}
	if stayed < len(s.Pods)+60 || moved != 0 {
		t.Errorf("%d huts stayed, %d moved", stayed, moved)
	}
	if got := len(r.World().Huts); got != len(s.Pods)+100-33+100 {
		t.Errorf("%d huts after the stream", got)
	}
}

func TestOnlyANewKeepRedrawsTheRealmAndANewFiefFindsRoomWhereItStands(t *testing.T) {
	s := kindSnapshot(t)
	r, err := OpenRecorded(s)
	if err != nil {
		t.Fatal(err)
	}
	w := r.World()
	stream(t, r, "namespaces", watchAdded, corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "newfief", UID: "uid-newfief", CreationTimestamp: metav1.NewTime(s.TakenAt)}})
	p := syntheticPod(900, w.Keeps[1].Node, s.TakenAt.Add(time.Hour))
	p.Namespace = "newfief"
	stream(t, r, "pods", watchAdded, p)
	r.Pump(PumpEvery)
	if r.World() != w {
		t.Fatal("a new fief redrew the realm")
	}
	found := false
	for _, d := range r.World().Districts {
		found = found || d.Namespace == "newfief" && d.Pods == 1
	}
	if h := hutsAt(r.World())[p.UID]; !h.Placed || !found {
		t.Errorf("the new fief's hut %+v, district found %v", h, found)
	}
	drawn := r.Drawn()
	node := s.Nodes[1].DeepCopy()
	node.Name, node.UID = "kubernites-worker3", "uid-worker3"
	stream(t, r, "nodes", watchAdded, node)
	r.Pump(PumpEvery)
	if r.World() == w || len(r.World().Keeps) != len(s.Nodes)+1 || r.Drawn() == drawn {
		t.Fatalf("a new Keep did not redraw the realm: %d keeps", len(r.World().Keeps))
	}
}

func TestStreamedEventsNeverWriteIntoTheSnapshotTheRealmWasOpenedFrom(t *testing.T) {
	s := calmSnapshot()
	r, err := OpenRecorded(s)
	if err != nil {
		t.Fatal(err)
	}
	broken := calmSnapshot().Pods[0]
	waiting(&broken, reasonCrashLoop, crashWords, ended("Error", 1))
	stream(t, r, "pods", watchModified, broken)
	if s.Pods[0].Status.ContainerStatuses[0].State.Waiting != nil {
		t.Fatal("a watch event changed the caller's snapshot")
	}
}
