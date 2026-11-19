package kubectl

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func TestGetDaemonSetsTable(t *testing.T) {
	f := newCluster(t)
	f.wardenSet(t, appsv1.DaemonSetStatus{DesiredNumberScheduled: 3, CurrentNumberScheduled: 3, NumberReady: 2, UpdatedNumberScheduled: 3, NumberAvailable: 2, ObservedGeneration: 1})
	f.clk.Advance(90)
	for _, cmd := range []string{"kubectl get ds", "kubectl get daemonsets", "kubectl get daemonset.apps warden"} {
		res := run(f, cmd)
		lines := strings.Split(strings.TrimRight(res.Output, "\n"), "\n")
		if !res.OK || len(lines) != 2 || strings.Join(strings.Fields(lines[0]), " ") != "NAME DESIRED CURRENT READY UP-TO-DATE AVAILABLE NODE SELECTOR AGE" {
			t.Fatalf("%s:\n%s", cmd, res.Output)
		}
		if got := strings.Join(strings.Fields(lines[1]), " "); got != "warden 3 3 2 3 2 trade=couriers 90s" {
			t.Fatalf("%s row = %q", cmd, got)
		}
	}
	wide := run(f, "kubectl get ds -o wide").Output
	if !strings.Contains(wide, "CONTAINERS") || !strings.Contains(wide, "warden:1.0") || !strings.Contains(wide, "app=warden") {
		t.Fatalf("wide:\n%s", wide)
	}
}

func TestRolloutStatusDaemonSet(t *testing.T) {
	f := newCluster(t)
	ds := f.wardenSet(t, appsv1.DaemonSetStatus{DesiredNumberScheduled: 3, UpdatedNumberScheduled: 1, NumberAvailable: 3, ObservedGeneration: 1})
	if out := mustRun(t, f, "rollout status ds/warden"); out != "Waiting for daemon set \"warden\" rollout to finish: 1 out of 3 new pods have been updated...\n" {
		t.Fatalf("out = %q", out)
	}
	ds.Status.UpdatedNumberScheduled, ds.Status.NumberAvailable = 3, 2
	ds, _ = mustStatus(t, f, ds)
	if out := mustRun(t, f, "rollout status daemonset warden"); out != "Waiting for daemon set \"warden\" rollout to finish: 2 of 3 updated pods are available...\n" {
		t.Fatalf("out = %q", out)
	}
	ds.Status.NumberAvailable = 3
	ds, _ = mustStatus(t, f, ds)
	if out := mustRun(t, f, "rollout status ds/warden"); out != "daemon set \"warden\" successfully rolled out\n" {
		t.Fatalf("out = %q", out)
	}
	ds.Status.ObservedGeneration = 0
	_, _ = mustStatus(t, f, ds)
	if out := mustRun(t, f, "rollout status ds/warden"); out != "Waiting for daemon set spec update to be observed...\n" {
		t.Fatalf("out = %q", out)
	}
}

func TestDescribeDaemonSet(t *testing.T) {
	f := newCluster(t)
	ds := f.wardenSet(t, appsv1.DaemonSetStatus{DesiredNumberScheduled: 2, CurrentNumberScheduled: 2, UpdatedNumberScheduled: 2, NumberAvailable: 1, NumberReady: 1})
	yes := true
	pod := f.runningPod(t, "warden-abcde", 0).DeepCopy()
	pod.Labels = map[string]string{"app": "warden"}
	pod.OwnerReferences = []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "DaemonSet", Name: "warden", UID: ds.UID, Controller: &yes}}
	_, _ = f.st.Update(pod)
	out := mustRun(t, f, "describe ds warden")
	for _, want := range []string{
		"Name:           warden\n",
		"Selector:       app=warden\n",
		"Node-Selector:  trade=couriers\n",
		"Annotations:    deprecated.daemonset.template.generation: 1\n",
		"Desired Number of Nodes Scheduled: 2\n",
		"Current Number of Nodes Scheduled: 2\n",
		"Number of Nodes Scheduled with Up-to-date Pods: 2\n",
		"Number of Nodes Scheduled with Available Pods: 1\n",
		"Number of Nodes Misscheduled: 0\n",
		"Pods Status:  1 Running / 0 Waiting / 0 Succeeded / 0 Failed\n",
		"Pod Template:\n  Labels:  app=warden\n  Containers:\n   warden:\n    Image:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("describe lacks %q:\n%s", want, out)
		}
	}
	if _, ok := store.Get[*appsv1.DaemonSet](f.st, store.KindDaemonSet, "default", "warden"); !ok {
		t.Fatal("daemonset missing")
	}
}
