package kubectl

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func (f *fakeCluster) systemPod(t *testing.T, name string) {
	t.Helper()
	_, _ = f.st.Create(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive}})
	p := f.runningPod(t, name+"-tmp", 0)
	sys := p.DeepCopy()
	sys.ResourceVersion, sys.UID = "", ""
	sys.Name, sys.Namespace = name, "kube-system"
	if _, err := f.st.Create(sys); err != nil {
		t.Fatal(err)
	}
	if err := f.st.Delete(store.KeyOf(p)); err != nil {
		t.Fatal(err)
	}
}

func TestNamespaceFlagMayComeBeforeTheVerb(t *testing.T) {
	f := newCluster(t)
	f.systemPod(t, "etcd-lone-tower")
	f.runningPod(t, "porter-abcde", 0)
	for _, cmd := range []string{
		"kubectl -n kube-system get pods", "-n kube-system get pods", "kubectl --namespace kube-system get pods",
		"kubectl --namespace=kube-system get pods", "kubectl -n=kube-system get pods", "kubectl -nkube-system get pods",
		"kubectl get pods -nkube-system", "kubectl -n kube-system get pods -o wide", "kubectl -n kube-system describe pod etcd-lone-tower",
	} {
		res := run(f, cmd)
		if !res.OK || !strings.Contains(res.Output, "etcd-lone-tower") || strings.Contains(res.Output, "porter-abcde") {
			t.Errorf("%q = %+v", cmd, res)
		}
		if res.Verb != "get" && res.Verb != "describe" {
			t.Errorf("%q verb = %q", cmd, res.Verb)
		}
	}
}

func TestLeadingFlagsThatAreNotGlobalStayErrors(t *testing.T) {
	f := newCluster(t)
	for cmd, want := range map[string]string{
		"kubectl -A get pods":       `unknown command "-A"`,
		"kubectl -n":                `unknown command "-n"`,
		"kubectl -n kube-system":    `unknown command "-n"`,
		"kubectl --output wide get": `unknown command "--output"`,
	} {
		if res := run(f, cmd); res.OK || !strings.Contains(res.Output, want) {
			t.Errorf("%q = %+v, want %q", cmd, res, want)
		}
	}
}

func TestAttachedShorthandValues(t *testing.T) {
	f := newCluster(t)
	f.runningPod(t, "porter-abcde", 0)
	if res := run(f, "get pods -lapp=lantern -owide"); !res.OK || !strings.Contains(res.Output, "NODE") || !strings.Contains(res.Output, "porter-abcde") {
		t.Errorf("attached -l and -o = %+v", res)
	}
	if res := run(f, "get pods -lapp=nobody"); !res.OK || strings.Contains(res.Output, "porter-abcde") {
		t.Errorf("attached selector filters = %+v", res)
	}
	if res := run(f, "get pods -zz"); res.OK || !strings.Contains(res.Output, "unknown flag: -zz") {
		t.Errorf("unknown shorthand = %+v", res)
	}
	if res := run(f, "logs porter-abcde -p"); strings.Contains(res.Output, "unknown flag") {
		t.Errorf("bool shorthand still parses = %+v", res)
	}
}
