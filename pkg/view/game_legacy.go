package view

type Legacy struct {
	Reforges     int            `json:"reforges,omitempty"`
	Threads      float64        `json:"threads,omitempty"`
	Spent        float64        `json:"spent,omitempty"`
	Patterns     map[string]int `json:"patterns,omitempty"`
	LifetimeGold float64        `json:"lifetimeGold,omitempty"`
	Tapestry     []string       `json:"tapestry,omitempty"`
}

type ReforgeView struct {
	Available    bool          `json:"available"`
	Earn         float64       `json:"earn"`
	Weave        float64       `json:"weave"`
	Nines        int           `json:"nines"`
	Threads      float64       `json:"threads"`
	Reforges     int           `json:"reforges"`
	Patterns     []PatternView `json:"patterns"`
	RunThreads   float64       `json:"runThreads"`
	NextThreadAt string        `json:"nextThreadAt"`
	Spent        float64       `json:"spent"`
	Bonus        float64       `json:"bonus"`
}

type IncidentView struct {
	Calamity string `json:"calamity"`
	K8s      string `json:"k8s"`
}

type MasteryView struct {
	Verbs         []string       `json:"verbs"`
	Incidents     []IncidentView `json:"incidents"`
	Declared      int            `json:"declared"`
	DeclaredKinds []string       `json:"declaredKinds"`
}
