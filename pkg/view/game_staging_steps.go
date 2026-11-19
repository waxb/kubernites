package view

type ShrinkFact struct {
	Crew string `json:"crew,omitempty"`
	From int    `json:"from,omitempty"`
	To   int    `json:"to,omitempty"`
	Need int    `json:"need,omitempty"`
}
