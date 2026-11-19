package podutil

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var always = corev1.ContainerRestartPolicyAlways

func sidecar(name string, req, lim corev1.ResourceList) corev1.Container {
	return corev1.Container{Name: name, Image: name + ":1.0", RestartPolicy: &always, Resources: corev1.ResourceRequirements{Requests: req, Limits: lim}}
}

func squire(name string, req, lim corev1.ResourceList) corev1.Container {
	return corev1.Container{Name: name, Image: name + ":1.0", Resources: corev1.ResourceRequirements{Requests: req, Limits: lim}}
}

func withInit(p *corev1.Pod, inits ...corev1.Container) *corev1.Pod {
	next := p.DeepCopy()
	next.Spec.InitContainers = inits
	return next
}

func TestIsRestartableInitContainer(t *testing.T) {
	never := corev1.ContainerRestartPolicyNever
	c := sidecar("raven", nil, nil)
	if !IsRestartableInitContainer(&c) || IsRestartableInitContainer(&corev1.Container{}) || IsRestartableInitContainer(&corev1.Container{RestartPolicy: &never}) || IsRestartableInitContainer(nil) {
		t.Fatal("restartable detection wrong")
	}
	p := withInit(podWith(nil, nil), squire("setup", nil, nil), sidecar("raven", nil, nil))
	if got := Sidecars(p); len(got) != 1 || got[0].Name != "raven" {
		t.Fatalf("sidecars = %+v", got)
	}
}

func TestRequestsAddSidecarsToApp(t *testing.T) {
	p := withInit(podWith(rl("100m", "64Mi"), nil), sidecar("raven", rl("50m", ""), rl("", "32Mi")))
	cpu, mem := Requests(p)
	if cpu != 150 || mem != 96*1024*1024 {
		t.Fatalf("cpu=%d mem=%d", cpu, mem)
	}
}

func TestRequestsTakeTheHighestInitUse(t *testing.T) {
	sidecarFirst := withInit(podWith(rl("100m", "64Mi"), nil), sidecar("raven", rl("50m", "32Mi"), nil), squire("setup", rl("500m", "16Mi"), nil))
	if cpu, mem := Requests(sidecarFirst); cpu != 550 || mem != 96*1024*1024 {
		t.Fatalf("sidecar first: cpu=%d mem=%d", cpu, mem)
	}
	squireFirst := withInit(podWith(rl("100m", "64Mi"), nil), squire("setup", rl("500m", "16Mi"), nil), sidecar("raven", rl("50m", "32Mi"), nil))
	if cpu, mem := Requests(squireFirst); cpu != 500 || mem != 96*1024*1024 {
		t.Fatalf("squire first: cpu=%d mem=%d", cpu, mem)
	}
}

func TestLimitsSumWhatIsDeclaredLikeDescribeNode(t *testing.T) {
	bound := withInit(podWith(nil, rl("200m", "128Mi")), sidecar("hound", nil, rl("100m", "96Mi")))
	if cpu, mem := Limits(bound); cpu != 300 || mem != 224*1024*1024 {
		t.Fatalf("bound limits cpu=%d mem=%d", cpu, mem)
	}
	loose := withInit(podWith(nil, rl("200m", "128Mi")), sidecar("hound", nil, rl("100m", "")))
	if cpu, mem := Limits(loose); cpu != 300 || mem != 128*1024*1024 {
		t.Fatalf("loose limits cpu=%d mem=%d", cpu, mem)
	}
	squireHigh := withInit(podWith(nil, rl("200m", "128Mi")), squire("setup", nil, rl("1", "64Mi")))
	if cpu, mem := Limits(squireHigh); cpu != 1000 || mem != 128*1024*1024 {
		t.Fatalf("squire limits cpu=%d mem=%d", cpu, mem)
	}
	app := podWith(nil, rl("200m", "128Mi")).Spec.Containers[0]
	side := sidecar("raven", rl("10m", ""), rl("", "64Mi"))
	if cpu, mem := ContainerLimits(app, side); cpu != 0 || mem != 192*1024*1024 {
		t.Fatalf("container limits cpu=%d mem=%d", cpu, mem)
	}
	if cpu, mem := ContainerRequests(app, side); cpu != 210 || mem != 192*1024*1024 {
		t.Fatalf("container requests cpu=%d mem=%d", cpu, mem)
	}
}

func TestQoSCountsSidecars(t *testing.T) {
	guaranteed := rl("100m", "64Mi")
	cases := []struct {
		name string
		pod  *corev1.Pod
		want corev1.PodQOSClass
	}{
		{"both guaranteed", withInit(podWith(guaranteed, guaranteed), sidecar("raven", guaranteed, guaranteed)), corev1.PodQOSGuaranteed},
		{"bare familiar", withInit(podWith(guaranteed, guaranteed), sidecar("raven", nil, nil)), corev1.PodQOSBurstable},
		{"nothing anywhere", withInit(podWith(nil, nil), sidecar("raven", nil, nil)), corev1.PodQOSBestEffort},
		{"only the familiar pledges", withInit(podWith(nil, nil), sidecar("raven", rl("10m", ""), nil)), corev1.PodQOSBurstable},
	}
	for _, tc := range cases {
		if got := QoSClass(tc.pod); got != tc.want {
			t.Errorf("%s: got %s want %s", tc.name, got, tc.want)
		}
	}
}

