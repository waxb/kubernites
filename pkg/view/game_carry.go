package view

type CarryFacts struct {
	Known    bool    `json:"known,omitempty"`
	Room     float64 `json:"room,omitempty"`
	Strained bool    `json:"strained,omitempty"`
	Crew     string  `json:"crew,omitempty"`
	Scale    int     `json:"scale,omitempty"`
	Keeper   string  `json:"keeper,omitempty"`
}

func (f CarryFacts) Scales() bool {
	return f.Known && f.Crew != "" && f.Scale > 0
}

func (f CarryFacts) Command() string {
	return ScaleCommandOf(f.Keeper, f.Crew, f.Scale)
}
