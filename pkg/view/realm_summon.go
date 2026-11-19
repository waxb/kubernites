package view

type SummonView struct {
	Running int  `json:"running"`
	Waiting int  `json:"waiting"`
	Room    int  `json:"room"`
	Full    bool `json:"full,omitempty"`
}
