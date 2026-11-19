package write

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/clientcmd"
)

type kindRig struct {
	ns         string
	t          *testing.T
	kubeconfig string
	kubectl    string
	gate       *Gate
}

func kindGate(t *testing.T, level Level) *kindRig {
	t.Helper()
	path, kubectl := os.Getenv("KUBERNITES_WRITES_KUBECONFIG"), os.Getenv("KUBERNITES_WRITES_KUBECTL")
	if path == "" || kubectl == "" {
		t.Skip("KUBERNITES_WRITES_KUBECONFIG and KUBERNITES_WRITES_KUBECTL name a kind cluster made in this VM")
	}
	raw, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := clientcmd.NewNonInteractiveClientConfig(*raw, raw.CurrentContext, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		t.Fatal(err)
	}
	c := raw.Contexts[raw.CurrentContext]
	g, err := New(Choice{Context: Context{Name: raw.CurrentContext, Server: cfg.Host, User: c.AuthInfo}, Level: level, Classes: AllClasses})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Client(cfg); err != nil {
		t.Fatal(err)
	}
	return &kindRig{ns: kindNS, t: t, kubeconfig: path, kubectl: kubectl, gate: g}
}

func (k *kindRig) run(args ...string) string {
	k.t.Helper()
	out, err := exec.Command(k.kubectl, append([]string{"--kubeconfig", k.kubeconfig}, args...)...).CombinedOutput()
	if err != nil {
		k.t.Fatalf("kubectl %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func (k *kindRig) apply(manifest string) {
	k.t.Helper()
	_, _ = exec.Command(k.kubectl, "--kubeconfig", k.kubeconfig, "wait", "--for=delete", "namespace/"+k.ns, "--timeout=180s").CombinedOutput()
	cmd := exec.Command(k.kubectl, "--kubeconfig", k.kubeconfig, "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	if out, err := cmd.CombinedOutput(); err != nil {
		k.t.Fatalf("apply: %v\n%s", err, out)
	}
}

func (k *kindRig) get(into any, args ...string) {
	k.t.Helper()
	if err := json.Unmarshal([]byte(k.run(append(args, "-o", "json")...)), into); err != nil {
		k.t.Fatal(err)
	}
}

func (k *kindRig) send(o Order) Answer {
	k.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return k.gate.Send(ctx, o)
}

func (k *kindRig) preview(o Order) (Answer, []Field) {
	k.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return k.gate.Preview(ctx, o)
}

const kindNS = "orders"

const kindManifest = `apiVersion: v1
kind: Namespace
metadata:
  name: orders
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: orders
spec:
  replicas: 1
  selector:
    matchLabels: {app: web}
  template:
    metadata:
      labels: {app: web}
    spec:
      terminationGracePeriodSeconds: 2
      containers:
      - name: web
        image: busybox:1.37
        command: ["sh", "-c", "sleep 3600"]
        resources:
          requests: {cpu: 10m, memory: 16Mi}
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: web
  namespace: orders
spec:
  scaleTargetRef: {apiVersion: apps/v1, kind: Deployment, name: web}
  minReplicas: 1
  maxReplicas: 3
  metrics:
  - type: Resource
    resource: {name: cpu, target: {type: Utilization, averageUtilization: 80}}
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: nightly-report
  namespace: orders
spec:
  schedule: "0 3 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: Never
          containers:
          - name: report
            image: busybox:1.37
            command: ["sh", "-c", "date"]
`

func (k *kindRig) deployment() *appsv1.Deployment {
	var d appsv1.Deployment
	k.get(&d, "-n", k.ns, "get", "deployment", "web")
	return &d
}

func settled[T any, P interface {
	*T
	GetResourceVersion() string
	GetGeneration() int64
}](k *kindRig, observed func(P) int64, args ...string) P {
	k.t.Helper()
	last, same := "", 0
	for range 90 {
		p := P(new(T))
		k.get(p, args...)
		v := p.GetResourceVersion()
		switch {
		case observed != nil && observed(p) != p.GetGeneration():
			last, same = "", 0
		case v == last:
			same++
		default:
			last, same = v, 0
		}
		if same >= 2 {
			return p
		}
		time.Sleep(time.Second)
	}
	k.t.Fatalf("%v never settled", args)
	return nil
}

func (k *kindRig) settledDeployment() *appsv1.Deployment {
	k.t.Helper()
	return settled(k, func(d *appsv1.Deployment) int64 { return d.Status.ObservedGeneration }, "-n", k.ns, "get", "deployment", "web")
}

func (k *kindRig) waitRollout() {
	k.run("-n", k.ns, "rollout", "status", "deployment/web", "--timeout=120s")
}

func (k *kindRig) settledPod(name string) *corev1.Pod {
	k.t.Helper()
	last, same := "", 0
	for range 60 {
		p := &corev1.Pod{}
		k.get(p, "-n", k.ns, "get", "pod", name)
		if p.ResourceVersion == last {
			same++
		} else {
			last, same = p.ResourceVersion, 0
		}
		if same >= 3 {
			return p
		}
		time.Sleep(time.Second)
	}
	k.t.Fatalf("pod %s never settled", name)
	return nil
}

func (k *kindRig) firstPod() *corev1.Pod {
	k.t.Helper()
	for range 120 {
		var list corev1.PodList
		k.get(&list, "-n", k.ns, "get", "pods", "-l", "app=web")
		for i := range list.Items {
			p := &list.Items[i]
			if p.DeletionTimestamp != nil || p.Status.Phase != corev1.PodRunning {
				continue
			}
			for _, c := range p.Status.Conditions {
				if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
					return k.settledPod(p.Name)
				}
			}
		}
		time.Sleep(time.Second)
	}
	k.t.Fatalf("no running web pod:\n%s", k.run("-n", k.ns, "get", "pods", "-o", "wide"))
	return nil
}

func TestEveryOrderOnKindSendsItsRequestAndAStaleOneGets409(t *testing.T) {
	k := kindGate(t, Seamless)
	k.apply(kindManifest)
	defer k.run("delete", "namespace", kindNS, "--wait=false")
	k.waitRollout()

	d := k.settledDeployment()
	stale := d.DeepCopy()
	o, _ := Scale("Deployment", d, 1, 2)
	if a := k.send(o); !a.OK() {
		t.Fatalf("scale: %+v", a)
	}
	if again := k.send(o); again.Code != 409 {
		t.Fatalf("a scale drawn from a stale version: %d %s", again.Code, again.Text())
	}
	d = k.settledDeployment()
	o, _ = SetImage("Deployment", d, d.Spec.Template.Spec.Containers, "web", "busybox:1.36")
	if a, fields := k.preview(o); !a.OK() || fields[0].After != "busybox:1.36" {
		t.Fatalf("set image preview: %+v %+v", a, fields)
	}
	if a := k.send(o); !a.OK() {
		t.Fatalf("set image: %+v", a)
	}
	k.waitRollout()
	d = k.settledDeployment()
	var rs appsv1.ReplicaSetList
	k.get(&rs, "-n", kindNS, "get", "replicasets", "-l", "app=web")
	o, err := Rollback(d, rs.Items, 0)
	if err != nil {
		t.Fatal(err)
	}
	old := d.DeepCopy()
	old.ResourceVersion = stale.ResourceVersion
	staleBack, err := Rollback(old, rs.Items, 0)
	if err != nil {
		t.Fatal(err)
	}
	if a := k.send(staleBack); a.Code != 409 {
		t.Fatalf("a JSON-patch roll back from a stale version: %d %s", a.Code, a.Text())
	}
	if a, fields := k.preview(o); !a.OK() || fields[1].After != "busybox:1.37" {
		t.Fatalf("roll back preview: %+v %+v", a, fields)
	}
	if a := k.send(o); !a.OK() {
		t.Fatalf("roll back: %+v", a)
	}
	k.waitRollout()
	d = k.settledDeployment()
	o, _ = SetResources("Deployment", d, d.Spec.Template.Spec.Containers, "web", corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("20m")}, corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("64Mi")})
	if a := k.send(o); !a.OK() {
		t.Fatalf("set resources: %+v", a)
	}
	k.waitRollout()
	d = k.settledDeployment()
	o, _ = Restart("Deployment", d, d.Spec.Paused, d.Spec.Template.Annotations, time.Now())
	if a := k.send(o); !a.OK() {
		t.Fatalf("restart: %+v", a)
	}
	k.waitRollout()
	d = k.settledDeployment()
	o, _ = Pause(d, true)
	if a := k.send(o); !a.OK() {
		t.Fatalf("pause: %+v", a)
	}
	d = k.settledDeployment()
	if !d.Spec.Paused {
		t.Fatal("not paused")
	}
	if _, err := Restart("Deployment", d, d.Spec.Paused, nil, time.Now()); err == nil || err.Error() != PausedRestart {
		t.Fatalf("restart of a paused deployment: %v", err)
	}
	o, _ = Pause(d, false)
	if a := k.send(o); !a.OK() {
		t.Fatalf("resume: %+v", a)
	}
	if d = k.deployment(); d.Spec.Paused {
		t.Fatal("still paused")
	}
	k.get(d, "-n", k.ns, "get", "deployment", "web", "--show-managed-fields")
	managers := map[string]bool{}
	for _, m := range d.ManagedFields {
		managers[m.Manager] = true
	}
	if !managers[FieldManager] {
		t.Fatalf("the cluster does not name Kubernites: %v", managers)
	}
	h := settled[autoscalingv2.HorizontalPodAutoscaler](k, nil, "-n", kindNS, "get", "hpa", "web")
	o, _ = Retune(h, 1, 4)
	if a := k.send(o); !a.OK() {
		t.Fatalf("retune: %+v", a)
	}
	cj := settled[batchv1.CronJob](k, nil, "-n", kindNS, "get", "cronjob", "nightly-report")
	o, _ = Trigger(cj, "x7k")
	if a, fields := k.preview(o); !a.OK() || fields[0].After != "nightly-report-manual-x7k" {
		t.Fatalf("trigger preview: %+v %+v", a, fields)
	}
	if a := k.send(o); !a.OK() {
		t.Fatalf("trigger: %+v", a)
	}
	k.run("-n", kindNS, "get", "job", "nightly-report-manual-x7k")

	p := k.firstPod()
	stalePod := p.DeepCopy()
	stalePod.ResourceVersion = "1"
	o, _ = Delete(RefOf(Pods, "Pod", stalePod), Background, false)
	if a := k.send(o); a.Code != 409 {
		t.Fatalf("a delete with a stale precondition: %d %s", a.Code, a.Text())
	}
	o, _ = Evict(stalePod)
	if a := k.send(o); a.Code != 409 {
		t.Fatalf("an eviction with a stale precondition: %d %s", a.Code, a.Text())
	}
	o, _ = Evict(p)
	if a, _ := k.preview(o); !a.OK() {
		t.Fatalf("eviction preview: %+v", a)
	}
	if a := k.send(o); !a.OK() {
		t.Fatalf("eviction: %+v", a)
	}
	p = k.firstPod()
	o, _ = Kill(p)
	if a := k.send(o); !a.OK() {
		t.Fatalf("kill: %+v", a)
	}
	k.run("-n", k.ns, "run", "solo", "--image=busybox:1.37", "--restart=Never", "--command", "--", "sleep", "3600")
	k.run("-n", k.ns, "wait", "--for=condition=Ready", "pod/solo", "--timeout=120s")
	p = k.settledPod("solo")
	o, _ = Delete(RefOf(Pods, "Pod", p), Foreground, false)
	if a, fields := k.preview(o); !a.OK() || fields[0].After == "" {
		t.Fatalf("delete preview: %+v %+v", a, fields)
	}
	if after := k.settledPod("solo"); after.DeletionTimestamp != nil || after.ResourceVersion != p.ResourceVersion {
		t.Fatalf("the preview of a delete changed the Pod: %v %s -> %s", after.DeletionTimestamp, p.ResourceVersion, after.ResourceVersion)
	}
	if a := k.send(o); !a.OK() {
		t.Fatalf("delete: %+v", a)
	}
	n := settled[corev1.Node](k, nil, "get", "node", "kubernites-worker")
	o, _ = Cordon(n, true)
	if a := k.send(o); !a.OK() {
		t.Fatalf("cordon: %+v", a)
	}
	n = settled[corev1.Node](k, nil, "get", "node", "kubernites-worker")
	o, _ = Cordon(n, false)
	if a := k.send(o); !a.OK() {
		t.Fatalf("uncordon: %+v", a)
	}
	d = k.settledDeployment()
	o, _ = Delete(RefOf(Deployments, "Deployment", d), Background, false)
	if a := k.send(o); !a.OK() {
		t.Fatalf("delete deployment: %+v", a)
	}
}

func TestAtWritesOnKindAnUnconfirmedOrderNeverReachesTheServer(t *testing.T) {
	k := kindGate(t, Writes)
	k.ns = "confirm"
	k.apply(strings.ReplaceAll(kindManifest, kindNS, k.ns))
	defer k.run("delete", "namespace", k.ns, "--wait=false")
	k.waitRollout()
	d := k.settledDeployment()
	o, _ := Scale("Deployment", d, 1, 3)
	if a, fields := k.preview(o); !a.OK() || fields[0].After != "3" {
		t.Fatalf("preview: %+v %+v", a, fields)
	}
	if a := k.send(o); a.Err == nil {
		t.Fatalf("an unconfirmed scale was sent: %+v", a)
	}
	if got := k.deployment(); *got.Spec.Replicas != 1 {
		t.Fatalf("the preview changed the cluster: %d replicas", *got.Spec.Replicas)
	}
	k.gate.Issue(o.Request)
	if a := k.send(o); !a.OK() {
		t.Fatalf("the confirmed scale: %+v", a)
	}
	if got := k.deployment(); *got.Spec.Replicas != 3 {
		t.Fatalf("%d replicas after CONFIRM", *got.Spec.Replicas)
	}
}
