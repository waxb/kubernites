package view

type UsePeakView struct {
	CPU    int64   `json:"cpu,omitempty"`
	Memory int64   `json:"memory,omitempty"`
	Window float64 `json:"window,omitempty"`
	Noon   bool    `json:"noon,omitempty"`
	Bound  bool    `json:"bound,omitempty"`
	Limit  int64   `json:"limit,omitempty"`
	Raise  int64   `json:"raise,omitempty"`
	Waits  int     `json:"waits,omitempty"`
}

func (v UsePeakView) Known() bool {
	return v.Noon && v.CPU > 0
}
