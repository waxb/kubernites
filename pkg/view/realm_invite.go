package view

type Invitation struct {
	Label  string  `json:"label"`
	Page   string  `json:"page"`
	Rank   string  `json:"rank"`
	Gold   float64 `json:"gold"`
	Wisdom float64 `json:"wisdom"`
}