func started(v bool) *bool {
	return &v
}

func familiarPod(initState corev1.ContainerState, initStarted, initReady bool, appState corev1.ContainerState, appReady bool) *corev1.Pod {
	p := withInit(podWith(nil, nil), sidecar("raven", nil, nil))
	p.Status.Phase = corev1.PodPending
	p.Status.InitContainerStatuses = []corev1.ContainerStatus{{Name: "raven", State: initState, Started: started(initStarted), Ready: initReady}}
	p.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "lantern", State: appState, Ready: appReady}}
	return p
}

func waitingState(reason string) corev1.ContainerState {
	return corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason}}
}

func runningState() corev1.ContainerState {
	return corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
}

func TestStatusReadyAndRestartsFollowKubectlForFamiliars(t *testing.T) {
	initialized := func(p *corev1.Pod) *corev1.Pod {
		p.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodInitialized, Status: corev1.ConditionTrue}}
		p.Status.Phase = corev1.PodRunning
		return p
	}
	cases := []struct {
		name   string
		pod    *corev1.Pod
		status string
		ready  int
	}{
		{"pulling the familiar", familiarPod(waitingState("PodInitializing"), false, false, waitingState("PodInitializing"), false), "Init:0/1", 0},
		{"familiar never arrived", familiarPod(waitingState("ImagePullBackOff"), false, false, waitingState("PodInitializing"), false), "Init:ImagePullBackOff", 0},
		{"familiar up, app initializing", familiarPod(runningState(), true, true, waitingState("PodInitializing"), false), "PodInitializing", 1},
		{"all burning", initialized(familiarPod(runningState(), true, true, runningState(), true)), "Running", 2},
		{"familiar crashlooping", initialized(familiarPod(waitingState("CrashLoopBackOff"), false, false, runningState(), true)), "Init:CrashLoopBackOff", 1},
		{"familiar failed init", familiarPod(corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 3}}, false, false, waitingState("PodInitializing"), false), "Init:ExitCode:3", 0},
		{"familiar failed with reason", familiarPod(corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 137, Reason: "OOMKilled"}}, false, false, waitingState("PodInitializing"), false), "Init:OOMKilled", 0},
	}
	for _, tc := range cases {
		if got := StatusReason(tc.pod); got != tc.status {
			t.Errorf("%s: status %q want %q", tc.name, got, tc.status)
		}
		if ready, total := ReadyCount(tc.pod); ready != tc.ready || total != 2 {
			t.Errorf("%s: ready %d/%d want %d/2", tc.name, ready, total, tc.ready)
		}
	}
}

func TestRestartsCountFamiliarsOnceInitialized(t *testing.T) {
	early := time.Date(2026, 9, 23, 1, 0, 0, 0, time.UTC)
	late := early.Add(time.Minute)
	p := familiarPod(runningState(), true, true, runningState(), true)
	p.Status.InitContainerStatuses[0].RestartCount = 2
	p.Status.InitContainerStatuses[0].LastTerminationState = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{FinishedAt: metav1.NewTime(late)}}
	p.Status.ContainerStatuses[0].RestartCount = 1
	p.Status.ContainerStatuses[0].LastTerminationState = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{FinishedAt: metav1.NewTime(early)}}
	if got := RestartCount(p); got != 3 {
		t.Fatalf("restarts = %d", got)
	}
	if got := LastRestart(p); !got.Time.Equal(late) {
		t.Fatalf("last restart = %v", got)
	}
	stuck := familiarPod(waitingState("CrashLoopBackOff"), false, false, waitingState("PodInitializing"), false)
	stuck.Status.InitContainerStatuses[0].RestartCount = 4
	stuck.Status.ContainerStatuses[0].RestartCount = 9
	if got := RestartCount(stuck); got != 4 {
		t.Fatalf("initializing restarts = %d", got)
	}
	plain := podWith(nil, nil)
	plain.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "lantern", RestartCount: 2, LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{FinishedAt: metav1.NewTime(early)}}}}
	if RestartCount(plain) != 2 || !LastRestart(plain).Time.Equal(early) {
		t.Fatal("plain restarts changed")
	}
}

func TestContainerStatusLookups(t *testing.T) {
	p := familiarPod(runningState(), true, true, waitingState("PodInitializing"), false)
	if !ContainerRunning(p.Status.InitContainerStatuses, "raven") || ContainerRunning(p.Status.ContainerStatuses, "lantern") || ContainerRunning(nil, "raven") {
		t.Fatal("running lookup wrong")
	}
	if cs, ok := ContainerStatusNamed(p.Status.InitContainerStatuses, "raven"); !ok || cs.Name != "raven" {
		t.Fatal("status lookup wrong")
	}
	if _, ok := ContainerStatusNamed(p.Status.InitContainerStatuses, "ghost"); ok {
		t.Fatal("ghost found")
	}
}
