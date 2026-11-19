package view

type MilestoneView struct {
	Milestone      int     `json:"milestone"`
	Multiplier     float64 `json:"multiplier"`
	Next           int     `json:"next,omitempty"`
	NextMultiplier float64 `json:"nextMultiplier,omitempty"`
	Holds          int     `json:"holds,omitempty"`
}
