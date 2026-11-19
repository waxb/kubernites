package view

type AutonomyView struct {
	Kind   string  `json:"kind"`
	Actor  string  `json:"actor"`
	Text   string  `json:"text"`
	At     float64 `json:"at"`
	Object string  `json:"object"`
}
