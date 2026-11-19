package write

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var update = flag.Bool("update", false, "write the golden files")

func meta(ns, name, uid, rv string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Namespace: ns, Name: name, UID: types.UID("uid-" + uid), ResourceVersion: rv}
}

func webTemplate(image, rev string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web", hashLabel: "h" + rev}},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: image, Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("64Mi")},
			Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("128Mi")},
		}}}},
	}
}

func fixtureDeployment() *appsv1.Deployment {
	d := &appsv1.Deployment{ObjectMeta: meta("shop", "web", "web", "4711")}
	d.Annotations = map[string]string{revisionAnnotation: "3", "kubectl.kubernetes.io/last-applied-configuration": "{}", "team": "shop"}
	replicas := int32(2)
	d.Spec.Replicas = &replicas
	d.Spec.Template = webTemplate("shop/web:1.4", "3")
	d.Spec.Template.Annotations = map[string]string{restartedAt: "2026-11-01T08:00:00Z"}
	return d
}

func fixtureReplicaSets() []appsv1.ReplicaSet {
	rs := func(rev, image string) appsv1.ReplicaSet {
		r := appsv1.ReplicaSet{ObjectMeta: meta("shop", "web-h"+rev, "rs"+rev, "90"+rev)}
		r.Annotations = map[string]string{revisionAnnotation: rev, "deployment.kubernetes.io/desired-replicas": "2", "release": "r" + rev}
		r.Spec.Template = webTemplate(image, rev)
		return r
	}
	return []appsv1.ReplicaSet{rs("1", "shop/web:1.2"), rs("3", "shop/web:1.4"), rs("2", "shop/web:1.3")}
}

func fixturePod() *corev1.Pod {
	return &corev1.Pod{ObjectMeta: meta("shop", "web-h3-abcde", "pod", "5150")}
}

func fixtureNode(cordoned bool) *corev1.Node {
	n := &corev1.Node{ObjectMeta: meta("", "kind-worker", "node", "777")}
	n.Spec.Unschedulable = cordoned
	return n
}

func fixtureHPA() *autoscalingv2.HorizontalPodAutoscaler {
	min := int32(1)
	return &autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: meta("shop", "web", "hpa", "321"), Spec: autoscalingv2.HorizontalPodAutoscalerSpec{MinReplicas: &min, MaxReplicas: 4}}
}

func fixtureCronJob() *batchv1.CronJob {
	cj := &batchv1.CronJob{ObjectMeta: meta("batch", "nightly-report", "cron", "88")}
	cj.Spec.JobTemplate.Labels = map[string]string{"app": "report"}
	cj.Spec.JobTemplate.Annotations = map[string]string{"owner": "ops"}
	cj.Spec.JobTemplate.Spec.Template.Spec = corev1.PodSpec{RestartPolicy: corev1.RestartPolicyOnFailure, Containers: []corev1.Container{{Name: "report", Image: "busybox:1.37", Command: []string{"sh", "-c", "date"}}}}
	return cj
}

func golden(o Order) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", o.Request.Method, o.Request.Path)
	if q := o.Request.Query.Encode(); q != "" {
		fmt.Fprintf(&b, "?%s", q)
	}
	fmt.Fprintf(&b, "\nverb: %s\ncontent-type: %s\n", o.Verb, o.ContentType)
	access, _ := json.Marshal(o.Access)
	fmt.Fprintf(&b, "access review: %s\nclasses: %v\n", access, o.Classes)
	for _, f := range o.Fields {
		fmt.Fprintf(&b, "preview field: %s %s (was %q)\n", f.Label, strings.Join(f.Path, "."), f.Before)
	}
	fmt.Fprintf(&b, "kubectl: %s\nbody:\n", o.Command)
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, o.Request.Body, "", "  "); err != nil {
		pretty.Write(o.Request.Body)
	}
	b.Write(pretty.Bytes())
	b.WriteString("\n")
	return b.String()
}

func everyOrder(t *testing.T) map[string]Order {
	t.Helper()
	d := fixtureDeployment()
	must := func(o Order, err error) Order {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	at := time.Date(2026, 11, 15, 21, 4, 5, 0, time.UTC)
	reqs := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m")}
	lims := corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")}
	sts := &appsv1.StatefulSet{ObjectMeta: meta("shop", "ledger", "sts", "64")}
	return map[string]Order{
		"crew-scale":             must(Scale("Deployment", d, 2, 5)),
		"crew-scale-statefulset": must(Scale("StatefulSet", sts, 3, 0)),
		"crew-restart":           must(Restart("Deployment", d, false, d.Spec.Template.Annotations, at)),
		"crew-pause":             must(Pause(d, true)),
		"crew-resume":            must(Pause(&appsv1.Deployment{ObjectMeta: d.ObjectMeta, Spec: appsv1.DeploymentSpec{Paused: true}}, false)),
		"crew-rollback":          must(Rollback(d, fixtureReplicaSets(), 0)),
		"crew-set-image":         must(SetImage("Deployment", d, d.Spec.Template.Spec.Containers, "web", "shop/web:1.5")),
		"crew-set-resources":     must(SetResources("Deployment", d, d.Spec.Template.Spec.Containers, "web", reqs, lims)),
		"horn-retune":            must(Retune(fixtureHPA(), 2, 6)),
		"hut-remove":             must(Delete(RefOf(Pods, "Pod", fixturePod()), Background, false)),
		"hut-remove-foreground":  must(Delete(RefOf(Pods, "Pod", fixturePod()), Foreground, false)),
		"hut-remove-orphan":      must(Delete(RefOf(Pods, "Pod", fixturePod()), Orphan, false)),
		"hut-remove-none-force":  must(Delete(RefOf(Pods, "Pod", fixturePod()), PropagationNone, true)),
		"hut-kill":               must(Kill(fixturePod())),
		"keep-cordon":            must(Cordon(fixtureNode(false), true)),
		"keep-uncordon":          must(Cordon(fixtureNode(true), false)),
		"keep-drain-evict":       must(Evict(fixturePod())),
		"object-delete":          must(Delete(RefOf(Deployments, "Deployment", d), Background, false)),
		"object-delete-node":     must(Delete(RefOf(Nodes, "Node", fixtureNode(false)), Background, false)),
		"cronjob-trigger":        must(Trigger(fixtureCronJob(), "x7k")),
	}
}

func TestEachOrderSendsExactlyItsGoldenRequest(t *testing.T) {
	for name, o := range everyOrder(t) {
		path := filepath.Join("testdata", "orders", name+".golden")
		got := golden(o)
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v (run with -update)", name, err)
		}
		if got != string(want) {
			t.Errorf("%s:\n%s\nwant\n%s", name, got, want)
		}
	}
}

