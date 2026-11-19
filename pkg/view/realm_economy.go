package view

const FlowNone = ""

const FlowCrew = "crew"

const FlowCPU = "cpu"

const FlowMemory = "memory"

const FlowService = "service"

const FlowQuay = "quay"

const FlowCrash = "crash"

const FlowPending = "pending"

const FlowStarting = "starting"

const FlowBurst = "burst"

type ResourcePair struct {
	CPU    int64 `json:"cpu"`
	Memory int64 `json:"memory"`
}

type IncomeGoal struct {
	Target   float64 `json:"target"`
	Progress float64 `json:"progress"`
}

type EconomyView struct {
	Income     float64       `json:"income"`
	Upkeep     float64       `json:"upkeep"`
	Profit     float64       `json:"profit"`
	Hand       float64       `json:"hand,omitempty"`
	Arriving   float64       `json:"arriving"`
	Carried    float64       `json:"carried"`
	Lost       float64       `json:"lost"`
	Bottleneck string        `json:"bottleneck"`
	Why        string        `json:"why,omitempty"`
	Fix        *CommandOffer `json:"fix,omitempty"`
	FixMode    string        `json:"fixMode"`
	Milestone  *IncomeGoal   `json:"milestone,omitempty"`
	UpkeepRule string        `json:"upkeepRule,omitempty"`
}

type ReplicaCost struct {
	Income      float64 `json:"income"`
	CPU         int64   `json:"cpu"`
	Memory      int64   `json:"memory"`
	UpkeepShare float64 `json:"upkeepShare"`
	SquireCPU   int64   `json:"squireCpu,omitempty"`
	SquireMem   int64   `json:"squireMemory,omitempty"`
}

type LimitRangeView struct {
	CPU         int64 `json:"cpu"`
	Memory      int64 `json:"memory"`
	CPULimit    int64 `json:"cpuLimit,omitempty"`
	MemoryLimit int64 `json:"memoryLimit,omitempty"`
}
