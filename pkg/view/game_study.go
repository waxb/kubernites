package view

type StudyView struct {
	Every     float64 `json:"every"`
	Progress  float64 `json:"progress"`
	Earned    int     `json:"earned"`
	Objective float64 `json:"objective"`
}
