package view

type PatternView struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Tier        int     `json:"tier"`
	Rank        int     `json:"rank"`
	MaxRank     int     `json:"maxRank"`
	Repeatable  bool    `json:"repeatable,omitempty"`
	Cost        float64 `json:"cost"`
	Affordable  bool    `json:"affordable"`
	Locked      bool    `json:"locked,omitempty"`
	OpensAt     float64 `json:"opensAt,omitempty"`
}
