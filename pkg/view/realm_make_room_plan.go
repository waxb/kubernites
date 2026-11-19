package view

const BoundSurge = "surge"

const BoundEnd = "end"

const BoundCrowd = "crowd"

const BoundStranded = "stranded"

type RoomPlan struct {
	Shrink    int    `json:"shrink,omitempty"`
	To        int    `json:"to,omitempty"`
	Ceiling   int    `json:"ceiling,omitempty"`
	Floor     int    `json:"floor,omitempty"`
	Horned    bool   `json:"horned,omitempty"`
	Surgeless string `json:"surgeless,omitempty"`
	Settling  bool   `json:"settling,omitempty"`
	Starves   int    `json:"starves,omitempty"`
	Rising    bool   `json:"rising,omitempty"`
	Reserve   bool   `json:"reserve,omitempty"`
	Before    string `json:"before,omitempty"`
}

func (v RolloutRoomView) Lacks() bool {
	return !v.Fits || v.Fit < v.Huts
}

func (v RolloutRoomView) MakeRoom(crew string) (Action, bool) {
	switch {
	case v.To <= 0:
		return Action{}, false
	case v.Horned:
		return Action{Action: ActRetuneHorn, Name: crew, Crew: CrewOrder{Min: v.Floor, Max: v.Ceiling, Before: v.Before}}, true
	}
	return Action{Action: ActScaleCrew, Name: crew, Replicas: v.To}, true
}
