package view

type IncidentTimeline struct {
	StruckAt  float64 `json:"struckAt"`
	CauseAt   float64 `json:"causeAt,omitempty"`
	FixedAt   float64 `json:"fixedAt,omitempty"`
	BoxesLost float64 `json:"boxesLost"`
}

type IncidentClueView struct {
	ID      string  `json:"id"`
	Text    string  `json:"text"`
	K8s     string  `json:"k8s"`
	Found   bool    `json:"found"`
	FoundAt float64 `json:"foundAt,omitempty"`
}

type IncidentReport struct {
	ID        string             `json:"id"`
	Kind      string             `json:"kind"`
	Name      string             `json:"name"`
	K8s       string             `json:"k8s"`
	Symptom   string             `json:"symptom"`
	Pod       string             `json:"pod"`
	Crew      string             `json:"crew,omitempty"`
	Node      string             `json:"node,omitempty"`
	Place     string             `json:"place"`
	State     string             `json:"state"`
	Guided    bool               `json:"guided"`
	Step      string             `json:"step,omitempty"`
	StepText  string             `json:"stepText,omitempty"`
	Clues     []IncidentClueView `json:"clues"`
	Fix       string             `json:"fix,omitempty"`
	FixText   string             `json:"fixText"`
	Timeline  IncidentTimeline   `json:"timeline"`
	Chronicle string             `json:"chronicle,omitempty"`
	Ravens    bool               `json:"ravens,omitempty"`
}
