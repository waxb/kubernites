package view

import (
	"fmt"
)

const AlreadyDone = "Already done: "

type Facts struct {
	Fiefs           int
	RunningHearths  int
	DoorHearths     int
	DoorPlaced      int
	BarePods        int
	RunningBare     int
	BareWaiting     string
	BareUnplaced    bool
	BarePlacing     bool
	ImpOmen         string
	HandPorters     int
	HandHired       int
	OldHut          string
	Roll            RollFacts
	Musters         int
	MusterReplicas  int
	PledgedLevies   int
	UpgradedHearths int
	CrewRolling     bool
	CrewStale       bool
	Keeps           int
	HearthsOffTower int
	SLO             float64
	SLOSamples      float64
	Drought         bool
	Built           map[string]int
	Raising         map[string]int
	Weave           float64
	Nines           int
	Woven           map[string]int
	HornsSounding   int
	Bottleneck      string
	KeepJoining     bool
	KeepTypes       map[string]int
	Zones           int
	Familiars       int
	Augury          bool
	MusterImage     string
	MusterPledged   bool
	SurgeShort      string
	SurgeWaits      int
	RollShort       string
	RollWaits       int
	RollPlan        CrewRoomPlan
	CrewRolls       map[string]CrewRoll
	TryRolls        map[string]CrewRoll
	PledgeShort     string
	PledgeWaits     int
	PledgePlan      CrewRoomPlan
	PledgeAt        int64
	PledgeRoomKnown bool
	Shrink          ShrinkFact
	SaveJob         SaveJob
	NoonNeed        [DepotCount]int
	PorterRoom      int
	RollDips        bool
	PledgeDips      bool
	StaleLevy       string
	UnhousedPod     string
	RoomShort       bool
	IdlePod         string
	CrewPod         string
	ForemanRaised   string
	ClassRising     string
	WardedKeep      string
	WardTurnsAway   string
	RetireKeep      string
	RetireDrained   bool
	RetireCommand   string
	RetireCrowded   string
	NoKeepLand      bool
	TideShort       string
	TideKeepPays    bool
	TideLever       TideLever
	KeepRoomPays    bool
	Freight         FreightFacts
	Harbors         []HarborSpot
	Speed           float64
	ShipsDocked     int
	ShipCargo       float64
	CodexWriteDelay float64
	CodexLateWrites int
	RatingFacts
	DaemonFacts
	StorageFacts
	WatchFacts
	Carry CarryFacts
}

const DepotCount = KindCount + 1

type FreightFacts struct {
	Arrivals      [KindCount]float64
	Served        [KindCount]float64
	Spoiled       [KindCount]float64
	TimedOutTotal float64
	Waiting       [KindCount]float64
	Depots        [DepotCount]bool
	Crews         [DepotCount]int
	Declared      [DepotCount]bool
	Keeper        [DepotCount]string
	Replicas      [DepotCount]int
	Ready         [DepotCount]int
	Pending       [DepotCount]int
	Stalled       [DepotCount]int
	Away          [DepotCount]int
	Asks          [DepotCount]int64
	Pledges       [DepotCount]Pledge
	Class         [DepotCount]string
	Settled       [DepotCount]bool
	Rolled        [DepotCount]bool
	Hungry        [DepotCount]bool
	DemandLimited [DepotCount]bool
	Horns         [DepotCount]HornFact
	Door          bool
	Sealed        float64
}

type HornFact struct {
	Sounding bool
	Goal     int
	Housing  bool
}

func (h HornFact) Trigger() string {
	switch {
	case h.Goal <= 0:
		return "outgrow their pledge"
	case h.Housing:
		return fmt.Sprintf("fill past %d%% of their housing pledge", h.Goal)
	}
	return fmt.Sprintf("eat past %d%% of their food pledge", h.Goal)
}

func (f FreightFacts) Quay() float64 {
	total := 0.0
	for _, w := range f.Waiting {
		total += w
	}
	return total
}

type ObjectiveView struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Why       string `json:"why"`
	Hint      string `json:"hint"`
	Target    string `json:"target,omitempty"`
	Button    string `json:"button,omitempty"`
	Build     string `json:"build,omitempty"`
	Command   string `json:"command,omitempty"`
	Progress  string `json:"progress,omitempty"`
	Earn      string `json:"earn,omitempty"`
	Page      string `json:"page,omitempty"`
	Purpose   string `json:"purpose,omitempty"`
	Quest     string `json:"quest,omitempty"`
	Lesson    string `json:"lesson,omitempty"`
	Wait      bool   `json:"wait,omitempty"`
	Done      bool   `json:"done"`
	Skipped   bool   `json:"skipped,omitempty"`
	Ahead     bool   `json:"ahead,omitempty"`
	Acted     bool   `json:"acted,omitempty"`
	Practiced bool   `json:"practiced,omitempty"`
	DoneAct   string `json:"doneAct,omitempty"`
	Major     bool   `json:"major,omitempty"`
	PastDemo  bool   `json:"pastDemo,omitempty"`
	SaveFor   int    `json:"saveFor,omitempty"`
}
