package kubectl

import (
	"strings"
	"testing"
)

func TestGetPodsTable(t *testing.T) {
	f := newCluster(t)
	f.runningPod(t, "lantern-abcde", 0)
	f.clk.Advance(75)
	res := run(f, "kubectl get pods")
	if !res.OK || res.Verb != "get" || res.Resource != "pods" {
		t.Fatalf("res = %+v", res)
	}
	lines := strings.Split(strings.TrimRight(res.Output, "\n"), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "NAME") || !strings.Contains(lines[0], "RESTARTS") {
		t.Fatalf("output:\n%s", res.Output)
	}
	fields := strings.Fields(lines[1])
	if fields[0] != "lantern-abcde" || fields[1] != "1/1" || fields[2] != "Running" || fields[3] != "0" || fields[4] != "75s" {
		t.Fatalf("row = %v", fields)
	}
}

func TestGetEmptyAndNotFound(t *testing.T) {
	f := newCluster(t)
	if res := run(f, "get po"); res.Output != "No resources found in default namespace.\n" {
		t.Fatalf("output = %q", res.Output)
	}
	res := run(f, "get pod ghost")
	if res.OK || res.Output != "Error from server (NotFound): pods \"ghost\" not found\n" {
		t.Fatalf("res = %+v", res)
	}
	res = run(f, "get cronjobs")
	if res.OK || !strings.Contains(res.Output, "doesn't have a resource type \"cronjobs\"") {
		t.Fatalf("res = %+v", res)
	}
}

func TestGetWideYamlNodesAndLabels(t *testing.T) {
	f := newCluster(t)
	f.runningPod(t, "lantern-abcde", 3)
	wide := run(f, "get pods -o wide")
	if !strings.Contains(wide.Output, "10.244.0.2") || !strings.Contains(wide.Output, "lone-tower") {
		t.Fatalf("wide:\n%s", wide.Output)
	}
	y := run(f, "get pod lantern-abcde -o yaml")
	if !strings.Contains(y.Output, "kind: Pod") || !strings.Contains(y.Output, "apiVersion: v1") {
		t.Fatalf("yaml:\n%s", y.Output)
	}
	n := run(f, "get nodes")
	if !strings.Contains(n.Output, "control-plane") || !strings.Contains(n.Output, "v1.37.0") {
		t.Fatalf("nodes:\n%s", n.Output)
	}
	l := run(f, "get pods --show-labels -l app=lantern")
	if !strings.Contains(l.Output, "app=lantern") {
		t.Fatalf("labels:\n%s", l.Output)
	}
	none := run(f, "get pods -l app=nope")
	if !strings.Contains(none.Output, "No resources found") {
		t.Fatalf("selector:\n%s", none.Output)
	}
	names := run(f, "get pods -o name")
	if names.Output != "pod/lantern-abcde\n" {
		t.Fatalf("names = %q", names.Output)
	}
}

func TestDescribePodAndNode(t *testing.T) {
	f := newCluster(t)
	p := f.runningPod(t, "lantern-abcde", 2)
	d := run(f, "describe pod lantern-abcde")
	for _, want := range []string{"Name:", "lantern-abcde", "Node:", "lone-tower", "Restart Count:  2", "Requests:", "cpu:  100m", "Events:"} {
		if !strings.Contains(d.Output, want) {
			t.Fatalf("describe missing %q:\n%s", want, d.Output)
		}
	}
	_ = p
	n := run(f, "describe node lone-tower")
	for _, want := range []string{"Taints:", "node-role.kubernetes.io/control-plane:NoSchedule", "Allocatable:", "Non-terminated Pods:", "Allocated resources:"} {
		if !strings.Contains(n.Output, want) {
			t.Fatalf("describe node missing %q:\n%s", want, n.Output)
		}
	}
	if n.Resource != "node" {
		t.Fatalf("resource = %q", n.Resource)
	}
}

func TestLogsAndPrevious(t *testing.T) {
	f := newCluster(t)
	p := f.runningPod(t, "lantern-abcde", 1)
	f.logs[p.UID] = []string{"line one", "line two", "line three"}
	res := run(f, "logs lantern-abcde --tail=2")
	if res.Output != "line two\nline three\n" {
		t.Fatalf("logs = %q", res.Output)
	}
	res = run(f, "logs lantern-abcde --previous")
	if res.OK || !strings.Contains(res.Output, "previous terminated container") {
		t.Fatalf("previous = %+v", res)
	}
	f.prev[p.UID] = []string{"panic: scroll key LANTERN_WICK missing"}
	if res := run(f, "logs -p lantern-abcde"); !strings.Contains(res.Output, "LANTERN_WICK") {
		t.Fatalf("previous = %+v", res)
	}
}

func TestTopWithoutMetricsSaysKubectlsOwnWords(t *testing.T) {
	f := newCluster(t)
	p := f.runningPod(t, "lantern-abcde", 0)
	if res := run(f, "top pods"); res.OK || res.Output != "error: Metrics API not available\n" {
		t.Fatalf("res = %+v", res)
	}
	f.unlocks["augury"] = true
	f.usage[string(p.UID)] = Usage{CPUMilli: 42, MemBytes: 50 * 1024 * 1024}
	f.usage["node:lone-tower"] = Usage{CPUMilli: 500, MemBytes: 1024 * 1024 * 1024}
	pods := run(f, "top pods")
	if !strings.Contains(pods.Output, "42m") || !strings.Contains(pods.Output, "50Mi") {
		t.Fatalf("top pods:\n%s", pods.Output)
	}
	nodes := run(f, "top nodes")
	if !strings.Contains(nodes.Output, "500m") || !strings.Contains(nodes.Output, "25%") {
		t.Fatalf("top nodes:\n%s", nodes.Output)
	}
}

func TestTokenize(t *testing.T) {
	got, err := tokenize(`kubectl label pod "my pod" note='a b'`)
	if err != nil || len(got) != 5 || got[3] != "my pod" || got[4] != "note=a b" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := tokenize(`get "unterminated`); err == nil {
		t.Fatal("expected error")
	}
}

func TestAReachVerbSaysTheSimulatedClusterRunsNoContainer(t *testing.T) {
	f := newCluster(t)
	for _, verb := range []string{"exec", "attach", "port-forward", "cp", "debug"} {
		want := "error: kubectl " + verb + " reaches into a running container: this simulated cluster runs none\n"
		if res := run(f, verb+" web-1 -- sh"); res.OK || res.Output != want {
			t.Errorf("%s answered %q", verb, res.Output)
		}
	}
}

func TestExplainSaysWhatAnInitContainerIsInTheRealm(t *testing.T) {
	f := newCluster(t)
	res := run(f, "explain pod.spec.initContainers")
	if !strings.Contains(res.Output, "IN THE REALM:") || strings.Contains(res.Output, "GRIMOIRE") || !strings.Contains(res.Output, "before the hut's workers") || strings.Contains(res.Output, "later Age") {
		t.Fatalf("explain initContainers:\n%s", res.Output)
	}
	res = run(f, "explain pod.spec.initContainers.restartPolicy")
	if !strings.Contains(strings.Join(strings.Fields(res.Output), " "), "Without it, it is a Squire: it runs to completion before the workers start.") {
		t.Fatalf("explain restartPolicy:\n%s", res.Output)
	}
}