func TestEveryOrderCarriesItsPreconditionAndTheGamesFieldManager(t *testing.T) {
	for name, o := range everyOrder(t) {
		body := string(o.Request.Body)
		switch o.Request.Method {
		case "PATCH":
			if !strings.Contains(body, `"resourceVersion":"`+o.Target.ResourceVersion+`"`) && !strings.Contains(body, `"path":"/metadata/resourceVersion","value":"`+o.Target.ResourceVersion+`"`) {
				t.Errorf("%s: the patch does not carry the version it was drawn from: %s", name, body)
			}
			if o.Request.Query.Get("fieldManager") != FieldManager {
				t.Errorf("%s: fieldManager %q", name, o.Request.Query.Get("fieldManager"))
			}
		case "DELETE":
			if !strings.Contains(body, `"preconditions":{"resourceVersion":"`+o.Target.ResourceVersion+`","uid":"`+o.Target.UID+`"}`) {
				t.Errorf("%s: the delete carries no uid and version preconditions: %s", name, body)
			}
		case "POST":
			if o.Verb == OrderEvict && !strings.Contains(body, `"preconditions":{"resourceVersion":"`+o.Target.ResourceVersion+`","uid":"`+o.Target.UID+`"}`) {
				t.Errorf("%s: the eviction carries no preconditions: %s", name, body)
			}
			if o.Verb == OrderTrigger && o.Request.Query.Get("fieldManager") != FieldManager {
				t.Errorf("%s: fieldManager %q", name, o.Request.Query.Get("fieldManager"))
			}
		}
		if o.Access.Verb == "" || o.Access.Resource == "" {
			t.Errorf("%s: no access review attributes", name)
		}
	}
}

func TestOrdersRefuseWhatKubectlRefuses(t *testing.T) {
	d := fixtureDeployment()
	d.Spec.Paused = true
	if _, err := Restart("Deployment", d, true, nil, time.Now()); err == nil || err.Error() != PausedRestart {
		t.Errorf("restart of a paused deployment: %v", err)
	}
	if _, err := Rollback(d, fixtureReplicaSets(), 0); err == nil || err.Error() != PausedRollback {
		t.Errorf("roll back of a paused deployment: %v", err)
	}
	d.Spec.Paused = false
	d.Spec.Template.Annotations = nil
	if _, err := Rollback(d, fixtureReplicaSets(), 3); err == nil || !strings.Contains(err.Error(), "already matches revision 3") {
		t.Errorf("roll back to the current revision: %v", err)
	}
	if _, err := Rollback(d, fixtureReplicaSets()[:1], 0); err == nil {
		t.Error("roll back with one revision")
	}
	if _, err := Scale("DaemonSet", d, 1, 2); err == nil {
		t.Error("a DaemonSet scaled")
	}
	if _, err := Restart("StatefulSet", d, true, nil, time.Now()); err != nil {
		t.Errorf("only a Deployment is refused for its pause: %v", err)
	}
	if _, err := Delete(Ref{GVR: Pods, Name: "x"}, Background, false); err == nil {
		t.Error("a delete without preconditions")
	}
	if _, err := Cordon(fixtureNode(true), true); err == nil {
		t.Error("cordoned twice")
	}
	if _, err := Retune(fixtureHPA(), 3, 2); err == nil {
		t.Error("a Horn tuned to an empty range")
	}
	if got := TriggeredJobName(strings.Repeat("a", 50), "x7k"); got != strings.Repeat("a", 42)+"-manual-x7k" {
		t.Errorf("trigger name %s", got)
	}
}

func TestTheClassesOfAnOrderAreTheGates(t *testing.T) {
	orders := everyOrder(t)
	if c := orders["hut-remove-none-force"].Classes; len(c) != 1 || c[0] != Destroy {
		t.Errorf("a force delete needs Destroy, has %v", c)
	}
	if c := orders["object-delete-node"].Classes; len(c) != 1 || c[0] != Destroy {
		t.Errorf("deleting a Node needs Destroy, has %v", c)
	}
	for _, name := range []string{"crew-scale", "hut-kill", "hut-remove", "keep-drain-evict", "cronjob-trigger"} {
		if c := orders[name].Classes; len(c) != 0 {
			t.Errorf("%s needs %v", name, c)
		}
	}
}
