package kubectl

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func upstreamShorten(s string, maxLength int) string {
	if len(s) > maxLength {
		return s[:maxLength] + "..."
	}
	return s
}

var upstreamEscaper = strings.NewReplacer("\x1b", "^[", "\r", "\\r")

func upstreamWrite(w io.Writer, format string, a ...any) {
	_, _ = io.WriteString(w, upstreamEscaper.Replace(fmt.Sprintf(format, a...)))
}

func upstreamAnnotations(w io.Writer, title string, annotations map[string]string) {
	upstreamWrite(w, "%s:\t", title)
	keys := make([]string, 0, len(annotations))
	for key := range annotations {
		if key == corev1.LastAppliedConfigAnnotation {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		upstreamWrite(w, "%s", fmt.Sprintln("<none>"))
		return
	}
	sort.Strings(keys)
	indent := "\t"
	for i, key := range keys {
		if i != 0 {
			upstreamWrite(w, "%s", indent)
		}
		value := strings.TrimSuffix(annotations[key], "\n")
		if (len(value)+len(key)+2) > 140 || strings.Contains(value, "\n") {
			upstreamWrite(w, "%s:\n", key)
			for _, s := range strings.Split(value, "\n") {
				upstreamWrite(w, "%s  %s\n", indent, upstreamShorten(s, 140-2))
			}
		} else {
			upstreamWrite(w, "%s: %s\n", key, value)
		}
	}
}

func annotationCases() map[string]map[string]string {
	return map[string]map[string]string{
		"nil":                  nil,
		"empty":                {},
		"only last-applied":    {corev1.LastAppliedConfigAnnotation: "{}"},
		"sorted by byte":       {"b": "2", "a": "1\n", "B": "0", corev1.LastAppliedConfigAnnotation: "{}"},
		"exactly the limit":    {"k": strings.Repeat("x", 137)},
		"one past the limit":   {"k": strings.Repeat("x", 138)},
		"cut to the limit":     {"k": strings.Repeat("x", 139)},
		"long and short":       {"a": "short", "b": strings.Repeat("x", 150), "c": "one\ntwo"},
		"trailing blank line":  {"c": "one\n\n"},
		"only a newline":       {"c": "\n"},
		"cut inside a rune":    {"k": "a" + strings.Repeat("é", 100)},
		"a percent sign":       {"cpu": "50%", "text": "%s %d"},
		"the limit ranger":     {"kubernetes.io/limit-ranger": "LimitRanger plugin set: cpu, memory request for container porter; cpu, memory limit for container porter"},
		"a long key and value": {strings.Repeat("k", 70): strings.Repeat("v", 69)},
		"carriage returns":     {"crlf": "one\r\ntwo\r\n", "cr": "a\rb"},
		"terminal escapes":     {"esc": "\x1b[31mred\x1b[0m"},
		"a return at the cut":  {"k": strings.Repeat("x", 137) + "\r\r"},
	}
}

func TestAnnotationsAreWrittenAsUpstreamKubectlWritesThem(t *testing.T) {
	for name, annotations := range annotationCases() {
		var got, want strings.Builder
		WriteAnnotations(&got, "Annotations", annotations)
		upstreamAnnotations(&want, "Annotations", annotations)
		if got.String() != want.String() {
			t.Errorf("%s:\ngot  %q\nwant %q", name, got.String(), want.String())
		}
		tb := newTabbed()
		tb.annotations("  Annotations", annotations)
		ref := newTabbed()
		upstreamAnnotations(ref.w, "  Annotations", annotations)
		if tb.String() != ref.String() {
			t.Errorf("%s in a template:\ngot  %q\nwant %q", name, tb.String(), ref.String())
		}
	}
}

func TestDescribePodWritesLabelsAndAnnotationsOnePerLine(t *testing.T) {
	f := newCluster(t)
	f.placed(t, "default", "lantern-lone-tower", "lone-tower", func(p *corev1.Pod) {
		mirrored("lone-tower")(p)
		p.Labels["tier"] = "door"
		p.Annotations["note"] = strings.Repeat("x", 150)
	})
	out := run(f, "kubectl describe pod lantern-lone-tower").Output
	pad := strings.Repeat(" ", 16)
	want := "\nLabels:         app=lantern\n" + pad + "tier=door\n" +
		"Annotations:    kubernetes.io/config.hash: 9c4e\n" +
		pad + "kubernetes.io/config.mirror: 9c4e\n" +
		pad + "kubernetes.io/config.source: file\n" +
		pad + "note:\n" +
		pad + "  " + strings.Repeat("x", 138) + "...\n" +
		"Status:         Running\n"
	if !strings.Contains(out, want) {
		t.Fatalf("describe pod:\n%s\nwant the block:\n%s", out, want)
	}
	f.placed(t, "default", "bare", "lone-tower", nil)
	if out := run(f, "kubectl describe pod bare").Output; !strings.Contains(out, "\nLabels:           <none>\nAnnotations:      <none>\nStatus:") {
		t.Fatalf("a pod without labels or annotations:\n%s", out)
	}
}

func TestDescribeReplicaSetWritesLabelsAndAnnotationsOnePerLine(t *testing.T) {
	f := newCluster(t)
	one := int32(1)
	labels := map[string]string{"app": "lantern", "pod-template-hash": "5d8f7c"}
	_, err := f.st.Create(&appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "lantern-5d8f7c", Namespace: "default", Labels: labels,
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1", "deployment.kubernetes.io/desired-replicas": "1"}},
		Spec: appsv1.ReplicaSetSpec{Replicas: &one, Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "lantern", Image: "porter:1.0"}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := run(f, "kubectl describe rs lantern-5d8f7c").Output
	pad := strings.Repeat(" ", 14)
	want := "\nLabels:       app=lantern\n" + pad + "pod-template-hash=5d8f7c\n" +
		"Annotations:  deployment.kubernetes.io/desired-replicas: 1\n" +
		pad + "deployment.kubernetes.io/revision: 1\n" +
		"Replicas:     "
	if !strings.Contains(out, want) {
		t.Fatalf("describe rs:\n%s\nwant the block:\n%s", out, want)
	}
	if !strings.Contains(out, "\nPod Template:\n  Labels:  app=lantern\n           pod-template-hash=5d8f7c\n  Containers:\n") {
		t.Fatalf("describe rs template labels:\n%s", out)
	}
}

func TestDescribeServiceAndNodeWriteLabelsAndAnnotationsAsUpstream(t *testing.T) {
	f := newCluster(t)
	_, err := f.st.Create(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "parcels", Namespace: "default", Labels: map[string]string{"app": "porters", "tier": "depot"},
			Annotations: map[string]string{"note": "sound at dawn\nand dusk", corev1.LastAppliedConfigAnnotation: "{}"}},
		Spec: corev1.ServiceSpec{Selector: map[string]string{"app": "porters"}, Ports: []corev1.ServicePort{{Port: 80, Protocol: corev1.ProtocolTCP}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := run(f, "kubectl describe svc parcels").Output
	pad := strings.Repeat(" ", 19)
	if want := "\nLabels:            app=porters\n" + pad + "tier=depot\nAnnotations:       note:\n" + pad + "  sound at dawn\n" + pad + "  and dusk\nSelector:          app=porters\n"; !strings.Contains(out, want) {
		t.Fatalf("describe svc:\n%s\nwant the block:\n%s", out, want)
	}
	node, _ := store.Get[*corev1.Node](f.st, store.KindNode, "", "lone-tower")
	next := node.DeepCopy()
	next.Annotations = map[string]string{"kubernites.dev/ready-at": "12.00"}
	if _, err := f.st.Update(next); err != nil {
		t.Fatal(err)
	}
	out = run(f, "kubectl describe node lone-tower").Output
	if want := "\nLabels:             node-role.kubernetes.io/control-plane=\nAnnotations:        kubernites.dev/ready-at: 12.00\nCreationTimestamp:"; !strings.Contains(out, want) {
		t.Fatalf("describe node:\n%s\nwant the block:\n%s", out, want)
	}
}
