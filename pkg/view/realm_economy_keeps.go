package view

type KeepView struct {
	Name         string         `json:"name"`
	ID           string         `json:"id,omitempty"`
	KeepType     string         `json:"keepType"`
	Instance     string         `json:"instance"`
	Ready        bool           `json:"ready"`
	Open         bool           `json:"open"`
	Upkeep       float64        `json:"upkeep"`
	UpkeepLine   string         `json:"upkeepLine,omitempty"`
	FullUpkeep   float64        `json:"fullUpkeep,omitempty"`
	Allocatable  ResourcePair   `json:"allocatable"`
	Requested    ResourcePair   `json:"requested"`
	Used         ResourcePair   `json:"used"`
	Daemons      []DaemonOnKeep `json:"daemons"`
	Stranded     int            `json:"stranded"`
	StrandedGold float64        `json:"strandedGold,omitempty"`
	DrainCommand string         `json:"drainCommand,omitempty"`
}
