package view

type SiegeState struct {
	Phase    string  `json:"phase,omitempty"`
	StartsAt float64 `json:"startsAt,omitempty"`
	EndsAt   float64 `json:"endsAt,omitempty"`
	Served   float64 `json:"served,omitempty"`
	Dropped  float64 `json:"dropped,omitempty"`
	Cause    string  `json:"cause,omitempty"`
	RetryAt  float64 `json:"retryAt,omitempty"`
	Seals    int     `json:"seals,omitempty"`
	Surge    float64 `json:"surge,omitempty"`
	Won      int     `json:"won,omitempty"`
	Lost     int     `json:"lost,omitempty"`
}

type SiegeView struct {
	Available bool    `json:"available"`
	Reason    string  `json:"reason,omitempty"`
	Phase     string  `json:"phase,omitempty"`
	TimeLeft  float64 `json:"timeLeft,omitempty"`
	Served    float64 `json:"served"`
	Target    float64 `json:"target"`
	Crowd     float64 `json:"crowd"`
	Extra     float64 `json:"extra,omitempty"`
	Forecast  string  `json:"forecast,omitempty"`
	Hands     float64 `json:"hands"`
	Spare     float64 `json:"spare"`
	Crowded   string  `json:"crowded,omitempty"`
	Reward    float64 `json:"reward"`
	Seals     int     `json:"seals"`
	SealsGoal int     `json:"sealsGoal"`
	RetryIn   float64 `json:"retryIn,omitempty"`
}

const TideFloodForecast = "Your Keeps cannot place the tide's extra huts: expect the quay to flood."

type TideLever struct {
	Text    string
	Command string
	Crew    string
	Button  string
}
