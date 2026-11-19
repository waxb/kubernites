package view

type ProbeSetting struct {
	Path      string `json:"path"`
	Period    int32  `json:"period"`
	Threshold int32  `json:"threshold"`
	Timeout   int32  `json:"timeout"`
}

type CrewWatchView struct {
	Crew      string        `json:"crew"`
	Container string        `json:"container"`
	Readiness *ProbeSetting `json:"readiness,omitempty"`
	Liveness  *ProbeSetting `json:"liveness,omitempty"`
	Patience  int           `json:"patience,omitempty"`
	Delay     int           `json:"delay,omitempty"`
}

type HutWatchView struct {
	Pod     string  `json:"pod"`
	Crew    string  `json:"crew"`
	Node    string  `json:"node"`
	Warming bool    `json:"warming"`
	Frozen  bool    `json:"frozen"`
	Shut    bool    `json:"shut"`
	Knocks  int     `json:"knocks"`
	Busy    float64 `json:"busy"`
}

type WatchView struct {
	Crews []CrewWatchView `json:"crews"`
	Huts  []HutWatchView  `json:"huts"`
}
