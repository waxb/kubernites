package talk

import "github.com/waxb/kubernites/pkg/client/model"

type Facts struct {
	Names      bool
	Carrier    bool
	Trade      string
	Familiar   string
	Worker     model.Kubernite
	Hut        Hut
	Use        Use
	Doing      Doing
	Around     Around
	Problem    string
	Known      map[string]bool
	Now        float64
	FirstClick bool
}

type Hut struct {
	Name          string
	System        bool
	Rites         bool
	Static        bool
	Owner         string
	Crew          string
	OnRoll        bool
	Depot         string
	Rollout       bool
	OldClass      bool
	JoinedAgo     float64
	LeftAgo       float64
	App           string
	Claims        int
	Node          string
	LoneWard      bool
	RestartPolicy string
	Familiars     int
	Workers       int
	Frozen        bool
	ImageReady    bool
}

type Use struct {
	Measured  bool
	Alone     bool
	CPUMilli  float64
	MemBytes  float64
	Served    float64
	Throttled bool
}

type Doing struct {
	Activity string
	Load     string
	Toward   string
}

const (
	DoCarrying  = "carrying"
	DoReturning = "returning"
	DoWaiting   = "waiting"
	DoResting   = "resting"
	DoWalking   = "walking"
	DoServing   = "serving"
)

type Around struct {
	Night       bool
	Tide        bool
	Gilded      bool
	ImpAgo      float64
	ImpReplaced bool
	AgeAgo      float64
}

const never = -1

const (
	recentSeconds = 120.0
	freshSeconds  = 120.0
	crampedShare  = 0.9
	impSeconds    = 300.0
)

func (f Facts) knows(needs []string) bool {
	for _, n := range needs {
		if !f.Known[n] {
			return false
		}
	}
	return true
}

func (f Facts) running() bool {
	return f.Worker.State.Phase == model.ContainerRunning
}

func (f Facts) waitingFor(reason string) bool {
	return f.Worker.State.Phase == model.ContainerWaiting && f.Worker.State.Reason == reason
}

func (f Facts) Kindling() bool {
	w := f.Worker.State
	return w.Phase == "" || w.Phase == model.ContainerWaiting && (w.Reason == "ContainerCreating" || w.Reason == "PodInitializing" || w.Reason == "")
}

func (f Facts) Guttering() bool {
	return f.waitingFor("CrashLoopBackOff")
}

func (f Facts) Voidtaken() bool {
	return f.Worker.Last.Reason == "OOMKilled" || f.Worker.State.Reason == "OOMKilled"
}

func (f Facts) NeverArrived() bool {
	return f.waitingFor("ImagePullBackOff") || f.waitingFor("ErrImagePull")
}

func (f Facts) Hungry() bool {
	return f.running() && f.Use.Measured && f.Use.Alone && f.Use.Throttled
}

func (f Facts) AtCPULimit() bool {
	limit := f.Worker.Resources.CPULimit
	return f.Hungry() && limit > 0 && f.Use.CPUMilli >= 0.95*float64(limit)
}

func (f Facts) Cramped() bool {
	limit := f.Worker.Resources.MemLimit
	return f.running() && f.Use.Measured && f.Use.Alone && limit > 0 && f.Use.MemBytes >= crampedShare*float64(limit)
}

func (f Facts) Shuttered() bool {
	return f.running() && f.Worker.Readiness && !f.Worker.Ready
}

func (f Facts) Burning() bool {
	return f.running() && (f.Worker.Ready || !f.Worker.Readiness)
}

func (f Facts) Waiting() bool {
	return f.Burning() && f.Hut.OnRoll && f.Use.Measured && f.Use.Served < 0.01 && f.Doing.Activity != DoCarrying
}

func (f Facts) Uptime() float64 {
	if !f.running() || f.Worker.State.Started == 0 || f.Now <= 0 {
		return never
	}
	return max(f.Now-f.Worker.State.Started, 0)
}

func (f Facts) RelitAgo() float64 {
	if f.Worker.Restarts == 0 || f.Worker.Last.Finished == 0 || f.Now <= 0 {
		return never
	}
	return max(f.Now-f.Worker.Last.Finished, 0)
}

func (f Facts) Relit() bool {
	ago := f.RelitAgo()
	return ago >= 0 && ago < recentSeconds
}

func (f Facts) ReplicaSet() bool {
	return len(f.Hut.Owner) > len("ReplicaSet/") && f.Hut.Owner[:len("ReplicaSet/")] == "ReplicaSet/"
}

func (f Facts) DaemonSet() bool {
	return len(f.Hut.Owner) > len("DaemonSet/") && f.Hut.Owner[:len("DaemonSet/")] == "DaemonSet/"
}

func (f Facts) HandHired() bool {
	return f.Hut.Owner == "" && !f.Hut.Static
}

func (f Facts) Restarts() bool {
	return f.Hut.RestartPolicy != "Never"
}

func recent(ago float64) bool {
	return ago >= 0 && ago < recentSeconds
}
