package view

type CharterView struct {
	Cost      float64 `json:"cost"`
	Available bool    `json:"available"`
	Reason    string  `json:"reason,omitempty"`
}
