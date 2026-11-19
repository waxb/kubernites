package view

type Cohort struct {
	At    float64 `json:"at"`
	Count float64 `json:"count"`
	TTL   float64 `json:"ttl"`
}

type CohortView struct {
	Count     float64 `json:"count"`
	Age       float64 `json:"age"`
	Remaining float64 `json:"remaining"`
	TTL       float64 `json:"ttl"`
	Sealed    bool    `json:"sealed,omitempty"`
	Crowned   bool    `json:"crowned,omitempty"`
}
