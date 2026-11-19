package store

import (
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func storedPod(t *testing.T, s *Store, name string) *corev1.Pod {
	t.Helper()
	p, ok := Get[*corev1.Pod](s, KindPod, "default", name)
	if !ok {
		t.Fatalf("pod %s missing", name)
	}
	return p
}

func TestCreateAddsDefaultTolerationSeconds(t *testing.T) {
	s := cluster()
	in := newPod("a")
	created, err := s.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	tols := created.(*corev1.Pod).Spec.Tolerations
	if len(tols) != 2 {
		t.Fatalf("tolerations = %+v", tols)
	}
	for i, key := range []string{corev1.TaintNodeNotReady, corev1.TaintNodeUnreachable} {
		tol := tols[i]
		if tol.Key != key || tol.Operator != corev1.TolerationOpExists || tol.Effect != corev1.TaintEffectNoExecute || tol.TolerationSeconds == nil || *tol.TolerationSeconds != 300 {
			t.Fatalf("toleration %d = %+v", i, tol)
		}
	}
	if len(in.Spec.Tolerations) != 0 {
		t.Fatal("admission mutated the caller's pod")
	}
	if len(storedPod(t, s, "a").Spec.Tolerations) != 2 {
		t.Fatal("stored pod lacks the default tolerations")
	}
}

func TestDefaultTolerationSecondsRespectsExistingTolerations(t *testing.T) {
	long := int64(6000)
	cases := map[string]struct {
		tols []corev1.Toleration
		want int
	}{
		"tolerate everything":     {[]corev1.Toleration{{Operator: corev1.TolerationOpExists}}, 1},
		"tolerate all NoExecute":  {[]corev1.Toleration{{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute}}, 1},
		"own unreachable seconds": {[]corev1.Toleration{{Key: corev1.TaintNodeUnreachable, Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute, TolerationSeconds: &long}}, 2},
		"any effect not-ready":    {[]corev1.Toleration{{Key: corev1.TaintNodeNotReady, Operator: corev1.TolerationOpExists}}, 2},
		"NoSchedule only":         {[]corev1.Toleration{{Key: corev1.TaintNodeUnreachable, Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule}}, 3},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := cluster()
			p := newPod("a")
			p.Spec.Tolerations = tc.tols
			created, _ := s.Create(p)
			if got := created.(*corev1.Pod).Spec.Tolerations; len(got) != tc.want {
				t.Fatalf("tolerations = %+v", got)
			}
		})
	}
	s := cluster()
	p := newPod("b")
	p.Spec.Tolerations = []corev1.Toleration{{Key: corev1.TaintNodeUnreachable, Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute, TolerationSeconds: &long}}
	created, _ := s.Create(p)
	tols := created.(*corev1.Pod).Spec.Tolerations
	if *tols[0].TolerationSeconds != 6000 || tols[1].Key != corev1.TaintNodeNotReady {
		t.Fatalf("tolerations = %+v", tols)
	}
}

func TestUpdateAddsMissingDefaultTolerations(t *testing.T) {
	s := cluster()
	if err := s.Restore([]Object{newPod("old")}, 1, 1); err != nil {
		t.Fatal(err)
	}
	if len(storedPod(t, s, "old").Spec.Tolerations) != 0 {
		t.Fatal("restore must not run admission")
	}
	next := storedPod(t, s, "old")
	next.Labels = map[string]string{"app": "lantern"}
	if _, err := s.Update(next); err != nil {
		t.Fatal(err)
	}
	if len(storedPod(t, s, "old").Spec.Tolerations) != 2 {
		t.Fatal("update of a pod saved before admission did not add the tolerations")
	}
}

func serviceAccount(t *testing.T, s *Store, namespace, name string) {
	t.Helper()
	if _, err := s.Create(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}); err != nil {
		t.Fatal(err)
	}
}

func TestAPodWhoseServiceAccountDoesNotExistIsForbidden(t *testing.T) {
	s := New(fixedNow)
	_, err := s.Create(newPod("hello"))
	want := `pods "hello" is forbidden: error looking up service account default/default: serviceaccount "default" not found`
	var forbidden *ForbiddenError
	if !errors.As(err, &forbidden) || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
	if _, ok := Get[*corev1.Pod](s, KindPod, "default", "hello"); ok {
		t.Fatal("the refused Pod was stored")
	}
	serviceAccount(t, s, "default", "default")
	if _, err := s.Create(newPod("hello")); err != nil {
		t.Fatal(err)
	}
	named := newPod("clerk")
	named.Spec.ServiceAccountName = "clerk"
	if _, err := s.Create(named); err == nil || err.Error() != `pods "clerk" is forbidden: error looking up service account default/clerk: serviceaccount "clerk" not found` {
		t.Fatalf("err = %v", err)
	}
	elsewhere := newPod("guildsman")
	elsewhere.Namespace = "guild"
	if _, err := s.Create(elsewhere); err == nil || !strings.Contains(err.Error(), "guild/default") {
		t.Fatalf("a Pod in a namespace with no ServiceAccount: err = %v", err)
	}
}

func TestOnlyAMirrorPodNeedsNoServiceAccount(t *testing.T) {
	s := New(fixedNow)
	mirror := newPod("etcd-lone-tower")
	mirror.Annotations = map[string]string{corev1.MirrorPodAnnotationKey: "abc", "kubernetes.io/config.source": "file"}
	if _, err := s.Create(mirror); err != nil {
		t.Fatalf("%s: %v", mirror.Name, err)
	}
	claimed := newPod("kube-apiserver-lone-tower")
	claimed.Annotations = map[string]string{"kubernetes.io/config.source": "file"}
	if _, err := s.Create(claimed); err == nil || !strings.Contains(err.Error(), "default/default") {
		t.Fatalf("a Pod that only claims a file source skipped the ServiceAccount lookup: %v", err)
	}
}

func TestOnlyCreatingAPodLooksUpItsServiceAccount(t *testing.T) {
	s := New(fixedNow)
	if err := s.Restore([]Object{newPod("old")}, 1, 1); err != nil {
		t.Fatal(err)
	}
	next := storedPod(t, s, "old")
	next.Labels = map[string]string{"app": "porters"}
	if _, err := s.Update(next); err != nil {
		t.Fatalf("an update was refused for the ServiceAccount: %v", err)
	}
}
