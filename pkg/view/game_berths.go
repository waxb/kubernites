package view

type BerthView struct {
	Open       int     `json:"open"`
	NextCost   float64 `json:"nextCost"`
	Multiplier float64 `json:"multiplier"`
	Effect     string  `json:"effect"`
	Gate       string  `json:"gate,omitempty"`
}
