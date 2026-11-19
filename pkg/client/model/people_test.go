package model

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func household() *corev1.Pod {
	p := pod("porters-7d9f-x2kq", "n1", "porter:1.0")
	p.Spec.Containers = []corev1.Container{{Name: "porter", Image: "porter:1.0"}, {Name: "packer", Image: "hauler:1.0"}}
	p.Spec.InitContainers = []corev1.Container{sidecar("raven", "raven:1.0", true)}
	p.Status.ContainerStatuses = []corev1.ContainerStatus{
		{Name: "packer", Ready: false, RestartCount: 4, State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
			LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"}}},
		{Name: "porter", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
	}
	return p
}

func TestEveryAppContainerIsAPersonOfTheHearth(t *testing.T) {
	v := ViewOf(household())
	if len(v.People) != 2 {
		t.Fatalf("people = %+v", v.People)
	}
	porter, packer := v.People[0], v.People[1]
	if porter.Name != "porter" || porter.Image != "porter:1.0" || porter.Sprite != v.Sprite || !porter.Ready || porter.Reason != "" {
		t.Errorf("porter = %+v", porter)
	}
	if packer.Name != "packer" || packer.Image != "hauler:1.0" || packer.Sprite == porter.Sprite || packer.Ready {
		t.Errorf("packer = %+v", packer)
	}
	if packer.Restarts != 4 || packer.Reason != "CrashLoopBackOff" || packer.LastExit != "OOMKilled" || packer.Tier != 1 {
		t.Errorf("packer status = %+v", packer)
	}
	if len(v.Familiars) != 1 {
		t.Errorf("the raven is a pet, not a person: %v", v.Familiars)
	}
	if len(ViewOf(pod("bare", "n1", "porter:1.0")).People) != 1 {
		t.Error("a one-container Pod has one person")
	}
}

func TestFocusedViewNamesOneContainer(t *testing.T) {
	v := ViewOf(household())
	packer := v.Focused("packer")
	if packer.Focus != "packer" || packer.Image != "hauler:1.0" || packer.Sprite != v.People[1].Sprite || packer.Profession != v.People[1].Profession {
		t.Fatalf("packer view = %+v", packer)
	}
	if packer.Restarts != 4 || packer.Reason != "CrashLoopBackOff" || packer.LastExit != "OOMKilled" || packer.Ready {
		t.Errorf("packer view status = %+v", packer)
	}
	if packer.Name != v.Name || packer.State != v.State || len(packer.People) != 2 {
		t.Error("a person's view is still the same hut")
	}
	if got := v.Person(0); got.Focus != "porter" || got.Image != "porter:1.0" {
		t.Errorf("person 0 = %+v", got)
	}
	if got := v.Person(5); got.Focus != "" {
		t.Errorf("no sixth person: %+v", got)
	}
	back := packer.Focused("")
	if back.Focus != "" || back.Name != v.Name {
		t.Errorf("unfocused = %+v", back)
	}
	if got := v.Focused("ghost"); got.Focus != "" || got.Image != v.Image {
		t.Errorf("unknown container keeps the hut view: %+v", got)
	}
	bare := PodView{Name: "p", Image: "porter:1.0"}
	if got := bare.Person(0); got.Name != "p" || got.Focus != "" {
		t.Errorf("a view without people is its own person: %+v", got)
	}
	if bare.Persons() != 1 || v.Persons() != 2 {
		t.Errorf("persons = %d and %d", bare.Persons(), v.Persons())
	}
}

func owned(p *corev1.Pod, kind, name string) *corev1.Pod {
	yes := true
	p.OwnerReferences = []metav1.OwnerReference{{Kind: kind, Name: name, Controller: &yes}}
	return p
}

func labelled(p *corev1.Pod, labels map[string]string) *corev1.Pod {
	p.Labels = labels
	return p
}

func service(name string, selector map[string]string) *corev1.Service {
	return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}, Spec: corev1.ServiceSpec{Selector: selector}}
}

func replicaSet(name, deployment string, labels map[string]string) *appsv1.ReplicaSet {
	yes := true
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}}
	rs.Spec.Template.Labels = labels
	if deployment != "" {
		rs.OwnerReferences = []metav1.OwnerReference{{Kind: "Deployment", Name: deployment, Controller: &yes}}
	}
	return rs
}

func viewNamed(views []PodView, name string) PodView {
	for _, v := range views {
		if v.Name == name {
			return v
		}
	}
	return PodView{}
}

func TestCrewsAndDepotsFollowOwnersAndSelectors(t *testing.T) {
	c := NewCache()
	c.Reset([]store.Object{
		service("kubernetes", nil),
		service("parcels", map[string]string{"app": "porters"}),
		service("ledgers", map[string]string{"app": "clerks", "tier": "db"}),
		replicaSet("porters-7d9f", "porters", map[string]string{"app": "porters"}),
		replicaSet("clerks-a1", "", map[string]string{"app": "clerks", "tier": "db"}),
		owned(labelled(pod("porters-7d9f-a", "n1", "porter:1.0"), map[string]string{"app": "porters"}), "ReplicaSet", "porters-7d9f"),
		owned(pod("clerks-a1-b", "n1", "clerk:1.0"), "ReplicaSet", "clerks-a1"),
		owned(labelled(pod("orphan-rs", "n1", "clerk:1.0"), map[string]string{"app": "clerks"}), "ReplicaSet", "gone-rs"),
		labelled(pod("porter-declared", "n1", "porter:1.0"), map[string]string{"app": "porters"}),
		labelled(pod("stray", "n1", "porter:1.0"), map[string]string{"app": "reserve"}),
	})
	views := c.PodViews()
	cases := []struct{ pod, crew, depot string }{
		{"porters-7d9f-a", "porters", "parcels"},
		{"clerks-a1-b", "clerks-a1", "ledgers"},
		{"orphan-rs", "gone-rs", ""},
		{"porter-declared", "", "parcels"},
		{"stray", "", ""},
	}
	for _, tc := range cases {
		v := viewNamed(views, tc.pod)
		if v.Crew != tc.crew || v.Depot != tc.depot {
			t.Errorf("%s: crew %q depot %q, want %q and %q", tc.pod, v.Crew, v.Depot, tc.crew, tc.depot)
		}
	}
}
