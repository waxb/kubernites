package view

const NextMilestone = "milestone"

const NextPage = "page"

const NextClass = "class"

const NextKeep = "keep"

const NextBerth = "berth"

const NextIncome = "income"

type NextView struct {
	Kind     string  `json:"kind"`
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Gold     float64 `json:"gold"`
	Wisdom   float64 `json:"wisdom,omitempty"`
	Hands    float64 `json:"hands,omitempty"`
	Missing  float64 `json:"missing,omitempty"`
	Progress float64 `json:"progress"`
	Ready    bool    `json:"ready"`
	Target   string  `json:"target"`
	Build    string  `json:"build,omitempty"`
	Command  string  `json:"command,omitempty"`
	Hint     string  `json:"hint,omitempty"`
}
