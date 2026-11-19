package view

import (
	"time"
)

type SLOView struct {
	Current       float64 `json:"current"`
	Target        float64 `json:"target"`
	WindowSeconds int     `json:"windowSeconds"`
	Samples       float64 `json:"samples"`
	Nines         int     `json:"nines"`
}

type PilgrimView struct {
	ArrivalRate  float64 `json:"arrivalRate"`
	ServedRate   float64 `json:"servedRate"`
	DroppedRate  float64 `json:"droppedRate"`
	Queue        float64 `json:"queue"`
	HandServed   float64 `json:"handServed"`
	GateOpen     bool    `json:"gateOpen"`
	TotalServed  float64 `json:"totalServed"`
	TotalDropped float64 `json:"totalDropped"`
	DayPhase     float64 `json:"dayPhase"`
}

type ImpView struct {
	Pod  string  `json:"pod"`
	Node string  `json:"node"`
	In   float64 `json:"in"`
}

type GameState struct {
	Snapshot
	StorageState
	FoundedAt     time.Time       `json:"foundedAt"`
	RealmTime     float64         `json:"realmTime"`
	Speed         float64         `json:"speed"`
	SLO           SLOView         `json:"slo"`
	Pilgrims      PilgrimView     `json:"pilgrims"`
	Imp           *ImpView        `json:"imp,omitempty"`
	Ledger        LedgerView      `json:"ledger"`
	Bottleneck    string          `json:"bottleneck,omitempty"`
	Advice        string          `json:"advice,omitempty"`
	Counsel       string          `json:"counsel,omitempty"`
	Codex         CodexView       `json:"codex"`
	Friction      []FrictionView  `json:"friction"`
	AutonomyToday int             `json:"autonomyToday"`
	Summon        SummonView      `json:"summon"`
	Freight       FreightView     `json:"freight"`
	Calamities    []CalamityView  `json:"calamities"`
	CauseTags     []CauseTag      `json:"causeTags"`
	Commands      []CommandOffer  `json:"commands"`
	Crews         []CrewView      `json:"crews"`
	Next          *NextView       `json:"next,omitempty"`
	Economy       EconomyView     `json:"economy"`
	Keeps         []KeepView      `json:"keeps"`
	Room          RoomBook        `json:"room"`
	LimitRange    *LimitRangeView `json:"limitRange,omitempty"`
	DaemonSets    []DaemonSetView `json:"daemonSets"`
	Watch         *WatchView      `json:"watch,omitempty"`
	Roll          RollView        `json:"roll"`
}

type PodMetrics struct {
	CPUMilli       float64 `json:"cpuMilli"`
	MemoryBytes    float64 `json:"memoryBytes"`
	ServedRate     float64 `json:"servedRate"`
	Throttled      bool    `json:"throttled"`
	EphemeralBytes float64 `json:"ephemeralBytes"`
}

type NodeMetrics struct {
	CPUMilli               float64 `json:"cpuMilli"`
	MemoryBytes            float64 `json:"memoryBytes"`
	CPUAllocatableMilli    int64   `json:"cpuAllocatableMilli"`
	MemoryAllocatableBytes int64   `json:"memoryAllocatableBytes"`
	CPURequestedMilli      int64   `json:"cpuRequestedMilli"`
	MemoryRequestedBytes   int64   `json:"memoryRequestedBytes"`
	Pods                   int     `json:"pods"`
	EphemeralUsedBytes     int64   `json:"ephemeralUsedBytes"`
	EphemeralCapacityBytes int64   `json:"ephemeralCapacityBytes"`
	DiskPressure           bool    `json:"diskPressure"`
}

type MetricsSnapshot struct {
	Pods   map[string]PodMetrics  `json:"pods"`
	Nodes  map[string]NodeMetrics `json:"nodes"`
	Augury bool                   `json:"augury"`
}
