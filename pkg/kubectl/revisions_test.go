package kubectl

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetControllerRevisionsTable(t *testing.T) {
	f := newCluster(t)
	yes := true
	_, _ = f.st.Create(&appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{Name: "warden-6d4b9", Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "DaemonSet", Name: "warden", UID: "u", Controller: &yes}}},
		Data: runtime.RawExtension{Raw: []byte(`{}`)}, Revision: 2,
	})
	_, _ = f.st.Create(&appsv1.ControllerRevision{ObjectMeta: metav1.ObjectMeta{Name: "loose-1", Namespace: "default"}, Revision: 1})
	f.clk.Advance(30)
	for _, cmd := range []string{"kubectl get controllerrevisions", "kubectl get controllerrevision.apps"} {
		out := mustRun(t, f, cmd)
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		if len(lines) != 3 || strings.Join(strings.Fields(lines[0]), " ") != "NAME CONTROLLER REVISION AGE" {
			t.Fatalf("%s:\n%s", cmd, out)
		}
		if got := strings.Join(strings.Fields(lines[2]), " "); got != "warden-6d4b9 daemonset.apps/warden 2 30s" {
			t.Fatalf("row = %q", got)
		}
		if got := strings.Join(strings.Fields(lines[1]), " "); got != "loose-1 <none> 1 30s" {
			t.Fatalf("row = %q", got)
		}
	}
}
