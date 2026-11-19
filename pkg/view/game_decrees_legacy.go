package view

const DecreesFlag = "decrees"

type DecreeReward struct {
	Gold   float64 `json:"gold"`
	Wisdom float64 `json:"wisdom"`
	Favor  float64 `json:"favor"`
}

type DecreeState struct {
	ID       string       `json:"id"`
	Kind     string       `json:"kind"`
	State    string       `json:"state"`
	Offered  float64      `json:"offered"`
	Deadline float64      `json:"deadline"`
	Resolved float64      `json:"resolved,omitempty"`
	Reward   DecreeReward `json:"reward"`
	Tribute  float64      `json:"tribute,omitempty"`
	Nines    int          `json:"nines,omitempty"`
	Opens    float64      `json:"opens,omitempty"`
	Low      float64      `json:"low,omitempty"`
	Scope    string       `json:"scope,omitempty"`
	Held     float64      `json:"held,omitempty"`
}

type DecreeView struct {
	ID            string       `json:"id"`
	Title         string       `json:"title"`
	Text          string       `json:"text"`
	Kind          string       `json:"kind"`
	Deadline      float64      `json:"deadline"`
	Progress      string       `json:"progress"`
	ProgressDone  float64      `json:"progressDone"`
	ProgressTotal float64      `json:"progressTotal"`
	Reward        DecreeReward `json:"reward"`
	State         string       `json:"state"`
	Tribute       float64      `json:"tribute,omitempty"`
}
