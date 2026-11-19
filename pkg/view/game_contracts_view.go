package view

type ContractView struct {
	ID       string         `json:"id"`
	Merchant string         `json:"merchant"`
	Title    string         `json:"title"`
	Goal     string         `json:"goal"`
	Kind     string         `json:"kind"`
	Box      string         `json:"box,omitempty"`
	Sealed   bool           `json:"sealed,omitempty"`
	Crowned  bool           `json:"crowned,omitempty"`
	Size     string         `json:"size,omitempty"`
	Target   float64        `json:"target"`
	Progress string         `json:"progress"`
	Done     float64        `json:"done"`
	Total    float64        `json:"total"`
	Deadline float64        `json:"deadline"`
	Span     float64        `json:"span"`
	Reward   ContractReward `json:"reward"`
	State    string         `json:"state"`
}
